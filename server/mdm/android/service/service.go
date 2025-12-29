package service

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm"
	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/service/modules/activities"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
)

// Used for overriding the private key validation in testing
var testSetEmptyPrivateKey bool

const (
	DefaultSignupSSEInterval = 3 * time.Second
	SignupSSESuccess         = "Android Enterprise successfully connected"
)

type Service struct {
	logger           kitlog.Logger
	authz            *authz.Authorizer
	ds               fleet.AndroidDatastore
	fleetDS          fleet.Datastore
	androidAPIClient androidmgmt.Client
	activityModule   activities.ActivityModule
	serverPrivateKey string

	// SignupSSEInterval can be overwritten in tests.
	SignupSSEInterval time.Duration
	// AllowLocalhostServerURL is set during tests.
	AllowLocalhostServerURL bool
}

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.AndroidDatastore,
	licenseKey string,
	serverPrivateKey string,
	fleetDS fleet.Datastore,
	activityModule activities.ActivityModule,
) (android.Service, error) {
	client := newAMAPIClient(ctx, logger, licenseKey)
	return NewServiceWithClient(logger, ds, client, serverPrivateKey, fleetDS, activityModule)
}

func NewServiceWithClient(
	logger kitlog.Logger,
	ds fleet.AndroidDatastore,
	client androidmgmt.Client,
	serverPrivateKey string,
	fleetDS fleet.Datastore,
	activityModule activities.ActivityModule,
) (android.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	svc := &Service{
		logger:            logger,
		authz:             authorizer,
		ds:                ds,
		androidAPIClient:  client,
		serverPrivateKey:  serverPrivateKey,
		SignupSSEInterval: DefaultSignupSSEInterval,
		fleetDS:           fleetDS,
		activityModule:    activityModule,
	}

	// OK to use background context here because this function is only called during server bootstrap
	// Setting the secret here ensures that we don't have to configure it in lots of different places
	// when using the proxy client.
	ctx := context.Background()
	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

	return svc, nil
}

func newAMAPIClient(ctx context.Context, logger kitlog.Logger, licenseKey string) androidmgmt.Client {
	var client androidmgmt.Client
	if os.Getenv("FLEET_DEV_ANDROID_GOOGLE_CLIENT") == "1" || strings.ToUpper(os.Getenv("FLEET_DEV_ANDROID_GOOGLE_CLIENT")) == "ON" {
		client = androidmgmt.NewGoogleClient(ctx, logger, os.Getenv)
	} else {
		client = androidmgmt.NewProxyClient(ctx, logger, licenseKey, os.Getenv)
	}
	return client
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

	// Check if server private key is configured (required for Android MDM)
	if err := svc.checkServerPrivateKey(ctx); err != nil {
		return nil, err
	}

	// Before checking if Android is already configured, verify if existing enterprise still exists
	// This ensures we detect enterprise deletion even if user goes directly to signup page
	if err := svc.VerifyExistingEnterpriseIfAny(ctx); err != nil {
		// If verification returns NotFound (enterprise was deleted), continue with signup
		// Other errors should be returned as-is
		if !fleet.IsNotFound(err) {
			return nil, err
		}
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

func (svc *Service) checkServerPrivateKey(ctx context.Context) error {
	if testSetEmptyPrivateKey {
		return &fleet.BadRequestError{
			Message: "missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key",
		}
	}

	if svc.serverPrivateKey == "" {
		return &fleet.BadRequestError{
			Message: "missing required private key. Learn how to configure the private key here: https://fleetdm.com/learn-more-about/fleet-server-private-key",
		}
	}

	return nil
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

	policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), fmt.Sprintf("%d", android.DefaultAndroidPolicyID))
	_, err = svc.androidAPIClient.EnterprisesPoliciesPatch(ctx, policyName, &androidmanagement.Policy{
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
			// applicationReports take a lot of space in device status reports. They are not free -- our current cost is $40 per TiB (2025-02-20).
			ApplicationReportsEnabled:    true,
			ApplicationReportingSettings: nil,
		},
	}, androidmgmt.PoliciesPatchOpts{ExcludeApps: true})
	if err != nil && !androidmgmt.IsNotModifiedError(err) {
		return ctxerr.Wrapf(ctx, err, "patching %d policy", android.DefaultAndroidPolicyID)
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

	if err = svc.activityModule.NewActivity(ctx, user, fleet.ActivityTypeEnabledAndroidMDM{}); err != nil {
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

	err = svc.ds.MarkAllPendingAndroidVPPInstallsAsFailed(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marking pending android vpp installs as failed")
	}

	if err = svc.activityModule.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDisabledAndroidMDM{}); err != nil {
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
	IdpUUID      string // The UUID of the mdm_idp_account that was used if any, can be empty, will be taken from cookies
}

func (enrollmentTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	enrollSecret := r.URL.Query().Get("enroll_secret")
	if enrollSecret == "" {
		return nil, &fleet.BadRequestError{
			Message: "enroll_secret is required",
		}
	}

	byodIdpCookie, err := r.Cookie(mdm.BYODIdpCookieName)

	if err == http.ErrNoCookie {
		// We do not fail here if no cookie is found, we validate later down the line if it's required
		return &enrollmentTokenRequest{
			EnrollSecret: enrollSecret,
			IdpUUID:      "",
		}, nil
	}

	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "something went wrong parsing the boyd idp cookie",
			InternalErr: err,
		}
	}

	if err = byodIdpCookie.Valid(); err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "boyd idp cookie is not valid",
			InternalErr: err,
		}
	}

	return &enrollmentTokenRequest{
		EnrollSecret: enrollSecret,
		IdpUUID:      byodIdpCookie.Value,
	}, nil
}

func enrollmentTokenEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*enrollmentTokenRequest)
	token, err := svc.CreateEnrollmentToken(ctx, req.EnrollSecret, req.IdpUUID)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return android.EnrollmentTokenResponse{EnrollmentToken: token}
}

func (svc *Service) CreateEnrollmentToken(ctx context.Context, enrollSecret, idpUUID string) (*android.EnrollmentToken, error) {
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

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	requiresIdPUUID, err := shared_mdm.RequiresEnrollOTAAuthentication(ctx, svc.ds, enrollSecret, appCfg.MDM.MacOSSetup.EnableEndUserAuthentication)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking requirement of ota enrollment authentication")
	}

	if requiresIdPUUID && idpUUID == "" {
		return nil, fleet.NewAuthFailedError("required idp uuid to be set, but none found")
	}

	if idpUUID != "" {
		_, err := svc.ds.GetMDMIdPAccountByUUID(ctx, idpUUID)
		if err != nil {
			iae := &fleet.InvalidArgumentError{}
			iae.Append("IDP UUID", "Failed validating IDP account existence")
			return nil, ctxerr.Wrap(ctx, iae)
		}
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

	enrollmentTokenRequest, err := json.Marshal(enrollmentTokenRequest{
		EnrollSecret: enrollSecret,
		IdpUUID:      idpUUID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshalling enrollment token request")
	}

	token := &androidmanagement.EnrollmentToken{
		// Default duration is 1 hour

		AdditionalData:     string(enrollmentTokenRequest),
		AllowPersonalUsage: "PERSONAL_USAGE_ALLOWED",
		PolicyName:         fmt.Sprintf("%s/policies/%d", enterprise.Name(), android.DefaultAndroidPolicyID),
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

// verifyEnterpriseExistsWithGoogle verifies if the given enterprise still exists in Google API.
// Uses LIST-first approach for efficiency and better error handling.
// Returns fleet.IsNotFound error if enterprise was deleted, nil if verification passed.
func (svc *Service) verifyEnterpriseExistsWithGoogle(ctx context.Context, enterprise *android.Enterprise) error {
	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

	// Get server URL from app config
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}

	// Use LIST API as primary verification method
	enterprises, err := svc.androidAPIClient.EnterprisesList(ctx, appConfig.ServerSettings.ServerURL)
	if err != nil {
		var gerr *googleapi.Error
		if errors.As(err, &gerr) {
			switch gerr.Code {
			case http.StatusNotFound:
				// Special case: 404 from proxy with deletion confirmation
				if strings.Contains(gerr.Message, "PROXY_VERIFIED_DELETED:") {
					level.Info(svc.logger).Log("msg", "enterprise confirmed deleted by proxy", "enterpriseID", enterprise.EnterpriseID)
					svc.cleanupDeletedEnterprise(ctx, enterprise, enterprise.EnterpriseID)
					return fleet.NewInvalidArgumentError("enterprise", "Android Enterprise has been deleted").WithStatus(http.StatusNotFound)
				}
			case http.StatusBadRequest:
				// Bad request might indicate missing headers or invalid request format
				// Don't delete the enterprise in this case
				level.Error(svc.logger).Log("msg", "bad request when verifying enterprise", "error", err)
				return fmt.Errorf("verifying enterprise with Google: %s (check request headers and format)", err.Error())
			case http.StatusUnauthorized, http.StatusForbidden:
				// Authentication/authorization issues - don't delete the enterprise
				level.Error(svc.logger).Log("msg", "authentication/authorization error when verifying enterprise", "error", err)
				return fmt.Errorf("verifying enterprise with Google: authentication error: %w", err)
			}
		}
		// LIST failed - this is likely a technical issue, not deletion
		// Log the error but don't delete the enterprise
		level.Error(svc.logger).Log("msg", "failed to list enterprises", "error", err)
		return fmt.Errorf("verifying enterprise with Google: %s", err.Error())
	}

	// Check if our enterprise is in the list
	enterpriseID := strings.TrimPrefix(enterprise.EnterpriseID, "enterprises/")
	for _, ent := range enterprises {
		if strings.HasSuffix(ent.Name, enterpriseID) {
			// Enterprise exists - verification passed
			return nil
		}
	}

	// Enterprise NOT in list - it's deleted - perform cleanup
	level.Info(svc.logger).Log("msg", "enterprise confirmed deleted via LIST API", "enterpriseID", enterpriseID)
	svc.cleanupDeletedEnterprise(ctx, enterprise, enterpriseID)
	return fleet.NewInvalidArgumentError("enterprise", "Android Enterprise has been deleted").WithStatus(http.StatusNotFound)
}

func (svc *Service) VerifyExistingEnterpriseIfAny(ctx context.Context) error {
	// Check if there's an existing enterprise
	enterprise, err := svc.ds.GetEnterprise(ctx)
	switch {
	case fleet.IsNotFound(err):
		// No enterprise exists - this is fine
		return nil
	case err != nil:
		return ctxerr.Wrap(ctx, err, "checking for existing enterprise")
	}

	// Enterprise exists - verify it using the shared method
	return svc.verifyEnterpriseExistsWithGoogle(ctx, enterprise)
}

// cleanupDeletedEnterprise performs the complete cleanup when an enterprise deletion is detected
func (svc *Service) cleanupDeletedEnterprise(ctx context.Context, enterprise *android.Enterprise, enterpriseID string) {
	// Clean up proxy database records by calling proxy DELETE endpoint
	// This ensures the proxy won't return conflicts when creating new signup URLs
	if deleteErr := svc.androidAPIClient.EnterpriseDelete(ctx, enterprise.Name()); deleteErr != nil {
		level.Warn(svc.logger).Log("msg", "failed to delete proxy records after enterprise deletion (may not exist)", "err", deleteErr)
	}

	// Delete local enterprise records
	if deleteErr := svc.ds.DeleteAllEnterprises(ctx); deleteErr != nil {
		level.Error(svc.logger).Log("msg", "failed to delete local enterprise records after deletion", "err", deleteErr)
	}

	// Turn off Android MDM
	if setErr := svc.ds.SetAndroidEnabledAndConfigured(ctx, false); setErr != nil {
		level.Error(svc.logger).Log("msg", "failed to turn off Android MDM after enterprise deletion", "err", setErr)
	}

	// Unenroll Android hosts
	if unenrollErr := svc.ds.BulkSetAndroidHostsUnenrolled(ctx); unenrollErr != nil {
		level.Error(svc.logger).Log("msg", "failed to unenroll Android hosts after enterprise deletion", "err", unenrollErr)
	}

	if err := svc.ds.MarkAllPendingAndroidVPPInstallsAsFailed(ctx); err != nil {
		level.Error(svc.logger).Log("msg", "failed to mark pending Android VPP installs as failed after enterprise deletion", "err", err)
	}
}

// UnenrollAndroidHost calls AMAPI to delete the device (work profile).
// The actual MDM status flip to Off is performed when Pub/Sub sends DELETED for the device.
func (svc *Service) UnenrollAndroidHost(ctx context.Context, hostID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.fleetDS.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting host for android unenrollment")
	}

	if !fleet.IsAndroidPlatform(host.Platform) {
		level.Debug(svc.logger).Log("msg", "Skipping Android unenrollment for non-Android host", "host_id", host.ID, "platform", host.Platform)
		return nil // no-op for non-Android hosts
	}

	// Check authorization again based on host info for team-based permissions.
	if err := svc.authz.Authorize(ctx, fleet.MDMCommandAuthz{
		TeamID: host.TeamID,
	}, fleet.ActionWrite); err != nil {
		return err
	}

	// Resolve Android device and enterprise
	ah, err := svc.ds.AndroidHostLiteByHostUUID(ctx, host.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting android host by uuid")
	}
	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting android enterprise")
	}
	if ah.Device == nil || ah.Device.DeviceID == "" || enterprise.EnterpriseID == "" {
		return &fleet.BadRequestError{Message: "missing android device or enterprise id"}
	}

	// Authenticate client and call AMAPI delete
	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Android authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)
	deviceName := fmt.Sprintf("enterprises/%s/devices/%s", enterprise.EnterpriseID, ah.Device.DeviceID)
	if err := svc.androidAPIClient.EnterprisesDevicesDelete(ctx, deviceName); err != nil {
		return ctxerr.Wrap(ctx, err, "amapi delete device")
	}

	return nil
}

