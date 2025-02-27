package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/proxy"
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
	logger   kitlog.Logger
	authz    *authz.Authorizer
	ds       android.Datastore
	fleetDS  fleet.Datastore
	proxy    android.Proxy
	fleetSvc fleet.Service

	// SignupSSEInterval can be overwritten in tests.
	SignupSSEInterval time.Duration
}

func NewService(
	ctx context.Context,
	logger kitlog.Logger,
	fleetDS fleet.Datastore,
	fleetSvc fleet.Service,
) (android.Service, error) {
	prx := proxy.NewProxy(ctx, logger)
	return NewServiceWithProxy(logger, fleetDS, prx, fleetSvc)
}

func NewServiceWithProxy(
	logger kitlog.Logger,
	fleetDS fleet.Datastore,
	proxy android.Proxy,
	fleetSvc fleet.Service,
) (android.Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}

	return &Service{
		logger:            logger,
		authz:             authorizer,
		ds:                fleetDS.GetAndroidDS(),
		fleetDS:           fleetDS,
		proxy:             proxy,
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

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	id, err := svc.ds.CreateEnterprise(ctx, vc.User.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	callbackURL := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/%d/connect", appConfig.ServerSettings.ServerURL, id)
	signupDetails, err := svc.proxy.SignupURLsCreate(callbackURL)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating signup url")
	}

	err = svc.ds.UpdateEnterprise(ctx, &android.EnterpriseDetails{
		Enterprise: android.Enterprise{
			ID: id,
		},
		SignupName: signupDetails.Name,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	return signupDetails, nil
}

func (svc *Service) checkIfAndroidAlreadyConfigured(ctx context.Context) (*fleet.AppConfig, error) {
	appConfig, err := svc.fleetDS.AppConfig(ctx)
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
	ID              uint   `url:"id"`
	EnterpriseToken string `query:"enterpriseToken"`
}

func enterpriseSignupCallbackEndpoint(ctx context.Context, request interface{}, svc android.Service) fleet.Errorer {
	req := request.(*enterpriseSignupCallbackRequest)
	err := svc.EnterpriseSignupCallback(ctx, req.ID, req.EnterpriseToken)
	return android.DefaultResponse{Err: err}
}

func (svc *Service) EnterpriseSignupCallback(ctx context.Context, id uint, enterpriseToken string) error {
	// Skip authorization because the callback is called by Google.
	// TODO(26218): Add some authorization here so random people can't bind random Android enterprises just for fun.
	// This call will fail if Proxy (Google Project) is not configured.
	svc.authz.SkipAuthorization(ctx)

	appConfig, err := svc.checkIfAndroidAlreadyConfigured(ctx)
	if err != nil {
		return err
	}

	enterprise, err := svc.ds.GetEnterpriseByID(ctx, id)
	switch {
	case fleet.IsNotFound(err):
		return fleet.NewInvalidArgumentError("id",
			fmt.Sprintf("Enterprise with ID %d not found", id)).WithStatus(http.StatusNotFound)
	case err != nil:
		return ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	// pubSubToken is used to authenticate the pubsub push endpoint -- to ensure that the push came from our Android enterprise
	pubSubToken, err := server.GenerateRandomURLSafeText(64)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating pubsub token")
	}
	err = svc.fleetDS.InsertOrReplaceMDMConfigAsset(ctx, fleet.MDMConfigAsset{
		Name:  fleet.MDMAssetAndroidPubSubToken,
		Value: []byte(pubSubToken),
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting pubsub authentication token")
	}

	name, topicName, err := svc.proxy.EnterprisesCreate(
		ctx,
		android.ProxyEnterprisesCreateRequest{
			Enterprise: androidmanagement.Enterprise{
				EnabledNotificationTypes: []string{
					string(android.PubSubEnrollment),
					string(android.PubSubStatusReport),
					string(android.PubSubCommand),
					string(android.PubSubUsageLogs),
				},
			},
			EnterpriseToken: enterpriseToken,
			SignupUrlName:   enterprise.SignupName,
			PubSubPushURL:   appConfig.ServerSettings.ServerURL + pubSubPushPath + "?token=" + pubSubToken,
		},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating enterprise")
	}

	enterpriseID := strings.TrimPrefix(name, "enterprises/")
	enterprise.EnterpriseID = enterpriseID
	topicID, err := topicIDFromName(topicName)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "parsing topic name")
	}
	enterprise.TopicID = topicID
	err = svc.ds.UpdateEnterprise(ctx, enterprise)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating enterprise")
	}

	err = svc.proxy.EnterprisesPoliciesPatch(enterprise.EnterpriseID, fmt.Sprintf("%d", defaultAndroidPolicyID), &androidmanagement.Policy{
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
			// by the fleetdm.com proxy or manually. The proxy could also enforce this report setting.
			ApplicationReportsEnabled:    false,
			ApplicationReportingSettings: nil,
		},
	})
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "patching %d policy", defaultAndroidPolicyID)
	}

	err = svc.ds.DeleteOtherEnterprises(ctx, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting temp enterprises")
	}

	err = svc.fleetDS.SetAndroidEnabledAndConfigured(ctx, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting android enabled and configured")
	}

	user, err := svc.fleetDS.UserOrDeletedUserByID(ctx, enterprise.UserID)
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
		err = svc.proxy.EnterpriseDelete(enterprise.EnterpriseID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting enterprise via Google API")
		}
	}

	err = svc.ds.DeleteAllEnterprises(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting enterprises")
	}

	err = svc.fleetDS.SetAndroidEnabledAndConfigured(ctx, false)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "clearing android enabled and configured")
	}

	if err = svc.fleetSvc.NewActivity(ctx, authz.UserFromContext(ctx), fleet.ActivityTypeDisabledAndroidMDM{}); err != nil {
		return ctxerr.Wrap(ctx, err, "create activity for disabled Android MDM")
	}

	err = svc.fleetDS.DeleteMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting pubsub token")
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

	_, err = svc.fleetDS.VerifyEnrollSecret(ctx, enrollSecret)
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

	token := &androidmanagement.EnrollmentToken{
		// Default duration is 1 hour

		AdditionalData:     enrollSecret,
		AllowPersonalUsage: "PERSONAL_USAGE_ALLOWED",
		PolicyName:         fmt.Sprintf("%s/policies/%d", enterprise.Name(), +defaultAndroidPolicyID),
		OneTimeOnly:        true,
	}
	token, err = svc.proxy.EnterprisesEnrollmentTokensCreate(enterprise.Name(), token)
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
	appConfig, err := svc.fleetDS.AppConfig(ctx)
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
	appConfig, err := svc.fleetDS.AppConfig(ctx)
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
