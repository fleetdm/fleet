package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"google.golang.org/api/androidmanagement/v1"
)

// zeroTouchTokenDuration is ~100 years in seconds.
const zeroTouchTokenDuration = "3153600000s"

func zeroTouchConfigurationEndpoint(ctx context.Context, _ any, svc android.Service) fleet.Errorer {
	resp, err := svc.GetZeroTouchConfiguration(ctx, nil)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return resp
}

type teamEnrollmentRequest struct {
	TeamID *uint `json:"team_id"`
}

func (svc *Service) GetZeroTouchConfiguration(ctx context.Context, teamID *uint) (*android.ZeroTouchConfigurationResponse, error) {
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
	_ = svc.androidAPIClient.SetAuthenticationSecret(secret)

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
	amapiToken, err = svc.androidAPIClient.EnterprisesEnrollmentTokensCreate(ctx, enterprise.Name(), amapiToken)
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

// One of two expected to resolve team properly
//   - {"team_id": <uint|null>} team_id key must be present
//   - {"EnrollSecret":"...", "IdpUUID":"..."}
func (svc *Service) resolveTeamFromEnrollmentData(ctx context.Context, enrollmentTokenData string) (teamID *uint, idpUUID string, err error) {
	var raw map[string]json.RawMessage
	if jsonErr := json.Unmarshal([]byte(enrollmentTokenData), &raw); jsonErr == nil {
		if _, hasTeamID := raw["team_id"]; hasTeamID {
			var ztData teamEnrollmentRequest
			if err := json.Unmarshal([]byte(enrollmentTokenData), &ztData); err != nil {
				return nil, "", ctxerr.Wrap(ctx, err, "unmarshalling zero-touch additional data")
			}
			// fall back to unassigned if not
			if ztData.TeamID != nil {
				exists, err := svc.fleetDS.TeamExists(ctx, *ztData.TeamID)
				if err != nil {
					return nil, "", ctxerr.Wrap(ctx, err, "checking if team exists")
				}
				if !exists {
					svc.logger.WarnContext(ctx, "zero-touch team_id does not exist, assigning to unassigned",
						"team_id", *ztData.TeamID)
					return nil, "", nil
				}
			}
			return ztData.TeamID, "", nil
		}
	}

	var etReq enrollmentTokenRequest
	if err := json.Unmarshal([]byte(enrollmentTokenData), &etReq); err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "unmarshalling enrollment token data")
	}
	enrollSecret, err := svc.ds.VerifyEnrollSecret(ctx, etReq.EnrollSecret)
	if err != nil {
		return nil, "", ctxerr.Wrap(ctx, err, "verifying enroll secret")
	}
	return enrollSecret.GetTeamID(), etReq.IdpUUID, nil
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
