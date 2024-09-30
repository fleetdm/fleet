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

type getSetupExperienceSoftwareRequest struct {
	fleet.ListOptions
	TeamID uint `query:"team_id"`
}

type getSetupExperienceSoftwareResponse struct {
	SoftwareTitles []fleet.SoftwareTitleListResult `json:"software_titles"`
	Count          int                             `json:"count"`
	Meta           *fleet.PaginationMetadata       `json:"meta"`
	Err            error                           `json:"error,omitempty"`
}

func (r getSetupExperienceSoftwareResponse) error() error { return r.Err }

func getSetupExperienceSoftware(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSetupExperienceSoftwareRequest)

	titles, count, meta, err := svc.ListSetupExperienceSoftware(ctx, req.TeamID, req.ListOptions)

	if err != nil {
		return &getSetupExperienceSoftwareResponse{Err: err}, nil
	}

	return &getSetupExperienceSoftwareResponse{SoftwareTitles: titles, Count: count, Meta: meta}, nil
}

func (svc *Service) ListSetupExperienceSoftware(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, 0, nil, fleet.ErrMissingLicense
}
