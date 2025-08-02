package service

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
)

// We use numbers for policy names for easier mapping/indexing with Fleet DB.
const (
	defaultAndroidPolicyID   = 1
	DefaultSignupSSEInterval = 3 * time.Second
	SignupSSESuccess         = "Android Enterprise successfully connected"
)

type Service struct {
	logger           kitlog.Logger
	authz            *authz.Authorizer
	ds               fleet.AndroidDatastore
	androidAPIClient androidmgmt.Client
	fleetSvc         fleet.Service

	// SignupSSEInterval can be overwritten in tests.
	SignupSSEInterval time.Duration
	// AllowLocalhostServerURL is set during tests.
	AllowLocalhostServerURL bool
}

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.AndroidDatastore,
	fleetSvc fleet.Service,
	licenseKey string,
) (android.Service, error) {
	var client androidmgmt.Client
	if os.Getenv("FLEET_DEV_ANDROID_GOOGLE_CLIENT") == "1" || strings.ToUpper(os.Getenv("FLEET_DEV_ANDROID_GOOGLE_CLIENT")) == "ON" {
		client = androidmgmt.NewGoogleClient(ctx, logger, os.Getenv)
	} else {
		client = androidmgmt.NewProxyClient(ctx, logger, licenseKey, os.Getenv)
	}
	return NewServiceWithClient(logger, ds, client, fleetSvc)
}

func NewServiceWithClient(
	logger kitlog.Logger,
	ds fleet.AndroidDatastore,
	client androidmgmt.Client,
	fleetSvc fleet.Service,
) (android.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	return &Service{
		logger:            logger,
		authz:             authorizer,
		ds:                ds,
		androidAPIClient:  client,
		fleetSvc:          fleetSvc,
		SignupSSEInterval: DefaultSignupSSEInterval,
	}, nil
}

func newErrResponse(err error) android.DefaultResponse {
	return android.DefaultResponse{Err: err}
}

func enterpriseSignupEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	result, err := svc.EnterpriseSignup(ctx)
	if err != nil {
		return newErrResponse(err)
	}
	return android.EnterpriseSignupResponse{Url: result.Url}
}

func (svc *Service) EnterpriseSignup(ctx context.Context) (*android.SignupDetails, error) {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	appConfig, err := svc.checkIfAndroidAlreadyConfigured(ctx)
	if err != nil {
		return nil, err
	}

	serverURL := appConfig.ServerSettings.ServerURL
	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: "parsing Fleet server URL: " + err.Error(), InternalErr: err}
	}
	if !svc.AllowLocalhostServerURL {
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" || host == "::1" {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Android Enterprise cannot be enabled with localhost server URL: %s", serverURL)}
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	id, err := svc.ds.CreateEnterprise(ctx, vc.User.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	// signupToken is used to authenticate the signup callback URL -- to ensure that the callback came from our Android enterprise signup flow
	signupToken, err := server.GenerateRandomURLSafeText(32)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generating Android enterprise signup token")
	}

	callbackURL := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/connect/%s", appConfig.ServerSettings.ServerURL, signupToken)
	signupDetails, err := svc.androidAPIClient.SignupURLsCreate(ctx, appConfig.ServerSettings.ServerURL, callbackURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating signup url")
	}

	err = svc.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		Enterprise: android.Enterprise{
			ID: id,
		},
		SignupName:  signupDetails.Name,
		SignupToken: signupToken,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	return signupDetails, nil
}

func (svc *Service) checkIfAndroidAlreadyConfigured(ctx context.Context) (*fleet.AppConfig, error) {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if appConfig.MDM.AndroidEnabledAndConfigured {
		return nil, fleet.NewInvalidArgumentError("android",
			"Android is already enabled and configured").WithStatus(http.StatusConflict)
	}
	return appConfig, nil
}

type enterpriseSignupCallbackRequest struct {
	SignupToken     string `url:"token"`
	EnterpriseToken string `query:"enterpriseToken"`
}