func (svc *Service) EnterprisesApplications(ctx context.Context, enterpriseName, applicationID string) (*androidmanagement.Application, error) {
	return svc.androidAPIClient.EnterprisesApplications(ctx, enterpriseName, applicationID)
}

// Adds the specified apps to the host-specific Android policy of the provided hosts, and
// returns a map of host UUID to the policy request object of their updated policy on success.
func (svc *Service) AddAppsToAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) (map[string]*android.MDMAndroidPolicyRequest, error) {
	var errs []error
	hostToPolicyRequest := make(map[string]*android.MDMAndroidPolicyRequest, len(hostUUIDs))
	for uuid, policyID := range hostUUIDs {
		policyName := fmt.Sprintf("%s/policies/%s", enterpriseName, policyID)
		policyRequest, err := newAndroidPolicyApplicationsRequest(policyID, policyName, appPolicies)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "prepare policy request %s", policyName)
		}

		policy, apiErr := svc.androidAPIClient.EnterprisesPoliciesModifyPolicyApplications(ctx, policyName, appPolicies)
		if _, err := recordAndroidRequestResult(ctx, svc.fleetDS, policyRequest, policy, nil, apiErr); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "save android policy request for host %s", uuid)
		}

		if apiErr != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, apiErr, "google api: modify policy applications for host %s", uuid))
		}
		hostToPolicyRequest[uuid] = policyRequest
	}

	return hostToPolicyRequest, errors.Join(errs...)
}

