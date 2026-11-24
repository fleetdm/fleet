package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type createStagedUploadRequest struct {
	Target  fleet.StagedUploadTarget `json:"target"`
	FleetID uint                     `json:"fleet_id"`
	Size    int64                    `json:"size"`
}

type createStagedUploadResponse struct {
	*fleet.StagedUpload
	Err error `json:"error,omitempty"`
}

func (r createStagedUploadResponse) Error() error { return r.Err }

func createStagedUploadEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createStagedUploadRequest)
	upload, err := svc.CreateStagedUpload(ctx, req.Target, req.FleetID, req.Size)
	if err != nil {
		return createStagedUploadResponse{Err: err}, nil
	}
	return createStagedUploadResponse{StagedUpload: upload}, nil
}

func (svc *Service) CreateStagedUpload(ctx context.Context, target fleet.StagedUploadTarget, teamID uint, size int64) (*fleet.StagedUpload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
