package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
)

const missingServerPrivateKeyMessage = "Missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key"

////////////////////////////////////////////////////////////////////////////////
// POST /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

type conditionalAccessGoogleConnectRequest struct {
	CustomerID        string          `json:"customer_id"`
	AdminEmail        string          `json:"admin_email"`
	ServiceAccountKey json.RawMessage `json:"service_account_key"`
}

type conditionalAccessGoogleStatusResponse struct {
	CustomerID string `json:"customer_id"`
	AdminEmail string `json:"admin_email"`
	fleet.ConditionalAccessGoogleStatus
	Err error `json:"error,omitempty"`
}

func (r conditionalAccessGoogleStatusResponse) Error() error { return r.Err }

func newConditionalAccessGoogleStatusResponse(
	integration *fleet.ConditionalAccessGoogleIntegration, status *fleet.ConditionalAccessGoogleStatus,
) conditionalAccessGoogleStatusResponse {
	var resp conditionalAccessGoogleStatusResponse
	if integration != nil {
		resp.CustomerID = integration.CustomerID
		resp.AdminEmail = integration.AdminEmail
	}
	if status != nil {
		resp.ConditionalAccessGoogleStatus = *status
	}
	return resp
}

func conditionalAccessGoogleConnectEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*conditionalAccessGoogleConnectRequest)
	if err := svc.ConditionalAccessGoogleConnect(ctx, req.CustomerID, req.AdminEmail, req.ServiceAccountKey); err != nil {
		return conditionalAccessGoogleStatusResponse{Err: err}, nil
	}
	// Read back from the primary so replica lag doesn't show the state from before the connect.
	integration, status, err := svc.ConditionalAccessGoogleStatus(ctxdb.RequirePrimary(ctx, true))
	if err != nil {
		return conditionalAccessGoogleStatusResponse{Err: err}, nil
	}
	return newConditionalAccessGoogleStatusResponse(integration, status), nil
}

func (svc *Service) ConditionalAccessGoogleConnect(ctx context.Context, customerID, adminEmail string, serviceAccountKey json.RawMessage) error {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessGoogleIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}
	if svc.googleConditionalAccess == nil {
		return ctxerr.New(ctx, "google conditional access client is not configured")
	}

	settings := fleet.GoogleConditionalAccessIntegration{Enabled: true, CustomerID: customerID, AdminEmail: adminEmail}
	invalid := &fleet.InvalidArgumentError{}
	fleet.ValidateGoogleConditionalAccessSettings(&settings, "", invalid)
	fleet.ValidateGoogleServiceAccountKey(serviceAccountKey, "service_account_key", invalid)
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}
	if svc.config.Server.PrivateKey == "" || testSetEmptyPrivateKey {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("service_account_key", missingServerPrivateKeyMessage))
	}

	ctx = ctxdb.RequirePrimary(ctx, true)
	appConfig, err := svc.ds.AppConfig(ctxdb.BypassCachedMysql(ctx, true))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config")
	}
	existing, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return err
	}
	storedKey, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return err
	}
	keyChanged := !bytes.Equal(storedKey, serviceAccountKey)
	if !keyChanged && googleConditionalAccessConnected(existing, settings) {
		return svc.saveGoogleConditionalAccessSettings(ctx, appConfig, settings)
	}

	if err := svc.verifyGoogleConditionalAccess(ctx, settings, serviceAccountKey, "service_account_key"); err != nil {
		return err
	}
	// The previous customer's devices can only be unmarked with the key stored for it,
	// so it is left before the key is replaced.
	if existing != nil && existing.CustomerID != settings.CustomerID {
		if err := svc.leaveGoogleConditionalAccessCustomer(ctx, appConfig.ServerSettings.ServerURL, settings, existing, storedKey); err != nil {
			return err
		}
		existing = nil
	}
	// The key and the setting are saved before connecting: if registering fails, the
	// integration is on with a stored key and no completed row, which the next config
	// update retries.
	if keyChanged {
		// Replace rather than compare-and-replace: a stored key that can't be decrypted
		// must still be replaceable.
		if err := svc.ds.ReplaceMDMConfigAssets(ctx, []fleet.MDMConfigAsset{{
			Name:  fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
			Value: serviceAccountKey,
		}}, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "store google conditional access service account key")
		}
	}
	if err := svc.saveGoogleConditionalAccessSettings(ctx, appConfig, settings); err != nil {
		return err
	}
	return svc.connectGoogleConditionalAccess(ctx, appConfig.ServerSettings.ServerURL, settings, existing)
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

func conditionalAccessGoogleGetEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	integration, status, err := svc.ConditionalAccessGoogleStatus(ctx)
	if err != nil {
		return conditionalAccessGoogleStatusResponse{Err: err}, nil
	}
	return newConditionalAccessGoogleStatusResponse(integration, status), nil
}

// ConditionalAccessGoogleStatus returns the connection row (nil when none) and its
// status. Both are nil on Fleet Free.
func (svc *Service) ConditionalAccessGoogleStatus(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, *fleet.ConditionalAccessGoogleStatus, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return nil, nil, nil
	}
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "load app config")
	}
	status := &fleet.ConditionalAccessGoogleStatus{}
	// The connection state and the key only exist while the integration is on.
	if !googleConditionalAccessEnabled(appConfig.Integrations.GoogleConditionalAccess) {
		return nil, status, nil
	}

	integration, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return nil, nil, err
	}
	keyStored, err := svc.googleConditionalAccessKeyStored(ctx)
	if err != nil {
		return nil, nil, err
	}
	status.ServiceAccountKeyConfigured = keyStored
	if integration != nil {
		status.State = integration.State
		status.ErrorMessage = integration.ErrorMessage
	}
	return integration, status, nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

type conditionalAccessGoogleDeleteResponse struct {
	Err error `json:"error,omitempty"`
}

func (r conditionalAccessGoogleDeleteResponse) Error() error { return r.Err }

func conditionalAccessGoogleDeleteEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	if err := svc.ConditionalAccessGoogleDelete(ctx); err != nil {
		return conditionalAccessGoogleDeleteResponse{Err: err}, nil
	}
	return conditionalAccessGoogleDeleteResponse{}, nil
}

func (svc *Service) ConditionalAccessGoogleDelete(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessGoogleIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}
	if svc.googleConditionalAccess == nil {
		return ctxerr.New(ctx, "google conditional access client is not configured")
	}

	ctx = ctxdb.RequirePrimary(ctx, true)
	existing, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return err
	}
	keyStored, err := svc.googleConditionalAccessKeyStored(ctx)
	if err != nil {
		return err
	}
	appConfig, err := svc.ds.AppConfig(ctxdb.BypassCachedMysql(ctx, true))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config")
	}
	enabled := googleConditionalAccessEnabled(appConfig.Integrations.GoogleConditionalAccess)
	if existing == nil && !keyStored && !enabled {
		return &fleet.BadRequestError{Message: "integration not found"}
	}
	if err := svc.removeGoogleConditionalAccess(ctx, appConfig.ServerSettings.ServerURL, existing); err != nil {
		return err
	}

	if enabled {
		gca := appConfig.Integrations.GoogleConditionalAccess
		gca.Enabled = false
		if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
			return ctxerr.Wrap(ctx, err, "save app config")
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Shared by the endpoints above.
////////////////////////////////////////////////////////////////////////////////

// GoogleConditionalAccessClient holds the clients of the Google conditional access
// integration: the fleetdm.com relay and the per-customer Cloud Identity factory.
// The service methods check it is set before the helpers below use it.
type GoogleConditionalAccessClient struct {
	Relay         conditional_access_google.Relay
	CloudIdentity conditional_access_google.CloudIdentityFactory
}

// NewGoogleConditionalAccessClient builds the production client for the fleetdm.com
// relay at relayURI. Dev mode accepts a plain HTTP relay for the mock relay.
func NewGoogleConditionalAccessClient(relayURI string, logger *slog.Logger) *GoogleConditionalAccessClient {
	var opts []conditional_access_google.RelayOption
	if dev_mode.IsEnabled {
		opts = append(opts, conditional_access_google.AllowHTTP())
	}
	return &GoogleConditionalAccessClient{
		Relay:         conditional_access_google.NewRelay(relayURI, opts...),
		CloudIdentity: conditional_access_google.NewCloudIdentityFactory(logger),
	}
}

func googleConditionalAccessEnabled(settings *fleet.GoogleConditionalAccessIntegration) bool {
	return settings != nil && settings.Enabled
}

func googleConditionalAccessConnected(existing *fleet.ConditionalAccessGoogleIntegration, settings fleet.GoogleConditionalAccessIntegration) bool {
	return existing != nil && existing.State == fleet.ConditionalAccessGoogleStateConnected &&
		existing.CustomerID == settings.CustomerID && existing.AdminEmail == settings.AdminEmail
}

// checkGoogleDelegation runs the delegation check. The error message is meant for the admin.
func (svc *Service) checkGoogleDelegation(ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, key []byte) error {
	creds := conditional_access_google.Credentials{ServiceAccountKey: key, AdminEmail: settings.AdminEmail, CustomerID: settings.CustomerID}
	client, err := svc.googleConditionalAccess.CloudIdentity(ctx, creds)
	if err != nil {
		return errors.New("Couldn't connect to Google: " + err.Error())
	}
	if err := client.CheckDelegation(ctx); err != nil {
		return errors.New("Couldn't connect to Google: " + err.Error())
	}
	return nil
}

// verifyGoogleConditionalAccess runs the delegation check and returns a 422 with
// Google's reason when it fails.
func (svc *Service) verifyGoogleConditionalAccess(ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, key []byte, field string) error {
	if err := svc.checkGoogleDelegation(ctx, settings, key); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(field, err.Error()))
	}
	return nil
}

