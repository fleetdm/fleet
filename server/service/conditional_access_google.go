package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"reflect"

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
	integration *fleet.ConditionalAccessGoogleIntegration, status *fleet.ConditionalAccessGoogleStatus,
) googleConditionalAccessResponse {
	var resp googleConditionalAccessResponse
	if integration != nil {
		resp.CustomerID = integration.CustomerID
		resp.AdminEmail = integration.AdminEmail
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
	integration, status, err := svc.GetGoogleConditionalAccess(ctxdb.RequirePrimary(ctx, true))
	if err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	return newGoogleConditionalAccessResponse(integration, status), nil
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
	existing, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return err
	}
	storedKey, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return err
	}
	keyChanged := !sameGoogleServiceAccountKey(storedKey, serviceAccountKey)
	if !keyChanged && googleConditionalAccessConnected(existing, settings) {
		return svc.saveGoogleConditionalAccessSettings(ctx, appConfig, settings)
	}

	if err := svc.verifyGoogleConditionalAccess(ctx, settings, serviceAccountKey, "service_account_key"); err != nil {
		return err
	}
	if existing != nil && existing.CustomerID != settings.CustomerID {
		if err := svc.leaveGoogleConditionalAccessCustomer(ctx, appConfig.ServerSettings.ServerURL, settings, existing); err != nil {
			return err
		}
		existing = nil
	}
	// Saved before registering, so a failed registration is retried by the next config
	// update.
	if keyChanged {
		if err := svc.storeGoogleConditionalAccessKey(ctx, serviceAccountKey); err != nil {
			return err
		}
	}
	if err := svc.saveGoogleConditionalAccessSettings(ctx, appConfig, settings); err != nil {
		return err
	}
	return svc.registerGoogleConditionalAccess(ctx, appConfig.ServerSettings.ServerURL, settings, existing)
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/_version_/fleet/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

func getGoogleConditionalAccessEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	integration, status, err := svc.GetGoogleConditionalAccess(ctx)
	if err != nil {
		return googleConditionalAccessResponse{Err: err}, nil
	}
	return newGoogleConditionalAccessResponse(integration, status), nil
}

func (svc *Service) GetGoogleConditionalAccess(ctx context.Context) (*fleet.ConditionalAccessGoogleIntegration, *fleet.ConditionalAccessGoogleStatus, error) {
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
	// The connection state and the key only exist while the integration is configured.
	if appConfig.Integrations.GoogleConditionalAccess == nil {
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
	configured := appConfig.Integrations.GoogleConditionalAccess != nil
	active := existing != nil && existing.State != fleet.ConditionalAccessGoogleStateDeleted
	if !active && !keyStored && !configured {
		return &fleet.BadRequestError{Message: "integration not found"}
	}

	// Nothing changes in Google or on fleetdm.com: the row keeps the registration and the
	// device rows, so re-creating the integration for the same customer reuses them. The
	// row is marked last, so a failed DELETE can be retried.
	if configured {
		appConfig.Integrations.GoogleConditionalAccess = nil
		if err := svc.ds.SaveAppConfig(ctx, appConfig); err != nil {
			return ctxerr.Wrap(ctx, err, "save app config")
		}
	}
	return svc.softDeleteGoogleConditionalAccess(ctx, existing)
}

// softDeleteGoogleConditionalAccess deletes the key and marks the row deleted, once the
// setting is cleared. The registration and the device rows are kept for a re-add for the
// same customer.
func (svc *Service) softDeleteGoogleConditionalAccess(ctx context.Context, existing *fleet.ConditionalAccessGoogleIntegration) error {
	if err := svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetGoogleConditionalAccessServiceAccountKey}); err != nil {
		return ctxerr.Wrap(ctx, err, "delete google conditional access service account key")
	}
	if existing == nil || existing.State == fleet.ConditionalAccessGoogleStateDeleted {
		return nil
	}
	existing.State = fleet.ConditionalAccessGoogleStateDeleted
	existing.ErrorMessage = nil
	if err := svc.ds.SaveGoogleConditionalAccessIntegration(ctx, existing); err != nil {
		return ctxerr.Wrap(ctx, err, "save google conditional access integration")
	}
	if existing.Registered() {
		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
		}
	}
	return nil
}