type enterpriseSignupCallbackResponse struct {
	Err error `json:"error,omitempty"`
}

func (res enterpriseSignupCallbackResponse) Error() error { return res.Err }

//go:embed enterpriseCallback.html
var enterpriseCallbackHTML []byte

func (res enterpriseSignupCallbackResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=UTF-8")
	_, _ = w.Write(enterpriseCallbackHTML)
}

func enterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*enterpriseSignupCallbackRequest)
	err := svc.EnterpriseSignupCallback(ctx, req.SignupToken, req.EnterpriseToken)
	return enterpriseSignupCallbackResponse{Err: err}
}

// EnterpriseSignupCallback handles the callback from Google UI during signup flow.
// signupToken is for authentication with Fleet server
// enterpriseToken is for authentication with Google
func (svc *Service) EnterpriseSignupCallback(ctx context.Context, signupToken string, enterpriseToken string) error {
	// Authorization is done by GetEnterpriseBySignupToken below.
	// We call SkipAuthorization here to avoid explicitly calling it when errors occur.
	// Also, this method call will fail if ProxyClient (Google Project) is not configured.
	svc.authz.SkipAuthorization(ctx)

	appConfig, err := svc.checkIfAndroidAlreadyConfigured(ctx)
	if err != nil {
		return err
	}

	enterprise, err := svc.ds.GetEnterpriseBySignupToken(ctx, signupToken)
	switch {
	case fleet.IsNotFound(err):
		return authz.ForbiddenWithInternal("invalid signup token", nil, nil, nil)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	// pubSubToken is used to authenticate the pubsub push endpoint -- to ensure that the push came from our Android enterprise
	pubSubToken, err := server.GenerateRandomURLSafeText(64)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating pubsub token")
	}
	err = svc.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
		Name:  fleet.MDMAssetAndroidPubSubToken,
		Value: []byte(pubSubToken),
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting pubsub authentication token")
	}

	createRsp, err := svc.androidAPIClient.EnterprisesCreate(
		ctx,
		androidmgmt.EnterprisesCreateRequest{
			Enterprise: androidmanagement.Enterprise{
				EnabledNotificationTypes: []string{
					string(android.PubSubEnrollment),
					string(android.PubSubStatusReport),
					string(android.PubSubCommand),
					string(android.PubSubUsageLogs),
				},
			},
			EnterpriseToken: enterpriseToken,
			SignupURLName:   enterprise.SignupName,
			PubSubPushURL:   appConfig.ServerSettings.ServerURL + pubSubPushPath + "?token=" + pubSubToken,
			ServerURL:       appConfig.ServerSettings.ServerURL,
		},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	if createRsp.FleetServerSecret != "" {
		err = svc.ds.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
			Name:  fleet.MDMAssetAndroidFleetServerSecret,
			Value: []byte(createRsp.FleetServerSecret),
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting pubsub authentication token")
		}
		_ = svc.androidAPIClient.SetAuthenticationSecret(createRsp.FleetServerSecret)
	}

	enterpriseID := strings.TrimPrefix(createRsp.EnterpriseName, "enterprises/")
	enterprise.EnterpriseID = enterpriseID
	if createRsp.TopicName != "" {
		topicID, err := topicIDFromName(createRsp.TopicName)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "parsing topic name")
		}
		enterprise.TopicID = topicID
	}
	err = svc.ds.UpdateEnterprise(ctx, enterprise)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), fmt.Sprintf("%d", defaultAndroidPolicyID))
	err = svc.androidAPIClient.EnterprisesPoliciesPatch(ctx, policyName, &androidmanagement.Policy{
		StatusReportingSettings: &androidmanagement.StatusReportingSettings{
			DeviceSettingsEnabled:        true,
			MemoryInfoEnabled:            true,
			NetworkInfoEnabled:           true,
			DisplayInfoEnabled:           true,
			PowerManagementEventsEnabled: true,
			HardwareStatusEnabled:        true,
			SystemPropertiesEnabled:      true,
			SoftwareInfoEnabled:          true, // Android OS version, etc.
			CommonCriteriaModeEnabled:    true,
			// Application inventory will likely be a Premium feature.
			// applicationReports take a lot of space in device status reports. They are not free -- our current cost is $40 per TiB (2025-02-20).
			// We should disable them for free accounts. To enable them for a server transitioning from Free to Premium, we will need to patch the existing policies.
			// For server transitioning from Premium to Free, we will need to patch the existing policies to disable software inventory, which could also be done
			// by the fleetdm.com androidAPIClient or manually. The androidAPIClient could also enforce this report setting.
			ApplicationReportsEnabled:    false,
			ApplicationReportingSettings: nil,
		},
	})
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "patching %d policy", defaultAndroidPolicyID)
	}

	err = svc.ds.DeleteOtherEnterprises(ctx, enterprise.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting temp enterprises")
	}

	err = svc.ds.SetAndroidEnabledAndConfigured(ctx, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting android enabled and configured")
	}

	user, err := svc.ds.UserOrDeletedUserByID(ctx, enterprise.UserID)
	switch {
	case fleet.IsNotFound(err):
		// This should never happen.
		level.Error(svc.logger).Log("msg", "User that created the Android enterprise was not found", "user_id", enterprise.UserID)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting user")
	}

	if err = svc.fleetSvc.NewActivity(ctx, user, fleet.ActivityTypeEnabledAndroidMDM{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for enabled Android MDM")
	}

	return nil
}