// newGoogleConditionalAccessRow builds the connection row for settings, carrying the
// registration over from existing when it is for the same customer.
func newGoogleConditionalAccessRow(
	settings fleet.GoogleConditionalAccessIntegration, existing *fleet.ConditionalAccessGoogleIntegration,
	state fleet.ConditionalAccessGoogleState, errorMessage *string,
) *fleet.ConditionalAccessGoogleIntegration {
	row := &fleet.ConditionalAccessGoogleIntegration{
		CustomerID:   settings.CustomerID,
		AdminEmail:   settings.AdminEmail,
		State:        state,
		ErrorMessage: errorMessage,
	}
	if existing != nil && existing.Registered() && existing.CustomerID == settings.CustomerID {
		row.ProxyServerSecret = existing.ProxyServerSecret
		row.OAuthClientID = existing.OAuthClientID
	}
	return row
}

func (svc *Service) saveGoogleConditionalAccessRow(
	ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, existing *fleet.ConditionalAccessGoogleIntegration,
	state fleet.ConditionalAccessGoogleState, errorMessage *string,
) error {
	if err := svc.ds.SaveGoogleConditionalAccessIntegration(ctx, newGoogleConditionalAccessRow(settings, existing, state, errorMessage)); err != nil {
		return ctxerr.Wrap(ctx, err, "save google conditional access integration")
	}
	return nil
}