type googleConditionalAccessAction int

const (
	googleConditionalAccessNoop googleConditionalAccessAction = iota
	googleConditionalAccessConnect
	googleConditionalAccessRemove
)

// googleConditionalAccessPlan is the work an app config update implies for the
// integration, decided before the dry-run return so the delegation check can run in
// a dry run too.
type googleConditionalAccessPlan struct {
	action   googleConditionalAccessAction
	settings fleet.GoogleConditionalAccessIntegration
	existing *fleet.ConditionalAccessGoogleIntegration
	key      []byte
	// storeKey is set when key came with the update and replaces the stored key.
	storeKey bool
	// oldServerURL is the server URL before the update, which an existing registration was
	// made with, so it is used to leave or remove it. serverURL is the server URL after the
	// update, used to register.
	oldServerURL string
	serverURL    string
	// retry is set when the settings didn't change and a previous connection didn't
	// complete. A retry never fails the config update: its error is recorded as the
	// integration's error message instead.
	retry       bool
	retryReason string
}

// planGoogleConditionalAccess compares the desired settings with the stored connection
// state, so a GitOps run that resends the same settings makes no Google or fleetdm.com
// call. Only an update that carries the Google settings (requested) is planned, so an
// unrelated config change never reads the integration or depends on Google or
// fleetdm.com being reachable. Nil settings mean the integration is off, and nothing is
// read when it is off before and after the update.
func (svc *Service) planGoogleConditionalAccess(
	ctx context.Context, oldSettings, settings *fleet.GoogleConditionalAccessIntegration, requested bool,
	newKey []byte, oldServerURL, serverURL string,
) (*googleConditionalAccessPlan, error) {
	if !requested || (oldSettings == nil && settings == nil) {
		return nil, nil
	}
	if svc.googleConditionalAccess == nil {
		return nil, ctxerr.New(ctx, "google conditional access client is not configured")
	}
	ctx = ctxdb.RequirePrimary(ctx, true)
	existing, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return nil, err
	}
	plan := &googleConditionalAccessPlan{existing: existing, oldServerURL: oldServerURL, serverURL: serverURL}

	if settings == nil {
		plan.action = googleConditionalAccessRemove
		return plan, nil
	}
	plan.settings = *settings

	if newKey == nil && googleConditionalAccessConnected(existing, *settings) {
		return plan, nil
	}
	key, err := svc.googleConditionalAccessKey(ctx)
	if err != nil {
		return nil, err
	}
	if newKey != nil {
		if sameGoogleServiceAccountKey(key, newKey) {
			if googleConditionalAccessConnected(existing, *settings) {
				return plan, nil
			}
		} else {
			plan.action = googleConditionalAccessConnect
			plan.key = newKey
			plan.storeKey = true
			return plan, nil
		}
	}
	if len(key) == 0 {
		// Removing the integration deletes its key, so adding it back needs a key too.
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("integrations.google_conditional_access.api_key_json",
			"api_key_json is required because no service account key is stored"))
	}
	plan.action = googleConditionalAccessConnect
	plan.key = key
	plan.retry = oldSettings != nil && *oldSettings == *settings
	return plan, nil
}

func (svc *Service) verifyGoogleConditionalAccessPlan(ctx context.Context, plan *googleConditionalAccessPlan) error {
	if plan == nil || plan.action != googleConditionalAccessConnect {
		return nil
	}
	if plan.retry {
		if err := svc.checkGoogleDelegation(ctx, plan.settings, plan.key); err != nil {
			plan.retryReason = err.Error()
		}
		return nil
	}
	return svc.verifyGoogleConditionalAccess(ctx, plan.settings, plan.key, "integrations.google_conditional_access")
}

