package service

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type conditionalAccessGoogleDeleteRequest struct{}

type conditionalAccessGoogleDeleteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r conditionalAccessGoogleDeleteResponse) Error() error { return r.Err }

func conditionalAccessGoogleDeleteEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	_ = request.(*conditionalAccessGoogleDeleteRequest)
	if err := svc.ConditionalAccessGoogleDelete(ctx); err != nil {
		return conditionalAccessGoogleDeleteResponse{Err: err}, nil
	}
	return conditionalAccessGoogleDeleteResponse{}, nil
}

// ConditionalAccessGoogleDelete disconnects Google conditional access, the same way
// setting integrations.google_conditional_access to [] does.
func (svc *Service) ConditionalAccessGoogleDelete(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessGoogleIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}

	integration, err := svc.ds.ConditionalAccessGoogleGet(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{Message: "integration not found"}
		}
		return ctxerr.Wrap(ctx, err, "failed to load the integration")
	}
	if err := svc.disconnectGoogleConditionalAccess(ctx, integration); err != nil {
		return err
	}

	// Clear the settings too, so GET config and GitOps agree with the deletion.
	appConfig, err := svc.ds.AppConfig(ctxdb.BypassCachedMysql(ctx, true))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config")
	}
	if appConfig.Integrations.GoogleConditionalAccess != nil {
		appConfig.Integrations.GoogleConditionalAccess = nil
		if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
			return ctxerr.Wrap(ctx, err, "save app config")
		}
	}
	return nil
}

// ConditionalAccessGoogleGet returns the integration for the app config mirror fields,
// or nil when there is none or the license isn't Premium.
func (svc *Service) ConditionalAccessGoogleGet(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return nil, nil
	}
	integration, err := svc.ds.ConditionalAccessGoogleGet(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access integration")
	}
	return integration, nil
}

// googleConditionalAccessKey returns the stored service account key, or nil if none is stored.
func (svc *Service) googleConditionalAccessKey(ctx context.Context) ([]byte, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}, nil)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get google conditional access service account key")
	}
	return assets[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey].Value, nil
}

// googleConditionalAccessUpdate is what saving integrations.google_conditional_access requires.
type googleConditionalAccessUpdate struct {
	// existing is the connection state before the update, nil if not connected.
	existing *fleet.ConditionalAccessGoogleIntegration
	// remove is set when the setting was cleared.
	remove bool

	entry fleet.GoogleConditionalAccessIntegration
	// key is the service account key to verify with: the incoming one, or the stored one.
	key []byte
	// storeKey is set when key differs from the stored key.
	storeKey bool
	// previousKey is the stored key before the update, nil if none, restored when the update fails.
	previousKey []byte
	// verify is set when the settings changed, which requires a delegation check.
	verify bool
	// register is set when the customer ID needs registering with fleetdm.com.
	register bool
}

// planGoogleConditionalAccessUpdate validates the incoming google_conditional_access
// setting (already merged into appConfig) and works out what saving it requires.
// It strips the key from appConfig, which is never saved with it. A nil result
// means there's nothing to do.
func (svc *Service) planGoogleConditionalAccessUpdate(
	ctx context.Context,
	incoming, old []*fleet.GoogleConditionalAccessIntegration,
	appConfig *fleet.AppConfig,
	lic *fleet.LicenseInfo,
	invalid *fleet.InvalidArgumentError,
) (*googleConditionalAccessUpdate, error) {
	// null (or absent) keeps the existing setting.
	if incoming == nil {
		appConfig.Integrations.GoogleConditionalAccess = old
		return nil, nil
	}
	if len(incoming) > 0 && !lic.IsPremium() {
		invalid.Append("integrations.google_conditional_access", ErrMissingLicense.Error())
		return nil, nil
	}
	// GitOps sends [] on every run when the integration isn't in YAML.
	if len(incoming) == 0 && len(old) == 0 {
		return nil, nil
	}

	existing, err := svc.ds.ConditionalAccessGoogleGet(ctx)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access integration")
	}
	storedKey, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return nil, err
	}

	merged := appConfig.Integrations.GoogleConditionalAccess
	if len(merged) == 0 {
		if existing == nil && storedKey == nil {
			return nil, nil
		}
		return &googleConditionalAccessUpdate{existing: existing, remove: true}, nil
	}

	// The merged entries reuse the stored ones, so the key only comes from the request.
	for i := range merged {
		merged[i].ApiKey = fleet.GoogleCalendarApiKey{}
		if i < len(incoming) {
			merged[i].ApiKey = incoming[i].ApiKey
		}
	}
	fleet.ValidateGoogleConditionalAccessIntegrations(merged, storedKey != nil, invalid)
	if invalid.HasErrors() {
		return nil, nil
	}

	update := &googleConditionalAccessUpdate{existing: existing, key: storedKey, previousKey: storedKey}
	if apiKey := merged[0].ApiKey; !apiKey.IsEmpty() && !apiKey.IsMasked() {
		if svc.config.Server.PrivateKey == "" {
			invalid.Append("integrations.google_conditional_access",
				"Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key")
			return nil, nil
		}
		newKey, err := json.Marshal(apiKey.Values)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "marshal google conditional access service account key")
		}
		update.storeKey = !bytes.Equal(newKey, storedKey)
		update.key = newKey
	}
	for _, intg := range merged {
		intg.ApiKey = fleet.GoogleCalendarApiKey{}
	}

	update.entry = *merged[0]
	update.register = existing == nil || existing.CustomerID != update.entry.CustomerID
	settingsChanged := len(old) == 0 ||
		old[0].CustomerID != update.entry.CustomerID ||
		old[0].AdminEmail != update.entry.AdminEmail
	// GitOps resends the same settings and key on every run, which must not contact Google or fleetdm.com.
	update.verify = update.register || update.storeKey || settingsChanged
	return update, nil
}

