package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// List Hosts
////////////////////////////////////////////////////////////////////////////////

type listHostsRequest struct {
	Opts fleet.HostListOptions `url:"host_options"`
}

type listHostsResponse struct {
	Hosts    []HostResponse  `json:"hosts"`
	Software *fleet.Software `json:"software,omitempty"`
	Err      error           `json:"error,omitempty"`
}

func (r listHostsResponse) error() error { return r.Err }

func listHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listHostsRequest)
	hosts, err := svc.ListHosts(ctx, req.Opts)
	if err != nil {
		return listHostsResponse{Err: err}, nil
	}

	var software *fleet.Software
	if req.Opts.SoftwareIDFilter != nil {
		software, err = svc.SoftwareByID(ctx, *req.Opts.SoftwareIDFilter)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}
	hostResponses := make([]HostResponse, len(hosts))
	for i, host := range hosts {
		h, err := hostResponseForHost(ctx, svc, host)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}

		hostResponses[i] = *h
	}
	return listHostsResponse{Hosts: hostResponses, Software: software}, nil
}

func (svc Service) ListHosts(ctx context.Context, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.ListHosts(ctx, filter, opt)
}

func (svc Service) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	return svc.ds.SoftwareByID(ctx, id)
}

/////////////////////////////////////////////////////////////////////////////////
// Delete
/////////////////////////////////////////////////////////////////////////////////

type deleteHostsRequest struct {
	IDs     []uint `json:"ids"`
	Filters struct {
		MatchQuery string           `json:"query"`
		Status     fleet.HostStatus `json:"status"`
		LabelID    *uint            `json:"label_id"`
		TeamID     *uint            `json:"team_id"`
	} `json:"filters"`
}

type deleteHostsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteHostsResponse) error() error { return r.Err }

func deleteHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteHostsRequest)
	listOpt := fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			MatchQuery: req.Filters.MatchQuery,
		},
		StatusFilter: req.Filters.Status,
		TeamFilter:   req.Filters.TeamID,
	}
	err := svc.DeleteHosts(ctx, req.IDs, listOpt, req.Filters.LabelID)
	if err != nil {
		return deleteHostsResponse{Err: err}, nil
	}
	return deleteHostsResponse{}, nil
}

func (svc Service) DeleteHosts(ctx context.Context, ids []uint, opts fleet.HostListOptions, lid *uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	if len(ids) > 0 && (lid != nil || !opts.Empty()) {
		return &badRequestError{"Cannot specify a list of ids and filters at the same time"}
	}

	if len(ids) > 0 {
		err := svc.checkWriteForHostIDs(ctx, ids)
		if err != nil {
			return err
		}
		return svc.ds.DeleteHosts(ctx, ids)
	}

	hostIDs, err := svc.hostIDsFromFilters(ctx, opts, lid)
	if err != nil {
		return err
	}

	if len(hostIDs) == 0 {
		return nil
	}

	err = svc.checkWriteForHostIDs(ctx, hostIDs)
	if err != nil {
		return err
	}
	return svc.ds.DeleteHosts(ctx, hostIDs)
}

/////////////////////////////////////////////////////////////////////////////////
// Count
/////////////////////////////////////////////////////////////////////////////////

type countHostsRequest struct {
	Opts    fleet.HostListOptions `url:"host_options"`
	LabelID *uint                 `query:"label_id,optional"`
}

type countHostsResponse struct {
	Count int   `json:"count"`
	Err   error `json:"error,omitempty"`
}

func (r countHostsResponse) error() error { return r.Err }

func countHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*countHostsRequest)
	count, err := svc.CountHosts(ctx, req.LabelID, req.Opts)
	if err != nil {
		return countHostsResponse{Err: err}, nil
	}
	return countHostsResponse{Count: count}, nil
}

func (svc Service) CountHosts(ctx context.Context, labelID *uint, opts fleet.HostListOptions) (int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return 0, err
	}

	return svc.countHostFromFilters(ctx, labelID, opts)
}

/////////////////////////////////////////////////////////////////////////////////
// Get host
/////////////////////////////////////////////////////////////////////////////////

func getHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getHostRequest)
	host, err := svc.GetHost(ctx, req.ID)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, host)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	return getHostResponse{Host: resp}, nil
}

func (svc Service) checkWriteForHostIDs(ctx context.Context, ids []uint) error {
	for _, id := range ids {
		host, err := svc.ds.Host(ctx, id, false)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host for delete")
		}

		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
			return err
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host Summary
////////////////////////////////////////////////////////////////////////////////

type getHostSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getHostSummaryResponse struct {
	fleet.HostSummary
	Err error `json:"error,omitempty"`
}

func (r getHostSummaryResponse) error() error { return r.Err }

func getHostSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getHostSummaryRequest)
	summary, err := svc.GetHostSummary(ctx, req.TeamID)
	if err != nil {
		return getHostSummaryResponse{Err: err}, nil
	}

	resp := getHostSummaryResponse{
		HostSummary: *summary,
	}
	return resp, nil
}

func (svc *Service) GetHostSummary(ctx context.Context, teamID *uint) (*fleet.HostSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID}

	summary, err := svc.ds.GenerateHostStatusStatistics(ctx, filter, svc.clock.Now())
	if err != nil {
		return nil, err
	}
	return summary, nil
}
