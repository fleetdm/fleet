package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gocarina/gocsv"
)

// HostResponse is the response struct that contains the full host information
// along with the host online status and the "display text" to be used when
// rendering in the UI.
type HostResponse struct {
	*fleet.Host
	Status           fleet.HostStatus   `json:"status" csv:"status"`
	DisplayText      string             `json:"display_text" csv:"display_text"`
	Labels           []fleet.Label      `json:"labels,omitempty" csv:"-"`
	Geolocation      *fleet.GeoLocation `json:"geolocation,omitempty" csv:"-"`
	CSVDeviceMapping string             `json:"-" db:"-" csv:"device_mapping"`
}

func hostResponseForHost(ctx context.Context, svc fleet.Service, host *fleet.Host) (*HostResponse, error) {
	return &HostResponse{
		Host:        host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
		Geolocation: svc.LookupGeoIP(ctx, host.PublicIP),
	}, nil
}

// HostDetailResponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	fleet.HostDetail
	Status      fleet.HostStatus   `json:"status"`
	DisplayText string             `json:"display_text"`
	Geolocation *fleet.GeoLocation `json:"geolocation,omitempty"`
}

func hostDetailResponseForHost(ctx context.Context, svc fleet.Service, host *fleet.HostDetail) (*HostDetailResponse, error) {
	return &HostDetailResponse{
		HostDetail:  *host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
		Geolocation: svc.LookupGeoIP(ctx, host.PublicIP),
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Hosts
////////////////////////////////////////////////////////////////////////////////

type listHostsRequest struct {
	Opts fleet.HostListOptions `url:"host_options"`
}

type listHostsResponse struct {
	Hosts    []HostResponse  `json:"hosts"`
	Software *fleet.Software `json:"software,omitempty"`
	// MDMSolution is populated with the MDM solution corresponding to the mdm_id
	// filter if one is provided with the request (and it exists in the
	// database). It is nil otherwise and absent of the JSON response payload.
	MDMSolution *fleet.AggregatedMDMSolutions `json:"mobile_device_management_solution,omitempty"`
	// MunkiIssue is populated with the munki issue corresponding to the
	// munki_issue_id filter if one is provided with the request (and it exists
	// in the database). It is nil otherwise and absent of the JSON response
	// payload.
	MunkiIssue *fleet.AggregatedMunkiIssue `json:"munki_issue,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r listHostsResponse) error() error { return r.Err }

func listHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listHostsRequest)

	var software *fleet.Software
	if req.Opts.SoftwareIDFilter != nil {
		var err error
		software, err = svc.SoftwareByID(ctx, *req.Opts.SoftwareIDFilter, false)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}

	var mdmSolution *fleet.AggregatedMDMSolutions
	if req.Opts.MDMIDFilter != nil {
		var err error
		mdmSolution, err = svc.AggregatedMDMSolutions(ctx, req.Opts.TeamFilter, *req.Opts.MDMIDFilter)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}

	var munkiIssue *fleet.AggregatedMunkiIssue
	if req.Opts.MunkiIssueIDFilter != nil {
		var err error
		munkiIssue, err = svc.AggregatedMunkiIssue(ctx, req.Opts.TeamFilter, *req.Opts.MunkiIssueIDFilter)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}

	hosts, err := svc.ListHosts(ctx, req.Opts)
	if err != nil {
		return listHostsResponse{Err: err}, nil
	}

	hostResponses := make([]HostResponse, len(hosts))
	for i, host := range hosts {
		h, err := hostResponseForHost(ctx, svc, host)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}

		hostResponses[i] = *h
	}
	return listHostsResponse{
		Hosts:       hostResponses,
		Software:    software,
		MDMSolution: mdmSolution,
		MunkiIssue:  munkiIssue,
	}, nil
}

func (svc *Service) AggregatedMDMSolutions(ctx context.Context, teamID *uint, mdmID uint) (*fleet.AggregatedMDMSolutions, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if teamID != nil {
		_, err := svc.ds.Team(ctx, *teamID)
		if err != nil {
			return nil, err
		}
	}

	// it is expected that there will be relatively few MDM solutions. This
	// returns the slice of all aggregated stats (one entry per mdm_id), and we
	// then iterate to return only the one that was requested (the slice is
	// stored as-is in a JSON field in the database).
	sols, _, err := svc.ds.AggregatedMDMSolutions(ctx, teamID)
	if err != nil {
		return nil, err
	}

	for _, sol := range sols {
		// don't take the address of the loop variable (although it could be ok
		// here, but just bad practice)
		sol := sol
		if sol.ID == mdmID {
			return &sol, nil
		}
	}
	return nil, nil
}

func (svc *Service) AggregatedMunkiIssue(ctx context.Context, teamID *uint, munkiIssueID uint) (*fleet.AggregatedMunkiIssue, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if teamID != nil {
		_, err := svc.ds.Team(ctx, *teamID)
		if err != nil {
			return nil, err
		}
	}

	// This returns the slice of all aggregated stats (one entry per
	// munki_issue_id), and we then iterate to return only the one that was
	// requested (the slice is stored as-is in a JSON field in the database).
	issues, _, err := svc.ds.AggregatedMunkiIssues(ctx, teamID)
	if err != nil {
		return nil, err
	}

	for _, iss := range issues {
		// don't take the address of the loop variable (although it could be ok
		// here, but just bad practice)
		iss := iss
		if iss.ID == munkiIssueID {
			return &iss, nil
		}
	}
	return nil, nil
}

func (svc *Service) ListHosts(ctx context.Context, opt fleet.HostListOptions) ([]*fleet.Host, error) {
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

/////////////////////////////////////////////////////////////////////////////////
// Delete Hosts
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

func (svc *Service) DeleteHosts(ctx context.Context, ids []uint, opts fleet.HostListOptions, lid *uint) error {
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

func (svc *Service) CountHosts(ctx context.Context, labelID *uint, opts fleet.HostListOptions) (int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return 0, err
	}

	return svc.countHostFromFilters(ctx, labelID, opts)
}

func (svc *Service) countHostFromFilters(ctx context.Context, labelID *uint, opt fleet.HostListOptions) (int, error) {
	filter, err := processHostFilters(ctx, opt, nil)
	if err != nil {
		return 0, err
	}

	var count int
	if labelID != nil {
		count, err = svc.ds.CountHostsInLabel(ctx, filter, *labelID, opt)
	} else {
		count, err = svc.ds.CountHosts(ctx, filter, opt)
	}
	if err != nil {
		return 0, err
	}

	return count, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Search
/////////////////////////////////////////////////////////////////////////////////

type searchHostsRequest struct {
	// MatchQuery is the query SQL
	MatchQuery string `json:"query"`
	// QueryID is the ID of a saved query to run (used to determine if this is a
	// query that observers can run).
	QueryID *uint `json:"query_id"`
	// ExcludedHostIDs is the list of IDs selected on the caller side
	// (e.g. the UI) that will be excluded from the returned payload.
	ExcludedHostIDs []uint `json:"excluded_host_ids"`
}

type searchHostsResponse struct {
	Hosts []*hostSearchResult `json:"hosts"`
	Err   error               `json:"error,omitempty"`
}

func (r searchHostsResponse) error() error { return r.Err }

func searchHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*searchHostsRequest)

	hosts, err := svc.SearchHosts(ctx, req.MatchQuery, req.QueryID, req.ExcludedHostIDs)
	if err != nil {
		return searchHostsResponse{Err: err}, nil
	}

	results := []*hostSearchResult{}

	for _, h := range hosts {
		results = append(results,
			&hostSearchResult{
				HostResponse{
					Host:   h,
					Status: h.Status(time.Now()),
				},
				h.Hostname,
			},
		)
	}

	return searchHostsResponse{
		Hosts: results,
	}, nil
}

func (svc *Service) SearchHosts(ctx context.Context, matchQuery string, queryID *uint, excludedHostIDs []uint) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	includeObserver := false
	if queryID != nil {
		canRun, err := svc.ds.ObserverCanRunQuery(ctx, *queryID)
		if err != nil {
			return nil, err
		}
		includeObserver = canRun
	}

	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: includeObserver}

	results := []*fleet.Host{}

	hosts, err := svc.ds.SearchHosts(ctx, filter, matchQuery, excludedHostIDs...)
	if err != nil {
		return nil, err
	}

	results = append(results, hosts...)

	return results, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get host
/////////////////////////////////////////////////////////////////////////////////

type getHostRequest struct {
	ID uint `url:"id"`
}

type getHostResponse struct {
	Host *HostDetailResponse `json:"host"`
	Err  error               `json:"error,omitempty"`
}

func (r getHostResponse) error() error { return r.Err }

func getHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getHostRequest)
	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  true, // intentionally true to preserve existing behavior
	}
	host, err := svc.GetHost(ctx, req.ID, opts)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, host)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	return getHostResponse{Host: resp}, nil
}

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken)
	if !alreadyAuthd {
		// First ensure the user has access to list hosts, then check the specific
		// host once team_id is loaded.
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}
	}

	host, err := svc.ds.Host(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host")
	}

	if !alreadyAuthd {
		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	hostDetails, err := svc.getHostDetails(ctx, host, opts)
	if err != nil {
		return nil, err
	}

	return hostDetails, nil
}

func (svc *Service) checkWriteForHostIDs(ctx context.Context, ids []uint) error {
	for _, id := range ids {
		host, err := svc.ds.HostLite(ctx, id)
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
	TeamID   *uint   `query:"team_id,optional"`
	Platform *string `query:"platform,optional"`
}

type getHostSummaryResponse struct {
	fleet.HostSummary
	Err error `json:"error,omitempty"`
}

func (r getHostSummaryResponse) error() error { return r.Err }

func getHostSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getHostSummaryRequest)
	summary, err := svc.GetHostSummary(ctx, req.TeamID, req.Platform)
	if err != nil {
		return getHostSummaryResponse{Err: err}, nil
	}

	resp := getHostSummaryResponse{
		HostSummary: *summary,
	}
	return resp, nil
}

func (svc *Service) GetHostSummary(ctx context.Context, teamID *uint, platform *string) (*fleet.HostSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID}

	hostSummary, err := svc.ds.GenerateHostStatusStatistics(ctx, filter, svc.clock.Now(), platform)
	if err != nil {
		return nil, err
	}

	linuxCount := uint(0)
	for _, p := range hostSummary.Platforms {
		if fleet.IsLinux(p.Platform) {
			linuxCount += p.HostsCount
		}
	}
	hostSummary.AllLinuxCount = linuxCount

	labelsSummary, err := svc.ds.LabelsSummary(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: should query for "All linux" label be updated to use `platform` from `os_version` table
	// so that the label tracks the way platforms are handled here in the host summary?
	var builtinLabels []*fleet.LabelSummary
	for _, l := range labelsSummary {
		if l.LabelType == fleet.LabelTypeBuiltIn {
			builtinLabels = append(builtinLabels, l)
		}
	}
	hostSummary.BuiltinLabels = builtinLabels

	return hostSummary, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host By Identifier
////////////////////////////////////////////////////////////////////////////////

type hostByIdentifierRequest struct {
	Identifier string `url:"identifier"`
}

func hostByIdentifierEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*hostByIdentifierRequest)
	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  true, // intentionally true to preserve existing behavior
	}
	host, err := svc.HostByIdentifier(ctx, req.Identifier, opts)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, host)
	if err != nil {
		return getHostResponse{Err: err}, nil
	}

	return getHostResponse{
		Host: resp,
	}, nil
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostByIdentifier(ctx, identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	hostDetails, err := svc.getHostDetails(ctx, host, opts)
	if err != nil {
		return nil, err
	}

	return hostDetails, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Host
////////////////////////////////////////////////////////////////////////////////

type deleteHostRequest struct {
	ID uint `url:"id"`
}

type deleteHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteHostResponse) error() error { return r.Err }

func deleteHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteHostRequest)
	err := svc.DeleteHost(ctx, req.ID)
	if err != nil {
		return deleteHostResponse{Err: err}, nil
	}
	return deleteHostResponse{}, nil
}

func (svc *Service) DeleteHost(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host for delete")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteHost(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Add Hosts to Team
////////////////////////////////////////////////////////////////////////////////

type addHostsToTeamRequest struct {
	TeamID  *uint  `json:"team_id"`
	HostIDs []uint `json:"hosts"`
}

type addHostsToTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addHostsToTeamResponse) error() error { return r.Err }

func addHostsToTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*addHostsToTeamRequest)
	err := svc.AddHostsToTeam(ctx, req.TeamID, req.HostIDs)
	if err != nil {
		return addHostsToTeamResponse{Err: err}, nil
	}

	return addHostsToTeamResponse{}, err
}

func (svc *Service) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.AddHostsToTeam(ctx, teamID, hostIDs)
}

////////////////////////////////////////////////////////////////////////////////
// Add Hosts to Team by Filter
////////////////////////////////////////////////////////////////////////////////

type addHostsToTeamByFilterRequest struct {
	TeamID  *uint `json:"team_id"`
	Filters struct {
		MatchQuery string           `json:"query"`
		Status     fleet.HostStatus `json:"status"`
		LabelID    *uint            `json:"label_id"`
	} `json:"filters"`
}

type addHostsToTeamByFilterResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addHostsToTeamByFilterResponse) error() error { return r.Err }

func addHostsToTeamByFilterEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*addHostsToTeamByFilterRequest)
	listOpt := fleet.HostListOptions{
		ListOptions: fleet.ListOptions{
			MatchQuery: req.Filters.MatchQuery,
		},
		StatusFilter: req.Filters.Status,
	}
	err := svc.AddHostsToTeamByFilter(ctx, req.TeamID, listOpt, req.Filters.LabelID)
	if err != nil {
		return addHostsToTeamByFilterResponse{Err: err}, nil
	}

	return addHostsToTeamByFilterResponse{}, err
}

func (svc *Service) AddHostsToTeamByFilter(ctx context.Context, teamID *uint, opt fleet.HostListOptions, lid *uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	hostIDs, err := svc.hostIDsFromFilters(ctx, opt, lid)
	if err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return nil
	}

	// Apply the team to the selected hosts.
	return svc.ds.AddHostsToTeam(ctx, teamID, hostIDs)
}

////////////////////////////////////////////////////////////////////////////////
// Refetch Host
////////////////////////////////////////////////////////////////////////////////

type refetchHostRequest struct {
	ID uint `url:"id"`
}

type refetchHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r refetchHostResponse) error() error { return r.Err }

func refetchHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*refetchHostRequest)
	err := svc.RefetchHost(ctx, req.ID)
	if err != nil {
		return refetchHostResponse{Err: err}, nil
	}
	return refetchHostResponse{}, nil
}

func (svc *Service) RefetchHost(ctx context.Context, id uint) error {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return err
		}

		host, err := svc.ds.HostLite(ctx, id)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "find host for refetch")
		}

		// We verify fleet.ActionRead instead of fleet.ActionWrite because we want to allow
		// observers to be able to refetch hosts.
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return err
		}
	}

	if err := svc.ds.UpdateHostRefetchRequested(ctx, id, true); err != nil {
		return ctxerr.Wrap(ctx, err, "save host")
	}

	return nil
}

func (svc *Service) getHostDetails(ctx context.Context, host *fleet.Host, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	if err := svc.ds.LoadHostSoftware(ctx, host, opts.IncludeCVEScores); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}

	labels, err := svc.ds.ListLabelsForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels for host")
	}

	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get packs for host")
	}

	bats, err := svc.ds.ListHostBatteries(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get batteries for host")
	}

	// Due to a known osquery issue with M1 Macs, we are ignoring the stored value in the db
	// and replacing it at the service layer with custom values determined by the cycle count.
	// See https://github.com/fleetdm/fleet/issues/6763.
	// TODO: Update once the underlying osquery issue has been resolved.
	for _, b := range bats {
		if b.CycleCount < 1000 {
			b.Health = "Normal"
		} else {
			b.Health = "Replacement recommended"
		}
	}

	var policies *[]*fleet.HostPolicy
	if opts.IncludePolicies {
		hp, err := svc.ds.ListPoliciesForHost(ctx, host)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get policies for host")
		}

		if hp == nil {
			hp = []*fleet.HostPolicy{}
		}

		policies = &hp
	}

	return &fleet.HostDetail{
		Host:      *host,
		Labels:    labels,
		Packs:     packs,
		Policies:  policies,
		Batteries: &bats,
	}, nil
}

func (svc *Service) hostIDsFromFilters(ctx context.Context, opt fleet.HostListOptions, lid *uint) ([]uint, error) {
	filter, err := processHostFilters(ctx, opt, lid)
	if err != nil {
		return nil, err
	}

	// Load hosts, either from label if provided or from all hosts.
	var hosts []*fleet.Host
	if lid != nil {
		hosts, err = svc.ds.ListHostsInLabel(ctx, filter, *lid, opt)
	} else {
		hosts, err = svc.ds.ListHosts(ctx, filter, opt)
	}
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, nil
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}
	return hostIDs, nil
}

func processHostFilters(ctx context.Context, opt fleet.HostListOptions, lid *uint) (fleet.TeamFilter, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.TeamFilter{}, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	if opt.StatusFilter != "" && lid != nil {
		return fleet.TeamFilter{}, fleet.NewInvalidArgumentError("status", "may not be provided with label_id")
	}

	opt.PerPage = fleet.PerPageUnlimited
	return filter, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Host Device Mappings
////////////////////////////////////////////////////////////////////////////////

type listHostDeviceMappingRequest struct {
	ID uint `url:"id"`
}

type listHostDeviceMappingResponse struct {
	HostID        uint                       `json:"host_id"`
	DeviceMapping []*fleet.HostDeviceMapping `json:"device_mapping"`
	Err           error                      `json:"error,omitempty"`
}

func (r listHostDeviceMappingResponse) error() error { return r.Err }

func listHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listHostDeviceMappingRequest)
	dms, err := svc.ListHostDeviceMapping(ctx, req.ID)
	if err != nil {
		return listHostDeviceMappingResponse{Err: err}, nil
	}
	return listHostDeviceMappingResponse{HostID: req.ID, DeviceMapping: dms}, nil
}

func (svc *Service) ListHostDeviceMapping(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}

		host, err := svc.ds.HostLite(ctx, id)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host")
		}

		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	return svc.ds.ListHostDeviceMapping(ctx, id)
}

////////////////////////////////////////////////////////////////////////////////
// Macadmins
////////////////////////////////////////////////////////////////////////////////

type getMacadminsDataRequest struct {
	ID uint `url:"id"`
}

type getMacadminsDataResponse struct {
	Err       error                `json:"error,omitempty"`
	Macadmins *fleet.MacadminsData `json:"macadmins"`
}

func (r getMacadminsDataResponse) error() error { return r.Err }

func getMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getMacadminsDataRequest)
	data, err := svc.MacadminsData(ctx, req.ID)
	if err != nil {
		return getMacadminsDataResponse{Err: err}, nil
	}
	return getMacadminsDataResponse{Macadmins: data}, nil
}

func (svc *Service) MacadminsData(ctx context.Context, id uint) (*fleet.MacadminsData, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}

		host, err := svc.ds.HostLite(ctx, id)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "find host for macadmins")
		}

		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	var munkiInfo *fleet.HostMunkiInfo
	switch version, err := svc.ds.GetMunkiVersion(ctx, id); {
	case err != nil && !fleet.IsNotFound(err):
		return nil, err
	case err == nil:
		munkiInfo = &fleet.HostMunkiInfo{Version: version}
	}

	var mdm *fleet.HostMDM
	switch hmdm, err := svc.ds.GetMDM(ctx, id); {
	case err != nil && !fleet.IsNotFound(err):
		return nil, err
	case err == nil:
		mdm = hmdm
	}

	var munkiIssues []*fleet.HostMunkiIssue
	switch issues, err := svc.ds.GetMunkiIssues(ctx, id); {
	case err != nil:
		return nil, err
	case err == nil:
		munkiIssues = issues
	}

	if munkiInfo == nil && mdm == nil && len(munkiIssues) == 0 {
		return nil, nil
	}

	data := &fleet.MacadminsData{
		Munki:       munkiInfo,
		MDM:         mdm,
		MunkiIssues: munkiIssues,
	}

	return data, nil
}

////////////////////////////////////////////////////////////////////////////////
// Aggregated Macadmins
////////////////////////////////////////////////////////////////////////////////

type getAggregatedMacadminsDataRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getAggregatedMacadminsDataResponse struct {
	Err       error                          `json:"error,omitempty"`
	Macadmins *fleet.AggregatedMacadminsData `json:"macadmins"`
}

func (r getAggregatedMacadminsDataResponse) error() error { return r.Err }

func getAggregatedMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getAggregatedMacadminsDataRequest)
	data, err := svc.AggregatedMacadminsData(ctx, req.TeamID)
	if err != nil {
		return getAggregatedMacadminsDataResponse{Err: err}, nil
	}
	return getAggregatedMacadminsDataResponse{Macadmins: data}, nil
}

func (svc *Service) AggregatedMacadminsData(ctx context.Context, teamID *uint) (*fleet.AggregatedMacadminsData, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if teamID != nil {
		_, err := svc.ds.Team(ctx, *teamID)
		if err != nil {
			return nil, err
		}
	}

	agg := &fleet.AggregatedMacadminsData{}

	versions, munkiUpdatedAt, err := svc.ds.AggregatedMunkiVersion(ctx, teamID)
	if err != nil {
		return nil, err
	}
	agg.MunkiVersions = versions

	issues, munkiIssUpdatedAt, err := svc.ds.AggregatedMunkiIssues(ctx, teamID)
	if err != nil {
		return nil, err
	}
	agg.MunkiIssues = issues

	status, mdmUpdatedAt, err := svc.ds.AggregatedMDMStatus(ctx, teamID)
	if err != nil {
		return nil, err
	}
	agg.MDMStatus = status

	solutions, mdmSolutionsUpdatedAt, err := svc.ds.AggregatedMDMSolutions(ctx, teamID)
	if err != nil {
		return nil, err
	}
	agg.MDMSolutions = solutions

	agg.CountsUpdatedAt = munkiUpdatedAt
	if munkiIssUpdatedAt.After(agg.CountsUpdatedAt) {
		agg.CountsUpdatedAt = munkiIssUpdatedAt
	}
	if mdmUpdatedAt.After(agg.CountsUpdatedAt) {
		agg.CountsUpdatedAt = mdmUpdatedAt
	}
	if mdmSolutionsUpdatedAt.After(agg.CountsUpdatedAt) {
		agg.CountsUpdatedAt = mdmSolutionsUpdatedAt
	}

	return agg, nil
}

////////////////////////////////////////////////////////////////////////////////
// Hosts Report in CSV downloadable file
////////////////////////////////////////////////////////////////////////////////

type hostsReportRequest struct {
	Opts    fleet.HostListOptions `url:"host_options"`
	LabelID *uint                 `query:"label_id,optional"`
	Format  string                `query:"format"`
	Columns string                `query:"columns,optional"`
}

type hostsReportResponse struct {
	Columns []string        `json:"-"` // used to control the generated csv, see the hijackRender method
	Hosts   []*HostResponse `json:"-"` // they get rendered explicitly, in csv
	Err     error           `json:"error,omitempty"`
}

func (r hostsReportResponse) error() error { return r.Err }

func (r hostsReportResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	// post-process the Device Mappings for CSV rendering
	for _, h := range r.Hosts {
		if h.DeviceMapping != nil {
			// return the list of emails, comma-separated, as part of that single CSV field
			var dms []struct {
				Email string `json:"email"`
			}
			if err := json.Unmarshal(*h.DeviceMapping, &dms); err != nil {
				// log the error but keep going
				logging.WithErr(ctx, err)
				continue
			}

			var sb strings.Builder
			for i, dm := range dms {
				if i > 0 {
					sb.WriteString(",")
				}
				sb.WriteString(dm.Email)
			}
			h.CSVDeviceMapping = sb.String()
		}
	}

	var buf bytes.Buffer
	if err := gocsv.Marshal(r.Hosts, &buf); err != nil {
		logging.WithErr(ctx, err)
		encodeError(ctx, ctxerr.New(ctx, "failed to generate CSV file"), w)
		return
	}

	returnAll := len(r.Columns) == 0

	var outRows [][]string
	if !returnAll {
		// read back the CSV to filter out any unwanted columns
		recs, err := csv.NewReader(&buf).ReadAll()
		if err != nil {
			logging.WithErr(ctx, err)
			encodeError(ctx, ctxerr.New(ctx, "failed to generate CSV file"), w)
			return
		}

		if len(recs) > 0 {
			// map the header names to their field index
			hdrs := make(map[string]int, len(recs))
			for i, hdr := range recs[0] {
				hdrs[hdr] = i
			}

			outRows = make([][]string, len(recs))
			for i, rec := range recs {
				for _, col := range r.Columns {
					colIx, ok := hdrs[col]
					if !ok {
						// invalid column name - it would be nice to catch this in the
						// endpoint before processing the results, but it would require
						// duplicating the list of columns from the Host's struct tags to a
						// map and keep this in sync, for what is essentially a programmer
						// mistake that should be caught and corrected early.
						encodeError(ctx, &badRequestError{message: fmt.Sprintf("invalid column name: %q", col)}, w)
						return
					}
					outRows[i] = append(outRows[i], rec[colIx])
				}
			}
		}
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="Hosts %s.csv"`, time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)

	var err error
	if returnAll {
		_, err = io.Copy(w, &buf)
	} else {
		err = csv.NewWriter(w).WriteAll(outRows)
	}
	if err != nil {
		logging.WithErr(ctx, err)
	}
}

func hostsReportEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*hostsReportRequest)

	// for now, only csv format is allowed
	if req.Format != "csv" {
		// prevent returning an "unauthorized" error, we want that specific error
		if az, ok := authz.FromContext(ctx); ok {
			az.SetChecked()
		}
		err := ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("format", "unsupported or unspecified report format").
			WithStatus(http.StatusUnsupportedMediaType))
		return hostsReportResponse{Err: err}, nil
	}

	req.Opts.DisableFailingPolicies = false
	req.Opts.AdditionalFilters = nil
	req.Opts.Page = 0
	req.Opts.PerPage = 0 // explicitly disable any limit, we want all matching hosts
	req.Opts.After = ""
	req.Opts.DeviceMapping = false

	rawCols := strings.Split(req.Columns, ",")
	var cols []string
	for _, rawCol := range rawCols {
		if rawCol = strings.TrimSpace(rawCol); rawCol != "" {
			cols = append(cols, rawCol)
			if rawCol == "device_mapping" {
				req.Opts.DeviceMapping = true
			}
		}
	}
	if len(cols) == 0 {
		// enable device_mapping retrieval, as no column means all columns
		req.Opts.DeviceMapping = true
	}

	var (
		hosts []*fleet.Host
		err   error
	)

	if req.LabelID == nil {
		hosts, err = svc.ListHosts(ctx, req.Opts)
	} else {
		hosts, err = svc.ListHostsInLabel(ctx, *req.LabelID, req.Opts)
	}
	if err != nil {
		return hostsReportResponse{Err: err}, nil
	}

	hostResps := make([]*HostResponse, len(hosts))
	for i, h := range hosts {
		hr, err := hostResponseForHost(ctx, svc, h)
		if err != nil {
			return hostsReportResponse{Err: err}, nil
		}
		hostResps[i] = hr
	}
	return hostsReportResponse{Columns: cols, Hosts: hostResps}, nil
}

