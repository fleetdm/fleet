package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type putSetupExperienceSoftwareRequest struct {
	TeamID   uint   `query:"team_id"`
	TitleIDs []uint `json:"title_ids"`
}

type putSetupExperienceSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r putSetupExperienceSoftwareResponse) error() error { return r.Err }

func putSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*putSetupExperienceSoftwareRequest)

	err := svc.SetSetupExperienceSoftware(ctx, req.TeamID, req.TitleIDs)

	if err != nil {
		return &putSetupExperienceSoftwareResponse{Err: err}, nil
	}

	return &putSetupExperienceSoftwareResponse{}, nil
}

func (svc *Service) SetSetupExperienceSoftware(ctx context.Context, teamID uint, titleIDs []uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