// connectGoogleConditionalAccess registers the server with fleetdm.com when it isn't
// registered for this customer yet, and stores the connection row. The delegation
// check must have passed and a previous customer must have been left (see
// leaveGoogleConditionalAccessCustomer). A failed registration is recorded in the row.
// serverURL is the app config's server URL.
func (svc *Service) connectGoogleConditionalAccess(
	ctx context.Context, serverURL string, settings fleet.GoogleConditionalAccessIntegration,
	existing *fleet.ConditionalAccessGoogleIntegration,
) error {
	if existing != nil && existing.Registered() && existing.CustomerID == settings.CustomerID {
		return svc.saveGoogleConditionalAccessRow(ctx, settings, existing, fleet.ConditionalAccessGoogleStateConnected, nil)
	}

	if err := svc.saveGoogleConditionalAccessRow(ctx, settings, nil, fleet.ConditionalAccessGoogleStatePending, nil); err != nil {
		return err
	}

	resp, err := svc.googleConditionalAccess.Relay.Register(ctx, serverURL, settings.CustomerID)
	if err != nil {
		gwErr := newFleetdmRelayError("Couldn't register with fleetdm.com", err)
		reason := gwErr.Error()
		if saveErr := svc.saveGoogleConditionalAccessRow(ctx, settings, nil, fleet.ConditionalAccessGoogleStateRegistrationFailed, &reason); saveErr != nil {
			svc.logger.ErrorContext(ctx, "record google conditional access registration error", "err", saveErr)
		}
		return ctxerr.Wrap(ctx, gwErr, "register with fleetdm.com")
	}
	row := newGoogleConditionalAccessRow(settings, nil, fleet.ConditionalAccessGoogleStateConnected, nil)
	row.ProxyServerSecret = resp.FleetServerSecret
	row.OAuthClientID = resp.OAuthClientID
	if err := svc.ds.SaveGoogleConditionalAccessIntegration(ctx, row); err != nil {
		// Without its saved secret the registration could never be removed, so drop it now.
		if derr := svc.googleConditionalAccess.Relay.Deregister(ctx, serverURL, settings.CustomerID, resp.FleetServerSecret); derr != nil {
			svc.logger.ErrorContext(ctx, "drop google conditional access registration that couldn't be saved", "err", derr)
		}
		return ctxerr.Wrap(ctx, err, "save google conditional access integration")
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for google conditional access integration")
	}
	return nil
}

// removeGoogleConditionalAccess unmarks every stored DeviceUser name (best effort),
// deregisters from fleetdm.com, and deletes the rows and the key. serverURL is the app
// config's server URL.
func (svc *Service) removeGoogleConditionalAccess(
	ctx context.Context, serverURL string, existing *fleet.ConditionalAccessGoogleIntegration,
) error {
	if existing != nil {
		key, err := svc.googleConditionalAccessKey(ctx)
		if err != nil {
			return err
		}
		if err := svc.unmarkGoogleConditionalAccessDevices(ctx, existing, key); err != nil {
			return err
		}
		if existing.Registered() {
			if err := svc.deleteGoogleConditionalAccessRegistration(ctx, serverURL, existing); err != nil {
				return err
			}
		}
	}
	if err := svc.ds.DeleteGoogleConditionalAccessIntegration(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "delete google conditional access integration")
	}
	if err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}); err != nil {
		return ctxerr.Wrap(ctx, err, "delete google conditional access service account key")
	}
	if existing != nil && existing.Registered() {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
		}
	}
	return nil
}

// leaveGoogleConditionalAccessCustomer drops the previous customer before a new one is
// connected. Its fleetdm.com registration goes first, so a failure changes nothing and
// keeps the old secret for the retry. Its devices are then unmarked (best effort) with
// the key stored for it, and their rows deleted.
func (svc *Service) leaveGoogleConditionalAccessCustomer(
	ctx context.Context, serverURL string, settings fleet.GoogleConditionalAccessIntegration,
	existing *fleet.ConditionalAccessGoogleIntegration, key []byte,
) error {
	if existing.Registered() {
		if err := svc.deleteGoogleConditionalAccessRegistration(ctx, serverURL, existing); err != nil {
			return err
		}
		// The secret is no longer valid on fleetdm.com, so the row must not keep it.
		if err := svc.saveGoogleConditionalAccessRow(ctx, settings, nil, fleet.ConditionalAccessGoogleStatePending, nil); err != nil {
			return err
		}
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
		}
	}
	if err := svc.unmarkGoogleConditionalAccessDevices(ctx, existing, key); err != nil {
		return err
	}
	if err := svc.ds.DeleteGoogleConditionalAccessDevices(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "delete google devices")
	}
	return nil
}