func topicIDFromName(name string) (string, error) {
	lastSlash := strings.LastIndex(name, "/")
	if lastSlash == -1 || lastSlash == len(name)-1 {
		return "", fmt.Errorf("topic name %s is not a fully-qualified name", name)
	}
	return name[lastSlash+1:], nil
}

func getEnterpriseEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	enterprise, err := svc.GetEnterprise(ctx)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return android.GetEnterpriseResponse{EnterpriseID: enterprise.EnterpriseID}
}

func (svc *Service) GetEnterprise(ctx context.Context) (*android.Enterprise, error) {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	enterprise, err := svc.ds.GetEnterprise(ctx)
	switch {
	case fleet.IsNotFound(err):
		return nil, fleet.NewInvalidArgumentError("enterprise", "No enterprise found").WithStatus(http.StatusNotFound)
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise")
	}
	return enterprise, nil
}

func deleteEnterpriseEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	err := svc.DeleteEnterprise(ctx)
	return android.DefaultResponse{Err: err}
}

func (svc *Service) DeleteEnterprise(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionWrite); err != nil {
		return err
	}

	// Get enterprise
	enterprise, err := svc.ds.GetEnterprise(ctx)
	switch {
	case fleet.IsNotFound(err):
		// No enterprise to delete
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	default:
		secret, err := svc.getClientAuthenticationSecret(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting client authentication secret")
		}
		_ = svc.androidAPIClient.SetAuthenticationSecret(secret)
		err = svc.androidAPIClient.EnterpriseDelete(ctx, enterprise.Name())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting enterprise via Google API")
		}
	}

	err = svc.ds.DeleteAllEnterprises(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting enterprises")
	}

	err = svc.ds.SetAndroidEnabledAndConfigured(ctx, false)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clearing android enabled and configured")
	}

	err = svc.ds.BulkSetAndroidHostsUnenrolled(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set android hosts as unenrolled")
	}

	if err = svc.fleetSvc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDisabledAndroidMDM{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for disabled Android MDM")
	}

	err = svc.ds.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken, fleet.MDMAssetAndroidFleetServerSecret})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting MDM Android encrypted config assets")
	}

	return nil
}

type enrollmentTokenRequest struct {
	EnrollSecret string `query:"enroll_secret"`
}

type enrollmentTokenResponse struct {
	*android.EnrollmentToken
	android.DefaultResponse
}

