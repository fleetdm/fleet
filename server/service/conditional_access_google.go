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

type connectGoogleConditionalAccessRequest struct {
	CustomerID        string          `json:"customer_id"`
	AdminEmail        string          `json:"admin_email"`
	ServiceAccountKey json.RawMessage `json:"service_account_key"`
}

type googleConditionalAccessResponse struct {
	CustomerID string `json:"customer_id"`
	AdminEmail string `json:"admin_email"`
	fleet.ConditionalAccessGoogleStatus
	Err error `json:"error,omitempty"`
}

func (r googleConditionalAccessResponse) Error() error { return r.Err }

func newGoogleConditionalAccessResponse(
	settings *fleet.GoogleConditionalAccessIntegration, status *fleet.ConditionalAccessGoogleStatus,
) googleConditionalAccessResponse {
	var resp googleConditionalAccessResponse
	if settings != nil {
		resp.CustomerID = settings.CustomerID
		resp.AdminEmail = settings.AdminEmail
	}
	if status != nil {
		resp.ConditionalAccessGoogleStatus = *status
	}
	return resp
}

func connectGoogleConditionalAccessEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*connectGoogleConditionalAccessRequest)
	if err := svc.ConnectGoogleConditionalAccess(ctx, req.CustomerID, req.AdminEmail, req.ServiceAccountKey); err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	// Read back from the primary so replica lag doesn't show the state from before the connect.
	settings, status, err := svc.GetGoogleConditionalAccess(ctxdb.RequirePrimary(ctx, true))
	if err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	return newGoogleConditionalAccessResponse(settings, status), nil
}

func (svc *Service) ConnectGoogleConditionalAccess(ctx context.Context, customerID, adminEmail string, serviceAccountKey json.RawMessage) error {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessGoogleIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}
	if svc.googleConditionalAccess == nil {
		return ctxerr.New(ctx, "google conditional access client is not configured")
	}

	settings := fleet.GoogleConditionalAccessIntegration{CustomerID: customerID, AdminEmail: adminEmail}
	invalid := &fleet.InvalidArgumentError{}
	settings.Validate(invalid)
	// The delegation check rejects any key Google can't use.
	if len(serviceAccountKey) == 0 || string(serviceAccountKey) == "null" {
		invalid.Append("service_account_key", "service_account_key is required")
	}
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
	storedKey, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return err
	}
	return svc.applyGoogleConditionalAccessSettings(ctx, appConfig, settings, serviceAccountKey, storedKey)
}

// applyGoogleConditionalAccessSettings connects the integration with settings and key,
// calling Google and fleetdm.com only for what differs from the config and storedKey.
func (svc *Service) applyGoogleConditionalAccessSettings(
	ctx context.Context, appConfig *fleet.AppConfig, settings fleet.GoogleConditionalAccessIntegration, key, storedKey []byte,
) error {
	existing, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return err
	}
	// A registration is always for the customer the config names.
	current := appConfig.Integrations.GoogleConditionalAccess
	keyChanged := !bytes.Equal(storedKey, key)
	changed := keyChanged || current == nil || *current != settings
	if !changed && existing != nil && existing.State == fleet.ConditionalAccessGoogleStateConnected {
		return nil
	}

	if err := svc.verifyGoogleConditionalAccess(ctx, settings, key, "service_account_key"); err != nil {
		return err
	}
	// From here fleetdm.com and the stored state change together, so a client that
	// disconnects must not stop it halfway: a registration whose secret isn't saved can't
	// be removed.
	ctx = context.WithoutCancel(ctx)
	wasRegistered := existing.Registered()
	registeredCustomerID := googleConditionalAccessCustomerID(current)
	leftCustomer := wasRegistered && registeredCustomerID != settings.CustomerID
	if leftCustomer {
		if err := svc.leaveGoogleConditionalAccessCustomer(ctx, appConfig.ServerSettings.ServerURL, registeredCustomerID,
			googleConditionalAccessSecret(existing)); err != nil {
			return err
		}
		existing = nil
	}
	// Saved before registering, so a failed registration is retried by the next config
	// update.
	if keyChanged {
		if err := svc.ds.ReplaceMDMConfigAssets(ctx, []fleet.MDMConfigAsset{{
			Name:  fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
			Value: key,
		}}, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "store google conditional access service account key")
		}
	}
	if err := svc.saveGoogleConditionalAccessSettings(ctx, appConfig, settings); err != nil {
		return err
	}
	if err := svc.registerGoogleConditionalAccess(ctx, appConfig.ServerSettings.ServerURL, settings, existing, registeredCustomerID); err != nil {
		if leftCustomer {
			// The previous registration is gone and nothing replaced it; the retry logs the
			// addition.
			deleted := fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}
			if actErr := svc.NewActivity(ctx, authz.UserFromContext(ctx), deleted); actErr != nil {
				svc.logger.ErrorContext(ctx, "create activity for deletion of google conditional access integration", "err", actErr)
			}
		}
		return err
	}
	// The first successful registration logs the addition, and a change to a registered
	// integration logs one edit, even when it replaced the registration.
	var activity fleet.ActivityDetails
	switch {
	case !wasRegistered:
		activity = fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}
	case changed:
		activity = fleet.ActivityTypeEditedConditionalAccessIntegrationGoogle{}
	default:
		return nil
	}
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return ctxerr.Wrapf(ctx, err, "create activity %s", activity.ActivityName())
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// PATCH /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