func (svc *Service) unmarkGoogleConditionalAccessDevices(
	ctx context.Context, existing *fleet.ConditionalAccessGoogleIntegration, key []byte,
) error {
	names, err := svc.ds.ListGoogleConditionalAccessDeviceUserNames(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list google device user names")
	}
	if len(names) == 0 {
		return nil
	}
	if len(key) == 0 {
		svc.logger.WarnContext(ctx, "no google service account key, skipping unmarking devices", "device_users", len(names))
		return nil
	}
	creds := conditional_access_google.Credentials{ServiceAccountKey: key, AdminEmail: existing.AdminEmail, CustomerID: existing.CustomerID}
	client, err := svc.googleConditionalAccess.CloudIdentity(ctx, creds)
	if err != nil {
		svc.logger.ErrorContext(ctx, "build google cloud identity client, skipping unmarking devices", "device_users", len(names), "err", err)
		return nil
	}
	for _, name := range names {
		if err := client.SetUnmanaged(ctx, name); err != nil {
			if fleet.IsNotFound(err) {
				svc.logger.DebugContext(ctx, "google device user already gone", "device_user", name)
				continue
			}
			svc.logger.ErrorContext(ctx, "unmark google device user", "device_user", name, "err", err)
		}
	}
	return nil
}

func (svc *Service) deleteGoogleConditionalAccessRegistration(
	ctx context.Context, serverURL string, existing *fleet.ConditionalAccessGoogleIntegration,
) error {
	if err := svc.googleConditionalAccess.Relay.Deregister(ctx, serverURL, existing.CustomerID, existing.ProxyServerSecret); err != nil {
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, newFleetdmRelayError("Couldn't delete the registration on fleetdm.com", err),
				"delete registration on fleetdm.com")
		}
		// A missing record on fleetdm.com must not block the removal in Fleet.
		svc.logger.WarnContext(ctx, "google conditional access registration not found on fleetdm.com, continuing")
	}
	return nil
}

// newFleetdmRelayError reports a failed fleetdm.com call as a 502. The relay's own
// status must not reach the admin's client: a 401 from fleetdm.com would read as an
// expired Fleet session.
func newFleetdmRelayError(message string, err error) *fleet.GatewayError {
	return fleet.NewBadGatewayError(message, err)
}

// saveGoogleConditionalAccessSettings writes the settings to appConfig and saves it, so
// the config and the connection state agree after a key upload.
func (svc *Service) saveGoogleConditionalAccessSettings(
	ctx context.Context, appConfig *fleet.AppConfig, settings fleet.GoogleConditionalAccessIntegration,
) error {
	if current := appConfig.Integrations.GoogleConditionalAccess; current != nil && *current == settings {
		return nil
	}
	appConfig.Integrations.GoogleConditionalAccess = &settings
	if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
		return ctxerr.Wrap(ctx, err, "save app config")
	}
	return nil
}

// googleConditionalAccessIntegration returns the connection row, or nil when there's none.
func (svc *Service) googleConditionalAccessIntegration(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, error) {
	integration, err := svc.ds.GetGoogleConditionalAccessIntegration(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access integration")
	}
	return integration, nil
}

// googleConditionalAccessKeyStored reports whether a key is stored, without decrypting
// it, so a server private key problem doesn't break reading the config.
func (svc *Service) googleConditionalAccessKeyStored(ctx context.Context) (bool, error) {
	hashes, err := svc.ds.GetAllMDMConfigAssetsHashes(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey})
	if err != nil {
		if fleet.IsNotFound(err) {
			return false, nil
		}
		return false, ctxerr.Wrap(ctx, err, "check google conditional access service account key")
	}
	_, ok := hashes[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey]
	return ok, nil
}

// googleConditionalAccessKey returns the stored service account key, or nil when there's
// none or it can't be decrypted: a damaged key must not stop the admin from replacing or
// removing the integration.
func (svc *Service) googleConditionalAccessKey(ctx context.Context) ([]byte, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}, nil)
	switch {
	case fleet.IsNotFound(err):
		return nil, nil
	case errors.Is(err, fleet.ErrMDMConfigAssetUndecryptable):
		svc.logger.WarnContext(ctx, "google conditional access service account key can't be decrypted, treating it as missing", "err", err)
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access service account key")
	}
	return assets[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey].Value, nil
}