// TODO IB This endpoint should be authenticated by MDM SSO in some cases
func enrollmentTokenEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*enrollmentTokenRequest)
	token, err := svc.CreateEnrollmentToken(ctx, req.EnrollSecret)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return enrollmentTokenResponse{EnrollmentToken: token}
}

func (svc *Service) CreateEnrollmentToken(ctx context.Context, enrollSecret string) (*android.EnrollmentToken, error) {
	// Authorization is done by VerifyEnrollSecret below.
	// We call SkipAuthorization here to avoid explicitly calling it when errors occur.
	svc.authz.SkipAuthorization(ctx)

	_, err := svc.checkIfAndroidNotConfigured(ctx)
	if err != nil {
		return nil, err
	}

	_, err = svc.ds.VerifyEnrollSecret(ctx, enrollSecret)
	switch {
	case fleet.IsNotFound(err):
		return nil, fleet.NewAuthFailedError("invalid secret")
	case err != nil:
		return nil, ctxerr.Wrap(ctx, err, "verifying enroll secret")
	}

	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

	token := &androidmanagement.EnrollmentToken{
		// Default duration is 1 hour

		AdditionalData:     enrollSecret,
		AllowPersonalUsage: "PERSONAL_USAGE_ALLOWED",
		PolicyName:         fmt.Sprintf("%s/policies/%d", enterprise.Name(), +defaultAndroidPolicyID),
		OneTimeOnly:        true,
	}
	token, err = svc.androidAPIClient.EnterprisesEnrollmentTokensCreate(ctx, enterprise.Name(), token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating Android enrollment token")
	}

	return &android.EnrollmentToken{
		EnrollmentToken: token.Value,
		EnrollmentURL:   "https://enterprise.google.com/android/enroll?et=" + token.Value,
	}, nil
}

func (svc *Service) checkIfAndroidNotConfigured(ctx context.Context) (*fleet.AppConfig, error) {
	// This call uses cached_mysql implementation, so it's safe to call it multiple times
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil, fleet.NewInvalidArgumentError("android",
			"Android MDM is NOT configured").WithStatus(http.StatusConflict)
	}
	return appConfig, nil
}

type enterpriseSSEResponse struct {
	android.DefaultResponse
	done chan string
}

func (r enterpriseSSEResponse) HijackRender(_ context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Transfer-Encoding", "chunked")
	if r.done == nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "Error: No SSE data available")
		return
	}
	w.WriteHeader(http.StatusOK)
	w.(http.Flusher).Flush()

	for {
		select {
		case data, ok := <-r.done:
			if ok {
				_, _ = fmt.Fprint(w, data)
				w.(http.Flusher).Flush()
			}
			return
		case <-time.After(5 * time.Second):
			// We send a heartbeat to prevent the load balancer from closing the (otherwise idle) connection.
			// The leading colon indicates this is a comment, and is ignored.
			// https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
			_, _ = fmt.Fprint(w, ":heartbeat\n")
			w.(http.Flusher).Flush()
		}
	}
}

func enterpriseSSE(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	done, err := svc.EnterpriseSignupSSE(ctx)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return enterpriseSSEResponse{done: done}
}

func (svc *Service) EnterpriseSignupSSE(ctx context.Context) (chan string, error) {
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	done := make(chan string)
	go func() {
		if svc.signupSSECheck(ctx, done) {
			return
		}
		for {
			select {
			case <-ctx.Done():
				level.Debug(svc.logger).Log("msg", "Context cancelled during Android signup SSE")
				return
			case <-time.After(svc.SignupSSEInterval):
				if svc.signupSSECheck(ctx, done) {
					return
				}
			}
		}
	}()

	return done, nil
}

func (svc *Service) signupSSECheck(ctx context.Context, done chan string) bool {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		done <- fmt.Sprintf("Error getting app config: %v", err)
		return true
	}
	if appConfig.MDM.AndroidEnabledAndConfigured {
		done <- SignupSSESuccess
		return true
	}
	return false
}