type updateGoogleConditionalAccessRequest struct {
	CustomerID        *string         `json:"customer_id"`
	AdminEmail        *string         `json:"admin_email"`
	ServiceAccountKey json.RawMessage `json:"service_account_key"`
}

func updateGoogleConditionalAccessEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*updateGoogleConditionalAccessRequest)
	if err := svc.UpdateGoogleConditionalAccess(ctx, req.CustomerID, req.AdminEmail, req.ServiceAccountKey); err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	// Read back from the primary so replica lag doesn't show the state from before the update.
	settings, status, err := svc.GetGoogleConditionalAccess(ctxdb.RequirePrimary(ctx, true))
	if err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	return newGoogleConditionalAccessResponse(settings, status), nil
}

func (svc *Service) UpdateGoogleConditionalAccess(ctx context.Context, customerID, adminEmail *string, serviceAccountKey json.RawMessage) error {
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
	appConfig, err := svc.ds.AppConfig(ctxdb.BypassCachedMysql(ctx, true))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load app config")
	}
	current := appConfig.Integrations.GoogleConditionalAccess
	if current == nil {
		return &fleet.BadRequestError{Message: "integration not found"}
	}

	settings := *current
	if customerID != nil {
		settings.CustomerID = *customerID
	}
	if adminEmail != nil {
		settings.AdminEmail = *adminEmail
	}
	invalid := &fleet.InvalidArgumentError{}
	settings.Validate(invalid)
	if invalid.HasErrors() {
		return ctxerr.Wrap(ctx, invalid)
	}
	if svc.config.Server.PrivateKey == "" || testSetEmptyPrivateKey {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("service_account_key", missingServerPrivateKeyMessage))
	}

	storedKey, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return err
	}
	key := []byte(serviceAccountKey)
	if len(key) == 0 || string(key) == "null" {
		key = storedKey
	}
	if len(key) == 0 {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("service_account_key",
			"service_account_key is required because no service account key is stored"))
	}
	return svc.applyGoogleConditionalAccessSettings(ctx, appConfig, settings, key, storedKey)
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

func getGoogleConditionalAccessEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	settings, status, err := svc.GetGoogleConditionalAccess(ctx)
	if err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	return newGoogleConditionalAccessResponse(settings, status), nil
}

func (svc *Service) GetGoogleConditionalAccess(ctx context.Context) (*fleet.GoogleConditionalAccessIntegration, *fleet.ConditionalAccessGoogleStatus, error) {
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
	settings := appConfig.Integrations.GoogleConditionalAccess
	// The connection state and the key only exist while the integration is configured.
	if settings == nil {
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
	return settings, status, nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

type deleteGoogleConditionalAccessResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteGoogleConditionalAccessResponse) Error() error { return r.Err }

func deleteGoogleConditionalAccessEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	if err := svc.DeleteGoogleConditionalAccess(ctx); err != nil {
		return deleteGoogleConditionalAccessResponse{Err: err}, nil
	}
	return deleteGoogleConditionalAccessResponse{}, nil
}

