package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service/conditional_access_google"
)

// googleRegistrationTokenTTL bounds the Google sign-in started from the web clip.
const googleRegistrationTokenTTL = 15 * time.Minute

// activeGoogleConditionalAccess is the integration as the device flows use it.
type activeGoogleConditionalAccess struct {
	settings    fleet.GoogleConditionalAccessIntegration
	integration *fleet.ConditionalAccessGoogleIntegration
	serverURL   string
}

// activeGoogleConditionalAccess returns nil when the device flows are off: on Fleet Free,
// without the setting, before fleetdm.com registered this server, and while the
// integration is deleted. appConfig is loaded when nil.
func (svc *Service) activeGoogleConditionalAccess(ctx context.Context, appConfig *fleet.AppConfig) (*activeGoogleConditionalAccess, error) {
	if lic, _ := license.FromContext(ctx); lic == nil || !lic.IsPremium() || svc.googleConditionalAccess == nil {
		return nil, nil
	}
	if appConfig == nil {
		var err error
		if appConfig, err = svc.ds.AppConfig(ctx); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load app config")
		}
	}
	settings := appConfig.Integrations.GoogleConditionalAccess
	if settings == nil {
		return nil, nil
	}
	integration, err := svc.googleConditionalAccessIntegration(ctx)
	if err != nil {
		return nil, err
	}
	if integration == nil || !integration.Registered() || integration.State == fleet.ConditionalAccessGoogleStateDeleted {
		return nil, nil
	}
	return &activeGoogleConditionalAccess{
		settings: *settings, integration: integration, serverURL: appConfig.ServerSettings.ServerURL,
	}, nil
}

// googleConditionalAccessEligible reports whether the host is a company-owned mobile
// device, the only kind whose Google account is let in.
func (svc *Service) googleConditionalAccessEligible(ctx context.Context, host *fleet.Host) (bool, error) {
	if !fleet.IsMobilePlatform(host.Platform) {
		return false, nil
	}
	hostMDM, err := svc.ds.GetHostMDM(ctx, host.ID)
	switch {
	case fleet.IsNotFound(err):
		return false, nil
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "get host mdm")
	}
	return hostMDM.EnrollmentStatus() == fleet.MDMEnrollmentStatusAutomatic, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /api/_version_/fleet/device/{token}/conditional_access/google
////////////////////////////////////////////////////////////////////////////////

type getDeviceGoogleConditionalAccessRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceGoogleConditionalAccessRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceGoogleConditionalAccessResponse struct {
	*fleet.GoogleConditionalAccessDeviceStatus
	Err error `json:"error,omitempty"`
}

func (r getDeviceGoogleConditionalAccessResponse) Error() error { return r.Err }

func getDeviceGoogleConditionalAccessEndpoint(ctx context.Context, _ any, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceGoogleConditionalAccessResponse{Err: err}, nil
	}
	status, err := svc.GetGoogleConditionalAccessDeviceStatus(ctx, host)
	if err != nil {
		return getDeviceGoogleConditionalAccessResponse{Err: err}, nil
	}
	return getDeviceGoogleConditionalAccessResponse{GoogleConditionalAccessDeviceStatus: status}, nil
}

func (svc *Service) GetGoogleConditionalAccessDeviceStatus(ctx context.Context, host *fleet.Host) (*fleet.GoogleConditionalAccessDeviceStatus, error) {
	// skipauth: The host is authenticated by its device token or UUID.
	svc.authz.SkipAuthorization(ctx)

	status := &fleet.GoogleConditionalAccessDeviceStatus{}
	// Android hosts don't sign in: the sync cron registers them.
	if !fleet.IsAppleMobilePlatform(host.Platform) {
		return status, nil
	}
	active, err := svc.activeGoogleConditionalAccess(ctx, nil)
	if err != nil || active == nil {
		return status, err
	}
	status.Enabled = true

	// Counting an attempt is a read-then-write.
	ctx = ctxdb.RequirePrimary(ctx, true)
	if status.Eligible, err = svc.googleConditionalAccessEligible(ctx, host); err != nil || !status.Eligible {
		return status, err
	}
	customerID := active.settings.CustomerID
	device, err := svc.ds.GetGoogleConditionalAccessDevice(ctx, host.UUID, customerID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "get google conditional access device")
	}
	if device.Registered() {
		status.Registered = true
		status.Managed = device.Managed()
		return status, nil
	}

	token, tokenHash, err := newGoogleRegistrationToken()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create google registration token")
	}
	started, err := svc.ds.StartGoogleConditionalAccessRegistration(ctx, host.UUID, customerID, tokenHash,
		svc.clock.Now().Add(googleRegistrationTokenTTL))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "start google conditional access registration")
	}
	if !started {
		status.RetryBlocked = true
		return status, nil
	}
	if status.SignInURL, err = svc.googleConditionalAccess.Relay.SignInURL(customerID, host.UUID, token); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build google sign-in url")
	}
	return status, nil
}

func newGoogleRegistrationToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, hashGoogleRegistrationToken(token), nil
}

func hashGoogleRegistrationToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/_version_/fleet/google_conditional_access/register
////////////////////////////////////////////////////////////////////////////////

type registerGoogleConditionalAccessDeviceRequest struct {
	fleet.GoogleConditionalAccessRegistration
	secret string
}

func (registerGoogleConditionalAccessDeviceRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var req registerGoogleConditionalAccessDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req.GoogleConditionalAccessRegistration); err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{Message: "invalid request body", InternalErr: err})
	}
	req.secret, _ = strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	return &req, nil
}

type registerGoogleConditionalAccessDeviceResponse struct {
	Result fleet.GoogleConditionalAccessRegisterResult `json:"result,omitempty"`
	Err    error                                       `json:"error,omitempty"`
}

func (r registerGoogleConditionalAccessDeviceResponse) Error() error { return r.Err }

func registerGoogleConditionalAccessDeviceEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*registerGoogleConditionalAccessDeviceRequest)
	result, err := svc.RegisterGoogleConditionalAccessDevice(ctx, req.secret, req.GoogleConditionalAccessRegistration)
	if err != nil {
		return registerGoogleConditionalAccessDeviceResponse{Err: err}, nil
	}
	return registerGoogleConditionalAccessDeviceResponse{Result: result}, nil
}

func (svc *Service) RegisterGoogleConditionalAccessDevice(
	ctx context.Context, secret string, reg fleet.GoogleConditionalAccessRegistration,
) (fleet.GoogleConditionalAccessRegisterResult, error) {
	// skipauth: fleetdm.com authenticates with this server's secret, checked below.
	svc.authz.SkipAuthorization(ctx)

	ctx = ctxdb.RequirePrimary(ctx, true)
	active, err := svc.activeGoogleConditionalAccess(ctx, nil)
	if err != nil {
		return "", err
	}
	if active == nil || subtle.ConstantTimeCompare([]byte(secret), []byte(active.integration.ProxyServerSecret)) != 1 {
		return "", fleet.NewAuthFailedError("invalid google conditional access secret")
	}
	result, err := svc.registerGoogleConditionalAccessDevice(ctx, active, reg)
	if err != nil {
		// fleetdm.com shows the result page, so the failure is recorded here.
		svc.logger.ErrorContext(ctx, "register google conditional access device", "err", err)
		ctxerr.Handle(ctx, err)
		return fleet.GoogleConditionalAccessRegisterError, nil
	}
	return result, nil
}

func (svc *Service) registerGoogleConditionalAccessDevice(
	ctx context.Context, active *activeGoogleConditionalAccess, reg fleet.GoogleConditionalAccessRegistration,
) (fleet.GoogleConditionalAccessRegisterResult, error) {
	customerID := active.settings.CustomerID
	if reg.Token == "" {
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	}
	// Consuming the token resets the attempt counter whatever the result.
	hostUUID, err := svc.ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customerID, hashGoogleRegistrationToken(reg.Token))
	switch {
	case fleet.IsNotFound(err):
		svc.logger.InfoContext(ctx, "google conditional access register with an unknown, used or expired token")
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	case err != nil:
		return "", ctxerr.Wrap(ctx, err, "consume google registration token")
	}
	host, err := svc.ds.HostByUUID(ctx, hostUUID)
	switch {
	case fleet.IsNotFound(err):
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	case err != nil:
		return "", ctxerr.Wrap(ctx, err, "get host by uuid")
	}
	if !fleet.IsAppleMobilePlatform(host.Platform) {
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	}
	return svc.registerGoogleDeviceUsers(ctx, active, host, reg.Customer, reg.Email, reg.Names)
}

// registerGoogleDeviceUsers stores the names Google's lookup returned on the host, which
// binds it to their device, and marks them.
func (svc *Service) registerGoogleDeviceUsers(
	ctx context.Context, active *activeGoogleConditionalAccess, host *fleet.Host, lookupCustomer, email string, names []string,
) (fleet.GoogleConditionalAccessRegisterResult, error) {
	customerID := active.settings.CustomerID
	eligible, err := svc.googleConditionalAccessEligible(ctx, host)
	if err != nil {
		return "", err
	}
	if !eligible {
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	}
	if !sameGoogleCustomer(lookupCustomer, customerID) {
		svc.logger.WarnContext(ctx, "google conditional access sign-in from another Workspace",
			"host_id", host.ID, "customer", lookupCustomer)
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	}
	names = dedupeDeviceUserNames(names)
	if len(names) == 0 {
		return fleet.GoogleConditionalAccessRegisterNotFound, nil
	}

	err = svc.ds.RecordGoogleConditionalAccessRegistration(ctx, host.UUID, customerID, names)
	if conflict, ok := errors.AsType[*fleet.GoogleConditionalAccessDeviceUserConflictError](err); ok {
		svc.logger.WarnContext(ctx, "refused a google device user registered to another host",
			"host_id", host.ID, "device_user_name", conflict.Name)
		return fleet.GoogleConditionalAccessRegisterNotAllowed, nil
	}
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "record google conditional access registration")
	}
	googleDeviceID, err := svc.ds.GetHostGoogleDeviceID(ctx, host.UUID, customerID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "get host google device id")
	}
	if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeSetHostGoogleDeviceID{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		GoogleDeviceID:  googleDeviceID,
		GoogleEmail:     email,
		Automated:       true,
	}); err != nil {
		return "", ctxerr.Wrap(ctx, err, "create activity for google device registration")
	}

	ci, err := svc.googleCloudIdentity(ctx, active)
	if err != nil {
		return "", err
	}
	if ci == nil {
		svc.logger.WarnContext(ctx, "google conditional access key missing, the cron marks the names once it is uploaded")
		return fleet.GoogleConditionalAccessRegisterDone, nil
	}
	svc.markGoogleDeviceUsers(ctx, ci, host, names)
	return fleet.GoogleConditionalAccessRegisterDone, nil
}

