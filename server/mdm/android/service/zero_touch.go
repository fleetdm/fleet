package service

import (
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
)

func zeroTouchConfigurationEndpoint(ctx context.Context, _ any, svc android.Service) fleet.Errorer {
	resp, err := svc.GetZeroTouchConfiguration(ctx, nil)
	if err != nil {
		return android.DefaultResponse{Err: err}
	}
	return resp
}

// GetZeroTouchConfiguration is a stub that returns ErrMissingLicense.
// The real implementation lives in ee/server/mdm/android/.
func (svc *Service) GetZeroTouchConfiguration(ctx context.Context, _ *uint) (*android.ZeroTouchConfigurationResponse, error) {
	svc.authz.SkipAuthorization(ctx)
	return nil, fleet.ErrMissingLicense
}

// teamEnrollmentRequest is the payload embedded in AMAPI enrollment token additionalData.
type teamEnrollmentRequest struct {
	TeamID *uint `json:"fleet_id"`
}

// Supports two formats:
//   - {"fleet_id": <uint|null>} — zero-touch and new QR tokens
//   - {"EnrollSecret":"...", "IdpUUID":"..."} — legacy QR tokens
//
// If the team no longer exists, falls back to unassigned (nil).
func (svc *Service) resolveTeamFromEnrollmentData(ctx context.Context, enrollmentTokenData string) (teamID *uint, idpUUID string, err error) {
	var raw map[string]json.RawMessage
	if jsonErr := json.Unmarshal([]byte(enrollmentTokenData), &raw); jsonErr == nil {
		if _, hasTeamID := raw["fleet_id"]; hasTeamID {
			var data teamEnrollmentRequest
			if err := json.Unmarshal([]byte(enrollmentTokenData), &data); err != nil {
				return nil, "", ctxerr.Wrap(ctx, err, "unmarshalling zero-touch additional data")
			}
			if data.TeamID != nil {
				exists, err := svc.fleetDS.TeamExists(ctx, *data.TeamID)
				if err != nil {
					return nil, "", ctxerr.Wrap(ctx, err, "checking if team exists")
				}
				if !exists {
					svc.logger.WarnContext(ctx, "enrollment token team does not exist, assigning to unassigned",
						"fleet_id", *data.TeamID)
					return nil, "", nil
				}
			}
			return data.TeamID, "", nil
		}
	}

	// Fall back to legacy QR enrollment format
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
