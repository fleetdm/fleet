package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// List Software Titles
/////////////////////////////////////////////////////////////////////////////////

type listSoftwareTitlesRequest struct {
	fleet.SoftwareTitleListOptions
}

type listSoftwareTitlesResponse struct {
	Meta            *fleet.PaginationMetadata `json:"meta"`
	Count           int                       `json:"count"`
	CountsUpdatedAt *time.Time                `json:"counts_updated_at"`
	SoftwareTitles  []fleet.SoftwareTitle     `json:"software_titles,omitempty"`
	Err             error                     `json:"error,omitempty"`
}

func (r listSoftwareTitlesResponse) error() error { return r.Err }

func listSoftwareTitlesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listSoftwareTitlesRequest)
	titles, count, meta, err := svc.ListSoftwareTitles(ctx, req.SoftwareTitleListOptions)
	if err != nil {
		return listSoftwareTitlesResponse{Err: err}, nil
	}

	var latest time.Time
	for _, sw := range titles {
		if !sw.CountsUpdatedAt.IsZero() && sw.CountsUpdatedAt.After(latest) {
			latest = sw.CountsUpdatedAt
		}
	}
	listResp := listSoftwareTitlesResponse{
		SoftwareTitles: titles,
		Count:          count,
		Meta:           meta,
	}
	if !latest.IsZero() {
		listResp.CountsUpdatedAt = &latest
	}

	return listResp, nil
}

func (svc *Service) ListSoftwareTitles(
	ctx context.Context,
	opt fleet.SoftwareTitleListOptions,
) ([]fleet.SoftwareTitle, int, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, 0, nil, err
	}

	if opt.TeamID != nil && *opt.TeamID != 0 {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, 0, nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, 0, nil, fleet.ErrMissingLicense
		}
	}

	// always include metadata for software titles
	opt.ListOptions.IncludeMetadata = true
	// cursor-based pagination is not supported for software titles
	opt.ListOptions.After = ""

	titles, count, meta, err := svc.ds.ListSoftwareTitles(ctx, opt)
	if err != nil {
		return nil, 0, nil, err
	}

	return titles, count, meta, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get a Software Title
/////////////////////////////////////////////////////////////////////////////////

type getSoftwareTitleRequest struct {
	ID uint `url:"id"`
}

type getSoftwareTitleResponse struct {
	SoftwareTitle *fleet.SoftwareTitle `json:"software_title,omitempty"`
	Err           error                `json:"error,omitempty"`
}

func (r getSoftwareTitleResponse) error() error { return r.Err }

func getSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareTitleRequest)

	software, err := svc.SoftwareTitleByID(ctx, req.ID)
	if err != nil {
		return getSoftwareTitleResponse{Err: err}, nil
	}

	return getSoftwareTitleResponse{SoftwareTitle: software}, nil
}

func (svc *Service) SoftwareTitleByID(ctx context.Context, id uint) (*fleet.SoftwareTitle, error) {
	// TODO: this is the autorization we do for GET /software, does it look right?
	// checking with product here: https://github.com/fleetdm/fleet/issues/14674#issuecomment-1841395788
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	software, err := svc.ds.SoftwareTitleByID(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software title by id")
	}

	return software, nil
}