func (svc *Service) DeleteGoogleConditionalAccess(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.ConditionalAccessGoogleIntegration{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "failed to authorize")
	}
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
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
	current := appConfig.Integrations.GoogleConditionalAccess
	active := existing != nil && existing.State != fleet.ConditionalAccessGoogleStateDeleted
	if !active && !keyStored && current == nil {
		return &fleet.BadRequestError{Message: "integration not found"}
	}

	// A client that disconnects must not stop the removal halfway, which would leave
	// Fleet with a registration fleetdm.com no longer has.
	ctx = context.WithoutCancel(ctx)

	// fleetdm.com goes first, so a failure changes nothing and frees the customer for
	// another server right away. Nothing is unmarked in Google, and the device rows are
	// kept for a re-add for the same customer. The setting is cleared last, so a failed
	// DELETE can be retried.
	registered := existing.Registered()
	if registered {
		if err := svc.deleteGoogleConditionalAccessRegistration(ctx, appConfig.ServerSettings.ServerURL,
			googleConditionalAccessCustomerID(current), googleConditionalAccessSecret(existing)); err != nil {
			return err
		}
	}
	if active {
		if err := svc.saveGoogleConditionalAccessRow(ctx, nil, fleet.ConditionalAccessGoogleStateDeleted, nil); err != nil {
			return err
		}
		if registered {
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
			}
		}
	}
	if keyStored {
		if err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}); err != nil {
			return ctxerr.Wrap(ctx, err, "delete google conditional access service account key")
		}
	}
	if current != nil {
		appConfig.Integrations.GoogleConditionalAccess = nil
		if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
			return ctxerr.Wrap(ctx, err, "save app config")
		}
	}
	return nil
}

// GoogleConditionalAccessClient holds the fleetdm.com relay and the Cloud Identity
// client factory.
type GoogleConditionalAccessClient struct {
	Relay         conditional_access_google.Relay
	CloudIdentity conditional_access_google.CloudIdentityFactory
}

// NewGoogleConditionalAccessClient accepts a plain HTTP relay in dev mode, for the
// mock relay.
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

func googleConditionalAccessSecret(existing *fleet.ConditionalAccessGoogleIntegration) string {
	if existing == nil {
		return ""
	}
	return existing.ProxyServerSecret
}

// googleConditionalAccessCustomerID is the customer a stored registration belongs to:
// the one the config names.
func googleConditionalAccessCustomerID(settings *fleet.GoogleConditionalAccessIntegration) string {
	if settings == nil {
		return ""
	}
	return settings.CustomerID
}

// checkGoogleDelegation's error message is shown to the admin.
func (svc *Service) checkGoogleDelegation(ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, key []byte) error {
	creds := conditional_access_google.Credentials{ServiceAccountKey: key, AdminEmail: settings.AdminEmail, CustomerID: settings.CustomerID}
	client, err := svc.googleConditionalAccess.CloudIdentity(ctx, creds)
	if err != nil {
		return errors.New("Couldn't connect to Google: " + err.Error())
	}
	if err := client.CheckDelegation(ctx); err != nil {
		// An outage says nothing about the settings, so it isn't reported against them.
		if conditional_access_google.IsUnavailable(err) {
			return fleet.NewBadGatewayError("Couldn't verify the settings with Google", err)
		}
		return errors.New("Couldn't connect to Google: " + err.Error())
	}
	return nil
}

func (svc *Service) verifyGoogleConditionalAccess(ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, key []byte, field string) error {
	if err := svc.checkGoogleDelegation(ctx, settings, key); err != nil {
		if gwErr, ok := errors.AsType[*fleet.GatewayError](err); ok {
			return ctxerr.Wrap(ctx, gwErr, "check google delegation")
		}
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(field, err.Error()))
	}
	return nil
}

// saveGoogleConditionalAccessRow saves the connection state, keeping the registration of
// registration (nil to drop it).
func (svc *Service) saveGoogleConditionalAccessRow(
	ctx context.Context, registration *fleet.ConditionalAccessGoogleIntegration,
	state fleet.ConditionalAccessGoogleState, errorMessage *string,
) error {
	row := &fleet.ConditionalAccessGoogleIntegration{State: state, ErrorMessage: errorMessage}
	if registration != nil {
		row.ProxyServerSecret = registration.ProxyServerSecret
	}
	if err := svc.ds.SaveGoogleConditionalAccessIntegration(ctx, row); err != nil {
		return ctxerr.Wrap(ctx, err, "save google conditional access integration")
	}
	return nil
}