// getFleetAgentPackageInfo returns the Fleet agent package name and SHA256 fingerprint.
// Returns empty strings if the package is not configured, or an error if the package is configured but SHA256 is missing.
func getFleetAgentPackageInfo(ctx context.Context) (packageName, sha256Fingerprint string, err error) {
	packageName = os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	if packageName == "" {
		return "", "", nil
	}

	sha256Fingerprint = os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	if sha256Fingerprint == "" {
		return "", "", ctxerr.New(ctx, "FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256 must be set when FLEET_DEV_ANDROID_AGENT_PACKAGE is set")
	}

	return packageName, sha256Fingerprint, nil
}

// buildFleetAgentAppPolicy builds an ApplicationPolicy for the Fleet agent from the given managed configuration.
func buildFleetAgentAppPolicy(packageName, sha256Fingerprint string, managedConfig android.AgentManagedConfiguration) (*androidmanagement.ApplicationPolicy, error) {
	managedConfigJSON, err := json.Marshal(managedConfig)
	if err != nil {
		return nil, err
	}

	return &androidmanagement.ApplicationPolicy{
		PackageName:             packageName,
		InstallType:             "FORCE_INSTALLED",
		DefaultPermissionPolicy: "GRANT",
		DelegatedScopes:         []string{"CERT_INSTALL"},
		ManagedConfiguration:    managedConfigJSON,
		SigningKeyCerts: []*androidmanagement.ApplicationSigningKeyCert{
			{
				SigningKeyCertFingerprintSha256: sha256Fingerprint,
			},
		},
		Roles: []*androidmanagement.Role{
			{
				RoleType: "COMPANION_APP",
			},
		},
	}, nil
}