type osVersionsRequest struct {
	TeamID   *uint   `query:"team_id,optional"`
	Platform *string `query:"platform,optional"`
	Name     *string `query:"os_name,optional"`
	Version  *string `query:"os_name,optional"`
}

type osVersionsResponse struct {
	CountsUpdatedAt *time.Time        `json:"counts_updated_at"`
	OSVersions      []fleet.OSVersion `json:"os_versions"`
	Err             error             `json:"error,omitempty"`
}

func (r osVersionsResponse) error() error { return r.Err }

func osVersionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*osVersionsRequest)

	osVersions, err := svc.OSVersions(ctx, req.TeamID, req.Platform, req.Name, req.Version)
	if err != nil {
		return &osVersionsResponse{Err: err}, nil
	}

	return &osVersionsResponse{
		CountsUpdatedAt: &osVersions.CountsUpdatedAt,
		OSVersions:      osVersions.OSVersions,
	}, nil
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string) (*fleet.OSVersions, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	if name != nil && version == nil {
		return nil, &badRequestError{"Cannot specify os_name without os_version"}
	}

	if name == nil && version != nil {
		return nil, &badRequestError{"Cannot specify os_version without os_name"}
	}

	osVersions, err := svc.ds.OSVersions(ctx, teamID, platform, name, version)
	if err != nil && fleet.IsNotFound(err) {
		// differentiate case where team was added after UpdateOSVersions last ran
		if teamID != nil {
			// most of the time, team should exist so checking here saves unnecessary db calls
			_, err := svc.ds.Team(ctx, *teamID)
			if err != nil {
				return nil, err
			}
		}
		// if team exists but stats have not yet been gathered, return empty JSON array
		osVersions = &fleet.OSVersions{}
	} else if err != nil {
		return nil, err
	}

	return osVersions, nil
}
