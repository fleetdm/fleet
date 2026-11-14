package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//////////////////////////////////////////////////////////////////////////////
// Get patch deadline settings
//////////////////////////////////////////////////////////////////////////////

type getPatchDeadlineSettingsRequest struct {
	FleetID *uint `query:"fleet_id,optional"`
}

type patchDeadlineSettingsResponse struct {
	PatchDeadlineSettings *fleet.PatchDeadlineSettings `json:"patch_deadline_settings,omitempty"`
	Err                   error                        `json:"error,omitempty"`
}

func (r patchDeadlineSettingsResponse) Error() error { return r.Err }

func getPatchDeadlineSettingsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getPatchDeadlineSettingsRequest)
	settings, err := svc.GetPatchDeadlineSettings(ctx, req.FleetID)
	if err != nil {
		return patchDeadlineSettingsResponse{Err: err}, nil
	}
	return patchDeadlineSettingsResponse{PatchDeadlineSettings: settings}, nil
}

func (svc *Service) GetPatchDeadlineSettings(ctx context.Context, _ *uint) (*fleet.PatchDeadlineSettings, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Update patch deadline settings
//////////////////////////////////////////////////////////////////////////////

type updatePatchDeadlineSettingsRequest struct {
	FleetID *uint `json:"fleet_id"`
	Days    any   `json:"days"`
	Time    any   `json:"time"`
}

func updatePatchDeadlineSettingsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*updatePatchDeadlineSettingsRequest)
	settings, err := svc.UpdatePatchDeadlineSettings(ctx, req.FleetID, req.Days, req.Time)
	if err != nil {
		return patchDeadlineSettingsResponse{Err: err}, nil
	}
	return patchDeadlineSettingsResponse{PatchDeadlineSettings: settings}, nil
}

func (svc *Service) UpdatePatchDeadlineSettings(ctx context.Context, _ *uint, _, _ any) (*fleet.PatchDeadlineSettings, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