// AddFleetAgentToAndroidPolicy adds the Fleet agent to the Android policy for the given enterprise.
// hostConfigs maps host UUIDs to managed configurations for the Fleet Agent.
// The UUID is BOTH the hostUUID and the policyID. We assume that the host UUID is the same as the policy ID.
func (svc *Service) AddFleetAgentToAndroidPolicy(ctx context.Context, enterpriseName string,
	hostConfigs map[string]android.AgentManagedConfiguration,
) error {
	packageName, sha256Fingerprint, err := getFleetAgentPackageInfo(ctx)
	if err != nil {
		return err
	}
	if packageName == "" {
		return nil
	}

	var errs []error
	for uuid, managedConfig := range hostConfigs {
		policyName := fmt.Sprintf("%s/policies/%s", enterpriseName, uuid)

		fleetAgentApp, err := buildFleetAgentAppPolicy(packageName, sha256Fingerprint, managedConfig)
		if err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "build fleet agent app policy for host %s", uuid))
			continue
		}

		_, err = svc.androidAPIClient.EnterprisesPoliciesModifyPolicyApplications(ctx, policyName,
			[]*androidmanagement.ApplicationPolicy{fleetAgentApp})
		if err != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, err, "google api: modify fleet agent application for host %s", uuid))
		}
	}
	return errors.Join(errs...)
}

// BuildFleetAgentApplicationPolicy builds the ApplicationPolicy for the Fleet agent for the given host.
func (svc *Service) BuildFleetAgentApplicationPolicy(ctx context.Context, hostUUID string) (*androidmanagement.ApplicationPolicy, error) {
	packageName, sha256Fingerprint, err := getFleetAgentPackageInfo(ctx)
	if err != nil {
		return nil, err
	}
	if packageName == "" {
		return nil, nil
	}

	// Build the managed configuration for this host
	managedConfig, err := svc.buildAgentManagedConfig(ctx, hostUUID)
	if err != nil {
		return nil, err
	}

	return buildFleetAgentAppPolicy(packageName, sha256Fingerprint, *managedConfig)
}

// buildAgentManagedConfig builds the AgentManagedConfiguration for the given host.
// This includes the server URL, enroll secret, and certificate template IDs.
func (svc *Service) buildAgentManagedConfig(ctx context.Context, hostUUID string) (*android.AgentManagedConfiguration, error) {
	appConfig, err := svc.fleetDS.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config")
	}

	androidHost, err := svc.ds.AndroidHostLiteByHostUUID(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "get android host %s", hostUUID)
	}

	enrollSecrets, err := svc.fleetDS.GetEnrollSecrets(ctx, androidHost.Host.TeamID)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "get enroll secrets for team %v", androidHost.Host.TeamID)
	}
	if len(enrollSecrets) == 0 {
		return nil, ctxerr.Errorf(ctx, "no enroll secrets found for team %v", androidHost.Host.TeamID)
	}

	// Get certificate templates for the host (all templates, regardless of status)
	certTemplates, err := svc.fleetDS.ListCertificateTemplatesForHosts(ctx, []string{hostUUID})
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "get certificate templates for host %s", hostUUID)
	}

	var certificateTemplateIDs []android.AgentCertificateTemplate
	for _, ct := range certTemplates {
		template := android.AgentCertificateTemplate{
			ID: ct.CertificateTemplateID,
		}
		if ct.Status != nil {
			template.Status = string(*ct.Status)
		}
		if ct.OperationType != nil {
			template.Operation = string(*ct.OperationType)
		}
		if ct.Version != nil {
			template.Version = *ct.Version
		}
		certificateTemplateIDs = append(certificateTemplateIDs, template)
	}

	return &android.AgentManagedConfiguration{
		ServerURL:              appConfig.ServerSettings.ServerURL,
		HostUUID:               hostUUID,
		EnrollSecret:           enrollSecrets[0].Secret,
		CertificateTemplateIDs: certificateTemplateIDs,
	}, nil
}

func (svc *Service) EnableAppReportsOnDefaultPolicy(ctx context.Context) error {
	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			// Then Android MDM isn't setup yet, so no-op
			level.Info(svc.logger).Log("msg", "skipping android default policy migration, Android MDM is not turned on")
			return nil
		}
		return ctxerr.Wrap(ctx, err, "getting android enterprise")
	}

	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

	policyName := fmt.Sprintf("%s/policies/%d", enterprise.Name(), android.DefaultAndroidPolicyID)
	_, err = svc.androidAPIClient.EnterprisesPoliciesPatch(ctx, policyName, &androidmanagement.Policy{
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
			ApplicationReportsEnabled:    true,
			ApplicationReportingSettings: nil,
		},
	}, androidmgmt.PoliciesPatchOpts{ExcludeApps: true})
	if err != nil && !androidmgmt.IsNotModifiedError(err) {
		return ctxerr.Wrapf(ctx, err, "enabling app reports on %d default policy", android.DefaultAndroidPolicyID)
	}
	return nil
}