// applyGoogleConditionalAccessPlan does the part of the plan that must come before the
// app config is saved, so a failure leaves the config unchanged. A changed customer ID
// leaves the previous customer here, as the key upload does before saving the settings.
// Nothing it does needs a new key upload to undo if the save then fails, which is why
// removal waits for finishGoogleConditionalAccessPlan.
func (svc *Service) applyGoogleConditionalAccessPlan(ctx context.Context, plan *googleConditionalAccessPlan) error {
	if plan == nil || plan.action != googleConditionalAccessConnect || plan.retry {
		return nil
	}
	ctx = ctxdb.RequirePrimary(ctx, true)
	if err := svc.leaveChangedGoogleConditionalAccessCustomer(ctx, plan); err != nil {
		return err
	}
	if plan.storeKey {
		// Stored before registering, so a failed registration is retried with this key.
		return svc.storeGoogleConditionalAccessKey(ctx, plan.key)
	}
	return nil
}

// finishGoogleConditionalAccessPlan does the rest of the plan once the app config holds
// the new settings: a failed registration is recorded against the settings the config
// shows, and a removed integration is soft deleted only once the config says it is off.
func (svc *Service) finishGoogleConditionalAccessPlan(ctx context.Context, plan *googleConditionalAccessPlan) error {
	if plan == nil {
		return nil
	}
	ctx = ctxdb.RequirePrimary(ctx, true)
	switch plan.action {
	case googleConditionalAccessRemove:
		return svc.softDeleteGoogleConditionalAccess(ctx, plan.existing)
	case googleConditionalAccessConnect:
	default:
		return nil
	}
	if !plan.retry {
		return svc.registerGoogleConditionalAccess(ctx, plan.serverURL, plan.settings, plan.existing)
	}

	if plan.retryReason != "" {
		if plan.existing.Registered() && plan.existing.CustomerID != plan.settings.CustomerID {
			// The previous customer's secret is still needed to leave it once the check passes.
			svc.logger.WarnContext(ctx, "google conditional access delegation check failed, keeping the previous customer", "err", plan.retryReason)
			return nil
		}
		return svc.saveGoogleConditionalAccessRow(ctx, plan.settings, plan.existing,
			fleet.ConditionalAccessGoogleStateDelegationFailed, &plan.retryReason)
	}
	err := svc.leaveChangedGoogleConditionalAccessCustomer(ctx, plan)
	if err == nil {
		err = svc.registerGoogleConditionalAccess(ctx, plan.serverURL, plan.settings, plan.existing)
	}
	if err != nil {
		svc.logger.ErrorContext(ctx, "retry google conditional access connection", "err", err)
	}
	return nil
}

// leaveChangedGoogleConditionalAccessCustomer leaves the previous customer when the
// customer ID changed.
func (svc *Service) leaveChangedGoogleConditionalAccessCustomer(ctx context.Context, plan *googleConditionalAccessPlan) error {
	if plan.existing == nil || plan.existing.CustomerID == plan.settings.CustomerID {
		return nil
	}
	if err := svc.leaveGoogleConditionalAccessCustomer(ctx, plan.oldServerURL, plan.settings, plan.existing); err != nil {
		return err
	}
	plan.existing = nil
	return nil
}

