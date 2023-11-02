package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listSoftwareRequest struct {
	fleet.SoftwareListOptions
}

type listSoftwareResponse struct {
	CountsUpdatedAt *time.Time       `json:"counts_updated_at"`
	Software        []fleet.Software `json:"software,omitempty"`
	Err             error            `json:"error,omitempty"`
}

func (r listSoftwareResponse) error() error { return r.Err }

func listSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listSoftwareRequest)
	resp, err := svc.ListSoftware(ctx, req.SoftwareListOptions)
	if err != nil {
		return listSoftwareResponse{Err: err}, nil
	}

	// calculate the latest counts_updated_at
	var latest time.Time
	for _, sw := range resp {
		if !sw.CountsUpdatedAt.IsZero() && sw.CountsUpdatedAt.After(latest) {
			latest = sw.CountsUpdatedAt
		}
	}
	listResp := listSoftwareResponse{Software: resp}
	if !latest.IsZero() {
		listResp.CountsUpdatedAt = &latest
	}

	return listResp, nil
}

func (svc *Service) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// default sort order to hosts_count descending
	if opt.ListOptions.OrderKey == "" {
		opt.ListOptions.OrderKey = "hosts_count"
		opt.ListOptions.OrderDirection = fleet.OrderDescending
	}
	opt.WithHostCounts = true

	softwares, err := svc.ds.ListSoftware(ctx, opt)
	if err != nil {
		return nil, err
	}

	return softwares, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get Software
/////////////////////////////////////////////////////////////////////////////////

type getSoftwareRequest struct {
	ID uint `url:"id"`
}

type getSoftwareResponse struct {
	Software *fleet.Software `json:"software,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r getSoftwareResponse) error() error { return r.Err }

func getSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareRequest)

	software, err := svc.SoftwareByID(ctx, req.ID, false)
	if err != nil {
		return getSoftwareResponse{Err: err}, nil
	}

	return getSoftwareResponse{Software: software}, nil
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint, includeCVEScores bool) (*fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	software, err := svc.ds.SoftwareByID(ctx, id, includeCVEScores)
	if err != nil {
		return nil, err
	}

	return software, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Count
/////////////////////////////////////////////////////////////////////////////////

type countSoftwareRequest struct {
	fleet.SoftwareListOptions
}

type countSoftwareResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r countSoftwareResponse) error() error { return r.Err }

func countSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*countSoftwareRequest)
	count, err := svc.CountSoftware(ctx, req.SoftwareListOptions)
	if err != nil {
		return countSoftwareResponse{Err: err}, nil
	}
	return countSoftwareResponse{Count: count}, nil
}

func (svc Service) CountSoftware(ctx context.Context, opt fleet.SoftwareListOptions) (int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return 0, err
	}

	return svc.ds.CountSoftware(ctx, opt)
}