func (svc *Service) PatchDevice(ctx context.Context, policyID, deviceName string, device *androidmanagement.Device) (skip bool, apiErr error) {
	deviceRequest, err := newAndroidDeviceRequest(policyID, deviceName, device)
	if err != nil {
		return false, ctxerr.Wrapf(ctx, err, "prepare device request %s", deviceName)
	}

	applied, apiErr := svc.androidAPIClient.EnterprisesDevicesPatch(ctx, deviceName, device)
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			deviceRequest.StatusCode = gerr.Code
		}
		deviceRequest.ErrorDetails.V = apiErr.Error()
		deviceRequest.ErrorDetails.Valid = true

		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		deviceRequest.StatusCode = http.StatusOK
		deviceRequest.AppliedPolicyVersion.V = applied.AppliedPolicyVersion
		deviceRequest.AppliedPolicyVersion.Valid = true
	}

	if err := svc.fleetDS.NewAndroidPolicyRequest(ctx, deviceRequest); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save android device request")
	}
	return skip, nil
}

func (svc *Service) PatchPolicy(ctx context.Context, policyID, policyName string,
	policy *androidmanagement.Policy, metadata map[string]string,
) (skip bool, err error) {
	policyRequest, err := newAndroidPolicyRequest(policyID, policyName, policy, metadata)
	if err != nil {
		return false, ctxerr.Wrapf(ctx, err, "prepare policy request %s", policyName)
	}

	applied, apiErr := svc.androidAPIClient.EnterprisesPoliciesPatch(ctx, policyName, policy, androidmgmt.PoliciesPatchOpts{ExcludeApps: true})
	if apiErr != nil {
		var gerr *googleapi.Error
		if errors.As(apiErr, &gerr) {
			policyRequest.StatusCode = gerr.Code
		}
		policyRequest.ErrorDetails.V = apiErr.Error()
		policyRequest.ErrorDetails.Valid = true

		// Note that from my tests, the "not modified" error is not reliable, the
		// AMAPI happily returned 200 even if the policy was the same (as
		// confirmed by the same version number being returned), so we do check
		// for this error, but do not build critical logic on top of it.
		//
		// Tests do show that the version number is properly incremented when the
		// policy changes, though.
		if skip = androidmgmt.IsNotModifiedError(apiErr); skip {
			apiErr = nil
		}
	} else {
		policyRequest.StatusCode = http.StatusOK
		policyRequest.PolicyVersion.V = applied.Version
		policyRequest.PolicyVersion.Valid = true
	}

	if err := svc.fleetDS.NewAndroidPolicyRequest(ctx, policyRequest); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save android policy request")
	}
	return skip, nil
}