// verifyGoogleConditionalAccess checks the delegation with Google when the settings changed.
func (svc *Service) verifyGoogleConditionalAccess(ctx context.Context, update *googleConditionalAccessUpdate) error {
	if update == nil || update.remove || !update.verify {
		return nil
	}
	client, err := svc.conditionalAccessGoogleClientFactory(ctx, update.key, update.entry.AdminEmail, update.entry.CustomerID)
	if err == nil {
		err = client.CheckDelegation(ctx)
	}
	if err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("integrations.google_conditional_access", "Couldn't connect to Google: "+err.Error()))
	}
	return nil
}

// applyGoogleConditionalAccess connects, re-registers, or disconnects Google conditional access as planned.
func (svc *Service) applyGoogleConditionalAccess(ctx context.Context, update *googleConditionalAccessUpdate) error {
	switch {
	case update == nil:
		return nil
	case update.remove:
		return svc.disconnectGoogleConditionalAccess(ctx, update.existing)
	case !update.verify:
		return nil
	}
	if svc.conditionalAccessGoogleProxy == nil {
		return ctxerr.New(ctx, "google conditional access proxy is not configured")
	}

	storeKey := func() error {
		if !update.storeKey {
			return nil
		}
		if err := svc.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
			Name:  fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
			Value: update.key,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "store google conditional access service account key")
		}
		return nil
	}
	if !update.register {
		return storeKey()
	}

	// Register the new customer ID before touching the stored key, the row, or the old
	// registration, so a failure leaves the previous integration working.
	created, err := svc.conditionalAccessGoogleProxy.Create(ctx, update.entry.CustomerID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "register with fleetdm.com")
	}
	if err := storeKey(); err != nil {
		svc.rollBackGoogleConditionalAccessRegistration(ctx, update.entry.CustomerID, created.FleetServerSecret)
		return err
	}
	if err := svc.ds.ConditionalAccessGoogleCreateIntegration(ctx, update.entry.CustomerID, created.FleetServerSecret, created.OAuthClientID); err != nil {
		// The previous row is untouched, so it must keep the key it was verified with.
		if update.storeKey {
			svc.restoreGoogleConditionalAccessKey(ctx, update.previousKey)
		}
		svc.rollBackGoogleConditionalAccessRegistration(ctx, update.entry.CustomerID, created.FleetServerSecret)
		return ctxerr.Wrap(ctx, err, "store google conditional access integration")
	}
	if update.existing != nil {
		// The new integration is in place, so a stale registration on fleetdm.com is only logged.
		if err := svc.deleteGoogleConditionalAccessRegistration(ctx, update.existing); err != nil {
			svc.logger.ErrorContext(ctx, "failed to delete previous google conditional access registration",
				"customer_id", update.existing.CustomerID, "err", err)
		}
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for google conditional access integration")
	}
	return nil
}

// rollBackGoogleConditionalAccessRegistration deletes a registration this save created, best effort,
// so retrying the save doesn't find an orphaned registration on fleetdm.com.
func (svc *Service) rollBackGoogleConditionalAccessRegistration(ctx context.Context, customerID, secret string) {
	if err := svc.conditionalAccessGoogleProxy.Delete(ctx, customerID, secret); err != nil {
		svc.logger.ErrorContext(ctx, "failed to roll back google conditional access registration", "err", err)
	}
}

// restoreGoogleConditionalAccessKey puts back the key stored before a failed save, best effort.
func (svc *Service) restoreGoogleConditionalAccessKey(ctx context.Context, previousKey []byte) {
	var err error
	if previousKey == nil {
		err = svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey})
	} else {
		err = svc.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
			Name:  fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
			Value: previousKey,
		})
	}
	if err != nil {
		svc.logger.ErrorContext(ctx, "failed to restore google conditional access service account key", "err", err)
	}
}

// disconnectGoogleConditionalAccess deregisters from fleetdm.com and deletes the connection state and
// the key. It doesn't unmark devices in Google.
func (svc *Service) disconnectGoogleConditionalAccess(ctx context.Context, existing *fleet.ConditionalAccessGoogleIntegration) error {
	if existing != nil {
		if err := svc.deleteGoogleConditionalAccessRegistration(ctx, existing); err != nil {
			return err
		}
		if err := svc.ds.ConditionalAccessGoogleDelete(ctx); err != nil {
			return ctxerr.Wrap(ctx, err, "delete google conditional access integration")
		}
	}
	if err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "delete google conditional access service account key")
	}
	if existing == nil {
		return nil
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
	}
	return nil
}

func (svc *Service) deleteGoogleConditionalAccessRegistration(ctx context.Context, existing *fleet.ConditionalAccessGoogleIntegration) error {
	if svc.conditionalAccessGoogleProxy == nil {
		return ctxerr.New(ctx, "google conditional access proxy is not configured")
	}
	if err := svc.conditionalAccessGoogleProxy.Delete(ctx, existing.CustomerID, existing.ProxyServerSecret); err != nil {
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "delete registration on fleetdm.com")
		}
		// A record missing on fleetdm.com must not block removal in Fleet.
		svc.logger.WarnContext(ctx, "google conditional access registration not found on fleetdm.com, continuing")
	}
	return nil
}
