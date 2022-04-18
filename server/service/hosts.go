package service

import (
	"context"
	"fmt"
	"net/http"
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
	Status      fleet.HostStatus   `json:"status"`
	DisplayText string             `json:"display_text"`
	Labels      []fleet.Label      `json:"labels,omitempty"`
	Geolocation *fleet.GeoLocation `json:"geolocation,omitempty"`
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

func (svc *Service) FlushSeenHosts(ctx context.Context) error {
	// No authorization check because this is used only internally.
	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(ctx, hostIDs, svc.clock.Now())
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

func (svc *Service) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	return svc.ds.SoftwareByID(ctx, id)
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

func (svc *Service) GetHost(ctx context.Context, id uint) (*fleet.HostDetail, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken)
	if !alreadyAuthd {
		// First ensure the user has access to list hosts, then check the specific
		// host once team_id is loaded.
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}
	}

	host, err := svc.ds.Host(ctx, id, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host")
	}

	if !alreadyAuthd {
		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	return svc.getHostDetails(ctx, host)
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

	summary, err := svc.ds.GenerateHostStatusStatistics(ctx, filter, svc.clock.Now(), platform)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host By Identifier
////////////////////////////////////////////////////////////////////////////////

type hostByIdentifierRequest struct {
	Identifier string `url:"identifier"`
}

func hostByIdentifierEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*hostByIdentifierRequest)
	host, err := svc.HostByIdentifier(ctx, req.Identifier)
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

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string) (*fleet.HostDetail, error) {
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

	return svc.getHostDetails(ctx, host)
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

func (svc *Service) getHostDetails(ctx context.Context, host *fleet.Host) (*fleet.HostDetail, error) {
	if err := svc.ds.LoadHostSoftware(ctx, host); err != nil {
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

	policies, err := svc.ds.ListPoliciesForHost(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies for host")
	}

	return &fleet.HostDetail{Host: *host, Labels: labels, Packs: packs, Policies: policies}, nil
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
	switch enrolled, serverURL, installedFromDep, err := svc.ds.GetMDM(ctx, id); {
	case err != nil && !fleet.IsNotFound(err):
		return nil, err
	case err == nil:
		enrollmentStatus := "Unenrolled"
		if enrolled && !installedFromDep {
			enrollmentStatus = "Enrolled (manual)"
		} else if enrolled && installedFromDep {
			enrollmentStatus = "Enrolled (automated)"
		}
		mdm = &fleet.HostMDM{
			EnrollmentStatus: enrollmentStatus,
			ServerURL:        serverURL,
		}
	}

	if munkiInfo == nil && mdm == nil {
		return nil, nil
	}

	data := &fleet.MacadminsData{
		Munki: munkiInfo,
		MDM:   mdm,
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

	status, mdmUpdatedAt, err := svc.ds.AggregatedMDMStatus(ctx, teamID)
	if err != nil {
		return nil, err
	}
	agg.MDMStatus = status

	agg.CountsUpdatedAt = munkiUpdatedAt
	if mdmUpdatedAt.After(munkiUpdatedAt) {
		agg.CountsUpdatedAt = mdmUpdatedAt
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
}

type hostsReportResponse struct {
	Hosts []*fleet.Host `json:"-"` // they get rendered explicitly, in csv
	Err   error         `json:"error,omitempty"`
}

func (r hostsReportResponse) error() error { return r.Err }

func (r hostsReportResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="Hosts %s.csv"`, time.Now().Format("2006-01-02")))
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)
	if err := gocsv.Marshal(r.Hosts, w); err != nil {
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

	// Those are not supported when listing hosts in a label, so that's just to
	// make the output consistent whether a label is used or not.
	req.Opts.DisableFailingPolicies = true
	req.Opts.AdditionalFilters = nil
	req.Opts.Page = 0
	req.Opts.PerPage = 0 // explicitly disable any limit, we want all matching hosts
	req.Opts.After = ""

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
	return hostsReportResponse{Hosts: hosts}, nil
}

type osVersionsRequest struct {
	TeamID   *uint   `query:"team_id,optional"`
	Platform *string `query:"platform,optional"`
}

type osVersionsResponse struct {
	CountsUpdatedAt *time.Time        `json:"counts_updated_at,omitempty"`
	OSVersions      []fleet.OSVersion `json:"os_versions,omitempty"`
	Err             error             `json:"error,omitempty"`
}

func (r osVersionsResponse) error() error { return r.Err }

func osVersionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*osVersionsRequest)

	osVersions, err := svc.OSVersions(ctx, req.TeamID, req.Platform)
	if err != nil {
		return &osVersionsResponse{Err: err}, nil
	}

	return &osVersionsResponse{
		CountsUpdatedAt: &osVersions.CountsUpdatedAt,
		OSVersions:      osVersions.OSVersions,
	}, nil
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string) (*fleet.OSVersions, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, err
	}

	osVersions, err := svc.ds.OSVersions(ctx, teamID, platform)
	if err != nil {
		return nil, err
	}

	return osVersions, nil
}