func (svc *Service) MigrateToPerDevicePolicy(ctx context.Context) error {
	hosts, err := svc.fleetDS.ListAndroidEnrolledDevicesForReconcile(ctx)
	if err != nil {
		return err
	}

	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		if fleet.IsNotFound(err) {
			// Android MDM is not on, so no-op
			return nil
		}
		return err
	}

	for _, h := range hosts {
		if h.AppliedPolicyID != nil && *h.AppliedPolicyID == "1" {
			var policy androidmanagement.Policy

			policy.StatusReportingSettings = &androidmanagement.StatusReportingSettings{
				DeviceSettingsEnabled:        true,
				MemoryInfoEnabled:            true,
				NetworkInfoEnabled:           true,
				DisplayInfoEnabled:           true,
				PowerManagementEventsEnabled: true,
				HardwareStatusEnabled:        true,
				SystemPropertiesEnabled:      true,
				SoftwareInfoEnabled:          true,
				CommonCriteriaModeEnabled:    true,
				ApplicationReportsEnabled:    true,
				ApplicationReportingSettings: nil, // only option is "includeRemovedApps", which I opted not to enable (we can diff apps to see removals)
			}

			if h.EnterpriseSpecificID != nil {

				policyName := fmt.Sprintf("%s/policies/%s", enterprise.Name(), *h.EnterpriseSpecificID)
				_, err := svc.PatchPolicy(ctx, *h.EnterpriseSpecificID, policyName, &policy, nil)
				if err != nil {
					return err
				}
				device := &androidmanagement.Device{
					PolicyName: policyName,
					// State must be specified when updating a device, otherwise it fails with
					// "Illegal state transition from ACTIVE to DEVICE_STATE_UNSPECIFIED"
					//
					// > Note that when calling enterprises.devices.patch, ACTIVE and
					// > DISABLED are the only allowable values.

					// TODO(ap): should we send whatever the previous state was? If it was DISABLED,
					// we probably don't want to re-enable it by accident. Those are the only
					// 2 valid states when patching a device.
					State: "ACTIVE",
				}
				androidHost, err := svc.ds.AndroidHostLiteByHostUUID(ctx, *h.EnterpriseSpecificID)
				if err != nil {
					return ctxerr.Wrapf(ctx, err, "get android host by host UUID %s", *h.EnterpriseSpecificID)
				}
				deviceName := fmt.Sprintf("%s/devices/%s", enterprise.Name(), androidHost.DeviceID)
				_, err = svc.PatchDevice(ctx, *h.EnterpriseSpecificID, deviceName, device)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// BuildAndSendFleetAgentConfig builds the complete AgentManagedConfiguration for the given hosts
// (including certificate templates) and sends it to the Android Management API.
//
// This function uses a state machine approach with the following states:
// - pending: Record exists, waiting for cron to process
// - delivering: Cron is actively sending to AMAPI
// - delivered: AMAPI confirmed receipt
func (svc *Service) BuildAndSendFleetAgentConfig(ctx context.Context, enterpriseName string, hostUUIDs []string, skipHostsWithoutNewCerts bool) error {
	if len(hostUUIDs) == 0 {
		return nil
	}

	appConfig, err := svc.fleetDS.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}

	// Cache enroll secrets by team ID to avoid repeated lookups
	enrollSecretsCache := make(map[uint][]*fleet.EnrollSecret)

	// Helper function to get enroll secrets with caching
	getEnrollSecretsForTeam := func(teamID *uint) ([]*fleet.EnrollSecret, error) {
		cacheKey := uint(0)
		if teamID != nil {
			cacheKey = *teamID
		}

		if secrets, ok := enrollSecretsCache[cacheKey]; ok {
			return secrets, nil
		}

		secrets, err := svc.fleetDS.GetEnrollSecrets(ctx, teamID)
		if err != nil {
			return nil, err
		}

		enrollSecretsCache[cacheKey] = secrets
		return secrets, nil
	}

	// Helper to build config for a single host using pre-fetched certificate templates
	buildHostConfig := func(hostUUID string, templates []fleet.HostCertificateTemplateForDelivery) (*android.AgentManagedConfiguration, error) {
		androidHost, err := svc.ds.AndroidHostLiteByHostUUID(ctx, hostUUID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "get android host %s", hostUUID)
		}

		enrollSecrets, err := getEnrollSecretsForTeam(androidHost.Host.TeamID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "get enroll secrets for team %v", androidHost.Host.TeamID)
		}
		if len(enrollSecrets) == 0 {
			return nil, ctxerr.Errorf(ctx, "no enroll secrets found for team %v", androidHost.Host.TeamID)
		}

		var certificateTemplateIDs []android.AgentCertificateTemplate
		for _, ct := range templates {
			certificateTemplateIDs = append(certificateTemplateIDs, android.AgentCertificateTemplate{
				ID:        ct.CertificateTemplateID,
				Status:    string(ct.Status),
				Operation: string(ct.OperationType),
				Version:   ct.Version,
			})
		}

		return &android.AgentManagedConfiguration{
			ServerURL:              appConfig.ServerSettings.ServerURL,
			HostUUID:               hostUUID,
			EnrollSecret:           enrollSecrets[0].Secret,
			CertificateTemplateIDs: certificateTemplateIDs,
		}, nil
	}

	for _, hostUUID := range hostUUIDs {
		// Step 1: Get all install templates and transition pending → delivering (atomically)
		// This prevents concurrent cron runs from processing the same templates
		certTemplates, err := svc.fleetDS.GetAndTransitionCertificateTemplatesToDelivering(ctx, hostUUID)
		if err != nil {
			level.Error(svc.logger).Log("msg", "failed to get and transition to delivering", "host_uuid", hostUUID, "err", err)
			return ctxerr.Wrapf(ctx, err, "get and transition certificate templates to delivering for host %s", hostUUID)
		}

		if len(certTemplates.DeliveringTemplateIDs) == 0 {
			// No pending templates for this host (another process got them, or none exist)
			if skipHostsWithoutNewCerts {
				continue
			}
			// Send config without new certificates (needed for new host enrollment)
			// There should be no other certificates either, but including them just in case.
			config, err := buildHostConfig(hostUUID, certTemplates.Templates)
			if err != nil {
				level.Error(svc.logger).Log("msg", "failed to build host config without certs", "host_uuid", hostUUID, "err", err)
				return ctxerr.Wrapf(ctx, err, "build host config without certs for host %s", hostUUID)
			}
			hostConfigs := map[string]android.AgentManagedConfiguration{hostUUID: *config}
			if err := svc.AddFleetAgentToAndroidPolicy(ctx, enterpriseName, hostConfigs); err != nil {
				level.Error(svc.logger).Log("msg", "failed to send AMAPI config without certs", "host_uuid", hostUUID, "err", err)
				// Not a critical failure. We will retry installing Fleet Agent when certificates are added to the host's team
			}
			continue
		}

		// Step 2: Build and send config to AMAPI with ALL certificate templates
		config, err := buildHostConfig(hostUUID, certTemplates.Templates)
		if err != nil {
			level.Error(svc.logger).Log("msg", "failed to build host config", "host_uuid", hostUUID, "err", err)
			return ctxerr.Wrapf(ctx, err, "build host config for %s", hostUUID)
		}

		hostConfigs := map[string]android.AgentManagedConfiguration{hostUUID: *config}
		if err := svc.AddFleetAgentToAndroidPolicy(ctx, enterpriseName, hostConfigs); err != nil {
			// AMAPI call failed, revert to pending for retry later
			level.Error(svc.logger).Log("msg", "failed to send to AMAPI", "host_uuid", hostUUID, "err", err)
			if revertErr := svc.fleetDS.RevertHostCertificateTemplatesToPending(ctx, hostUUID, certTemplates.DeliveringTemplateIDs); revertErr != nil {
				level.Error(svc.logger).Log("msg", "failed to revert to pending after AMAPI failure", "host_uuid", hostUUID, "err", revertErr)
				return ctxerr.Wrapf(ctx, revertErr, "revert certificate templates to pending after AMAPI failure for host %s", hostUUID)
			}
			continue
		}

		// Step 3: AMAPI succeeded - generate challenges for each newly delivering template
		// Note: Android app may try to fetch the certificate, but status is still delivering and no challenge is generated yet.
		// The app will retry until status turns to delivered.
		challenges := make(map[uint]string)
		for _, templateID := range certTemplates.DeliveringTemplateIDs {
			challenge, err := svc.fleetDS.NewChallenge(ctx)
			if err != nil {
				level.Error(svc.logger).Log("msg", "failed to generate challenge", "host_uuid", hostUUID, "template_id", templateID, "err", err)
				return ctxerr.Wrapf(ctx, err, "generate challenge for %s", hostUUID)
			}
			challenges[templateID] = challenge
		}

		// Step 4: Transition delivering → delivered with challenges
		if err := svc.fleetDS.TransitionCertificateTemplatesToDelivered(ctx, hostUUID, challenges); err != nil {
			level.Error(svc.logger).Log("msg", "failed to transition to delivered", "host_uuid", hostUUID, "err", err)
			return ctxerr.Wrap(ctx, err, "transition certificate templates to delivered")
		}
	}

	return nil
}