// googleCloudIdentity returns the client for the customer's Workspace, nil while no
// service account key is stored.
func (svc *Service) googleCloudIdentity(ctx context.Context, active *activeGoogleConditionalAccess) (conditional_access_google.CloudIdentity, error) {
	key, err := svc.googleConditionalAccessKey(ctx)
	if err != nil || len(key) == 0 {
		return nil, err
	}
	ci, err := svc.googleConditionalAccess.CloudIdentity(ctx, conditional_access_google.Credentials{
		ServiceAccountKey: key, AdminEmail: active.settings.AdminEmail, CustomerID: active.settings.CustomerID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create google cloud identity client")
	}
	return ci, nil
}

// connectedGoogleCloudIdentity returns the client and the customer of the connected
// integration, or a nil client when the device flows are off or no key is stored.
func (svc *Service) connectedGoogleCloudIdentity(ctx context.Context) (conditional_access_google.CloudIdentity, string, error) {
	active, err := svc.activeGoogleConditionalAccess(ctx, nil)
	if err != nil || active == nil {
		return nil, "", err
	}
	ci, err := svc.googleCloudIdentity(ctx, active)
	return ci, active.settings.CustomerID, err
}

// listGoogleDeviceUserNames returns the DeviceUser names of a device an admin named, with
// the errors the Google device ID endpoint answers with.
func listGoogleDeviceUserNames(ctx context.Context, ci conditional_access_google.CloudIdentity, googleDeviceID string) ([]string, error) {
	names, err := ci.ListDeviceUserNames(ctx, googleDeviceID)
	switch {
	case fleet.IsNotFound(err) || conditional_access_google.IsStale(err):
		return nil, fleet.NewInvalidArgumentError("google_device_id", "Google has no device with this ID.")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, fleet.NewBadGatewayError("Couldn't get the device's users from Google", err), "list google device users")
	}
	if names = dedupeDeviceUserNames(names); len(names) == 0 {
		return nil, fleet.NewInvalidArgumentError("google_device_id", "This device has no users in Google.")
	}
	return names, nil
}

// markGoogleDeviceUsers marks the names just stored on the host. They are stored
// either way: a failed mark stays pending with its error and the cron retries it.
func (svc *Service) markGoogleDeviceUsers(ctx context.Context, ci conditional_access_google.CloudIdentity, host *fleet.Host, names []string) {
	for _, name := range names {
		if _, err := markGoogleDeviceUser(ctx, svc.ds, ci, name, host.UUID); err != nil {
			svc.logger.ErrorContext(ctx, "mark google device user", "device_user_name", name, "err", err)
			ctxerr.Handle(ctx, err)
		}
	}
}

// markGoogleDeviceUser writes MANAGED on a pending name. A refusal from Google is
// recorded on the name, and stale reports whether Google no longer has it, which the
// sync cron acts on; registration leaves that to the cron.
func markGoogleDeviceUser(
	ctx context.Context, ds fleet.Datastore, ci conditional_access_google.CloudIdentity, name, hostUUID string,
) (stale bool, err error) {
	pending, managed := fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged
	du := fleet.GoogleConditionalAccessDeviceUser{HostUUID: hostUUID, Name: name, State: pending}
	if markErr := ci.SetClientState(ctx, name, fleet.GoogleConditionalAccessDesiredClientState(du)); markErr != nil {
		msg := markErr.Error()
		if _, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, name, pending, pending, &msg); err != nil {
			return false, ctxerr.Wrap(ctx, err, "record google device user error")
		}
		return conditional_access_google.IsStale(markErr), markErr
	}
	if _, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, name, pending, managed, nil); err != nil {
		return false, ctxerr.Wrap(ctx, err, "set google device user managed")
	}
	return false, nil
}

// sameGoogleCustomer compares customer IDs with or without their "customers/" prefix and
// leading "C", which Google's APIs use inconsistently.
func sameGoogleCustomer(a, b string) bool {
	norm := func(s string) string {
		s = strings.TrimPrefix(strings.TrimSpace(s), "customers/")
		return strings.TrimPrefix(s, "C")
	}
	return norm(a) != "" && norm(a) == norm(b)
}

func dedupeDeviceUserNames(names []string) []string {
	seen := make(map[string]struct{}, len(names))
	out := make([]string, 0, len(names))
	for _, n := range names {
		if _, ok := seen[n]; ok || n == "" {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
