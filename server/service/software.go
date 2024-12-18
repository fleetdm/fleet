package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

/////////////////////////////////////////////////////////////////////////////////
// List
/////////////////////////////////////////////////////////////////////////////////

type listSoftwareRequest struct {
	fleet.SoftwareListOptions
}

// Deprecated: listSoftwareResponse is the response struct for the deprecated
// listSoftwareEndpoint. It differs from listSoftwareVersionsResponse in that
// the latter includes a count of the total number of software items.
type listSoftwareResponse struct {
	CountsUpdatedAt *time.Time       `json:"counts_updated_at"`
	Software        []fleet.Software `json:"software,omitempty"`
	Err             error            `json:"error,omitempty"`
}

func (r listSoftwareResponse) error() error { return r.Err }

// Deprecated: use listSoftwareVersionsEndpoint instead
func listSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listSoftwareRequest)
	resp, _, err := svc.ListSoftware(ctx, req.SoftwareListOptions)
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

type listSoftwareVersionsResponse struct {
	Count           int                       `json:"count"`
	CountsUpdatedAt *time.Time                `json:"counts_updated_at"`
	Software        []fleet.Software          `json:"software,omitempty"`
	Meta            *fleet.PaginationMetadata `json:"meta"`
	Err             error                     `json:"error,omitempty"`
}

func (r listSoftwareVersionsResponse) error() error { return r.Err }

func listSoftwareVersionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listSoftwareRequest)

	// always include pagination for new software versions endpoint (not included by default in
	// legacy endpoint for backwards compatibility)
	req.SoftwareListOptions.ListOptions.IncludeMetadata = true

	resp, meta, err := svc.ListSoftware(ctx, req.SoftwareListOptions)
	if err != nil {
		return listSoftwareVersionsResponse{Err: err}, nil
	}

	// calculate the latest counts_updated_at
	var latest time.Time
	for _, sw := range resp {
		if !sw.CountsUpdatedAt.IsZero() && sw.CountsUpdatedAt.After(latest) {
			latest = sw.CountsUpdatedAt
		}
	}
	listResp := listSoftwareVersionsResponse{Software: resp, Meta: meta}
	if !latest.IsZero() {
		listResp.CountsUpdatedAt = &latest
	}

	c, err := svc.CountSoftware(ctx, req.SoftwareListOptions)
	if err != nil {
		return listSoftwareVersionsResponse{Err: err}, nil
	}
	listResp.Count = c

	return listResp, nil
}

func (svc *Service) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	// Vulnerability filters are only available in premium (opt.IncludeCVEScores is only true in premium)
	lic, err := svc.License(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !lic.IsPremium() && (opt.MaximumCVSS > 0 || opt.MinimumCVSS > 0 || opt.KnownExploit) {
		return nil, nil, fleet.ErrMissingLicense
	}

	// default sort order to hosts_count descending
	if opt.ListOptions.OrderKey == "" {
		opt.ListOptions.OrderKey = "hosts_count"
		opt.ListOptions.OrderDirection = fleet.OrderDescending
	}
	opt.WithHostCounts = true

	softwares, meta, err := svc.ds.ListSoftware(ctx, opt)
	if err != nil {
		return nil, nil, err
	}

	return softwares, meta, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get Software
/////////////////////////////////////////////////////////////////////////////////

type getSoftwareRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional"`
}

type getSoftwareResponse struct {
	Software *fleet.Software `json:"software,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r getSoftwareResponse) error() error { return r.Err }

func getSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareRequest)

	software, err := svc.SoftwareByID(ctx, req.ID, req.TeamID, false)
	if err != nil {
		return getSoftwareResponse{Err: err}, nil
	}

	return getSoftwareResponse{Software: software}, nil
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint, teamID *uint, includeCVEScores bool) (*fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if teamID != nil && *teamID > 0 {
		// This auth check ensures we return 403 if the user doesn't have access to the team
		if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking if team exists")
		} else if !exists {
			return nil, fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		}
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	software, err := svc.ds.SoftwareByID(ctx, id, teamID, includeCVEScores, &fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	})
	if err != nil {
		if fleet.IsNotFound(err) && teamID == nil {
			// here we use a global admin as filter because we want
			// to check if the software version exists
			filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}

			if _, err = svc.ds.SoftwareByID(ctx, id, teamID, includeCVEScores, &filter); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "checked using a global admin")
			}

			return nil, fleet.NewPermissionError("Error: You don’t have permission to view specified software. It is installed on hosts that belong to team you don’t have permissions to view.")
		}

		return nil, ctxerr.Wrap(ctx, err, "getting software version by id")
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

// Deprecated: counts are now included directly in the listSoftwareVersionsResponse. This
// endpoint is retained for backwards compatibility.
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

	lic, err := svc.License(ctx)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "get license")
	}

	// Vulnerability filters are only available in premium
	if !lic.IsPremium() && (opt.MaximumCVSS > 0 || opt.MinimumCVSS > 0 || opt.KnownExploit) {
		return 0, fleet.ErrMissingLicense
	}

	// required for vulnerability filters
	if lic.IsPremium() {
		opt.IncludeCVEScores = true
	}

	return svc.ds.CountSoftware(ctx, opt)
}