// SetAppsForAndroidPolicy sets the available apps for the given hosts' Android MDM policy to the given list of apps.
// Note that unlike AddAppsToAndroidPolicy, this method replaces the existing app list with the given one, it is
// not additive/PATCH semantics.
func (svc *Service) SetAppsForAndroidPolicy(ctx context.Context, enterpriseName string, appPolicies []*androidmanagement.ApplicationPolicy, hostUUIDs map[string]string) error {
	var errs []error
	for uuid, policyID := range hostUUIDs {
		policyName := fmt.Sprintf("%s/policies/%s", enterpriseName, policyID)
		policyRequest, err := newAndroidPolicyApplicationsRequest(policyID, policyName, appPolicies)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "prepare policy request %s", policyName)
		}

		var apiErr error
		policy := &androidmanagement.Policy{Applications: appPolicies}
		policy, apiErr = svc.androidAPIClient.EnterprisesPoliciesPatch(ctx, policyName, policy, androidmgmt.PoliciesPatchOpts{OnlyUpdateApps: true})
		if _, err := recordAndroidRequestResult(ctx, svc.fleetDS, policyRequest, policy, nil, apiErr); err != nil {
			return ctxerr.Wrapf(ctx, err, "save android policy request for host %s", uuid)
		}

		if apiErr != nil {
			errs = append(errs, ctxerr.Wrapf(ctx, apiErr, "google api: modify policy applications for host %s", uuid))
		}
	}
	return errors.Join(errs...)
}