// registerGoogleConditionalAccess registers with fleetdm.com unless existing is already
// registered for this customer (registeredCustomerID), and saves the row. Callers must
// have run the delegation check and left any previous customer.
func (svc *Service) registerGoogleConditionalAccess(
	ctx context.Context, serverURL string, settings fleet.GoogleConditionalAccessIntegration,
	existing *fleet.ConditionalAccessGoogleIntegration, registeredCustomerID string,
) error {
	if existing.Registered() && registeredCustomerID == settings.CustomerID {
		return svc.saveGoogleConditionalAccessRow(ctx, existing, fleet.ConditionalAccessGoogleStateConnected, nil)
	}

	if err := svc.saveGoogleConditionalAccessRow(ctx, nil, fleet.ConditionalAccessGoogleStatePending, nil); err != nil {
		return err
	}

	resp, err := svc.googleConditionalAccess.Relay.Register(ctx, serverURL, settings.CustomerID)
	if err != nil {
		gwErr := newFleetdmRelayError("Couldn't register with fleetdm.com", err)
		reason := gwErr.Error()
		if saveErr := svc.saveGoogleConditionalAccessRow(ctx, nil, fleet.ConditionalAccessGoogleStateRegistrationFailed, &reason); saveErr != nil {
			svc.logger.ErrorContext(ctx, "record google conditional access registration error", "err", saveErr)
		}
		return ctxerr.Wrap(ctx, gwErr, "register with fleetdm.com")
	}
	row := &fleet.ConditionalAccessGoogleIntegration{
		ProxyServerSecret: resp.FleetServerSecret, State: fleet.ConditionalAccessGoogleStateConnected,
	}
	if err := svc.ds.SaveGoogleConditionalAccessIntegration(ctx, row); err != nil {
		// Without its saved secret the registration could never be removed, so drop it now.
		if derr := svc.googleConditionalAccess.Relay.Deregister(ctx, serverURL, settings.CustomerID, resp.FleetServerSecret); derr != nil {
			svc.logger.ErrorContext(ctx, "drop google conditional access registration that couldn't be saved", "err", derr)
		}
		return ctxerr.Wrap(ctx, err, "save google conditional access integration")
	}
	return nil
}

// leaveGoogleConditionalAccessCustomer drops customerID's registration, so a failure
// changes nothing and keeps its secret for the retry. Google is left as it is, and the
// customer's device rows are left for the cron: deleting them here takes seconds on
// large fleets.
func (svc *Service) leaveGoogleConditionalAccessCustomer(ctx context.Context, serverURL, customerID, secret string) error {
	if secret == "" {
		return nil
	}
	if err := svc.deleteGoogleConditionalAccessRegistration(ctx, serverURL, customerID, secret); err != nil {
		return err
	}
	// The secret is no longer valid on fleetdm.com, so the row must not keep it.
	return svc.saveGoogleConditionalAccessRow(ctx, nil, fleet.ConditionalAccessGoogleStatePending, nil)
}

func (svc *Service) deleteGoogleConditionalAccessRegistration(ctx context.Context, serverURL, customerID, secret string) error {
	if customerID == "" {
		// Can't happen while the config names the registration's customer.
		svc.logger.WarnContext(ctx, "google conditional access registration without a customer in the config, dropping it in Fleet only")
		return nil
	}
	if err := svc.googleConditionalAccess.Relay.Deregister(ctx, serverURL, customerID, secret); err != nil {
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, newFleetdmRelayError("Couldn't delete the registration on fleetdm.com", err),
				"delete registration on fleetdm.com")
		}
		// A missing record on fleetdm.com must not block the removal in Fleet.
		svc.logger.WarnContext(ctx, "google conditional access registration not found on fleetdm.com, continuing")
	}
	return nil
}

// newFleetdmRelayError hides the relay's status behind a 502: a 401 from fleetdm.com
// would read as an expired Fleet session.
func newFleetdmRelayError(message string, err error) *fleet.GatewayError {
	return fleet.NewBadGatewayError(message, err)
}

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

// googleConditionalAccessKeyStored checks hashes only, so reading the config doesn't
// decrypt the key.
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

func (svc *Service) googleConditionalAccessKey(ctx context.Context) ([]byte, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx,
		[]fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}, nil)
	switch {
	case fleet.IsNotFound(err):
		return nil, nil
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "load google conditional access service account key")
	}
	return assets[fleet.MDMAssetGoogleConditionalAccessServiceAccountKey].Value, nil
}