// takeGoogleConditionalAccessKey clears the key from the settings, since it is stored in
// mdm_config_assets and never with the app config, and returns it when one was given.
func (svc *Service) takeGoogleConditionalAccessKey(gca *fleet.GoogleConditionalAccessIntegration, invalid *fleet.InvalidArgumentError) []byte {
	if gca == nil {
		return nil
	}
	key := gca.APIKeyJSON
	gca.APIKeyJSON = ""
	if !key.Provided() {
		return nil
	}
	if svc.config.Server.PrivateKey == "" || testSetEmptyPrivateKey {
		invalid.Append("integrations.google_conditional_access.api_key_json", missingServerPrivateKeyMessage)
	}
	return []byte(key)
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

func googleConditionalAccessConnected(existing *fleet.ConditionalAccessGoogleIntegration, settings fleet.GoogleConditionalAccessIntegration) bool {
	return existing != nil && existing.State == fleet.ConditionalAccessGoogleStateConnected &&
		existing.CustomerID == settings.CustomerID && existing.AdminEmail == settings.AdminEmail
}

// checkGoogleDelegation's error message is shown to the admin.
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

func (svc *Service) verifyGoogleConditionalAccess(ctx context.Context, settings fleet.GoogleConditionalAccessIntegration, key []byte, field string) error {
	if err := svc.checkGoogleDelegation(ctx, settings, key); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(field, err.Error()))
	}
	return nil
}

// newGoogleConditionalAccessRow keeps existing's registration when it is for the same
// customer.
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

// registerGoogleConditionalAccess registers with fleetdm.com unless already registered
// for this customer, and saves the row. Callers must have run the delegation check and
// left any previous customer.
func (svc *Service) registerGoogleConditionalAccess(
	ctx context.Context, serverURL string, settings fleet.GoogleConditionalAccessIntegration,
	existing *fleet.ConditionalAccessGoogleIntegration,
) error {
	if existing != nil && existing.Registered() && existing.CustomerID == settings.CustomerID {
		if err := svc.saveGoogleConditionalAccessRow(ctx, settings, existing, fleet.ConditionalAccessGoogleStateConnected, nil); err != nil {
			return err
		}
		if existing.State == fleet.ConditionalAccessGoogleStateDeleted {
			return svc.newGoogleConditionalAccessAddedActivity(ctx)
		}
		return nil
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
	return svc.newGoogleConditionalAccessAddedActivity(ctx)
}

func (svc *Service) newGoogleConditionalAccessAddedActivity(ctx context.Context) error {
	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeAddedConditionalAccessIntegrationGoogle{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for google conditional access integration")
	}
	return nil
}

// leaveGoogleConditionalAccessCustomer drops the previous customer's registration, so a
// failure changes nothing and keeps its secret for the retry. Google is left as it is,
// and the previous customer's device rows are left for the cron: deleting them here
// takes seconds on large fleets.
func (svc *Service) leaveGoogleConditionalAccessCustomer(
	ctx context.Context, serverURL string, settings fleet.GoogleConditionalAccessIntegration,
	existing *fleet.ConditionalAccessGoogleIntegration,
) error {
	if existing.Registered() {
		if err := svc.deleteGoogleConditionalAccessRegistration(ctx, serverURL, existing); err != nil {
			return err
		}
		// The secret is no longer valid on fleetdm.com, so the row must not keep it.
		if err := svc.saveGoogleConditionalAccessRow(ctx, settings, nil, fleet.ConditionalAccessGoogleStatePending, nil); err != nil {
			return err
		}
		// A deleted integration already logged its deletion.
		if existing.State != fleet.ConditionalAccessGoogleStateDeleted {
			if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDeletedConditionalAccessIntegrationGoogle{}); err != nil {
				return ctxerr.Wrap(ctx, err, "create activity for deletion of google conditional access integration")
			}
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

func (svc *Service) storeGoogleConditionalAccessKey(ctx context.Context, key []byte) error {
	if err := svc.ds.ReplaceMDMConfigAssets(ctx, []fleet.MDMConfigAsset{{
		Name:  fleet.MDMAssetGoogleConditionalAccessServiceAccountKey,
		Value: key,
	}}, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "store google conditional access service account key")
	}
	return nil
}

// sameGoogleServiceAccountKey compares the keys as JSON, so a GitOps run that resends the
// stored key with different formatting changes nothing.
func sameGoogleServiceAccountKey(a, b []byte) bool {
	var av, bv map[string]any
	if json.Unmarshal(a, &av) != nil || json.Unmarshal(b, &bv) != nil {
		return bytes.Equal(a, b)
	}
	return reflect.DeepEqual(av, bv)
}
