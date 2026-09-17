package android

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"google.golang.org/api/androidmanagement/v1"
)

var _ android.Service = (*Service)(nil)

// zeroTouchTokenDuration is ~100 years in seconds.
const zeroTouchTokenDuration = "3153600000s"

// Service wraps a core android.Service with premium feature implementations.
type Service struct {
	android.Service
	ds        fleet.AndroidDatastore
	fleetDS   fleet.Datastore
	apiClient androidmgmt.Client
	authz     *authz.Authorizer
	logger    *slog.Logger
}

func NewService(
	svc android.Service,
	ds fleet.AndroidDatastore,
	fleetDS fleet.Datastore,
	apiClient androidmgmt.Client,
	logger *slog.Logger,
) (*Service, error) {
	authorizer, err := authz.NewAuthorizer()
	if err != nil {
		return nil, fmt.Errorf("new authorizer: %w", err)
	}
	return &Service{
		Service:   svc,
		ds:        ds,
		fleetDS:   fleetDS,
		apiClient: apiClient,
		authz:     authorizer,
		logger:    logger,
	}, nil
}

// teamEnrollmentRequest is the payload embedded in AMAPI enrollment token additionalData.
type teamEnrollmentRequest struct {
	TeamID *uint `json:"fleet_id"`
}

func (svc *Service) GetZeroTouchConfiguration(ctx context.Context, teamID *uint) (*android.ZeroTouchConfigurationResponse, error) {
	if !licensectx.IsPremium(ctx) {
		return nil, fleet.ErrMissingLicense
	}

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

	ctx = ctxdb.RequirePrimary(ctx, true)

	// Check if a token already exists
	existing, err := svc.ds.GetZeroTouchEnrollmentToken(ctx, teamID)
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
	_ = svc.apiClient.SetAuthenticationSecret(secret)

	additionalData, err := json.Marshal(teamEnrollmentRequest{
		TeamID: teamID,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshalling zero-touch additional data")
	}

	amapiToken := &androidmanagement.EnrollmentToken{
		Duration:           zeroTouchTokenDuration,
		AdditionalData:     string(additionalData),
		AllowPersonalUsage: "PERSONAL_USAGE_DISALLOWED",
		PolicyName:         fmt.Sprintf("%s/policies/%d", enterprise.Name(), android.DefaultAndroidPolicyID),
		OneTimeOnly:        false,
	}
	amapiToken, err = svc.apiClient.EnterprisesEnrollmentTokensCreate(ctx, enterprise.Name(), amapiToken)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating zero-touch enrollment token via AMAPI")
	}

	expiresAt, err := time.Parse(time.RFC3339, amapiToken.ExpirationTimestamp)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing token expiration timestamp")
	}

	token := &android.ZeroTouchToken{
		TeamID:     teamID,
		TokenName:  amapiToken.Name,
		TokenValue: amapiToken.Value,
		ExpiresAt:  expiresAt,
	}
	token, err = svc.ds.CreateZeroTouchEnrollmentToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "persisting zero-touch enrollment token")
	}

	return buildDPCExtrasResponse(token), nil
}

func (svc *Service) getClientAuthenticationSecret(ctx context.Context) (string, error) {
	assets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidFleetServerSecret}, nil)
	switch {
	case fleet.IsNotFound(err):
		return "", nil
	case err != nil:
		return "", ctxerr.Wrap(ctx, err, "getting Android authentication secret")
	}
	return string(assets[fleet.MDMAssetAndroidFleetServerSecret].Value), nil
}

func buildDPCExtrasResponse(token *android.ZeroTouchToken) *android.ZeroTouchConfigurationResponse {
	dpcExtras := fmt.Sprintf(`{
  "android.app.extra.PROVISIONING_DEVICE_ADMIN_COMPONENT_NAME": "com.google.android.apps.work.clouddpc/.receivers.CloudDeviceAdminReceiver",
  "android.app.extra.PROVISIONING_DEVICE_ADMIN_SIGNATURE_CHECKSUM": "I5YvS0O5hXY46mb01BlRjq4oJJGs2kuUcHvVkAPEXlg",
  "android.app.extra.PROVISIONING_ADMIN_EXTRAS_BUNDLE": {
    "com.google.android.apps.work.clouddpc.EXTRA_ENROLLMENT_TOKEN": %q
  }
}`, token.TokenValue)

	return &android.ZeroTouchConfigurationResponse{
		DPCExtras: dpcExtras,
		ExpiresAt: token.ExpiresAt.Format(time.RFC3339),
	}
}
