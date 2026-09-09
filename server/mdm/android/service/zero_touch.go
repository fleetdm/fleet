package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
)

// zeroTouchTokenDuration is ~100 years in seconds.
const zeroTouchTokenDuration = "3153600000s"

func zeroTouchConfigurationEndpoint(ctx context.Context, _ interface{}, svc android.Service) fleet.Errorer {
	resp, err := svc.GetZeroTouchConfiguration(ctx)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return resp
}

func (svc *Service) GetZeroTouchConfiguration(ctx context.Context) (*android.ZeroTouchConfigurationResponse, error) {
	// Admin-only
	if err := svc.authz.Authorize(ctx, &android.Enterprise{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Android MDM must be configured
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil, fleet.NewInvalidArgumentError("android",
			"Android MDM is NOT configured").WithStatus(http.StatusNotFound)
	}

	// Check if a token already exists
	existing, err := svc.ds.GetZeroTouchEnrollmentToken(ctx, nil)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "getting existing zero-touch token")
	}
	if existing != nil {
		return buildDPCExtrasResponse(existing), nil
	}

	// No token exists — create one via AMAPI
	enterprise, err := svc.ds.GetEnterprise(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting enterprise")
	}

	secret, err := svc.getClientAuthenticationSecret(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting client authentication secret")
	}
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

	// Get a global enroll secret to embed in the token
	enrollSecrets, err := svc.fleetDS.GetEnrollSecrets(ctx, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting global enroll secrets")
	}
	if len(enrollSecrets) == 0 {
		return nil, &fleet.BadRequestError{Message: "No global enroll secret found. Please create one before setting up zero-touch enrollment."}
	}
	enrollSecret := enrollSecrets[0].Secret

	additionalData, err := json.Marshal(enrollmentTokenRequest{
		EnrollSecret: enrollSecret,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshalling enrollment token request")
	}

	amapiToken := &androidmanagement.EnrollmentToken{
		Duration:           zeroTouchTokenDuration,
		AdditionalData:     string(additionalData),
		AllowPersonalUsage: "PERSONAL_USAGE_DISALLOWED",
		PolicyName:         fmt.Sprintf("%s/policies/%d", enterprise.Name(), android.DefaultAndroidPolicyID),
		OneTimeOnly:        false,
	}
	amapiToken, err = svc.androidAPIClient.EnterprisesEnrollmentTokensCreate(ctx, enterprise.Name(), amapiToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating zero-touch enrollment token via AMAPI")
	}

	expiresAt, err := time.Parse(time.RFC3339, amapiToken.ExpirationTimestamp)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing token expiration timestamp")
	}

	token := &android.ZeroTouchToken{
		TokenName:    amapiToken.Name,
		TokenValue:   amapiToken.Value,
		EnrollSecret: enrollSecret,
		ExpiresAt:    expiresAt,
	}
	token, err = svc.ds.CreateZeroTouchEnrollmentToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "persisting zero-touch enrollment token")
	}

	return buildDPCExtrasResponse(token), nil
}

func buildDPCExtrasResponse(token *android.ZeroTouchToken) *android.ZeroTouchConfigurationResponse {
	dpcExtras := fmt.Sprintf(`{
  "android.app.extra.PROVISIONING_ADMIN_EXTRAS_BUNDLE": {
    "com.google.android.apps.work.clouddpc.EXTRA_ENROLLMENT_TOKEN": %q
  }
}`, token.TokenValue)

	return &android.ZeroTouchConfigurationResponse{
		DPCExtras: dpcExtras,
		ExpiresAt: token.ExpiresAt.Format(time.RFC3339),
	}
}
