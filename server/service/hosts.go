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

	"github.com/fleetdm/fleet/v4/server/authz"
	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/gocarina/gocsv"
)

// HostDetailResponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	fleet.HostDetail
	Status      fleet.HostStatus   `json:"status"`
	DisplayText string             `json:"display_text"`
	DisplayName string             `json:"display_name"`
	Geolocation *fleet.GeoLocation `json:"geolocation,omitempty"`
}

func hostDetailResponseForHost(ctx context.Context, svc fleet.Service, host *fleet.HostDetail) (*HostDetailResponse, error) {
	return &HostDetailResponse{
		HostDetail:  *host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
		DisplayName: host.DisplayName(),
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
	Hosts    []fleet.HostResponse `json:"hosts"`
	Software *fleet.Software      `json:"software,omitempty"`
	// MDMSolution is populated with the MDM solution corresponding to the mdm_id
	// filter if one is provided with the request (and it exists in the
	// database). It is nil otherwise and absent of the JSON response payload.
	MDMSolution *fleet.MDMSolution `json:"mobile_device_management_solution,omitempty"`
	// MunkiIssue is populated with the munki issue corresponding to the
	// munki_issue_id filter if one is provided with the request (and it exists
	// in the database). It is nil otherwise and absent of the JSON response
	// payload.
	MunkiIssue *fleet.MunkiIssue `json:"munki_issue,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r listHostsResponse) error() error { return r.Err }

func listHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostsRequest)

	var software *fleet.Software
	if req.Opts.SoftwareIDFilter != nil {
		var err error
		software, err = svc.SoftwareByID(ctx, *req.Opts.SoftwareIDFilter, false)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}

	var mdmSolution *fleet.MDMSolution
	if req.Opts.MDMIDFilter != nil {
		var err error
		mdmSolution, err = svc.GetMDMSolution(ctx, *req.Opts.MDMIDFilter)
		if err != nil && !fleet.IsNotFound(err) { // ignore not found, just return nil for the MDM solution in that case
			return listHostsResponse{Err: err}, nil
		}
	}

	var munkiIssue *fleet.MunkiIssue
	if req.Opts.MunkiIssueIDFilter != nil {
		var err error
		munkiIssue, err = svc.GetMunkiIssue(ctx, *req.Opts.MunkiIssueIDFilter)
		if err != nil && !fleet.IsNotFound(err) { // ignore not found, just return nil for the munki issue in that case
			return listHostsResponse{Err: err}, nil
		}
	}

	hosts, err := svc.ListHosts(ctx, req.Opts)
	if err != nil {
		return listHostsResponse{Err: err}, nil
	}

	hostResponses := make([]fleet.HostResponse, len(hosts))
	for i, host := range hosts {
		h := fleet.HostResponseForHost(ctx, svc, host)
		hostResponses[i] = *h
	}
	return listHostsResponse{
		Hosts:       hostResponses,
		Software:    software,
		MDMSolution: mdmSolution,
		MunkiIssue:  munkiIssue,
	}, nil
}

func (svc *Service) GetMDMSolution(ctx context.Context, mdmID uint) (*fleet.MDMSolution, error) {
	// require list hosts permission to view this information
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}
	return svc.ds.GetMDMSolution(ctx, mdmID)
}

func (svc *Service) GetMunkiIssue(ctx context.Context, munkiIssueID uint) (*fleet.MunkiIssue, error) {
	// require list hosts permission to view this information
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}
	return svc.ds.GetMunkiIssue(ctx, munkiIssueID)
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

	// TODO(Sarah): Are we missing any other filters here?
	if !license.IsPremium(ctx) {
		// the low disk space filter is premium-only
		opt.LowDiskSpaceFilter = nil
		// the bootstrap package filter is premium-only
		opt.MDMBootstrapPackageFilter = nil
	}

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

func deleteHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
		return &fleet.BadRequestError{Message: "Cannot specify a list of ids and filters at the same time"}
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

func countHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	if !license.IsPremium(ctx) {
		// the low disk space filter is premium-only
		opt.LowDiskSpaceFilter = nil
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
	Hosts []*fleet.HostResponse `json:"hosts"`
	Err   error                 `json:"error,omitempty"`
}

func (r searchHostsResponse) error() error { return r.Err }

func searchHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*searchHostsRequest)

	hosts, err := svc.SearchHosts(ctx, req.MatchQuery, req.QueryID, req.ExcludedHostIDs)
	if err != nil {
		return searchHostsResponse{Err: err}, nil
	}

	results := []*fleet.HostResponse{}

	for _, h := range hosts {
		results = append(results, fleet.HostResponseForHostCheap(h))
	}

	return searchHostsResponse{
		Hosts: results,
	}, nil
}

func (svc *Service) SearchHosts(ctx context.Context, matchQuery string, queryID *uint, excludedHostIDs []uint) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
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

func getHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken)
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
	TeamID       *uint   `query:"team_id,optional"`
	Platform     *string `query:"platform,optional"`
	LowDiskSpace *int    `query:"low_disk_space,optional"`
}

type getHostSummaryResponse struct {
	fleet.HostSummary
	Err error `json:"error,omitempty"`
}

func (r getHostSummaryResponse) error() error { return r.Err }

func getHostSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostSummaryRequest)
	if req.LowDiskSpace != nil {
		if *req.LowDiskSpace < 1 || *req.LowDiskSpace > 100 {
			err := ctxerr.Errorf(ctx, "invalid low_disk_space threshold, must be between 1 and 100: %d", *req.LowDiskSpace)
			return getHostSummaryResponse{Err: err}, nil
		}
	}

	summary, err := svc.GetHostSummary(ctx, req.TeamID, req.Platform, req.LowDiskSpace)
	if err != nil {
		return getHostSummaryResponse{Err: err}, nil
	}

	resp := getHostSummaryResponse{
		HostSummary: *summary,
	}
	return resp, nil
}

func (svc *Service) GetHostSummary(ctx context.Context, teamID *uint, platform *string, lowDiskSpace *int) (*fleet.HostSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true, TeamID: teamID}

	if !license.IsPremium(ctx) {
		lowDiskSpace = nil
	}

	hostSummary, err := svc.ds.GenerateHostStatusStatistics(ctx, filter, svc.clock.Now(), platform, lowDiskSpace)
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

func hostByIdentifierEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

func deleteHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

func addHostsToTeamEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	if err := svc.ds.AddHostsToTeam(ctx, teamID, hostIDs); err != nil {
		return err
	}
	if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDs, nil, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}
	serials, err := svc.ds.ListMDMAppleDEPSerialsInHostIDs(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in host ids")
	}
	if len(serials) > 0 {
		if err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantHostsTransferred,
			teamID,
			serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "queue macos setup assistant hosts transferred job")
		}
	}

	return nil
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

func addHostsToTeamByFilterEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	if err := svc.ds.AddHostsToTeam(ctx, teamID, hostIDs); err != nil {
		return err
	}
	if err := svc.ds.BulkSetPendingMDMAppleHostProfiles(ctx, hostIDs, nil, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}
	serials, err := svc.ds.ListMDMAppleDEPSerialsInHostIDs(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in host ids")
	}
	if len(serials) > 0 {
		if err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantHostsTransferred,
			teamID,
			serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "queue macos setup assistant hosts transferred job")
		}
	}
	return nil
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

func refetchHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*refetchHostRequest)
	err := svc.RefetchHost(ctx, req.ID)
	if err != nil {
		return refetchHostResponse{Err: err}, nil
	}
	return refetchHostResponse{}, nil
}

func (svc *Service) RefetchHost(ctx context.Context, id uint) error {
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) {
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

	// If Fleet MDM is enabled and configured, we want to include MDM profiles
	// and host's disk encryption status.
	var profiles []fleet.HostMDMAppleProfile
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for host mdm profiles")
	}
	if ac.MDM.EnabledAndConfigured {
		profs, err := svc.ds.GetHostMDMProfiles(ctx, host.UUID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host mdm profiles")
		}

		// determine disk encryption and action required here based on profiles and
		// raw decryptable key status.
		host.MDM.DetermineDiskEncryptionStatus(profs, mobileconfig.FleetFileVaultPayloadIdentifier)

		for _, p := range profs {
			if p.Identifier == mobileconfig.FleetFileVaultPayloadIdentifier {
				p.Status = host.MDM.ProfileStatusFromDiskEncryptionState(p.Status)
			}
			profiles = append(profiles, p)
		}
	}
	host.MDM.Profiles = &profiles

	var macOSSetup *fleet.HostMDMMacOSSetup
	if ac.MDM.EnabledAndConfigured && license.IsPremium(ctx) {
		macOSSetup, err = svc.ds.GetHostMDMMacOSSetup(ctx, host.ID)
		if err != nil {
			if !fleet.IsNotFound(err) {
				return nil, ctxerr.Wrap(ctx, err, "get host mdm macos setup")
			}
			// TODO(Sarah): What should we do for not found? Should we return an empty struct or nil?
			macOSSetup = &fleet.HostMDMMacOSSetup{}
		}
	}
	host.MDM.MacOSSetup = macOSSetup

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

func listHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listHostDeviceMappingRequest)
	dms, err := svc.ListHostDeviceMapping(ctx, req.ID)
	if err != nil {
		return listHostDeviceMappingResponse{Err: err}, nil
	}
	return listHostDeviceMappingResponse{HostID: req.ID, DeviceMapping: dms}, nil
}

func (svc *Service) ListHostDeviceMapping(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) {
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
// MDM
////////////////////////////////////////////////////////////////////////////////

type getHostMDMRequest struct {
	ID uint `url:"id"`
}

type getHostMDMResponse struct {
	*fleet.HostMDM
	Err error `json:"error,omitempty"`
}

func (r getHostMDMResponse) error() error { return r.Err }

func getHostMDM(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostMDMRequest)
	mdm, err := svc.MDMData(ctx, req.ID)
	if err != nil {
		return getHostMDMResponse{Err: err}, nil
	}
	return getHostMDMResponse{HostMDM: mdm}, nil
}

type getHostMDMSummaryResponse struct {
	fleet.AggregatedMDMData
	Err error `json:"error,omitempty"`
}

type getHostMDMSummaryRequest struct {
	TeamID   *uint  `query:"team_id,optional"`
	Platform string `query:"platform,optional"`
}

func (r getHostMDMSummaryResponse) error() error { return r.Err }

func getHostMDMSummary(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostMDMSummaryRequest)
	resp := getHostMDMSummaryResponse{}
	var err error

	resp.AggregatedMDMData, err = svc.AggregatedMDMData(ctx, req.TeamID, req.Platform)
	if err != nil {
		return getHostMDMSummaryResponse{Err: err}, nil
	}
	return resp, nil
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

func getMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMacadminsDataRequest)
	data, err := svc.MacadminsData(ctx, req.ID)
	if err != nil {
		return getMacadminsDataResponse{Err: err}, nil
	}
	return getMacadminsDataResponse{Macadmins: data}, nil
}

func (svc *Service) MacadminsData(ctx context.Context, id uint) (*fleet.MacadminsData, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) {
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
	switch version, err := svc.ds.GetHostMunkiVersion(ctx, id); {
	case err != nil && !fleet.IsNotFound(err):
		return nil, err
	case err == nil:
		munkiInfo = &fleet.HostMunkiInfo{Version: version}
	}

	var mdm *fleet.HostMDM
	switch hmdm, err := svc.ds.GetHostMDM(ctx, id); {
	case err != nil && !fleet.IsNotFound(err):
		return nil, err
	case err == nil:
		mdm = hmdm
	}

	var munkiIssues []*fleet.HostMunkiIssue
	switch issues, err := svc.ds.GetHostMunkiIssues(ctx, id); {
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

func getAggregatedMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

	var mdmUpdatedAt, mdmSolutionsUpdatedAt time.Time
	agg.MDMStatus, mdmUpdatedAt, err = svc.ds.AggregatedMDMStatus(ctx, teamID, "darwin")
	if err != nil {
		return nil, err
	}
	agg.MDMSolutions, mdmSolutionsUpdatedAt, err = svc.ds.AggregatedMDMSolutions(ctx, teamID, "darwin")
	if err != nil {
		return nil, err
	}
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

func (svc *Service) MDMData(ctx context.Context, id uint) (*fleet.HostMDM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find host for MDMData")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	hmdm, err := svc.ds.GetHostMDM(ctx, id)
	switch {
	case err == nil:
		return hmdm, nil
	case fleet.IsNotFound(err):
		return nil, nil
	default:
		return nil, ctxerr.Wrap(ctx, err, "get host mdm")
	}
}

func (svc *Service) AggregatedMDMData(ctx context.Context, teamID *uint, platform string) (fleet.AggregatedMDMData, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return fleet.AggregatedMDMData{}, err
	}

	mdmStatus, mdmStatusUpdatedAt, err := svc.ds.AggregatedMDMStatus(ctx, teamID, platform)
	if err != nil {
		return fleet.AggregatedMDMData{}, err
	}
	mdmSolutions, mdmSolutionsUpdatedAt, err := svc.ds.AggregatedMDMSolutions(ctx, teamID, platform)
	if err != nil {
		return fleet.AggregatedMDMData{}, err
	}

	countsUpdatedAt := mdmStatusUpdatedAt
	if mdmStatusUpdatedAt.Before(mdmSolutionsUpdatedAt) {
		countsUpdatedAt = mdmSolutionsUpdatedAt
	}

	return fleet.AggregatedMDMData{
		MDMStatus:       mdmStatus,
		MDMSolutions:    mdmSolutions,
		CountsUpdatedAt: countsUpdatedAt,
	}, nil
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
	Columns []string              `json:"-"` // used to control the generated csv, see the hijackRender method
	Hosts   []*fleet.HostResponse `json:"-"` // they get rendered explicitly, in csv
	Err     error                 `json:"error,omitempty"`
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
						encodeError(ctx, &fleet.BadRequestError{Message: fmt.Sprintf("invalid column name: %q", col)}, w)
						return
					}
					outRows[i] = append(outRows[i], rec[colIx])
				}
			}
		}
	}

	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="Hosts %s.csv"`, time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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

func hostsReportEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*hostsReportRequest)

	// for now, only csv format is allowed
	if req.Format != "csv" {
		// prevent returning an "unauthorized" error, we want that specific error
		if az, ok := authzctx.FromContext(ctx); ok {
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

	hostResps := make([]*fleet.HostResponse, len(hosts))
	for i, h := range hosts {
		hr := fleet.HostResponseForHost(ctx, svc, h)
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

func osVersionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
		return nil, &fleet.BadRequestError{Message: "Cannot specify os_name without os_version"}
	}

	if name == nil && version != nil {
		return nil, &fleet.BadRequestError{Message: "Cannot specify os_version without os_name"}
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

////////////////////////////////////////////////////////////////////////////////
// Encryption Key
////////////////////////////////////////////////////////////////////////////////

type getHostEncryptionKeyRequest struct {
	ID uint `url:"id"`
}

type getHostEncryptionKeyResponse struct {
	Err           error                        `json:"error,omitempty"`
	EncryptionKey *fleet.HostDiskEncryptionKey `json:"encryption_key,omitempty"`
	HostID        uint                         `json:"host_id,omitempty"`
}

func (r getHostEncryptionKeyResponse) error() error { return r.Err }

func getHostEncryptionKey(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostEncryptionKeyRequest)
	key, err := svc.HostEncryptionKey(ctx, req.ID)
	if err != nil {
		return getHostEncryptionKeyResponse{Err: err}, nil
	}
	return getHostEncryptionKeyResponse{EncryptionKey: key, HostID: req.ID}, nil
}

func (svc *Service) HostEncryptionKey(ctx context.Context, id uint) (*fleet.HostDiskEncryptionKey, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
	}

	// Permissions to read encryption keys are exactly the same
	// as the ones required to read hosts.
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	key, err := svc.ds.GetHostDiskEncryptionKey(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
	}

	if key.Decryptable == nil || !*key.Decryptable {
		return nil, ctxerr.Wrap(ctx, newNotFoundError(), "getting host encryption key")
	}

	cert, _, _, err := svc.config.MDM.AppleSCEP()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
	}

	decryptedKey, err := apple_mdm.DecryptBase64CMS(key.Base64Encrypted, cert.Leaf, cert.PrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
	}

	key.DecryptedValue = string(decryptedKey)

	err = svc.ds.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeReadHostDiskEncryptionKey{
			HostID:          host.ID,
			HostDisplayName: host.DisplayName(),
		},
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create read host disk encryption key activity")
	}

	return key, nil
}
