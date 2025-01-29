package service

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/assets"
	mdmlifecycle "github.com/fleetdm/fleet/v4/server/mdm/lifecycle"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/go-kit/log/level"
	"github.com/gocarina/gocsv"
	"github.com/google/uuid"
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
	Hosts []fleet.HostResponse `json:"hosts"`
	// Software is populated with the software version corresponding to the
	// software_version_id (or software_id) filter if one is provided with the
	// request (and it exists in the database). It is nil otherwise and absent of
	// the JSON response payload.
	Software *fleet.Software `json:"software,omitempty"`
	// SoftwareTitle is populated with the title corresponding to the
	// software_title_id filter if one is provided with the request (and it
	// exists in the database). It is nil otherwise and absent of the JSON
	// response payload.
	SoftwareTitle *fleet.SoftwareTitle `json:"software_title,omitempty"`
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
	if req.Opts.SoftwareVersionIDFilter != nil || req.Opts.SoftwareIDFilter != nil {
		var err error

		id := req.Opts.SoftwareVersionIDFilter
		if id == nil {
			id = req.Opts.SoftwareIDFilter
		}
		software, err = svc.SoftwareByID(ctx, *id, req.Opts.TeamFilter, false)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}
	}

	var softwareTitle *fleet.SoftwareTitle
	if req.Opts.SoftwareTitleIDFilter != nil {
		var err error

		softwareTitle, err = svc.SoftwareTitleByID(ctx, *req.Opts.SoftwareTitleIDFilter, req.Opts.TeamFilter)
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

		if req.Opts.PopulateLabels {
			labels, err := svc.ListLabelsForHost(ctx, h.ID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("failed to list labels for host %d", h.ID))
			}
			hostResponses[i].Labels = labels
		}
	}

	return listHostsResponse{
		Hosts:         hostResponses,
		Software:      software,
		SoftwareTitle: softwareTitle,
		MDMSolution:   mdmSolution,
		MunkiIssue:    munkiIssue,
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
	premiumLicense := license.IsPremium(ctx)
	if !premiumLicense {
		// the low disk space filter is premium-only
		opt.LowDiskSpaceFilter = nil
		// the bootstrap package filter is premium-only
		opt.MDMBootstrapPackageFilter = nil
		// including vulnerability details on software is premium-only
		opt.PopulateSoftwareVulnerabilityDetails = false
	}

	hosts, err := svc.ds.ListHosts(ctx, filter, opt)
	if err != nil {
		return nil, err
	}

	// If issues are enabled, we need to remove the critical vulnerabilities count for non-premium license.
	// If issues are disabled, we need to explicitly set the critical vulnerabilities count to 0 for premium license.
	if !opt.DisableIssues && !premiumLicense {
		// Remove critical vulnerabilities count if not premium license
		for _, host := range hosts {
			host.HostIssues.CriticalVulnerabilitiesCount = nil
		}
	} else if opt.DisableIssues && premiumLicense {
		var zero uint64
		for _, host := range hosts {
			host.HostIssues.CriticalVulnerabilitiesCount = &zero
		}
	}

	if opt.PopulateSoftware {
		for _, host := range hosts {
			if err = svc.ds.LoadHostSoftware(ctx, host, opt.PopulateSoftwareVulnerabilityDetails); err != nil {
				return nil, err
			}
		}
	}

	if opt.PopulatePolicies {
		for _, host := range hosts {
			hp, err := svc.ds.ListPoliciesForHost(ctx, host)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("get policies for host %d", host.ID))
			}
			host.Policies = &hp
		}
	}

	if opt.PopulateUsers {
		for _, host := range hosts {
			hu, err := svc.ds.ListHostUsers(ctx, host.ID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, fmt.Sprintf("get users for host %d", host.ID))
			}
			host.Users = hu
		}
	}

	return hosts, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Delete Hosts
/////////////////////////////////////////////////////////////////////////////////

// These values are modified during testing.
var (
	deleteHostsTimeout           = 30 * time.Second
	deleteHostsSkipAuthorization = false
)

type deleteHostsRequest struct {
	IDs []uint `json:"ids"`
	// Using a pointer to help determine whether an empty filter was passed, like: "filters":{}
	Filters *map[string]interface{} `json:"filters"`
}

type deleteHostsResponse struct {
	Err        error `json:"error,omitempty"`
	StatusCode int   `json:"-"`
}

func (r deleteHostsResponse) error() error { return r.Err }

// Status implements statuser interface to send out custom HTTP success codes.
func (r deleteHostsResponse) Status() int { return r.StatusCode }

func deleteHostsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteHostsRequest)

	// Since bulk deletes can take a long time, after DeleteHostsTimeout, we will return a 202 (Accepted) status code
	// and allow the delete operation to proceed.
	var err error
	deleteDone := make(chan bool, 1)
	ctx = context.WithoutCancel(ctx) // to make sure DB operations don't get killed after we return a 202
	go func() {
		err = svc.DeleteHosts(ctx, req.IDs, req.Filters)
		if err != nil {
			// logging the error for future debug in case we already sent http.StatusAccepted
			logging.WithErr(ctx, err)
		}
		deleteDone <- true
	}()
	select {
	case <-deleteDone:
		if err != nil {
			return deleteHostsResponse{Err: err}, nil
		}
		return deleteHostsResponse{StatusCode: http.StatusOK}, nil
	case <-time.After(deleteHostsTimeout):
		if deleteHostsSkipAuthorization {
			// Only called during testing.
			svc.(validationMiddleware).Service.(*Service).authz.SkipAuthorization(ctx)
		}
		return deleteHostsResponse{StatusCode: http.StatusAccepted}, nil
	}
}

func (svc *Service) DeleteHosts(ctx context.Context, ids []uint, filter *map[string]interface{}) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	opts, lid, err := hostListOptionsFromFilters(filter)
	if err != nil {
		return err
	}

	if len(ids) == 0 && lid == nil && opts == nil {
		return &fleet.BadRequestError{Message: "list of ids or filters must be specified"}
	}

	if len(ids) > 0 && (lid != nil || (opts != nil && !opts.Empty())) {
		return &fleet.BadRequestError{Message: "Cannot specify a list of ids and filters at the same time"}
	}

	doDelete := func(hostIDs []uint, hosts []*fleet.Host) error {
		if err := svc.ds.DeleteHosts(ctx, hostIDs); err != nil {
			return err
		}

		mdmLifecycle := mdmlifecycle.New(svc.ds, svc.logger)
		for _, host := range hosts {
			if fleet.MDMSupported(host.Platform) {
				err := mdmLifecycle.Do(ctx, mdmlifecycle.HostOptions{
					Action:   mdmlifecycle.HostActionDelete,
					Host:     host,
					Platform: host.Platform,
				})
				return err
			}
		}

		return nil
	}

	if len(ids) > 0 {
		if err := svc.checkWriteForHostIDs(ctx, ids); err != nil {
			return err
		}

		hosts, err := svc.ds.ListHostsLiteByIDs(ctx, ids)
		if err != nil {
			return err
		}

		return doDelete(ids, hosts)
	}

	if opts == nil {
		opts = &fleet.HostListOptions{}
	}
	opts.DisableIssues = true // don't check policies for hosts that are about to be deleted
	hostIDs, _, hosts, err := svc.hostIDsAndNamesFromFilters(ctx, *opts, lid)
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

	return doDelete(hostIDs, hosts)
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
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return 0, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	if !license.IsPremium(ctx) {
		// the low disk space filter is premium-only
		opt.LowDiskSpaceFilter = nil
	}

	var count int
	var err error
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
	ID              uint `url:"id"`
	ExcludeSoftware bool `query:"exclude_software,optional"`
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
		IncludePolicies:  true, // intentionally true to preserve existing behavior,
		ExcludeSoftware:  req.ExcludeSoftware,
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
	if !opts.IncludeCriticalVulnerabilitiesCount {
		host.HostIssues.CriticalVulnerabilitiesCount = nil
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

// //////////////////////////////////////////////////////////////////////////////
// Get Host Lite
// //////////////////////////////////////////////////////////////////////////////
func (svc *Service) GetHostLite(ctx context.Context, id uint) (*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host lite")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	return host, nil
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
	if lowDiskSpace != nil {
		if *lowDiskSpace < 1 || *lowDiskSpace > 100 {
			svc.authz.SkipAuthorization(ctx)
			return nil, ctxerr.Wrap(
				ctx, badRequest(fmt.Sprintf("invalid low_disk_space threshold, must be between 1 and 100: %d", *lowDiskSpace)),
			)
		}
	}
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
	Identifier      string `url:"identifier"`
	ExcludeSoftware bool   `query:"exclude_software,optional"`
}

func hostByIdentifierEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*hostByIdentifierRequest)
	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  true, // intentionally true to preserve existing behavior
		ExcludeSoftware:  req.ExcludeSoftware,
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
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionSelectiveList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostByIdentifier(ctx, identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionSelectiveRead); err != nil {
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

	if err := svc.ds.DeleteHost(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err, "delete host")
	}

	if fleet.MDMSupported(host.Platform) {
		mdmLifecycle := mdmlifecycle.New(svc.ds, svc.logger)
		err = mdmLifecycle.Do(ctx, mdmlifecycle.HostOptions{
			Action:   mdmlifecycle.HostActionDelete,
			Platform: host.Platform,
			UUID:     host.UUID,
			Host:     host,
		})
		return ctxerr.Wrap(ctx, err, "performing MDM actions after delete")
	}

	return nil
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
	err := svc.AddHostsToTeam(ctx, req.TeamID, req.HostIDs, false)
	if err != nil {
		return addHostsToTeamResponse{Err: err}, nil
	}

	return addHostsToTeamResponse{}, err
}

func (svc *Service) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint, skipBulkPending bool) error {
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
	if !skipBulkPending {
		if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, hostIDs, nil, nil, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
		}
	}
	serials, err := svc.ds.ListMDMAppleDEPSerialsInHostIDs(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in host ids")
	}
	if len(serials) > 0 {
		if _, err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantHostsTransferred,
			teamID,
			serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "queue macos setup assistant hosts transferred job")
		}
	}

	return svc.createTransferredHostsActivity(ctx, teamID, hostIDs, nil)
}

// creates the transferred hosts activity if hosts were transferred, taking
// care of loading the team name and the hosts names if necessary (hostNames
// may be passed as empty if they were not available to the caller, such as in
// AddHostsToTeam, while it may be provided if available, such as in
// AddHostsToTeamByFilter).
func (svc *Service) createTransferredHostsActivity(ctx context.Context, teamID *uint, hostIDs []uint, hostNames []string) error {
	if len(hostIDs) == 0 {
		return nil
	}

	var teamName *string
	if teamID != nil {
		tm, err := svc.ds.Team(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get team for activity")
		}
		teamName = &tm.Name
	}

	if len(hostNames) == 0 {
		hosts, err := svc.ds.ListHostsLiteByIDs(ctx, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "list hosts by ids")
		}

		// index the hosts by ids to get the names in the same order as hostIDs
		hostsByID := make(map[uint]*fleet.Host, len(hosts))
		for _, h := range hosts {
			hostsByID[h.ID] = h
		}

		hostNames = make([]string, 0, len(hostIDs))
		for _, hid := range hostIDs {
			if h, ok := hostsByID[hid]; ok {
				hostNames = append(hostNames, h.DisplayName())
			} else {
				// should not happen unless a host gets deleted just after transfer,
				// but this ensures hostNames always matches hostIDs at the same index
				hostNames = append(hostNames, "")
			}
		}
	}

	if err := svc.NewActivity(
		ctx,
		authz.UserFromContext(ctx),
		fleet.ActivityTypeTransferredHostsToTeam{
			TeamID:           teamID,
			TeamName:         teamName,
			HostIDs:          hostIDs,
			HostDisplayNames: hostNames,
		},
	); err != nil {
		return ctxerr.Wrap(ctx, err, "create transferred_hosts activity")
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Add Hosts to Team by Filter
////////////////////////////////////////////////////////////////////////////////

type addHostsToTeamByFilterRequest struct {
	TeamID  *uint                   `json:"team_id"`
	Filters *map[string]interface{} `json:"filters"`
}

type addHostsToTeamByFilterResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addHostsToTeamByFilterResponse) error() error { return r.Err }

func addHostsToTeamByFilterEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addHostsToTeamByFilterRequest)
	err := svc.AddHostsToTeamByFilter(ctx, req.TeamID, req.Filters)
	if err != nil {
		return addHostsToTeamByFilterResponse{Err: err}, nil
	}

	return addHostsToTeamByFilterResponse{}, err
}

func (svc *Service) AddHostsToTeamByFilter(ctx context.Context, teamID *uint, filters *map[string]interface{}) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	opt, lid, err := hostListOptionsFromFilters(filters)
	if err != nil {
		return err
	}

	if opt == nil {
		return &fleet.BadRequestError{Message: "filters must be specified"}
	}

	hostIDs, hostNames, _, err := svc.hostIDsAndNamesFromFilters(ctx, *opt, lid)
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
	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, hostIDs, nil, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}
	serials, err := svc.ds.ListMDMAppleDEPSerialsInHostIDs(ctx, hostIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list mdm dep serials in host ids")
	}
	if len(serials) > 0 {
		if _, err := worker.QueueMacosSetupAssistantJob(
			ctx,
			svc.ds,
			svc.logger,
			worker.MacosSetupAssistantHostsTransferred,
			teamID,
			serials...); err != nil {
			return ctxerr.Wrap(ctx, err, "queue macos setup assistant hosts transferred job")
		}
	}

	return svc.createTransferredHostsActivity(ctx, teamID, hostIDs, hostNames)
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

func (r refetchHostResponse) error() error {
	return r.Err
}

func refetchHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*refetchHostRequest)
	err := svc.RefetchHost(ctx, req.ID)
	if err != nil {
		return refetchHostResponse{Err: err}, nil
	}
	return refetchHostResponse{}, nil
}

func (svc *Service) RefetchHost(ctx context.Context, id uint) error {
	var host *fleet.Host
	// iOS and iPadOS refetch are not authenticated with device token because these devices do not have Fleet Desktop,
	// so we don't handle that case
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) {
		var err error
		if err = svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return err
		}

		host, err = svc.ds.HostLite(ctx, id)
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

	if host != nil && (host.Platform == "ios" || host.Platform == "ipados") {
		// Get MDM commands already sent
		commands, err := svc.ds.GetHostMDMCommands(ctx, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "get host MDM commands")
		}
		doAppRefetch := true
		doDeviceInfoRefetch := true
		for _, cmd := range commands {
			switch cmd.CommandType {
			case fleet.RefetchDeviceCommandUUIDPrefix:
				doDeviceInfoRefetch = false
			case fleet.RefetchAppsCommandUUIDPrefix:
				doAppRefetch = false
			}
		}
		if !doAppRefetch && !doDeviceInfoRefetch {
			// Nothing to do.
			return nil
		}
		err = svc.verifyMDMConfiguredAndConnected(ctx, host)
		if err != nil {
			return err
		}
		hostMDMCommands := make([]fleet.HostMDMCommand, 0, 2)
		cmdUUID := uuid.NewString()
		if doAppRefetch {
			err = svc.mdmAppleCommander.InstalledApplicationList(ctx, []string{host.UUID}, fleet.RefetchAppsCommandUUIDPrefix+cmdUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "refetch apps with MDM")
			}
			hostMDMCommands = append(hostMDMCommands, fleet.HostMDMCommand{
				HostID:      host.ID,
				CommandType: fleet.RefetchAppsCommandUUIDPrefix,
			})
		}
		if doDeviceInfoRefetch {
			// DeviceInformation is last because the refetch response clears the refetch_requested flag
			err = svc.mdmAppleCommander.DeviceInformation(ctx, []string{host.UUID}, fleet.RefetchDeviceCommandUUIDPrefix+cmdUUID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "refetch host with MDM")
			}
			hostMDMCommands = append(hostMDMCommands, fleet.HostMDMCommand{
				HostID:      host.ID,
				CommandType: fleet.RefetchDeviceCommandUUIDPrefix,
			})
		}
		// Add commands to the database to track the commands sent
		err = svc.ds.AddHostMDMCommands(ctx, hostMDMCommands)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "add host mdm commands")
		}
	}

	return nil
}

func (svc *Service) verifyMDMConfiguredAndConnected(ctx context.Context, host *fleet.Host) error {
	if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
		if errors.Is(err, fleet.ErrMDMNotConfigured) {
			err = fleet.NewInvalidArgumentError("id", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		}
		return ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
	}
	connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
	}
	if !connected {
		return ctxerr.Wrap(ctx,
			fleet.NewInvalidArgumentError("id", "Host does not have MDM turned on."))
	}
	return nil
}

func (svc *Service) getHostDetails(ctx context.Context, host *fleet.Host, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	if !opts.ExcludeSoftware {
		if err := svc.ds.LoadHostSoftware(ctx, host, opts.IncludeCVEScores); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load host software")
		}
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

	mws, err := svc.ds.ListUpcomingHostMaintenanceWindows(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list upcoming host maintenance windows")
	}
	// we are only interested in the next maintenance window. There should only be one for now, anyway.
	var nextMw *fleet.HostMaintenanceWindow
	if len(mws) > 0 {
		nextMw = mws[0]
	}

	// nil TimeZone is okay
	if nextMw != nil && nextMw.TimeZone != nil {
		// return the start time in the local timezone of the host's associated google calendar user
		gCalLoc, err := time.LoadLocation(*nextMw.TimeZone)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "list upcoming host maintenance windows - invalid google calendar timezone")
		}
		nextMw.StartsAt = nextMw.StartsAt.In(gCalLoc)
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

	// If Fleet MDM is enabled and configured, we want to include MDM profiles,
	// disk encryption status, and macOS setup details for non-linux hosts.
	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get app config for host mdm details")
	}

	var profiles []fleet.HostMDMProfile
	if ac.MDM.EnabledAndConfigured || ac.MDM.WindowsEnabledAndConfigured {
		host.MDM.OSSettings = &fleet.HostMDMOSSettings{}
		switch host.Platform {
		case "windows":
			if !ac.MDM.WindowsEnabledAndConfigured {
				break
			}

			if license.IsPremium(ctx) {
				// we include disk encryption status only for premium so initialize it to default struct
				host.MDM.OSSettings.DiskEncryption = fleet.HostMDMDiskEncryption{}
				// ensure host mdm info is loaded (we don't know if our caller populated it)
				_, err := svc.ds.GetHostMDM(ctx, host.ID)
				switch {
				case err != nil && fleet.IsNotFound(err):
					// assume host is unmanaged, log for debugging, and move on
					level.Debug(svc.logger).Log("msg", "cannot determine bitlocker status because no mdm info for host", "host_id", host.ID)
				case err != nil:
					return nil, ctxerr.Wrap(ctx, err, "ensure host mdm info")
				default:
					hde, err := svc.ds.GetMDMWindowsBitLockerStatus(ctx, host)
					if err != nil {
						return nil, ctxerr.Wrap(ctx, err, "get host mdm bitlocker status")
					}
					if hde != nil {
						// overwrite the default disk encryption status
						host.MDM.OSSettings.DiskEncryption = *hde
					}
				}
			}

			profs, err := svc.ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get host mdm windows profiles")
			}
			if profs == nil {
				profs = []fleet.HostMDMWindowsProfile{}
			}
			for _, p := range profs {
				p.Detail = fleet.HostMDMProfileDetail(p.Detail).Message()
				profiles = append(profiles, p.ToHostMDMProfile())
			}

		case "darwin", "ios", "ipados":
			if ac.MDM.EnabledAndConfigured {
				profs, err := svc.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get host mdm profiles")
				}

				// determine disk encryption and action required here based on profiles and
				// raw decryptable key status.
				host.MDM.PopulateOSSettingsAndMacOSSettings(profs, mobileconfig.FleetFileVaultPayloadIdentifier)

				for _, p := range profs {
					if p.Identifier == mobileconfig.FleetFileVaultPayloadIdentifier {
						p.Status = host.MDM.ProfileStatusFromDiskEncryptionState(p.Status)
					}
					p.Detail = fleet.HostMDMProfileDetail(p.Detail).Message()
					profiles = append(profiles, p.ToHostMDMProfile(host.Platform))
				}
			}
		}
	}
	host.MDM.Profiles = &profiles

	if host.IsLUKSSupported() {
		// since Linux hosts don't require MDM to be enabled & configured, explicitly check that disk encryption is
		// enabled for the host's team
		eDE, err := svc.ds.GetConfigEnableDiskEncryption(ctx, host.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host disk encryption enabled setting")
		}
		if eDE {
			status, err := svc.LinuxHostDiskEncryptionStatus(ctx, *host)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get host disk encryption status")
			}
			host.MDM.OSSettings = &fleet.HostMDMOSSettings{
				DiskEncryption: status,
			}

			if status.Status != nil && *status.Status == fleet.DiskEncryptionVerified {
				host.MDM.EncryptionKeyAvailable = true
			}
		}
	}

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

	mdmActions, err := svc.ds.GetHostLockWipeStatus(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host mdm lock/wipe status")
	}

	host.MDM.DeviceStatus = ptr.String(string(mdmActions.DeviceStatus()))
	host.MDM.PendingAction = ptr.String(string(mdmActions.PendingAction()))

	host.Policies = policies

	return &fleet.HostDetail{
		Host:              *host,
		Labels:            labels,
		Packs:             packs,
		Batteries:         &bats,
		MaintenanceWindow: nextMw,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host Query Report
////////////////////////////////////////////////////////////////////////////////

type getHostQueryReportRequest struct {
	ID      uint `url:"id"`
	QueryID uint `url:"query_id"`
}

type getHostQueryReportResponse struct {
	QueryID       uint                          `json:"query_id"`
	HostID        uint                          `json:"host_id"`
	HostName      string                        `json:"host_name"`
	LastFetched   *time.Time                    `json:"last_fetched"`
	ReportClipped bool                          `json:"report_clipped"`
	Results       []fleet.HostQueryReportResult `json:"results"`
	Err           error                         `json:"error,omitempty"`
}

func (r getHostQueryReportResponse) error() error { return r.Err }

func getHostQueryReportEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostQueryReportRequest)

	// Need to return hostname in response even if there are no report results
	host, err := svc.GetHostLite(ctx, req.ID)
	if err != nil {
		return getHostQueryReportResponse{Err: err}, nil
	}

	reportResults, lastFetched, err := svc.GetHostQueryReportResults(ctx, req.ID, req.QueryID)
	if err != nil {
		return getHostQueryReportResponse{Err: err}, nil
	}

	appConfig, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return getHostQueryReportResponse{Err: err}, nil
	}

	isClipped, err := svc.QueryReportIsClipped(ctx, req.QueryID, appConfig.ServerSettings.GetQueryReportCap())
	if err != nil {
		return getHostQueryReportResponse{Err: err}, nil
	}

	return getHostQueryReportResponse{
		QueryID:       req.QueryID,
		HostID:        host.ID,
		HostName:      host.DisplayName(),
		LastFetched:   lastFetched,
		ReportClipped: isClipped,
		Results:       reportResults,
	}, nil
}

func (svc *Service) GetHostQueryReportResults(ctx context.Context, hostID uint, queryID uint) ([]fleet.HostQueryReportResult, *time.Time, error) {
	query, err := svc.ds.Query(ctx, queryID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, nil, ctxerr.Wrap(ctx, err, "get query from datastore")
	}
	if err := svc.authz.Authorize(ctx, query, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	rows, err := svc.ds.QueryResultRowsForHost(ctx, queryID, hostID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get query result rows for host")
	}

	if len(rows) == 0 {
		return []fleet.HostQueryReportResult{}, nil, nil
	}

	var lastFetched *time.Time
	result := make([]fleet.HostQueryReportResult, 0, len(rows))
	for _, row := range rows {
		fetched := row.LastFetched // copy to avoid loop reuse issue
		lastFetched = &fetched     // need to return value even if data is nil

		if row.Data != nil {
			columns := map[string]string{}
			if err := json.Unmarshal(*row.Data, &columns); err != nil {
				return nil, nil, ctxerr.Wrap(ctx, err, "unmarshal query result row data")
			}
			result = append(result, fleet.HostQueryReportResult{Columns: columns})
		}
	}

	return result, lastFetched, nil
}

func (svc *Service) hostIDsAndNamesFromFilters(ctx context.Context, opt fleet.HostListOptions, lid *uint) ([]uint, []string, []*fleet.Host, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	// Load hosts, either from label if provided or from all hosts.
	var hosts []*fleet.Host
	var err error
	if lid != nil {
		hosts, err = svc.ds.ListHostsInLabel(ctx, filter, *lid, opt)
	} else {
		opt.DisableIssues = true // intentionally ignore failing policies
		hosts, err = svc.ds.ListHosts(ctx, filter, opt)
	}
	if err != nil {
		return nil, nil, nil, err
	}

	if len(hosts) == 0 {
		return nil, nil, nil, nil
	}

	hostIDs := make([]uint, 0, len(hosts))
	hostNames := make([]string, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
		hostNames = append(hostNames, h.DisplayName())
	}
	return hostIDs, hostNames, hosts, nil
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
// Put Custom Host Device Mapping
////////////////////////////////////////////////////////////////////////////////

type putHostDeviceMappingRequest struct {
	ID    uint   `url:"id"`
	Email string `json:"email"`
}

type putHostDeviceMappingResponse struct {
	HostID        uint                       `json:"host_id"`
	DeviceMapping []*fleet.HostDeviceMapping `json:"device_mapping"`
	Err           error                      `json:"error,omitempty"`
}

func (r putHostDeviceMappingResponse) error() error { return r.Err }

func putHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*putHostDeviceMappingRequest)
	dms, err := svc.SetCustomHostDeviceMapping(ctx, req.ID, req.Email)
	if err != nil {
		return putHostDeviceMappingResponse{Err: err}, nil
	}
	return putHostDeviceMappingResponse{HostID: req.ID, DeviceMapping: dms}, nil
}

func (svc *Service) SetCustomHostDeviceMapping(ctx context.Context, hostID uint, email string) ([]*fleet.HostDeviceMapping, error) {
	isInstallerSource := svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnOrbitToken)
	if !isInstallerSource {
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}

		host, err := svc.ds.HostLite(ctx, hostID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get host")
		}

		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
			return nil, err
		}
	}

	source := fleet.DeviceMappingCustomOverride
	if isInstallerSource {
		source = fleet.DeviceMappingCustomInstaller
	}
	return svc.ds.SetOrUpdateCustomHostDeviceMapping(ctx, hostID, email, source)
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

func (svc *Service) ListLabelsForHost(ctx context.Context, hostID uint) ([]*fleet.Label, error) {
	// require list hosts permission to view this information
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}
	return svc.ds.ListLabelsForHost(ctx, hostID)
}

type osVersionsRequest struct {
	fleet.ListOptions
	TeamID   *uint   `query:"team_id,optional"`
	Platform *string `query:"platform,optional"`
	Name     *string `query:"os_name,optional"`
	Version  *string `query:"os_version,optional"`
}

type osVersionsResponse struct {
	Meta            *fleet.PaginationMetadata `json:"meta,omitempty"`
	Count           int                       `json:"count"`
	CountsUpdatedAt *time.Time                `json:"counts_updated_at"`
	OSVersions      []fleet.OSVersion         `json:"os_versions"`
	Err             error                     `json:"error,omitempty"`
}

func (r osVersionsResponse) error() error { return r.Err }

func osVersionsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*osVersionsRequest)

	osVersions, count, metadata, err := svc.OSVersions(ctx, req.TeamID, req.Platform, req.Name, req.Version, req.ListOptions, false)
	if err != nil {
		return &osVersionsResponse{Err: err}, nil
	}

	return &osVersionsResponse{
		CountsUpdatedAt: &osVersions.CountsUpdatedAt,
		OSVersions:      osVersions.OSVersions,
		Meta:            metadata,
		Count:           count,
	}, nil
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts fleet.ListOptions, includeCVSS bool) (*fleet.OSVersions, int, *fleet.PaginationMetadata, error) {
	var count int
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, count, nil, err
	}

	if name != nil && version == nil {
		return nil, count, nil, &fleet.BadRequestError{Message: "Cannot specify os_name without os_version"}
	}

	if name == nil && version != nil {
		return nil, count, nil, &fleet.BadRequestError{Message: "Cannot specify os_version without os_name"}
	}

	if opts.OrderKey != "" && opts.OrderKey != "hosts_count" {
		return nil, count, nil, &fleet.BadRequestError{Message: "Invalid order key"}
	}

	if teamID != nil {
		// This auth check ensures we return 403 if the user doesn't have access to the team
		if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
			return nil, count, nil, err
		}

		if *teamID != 0 {
			exists, err := svc.ds.TeamExists(ctx, *teamID)
			if err != nil {
				return nil, count, nil, ctxerr.Wrap(ctx, err, "checking if team exists")
			} else if !exists {
				return nil, count, nil, fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
					WithStatus(http.StatusNotFound)
			}
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, count, nil, fleet.ErrNoContext
	}
	osVersions, err := svc.ds.OSVersions(
		ctx, &fleet.TeamFilter{
			User:            vc.User,
			IncludeObserver: true,
			TeamID:          teamID,
		}, platform, name, version,
	)
	if err != nil && fleet.IsNotFound(err) {
		// It is possible that os exists, but aggregation job has not run yet.
		osVersions = &fleet.OSVersions{}
	} else if err != nil {
		return nil, count, nil, err
	}

	for i := range osVersions.OSVersions {
		if err := svc.populateOSVersionDetails(ctx, &osVersions.OSVersions[i], includeCVSS); err != nil {
			return nil, count, nil, err
		}
	}

	if opts.OrderKey == "hosts_count" && opts.OrderDirection == fleet.OrderAscending {
		sort.Slice(osVersions.OSVersions, func(i, j int) bool {
			return osVersions.OSVersions[i].HostsCount < osVersions.OSVersions[j].HostsCount
		})
	} else {
		sort.Slice(osVersions.OSVersions, func(i, j int) bool {
			return osVersions.OSVersions[i].HostsCount > osVersions.OSVersions[j].HostsCount
		})
	}

	count = len(osVersions.OSVersions)

	var metaData *fleet.PaginationMetadata
	osVersions.OSVersions, metaData = paginateOSVersions(osVersions.OSVersions, opts)

	return osVersions, count, metaData, nil
}

func paginateOSVersions(slice []fleet.OSVersion, opts fleet.ListOptions) ([]fleet.OSVersion, *fleet.PaginationMetadata) {
	metaData := &fleet.PaginationMetadata{
		HasPreviousResults: opts.Page > 0,
	}

	if opts.PerPage == 0 {
		return slice, metaData
	}

	start := opts.Page * opts.PerPage
	if start >= uint(len(slice)) {
		return []fleet.OSVersion{}, metaData
	}

	end := start + opts.PerPage
	if end >= uint(len(slice)) {
		end = uint(len(slice))
	} else {
		metaData.HasNextResults = true
	}

	return slice[start:end], metaData
}

type getOSVersionRequest struct {
	ID     uint  `url:"id"`
	TeamID *uint `query:"team_id,optional"`
}

type getOSVersionResponse struct {
	CountsUpdatedAt *time.Time       `json:"counts_updated_at"`
	OSVersion       *fleet.OSVersion `json:"os_version"`
	Err             error            `json:"error,omitempty"`
}

func (r getOSVersionResponse) error() error { return r.Err }

func getOSVersionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getOSVersionRequest)

	osVersion, updateTime, err := svc.OSVersion(ctx, req.ID, req.TeamID, false)
	if err != nil {
		return getOSVersionResponse{Err: err}, nil
	}
	if osVersion == nil {
		osVersion = &fleet.OSVersion{}
	}

	return getOSVersionResponse{CountsUpdatedAt: updateTime, OSVersion: osVersion}, nil
}

func (svc *Service) OSVersion(ctx context.Context, osID uint, teamID *uint, includeCVSS bool) (*fleet.OSVersion, *time.Time, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionList); err != nil {
		return nil, nil, err
	}

	if teamID != nil && *teamID != 0 {
		// This auth check ensures we return 403 if the user doesn't have access to the team
		if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
			return nil, nil, err
		}
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "checking if team exists")
		} else if !exists {
			return nil, nil, fleet.NewInvalidArgumentError("team_id", fmt.Sprintf("team %d does not exist", *teamID)).
				WithStatus(http.StatusNotFound)
		}
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, nil, fleet.ErrNoContext
	}
	osVersion, updateTime, err := svc.ds.OSVersion(
		ctx, osID, &fleet.TeamFilter{
			User:            vc.User,
			IncludeObserver: true,
			TeamID:          teamID,
		},
	)
	if err != nil {
		if fleet.IsNotFound(err) {
			// We return an empty result here to be consistent with the fleet/os_versions behavior.
			// It is possible the os version exists, but the aggregation job has not run yet.
			return nil, nil, nil
		}
		return nil, nil, err
	}

	if osVersion != nil {
		if err = svc.populateOSVersionDetails(ctx, osVersion, includeCVSS); err != nil {
			return nil, nil, err
		}
	}

	return osVersion, updateTime, nil
}

// PopulateOSVersionDetails populates the GeneratedCPEs and Vulnerabilities for an OSVersion.
func (svc *Service) populateOSVersionDetails(ctx context.Context, osVersion *fleet.OSVersion, includeCVSS bool) error {
	// Populate GeneratedCPEs
	if osVersion.Platform == "darwin" {
		osVersion.GeneratedCPEs = []string{
			fmt.Sprintf("cpe:2.3:o:apple:macos:%s:*:*:*:*:*:*:*", osVersion.Version),
			fmt.Sprintf("cpe:2.3:o:apple:mac_os_x:%s:*:*:*:*:*:*:*", osVersion.Version),
		}
	}

	// Populate Vulnerabilities
	vulns, err := svc.ds.ListVulnsByOsNameAndVersion(ctx, osVersion.NameOnly, osVersion.Version, includeCVSS)
	if err != nil {
		return err
	}

	osVersion.Vulnerabilities = make(fleet.Vulnerabilities, 0) // avoid null in JSON
	for _, vuln := range vulns {
		vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE)
		osVersion.Vulnerabilities = append(osVersion.Vulnerabilities, vuln)
	}
	return nil
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

	var key *fleet.HostDiskEncryptionKey
	if host.IsLUKSSupported() {
		if svc.config.Server.PrivateKey == "" {
			return nil, ctxerr.Wrap(ctx, errors.New("private key is unavailable"), "getting host encryption key")
		}

		key, err = svc.ds.GetHostDiskEncryptionKey(ctx, id)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
		}
		if key.Base64Encrypted == "" {
			return nil, ctxerr.Wrap(ctx, newNotFoundError(), "host encryption key is not set")
		}

		decryptedKey, err := mdm.DecodeAndDecrypt(key.Base64Encrypted, svc.config.Server.PrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypt host encryption key")
		}
		key.DecryptedValue = decryptedKey
	} else {
		key, err = svc.decryptForMDMPlatform(ctx, host)
		if err != nil {
			return nil, err
		}
	}

	err = svc.NewActivity(
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

func (svc *Service) decryptForMDMPlatform(ctx context.Context, host *fleet.Host) (*fleet.HostDiskEncryptionKey, error) {
	// Here we must check if the appropriate MDM is enabled for that particular host's platform.
	var decryptCert *tls.Certificate
	if host.FleetPlatform() == "windows" {
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			return nil, err
		}

		// use Microsoft's WSTEP certificate for decrypting
		cert, _, _, err := svc.config.MDM.MicrosoftWSTEP()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting Microsoft WSTEP certificate to decrypt key")
		}
		decryptCert = cert
	} else {
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			return nil, err
		}

		// use Apple's SCEP certificate for decrypting
		cert, err := assets.CAKeyPair(ctx, svc.ds)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "loading existing assets from the database")
		}
		decryptCert = cert
	}

	key, err := svc.ds.GetHostDiskEncryptionKey(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting host encryption key")
	}
	if key.Decryptable == nil || !*key.Decryptable {
		return nil, ctxerr.Wrap(ctx, newNotFoundError(), "host encryption key is not decryptable")
	}

	decryptedKey, err := mdm.DecryptBase64CMS(key.Base64Encrypted, decryptCert.Leaf, decryptCert.PrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypt host encryption key")
	}

	key.DecryptedValue = string(decryptedKey)
	return key, nil
}

////////////////////////////////////////////////////////////////////////////////
// Host Health
////////////////////////////////////////////////////////////////////////////////

type getHostHealthRequest struct {
	ID uint `url:"id"`
}

type getHostHealthResponse struct {
	Err        error             `json:"error,omitempty"`
	HostID     uint              `json:"host_id,omitempty"`
	HostHealth *fleet.HostHealth `json:"health,omitempty"`
}

func (r getHostHealthResponse) error() error { return r.Err }

func getHostHealthEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostHealthRequest)
	hh, err := svc.GetHostHealth(ctx, req.ID)
	if err != nil {
		return getHostHealthResponse{Err: err}, nil
	}

	// remove TeamID as it's needed for authorization internally but is not part of the external API
	hh.TeamID = nil

	return getHostHealthResponse{HostID: req.ID, HostHealth: hh}, nil
}

func (svc *Service) GetHostHealth(ctx context.Context, id uint) (*fleet.HostHealth, error) {
	svc.authz.SkipAuthorization(ctx)
	hh, err := svc.ds.GetHostHealth(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := svc.authz.Authorize(ctx, hh, fleet.ActionRead); err != nil {
		return nil, err
	}

	return hh, nil
}

func (svc *Service) HostLiteByIdentifier(ctx context.Context, identifier string) (*fleet.HostLite, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLiteByIdentifier(ctx, identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	if err := svc.authz.Authorize(ctx, fleet.Host{
		TeamID: host.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return host, nil
}

func (svc *Service) HostLiteByID(ctx context.Context, id uint) (*fleet.HostLite, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostLiteByID(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by id")
	}

	if err := svc.authz.Authorize(ctx, fleet.Host{
		TeamID: host.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return host, nil
}

func hostListOptionsFromFilters(filter *map[string]interface{}) (*fleet.HostListOptions, *uint, error) {
	var labelID *uint

	if filter == nil {
		return nil, nil, nil
	}

	opt := fleet.HostListOptions{}

	for k, v := range *filter {
		switch k {
		case "label_id":
			if l, ok := v.(float64); ok { // json unmarshals numbers as float64
				lid := uint(l)
				labelID = &lid
			} else {
				return nil, nil, badRequest("label_id must be a number")
			}
		case "team_id":
			if teamID, ok := v.(float64); ok { // json unmarshals numbers as float64
				teamID := uint(teamID)
				opt.TeamFilter = &teamID
			} else {
				return nil, nil, badRequest("team_id must be a number")
			}
		case "status":
			status, ok := v.(string)
			if !ok {
				return nil, nil, badRequest("status must be a string")
			}
			if !fleet.HostStatus(status).IsValid() {
				return nil, nil, badRequest("status must be one of: new, online, offline, missing")
			}
			opt.StatusFilter = fleet.HostStatus(status)
		case "query":
			query, ok := v.(string)
			if !ok {
				return nil, nil, badRequest("query must be a string")
			}
			if query == "" {
				return nil, nil, badRequest("query must not be empty")
			}
			opt.MatchQuery = query

		default:
			return nil, nil, badRequest(fmt.Sprintf("unknown filter key: %s", k))
		}
	}

	return &opt, labelID, nil
}

////////////////////////////////////////////////////////////////////////////////
// Host Labels
////////////////////////////////////////////////////////////////////////////////

type addLabelsToHostRequest struct {
	ID     uint     `url:"id"`
	Labels []string `json:"labels"`
}

type addLabelsToHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addLabelsToHostResponse) error() error { return r.Err }

func addLabelsToHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addLabelsToHostRequest)
	if err := svc.AddLabelsToHost(ctx, req.ID, req.Labels); err != nil {
		return addLabelsToHostResponse{Err: err}, nil
	}
	return addLabelsToHostResponse{}, nil
}

func (svc *Service) AddLabelsToHost(ctx context.Context, id uint, labelNames []string) error {
	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "load host")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionWriteHostLabel); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	labelIDs, err := svc.validateLabelNames(ctx, "add", labelNames)
	if err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}

	if err := svc.ds.AddLabelsToHost(ctx, host.ID, labelIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "add labels to host")
	}

	return nil
}

type removeLabelsFromHostRequest struct {
	ID     uint     `url:"id"`
	Labels []string `json:"labels"`
}

type removeLabelsFromHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r removeLabelsFromHostResponse) error() error { return r.Err }

func removeLabelsFromHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*removeLabelsFromHostRequest)
	if err := svc.RemoveLabelsFromHost(ctx, req.ID, req.Labels); err != nil {
		return removeLabelsFromHostResponse{Err: err}, nil
	}
	return removeLabelsFromHostResponse{}, nil
}

func (svc *Service) RemoveLabelsFromHost(ctx context.Context, id uint, labelNames []string) error {
	host, err := svc.ds.HostLite(ctx, id)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "load host")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionWriteHostLabel); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	labelIDs, err := svc.validateLabelNames(ctx, "remove", labelNames)
	if err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}

	if err := svc.ds.RemoveLabelsFromHost(ctx, host.ID, labelIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "remove labels from host")
	}

	return nil
}

func (svc *Service) validateLabelNames(ctx context.Context, action string, labelNames []string) ([]uint, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	labelNames = server.RemoveDuplicatesFromSlice(labelNames)

	// Filter out empty label string.
	for i, labelName := range labelNames {
		if labelName == "" {
			labelNames = append(labelNames[:i], labelNames[i+1:]...)
			break
		}
	}
	if len(labelNames) == 0 {
		return nil, nil
	}

	labels, err := svc.ds.LabelIDsByName(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	var labelsNotFound []string
	for _, labelName := range labelNames {
		if _, ok := labels[labelName]; !ok {
			labelsNotFound = append(labelsNotFound, "\""+labelName+"\"")
		}
	}

	if len(labelsNotFound) > 0 {
		sort.Slice(labelsNotFound, func(i, j int) bool {
			// Ignore quotes to sort.
			return labelsNotFound[i][1:len(labelsNotFound[i])-1] < labelsNotFound[j][1:len(labelsNotFound[j])-1]
		})
		return nil, &fleet.BadRequestError{
			Message: fmt.Sprintf(
				"Couldn't %s labels. Labels not found: %s. All labels must exist.",
				action,
				strings.Join(labelsNotFound, ", "),
			),
		}
	}

	var dynamicLabels []string
	for labelName, labelID := range labels {
		// we use a global admin filter because we want to get that label
		// regardless of user roles
		filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		label, _, err := svc.ds.Label(ctx, labelID, filter)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "load label from id")
		}
		if label.LabelMembershipType != fleet.LabelMembershipTypeManual {
			dynamicLabels = append(dynamicLabels, "\""+labelName+"\"")
		}
	}

	if len(dynamicLabels) > 0 {
		sort.Slice(dynamicLabels, func(i, j int) bool {
			// Ignore quotes to sort.
			return dynamicLabels[i][1:len(dynamicLabels[i])-1] < dynamicLabels[j][1:len(dynamicLabels[j])-1]
		})
		return nil, &fleet.BadRequestError{
			Message: fmt.Sprintf(
				"Couldn't %s labels. Labels are dynamic: %s. Dynamic labels can not be assigned to hosts manually.",
				action,
				strings.Join(dynamicLabels, ", "),
			),
		}
	}

	labelIDs := make([]uint, 0, len(labels))
	for _, labelID := range labels {
		labelIDs = append(labelIDs, labelID)
	}

	return labelIDs, nil
}

////////////////////////////////////////////////////////////////////////////////
// Host Software
////////////////////////////////////////////////////////////////////////////////

type getHostSoftwareRequest struct {
	ID uint `url:"id"`
	fleet.HostSoftwareTitleListOptions
}

type getHostSoftwareResponse struct {
	Software []*fleet.HostSoftwareWithInstaller `json:"software"`
	Count    int                                `json:"count"`
	Meta     *fleet.PaginationMetadata          `json:"meta,omitempty"`
	Err      error                              `json:"error,omitempty"`
}

func (r getHostSoftwareResponse) error() error { return r.Err }

func getHostSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getHostSoftwareRequest)
	res, meta, err := svc.ListHostSoftware(ctx, req.ID, req.HostSoftwareTitleListOptions)
	if err != nil {
		return getHostSoftwareResponse{Err: err}, nil
	}
	if res == nil {
		res = []*fleet.HostSoftwareWithInstaller{}
	}
	return getHostSoftwareResponse{Software: res, Meta: meta, Count: int(meta.TotalResults)}, nil //nolint:gosec // dismiss G115
}

func (svc *Service) ListHostSoftware(ctx context.Context, hostID uint, opts fleet.HostSoftwareTitleListOptions) ([]*fleet.HostSoftwareWithInstaller, *fleet.PaginationMetadata, error) {
	// if the request is token-authenticated ("My device" page), we don't include
	// software that is not installed but for which there's an installer
	// available for that host (unless the request filters for self-service
	// software only).
	var includeAvailableForInstall bool

	var host *fleet.Host
	if !svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) {
		includeAvailableForInstall = true

		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, nil, err
		}

		h, err := svc.ds.HostLite(ctx, hostID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get host lite")
		}
		host = h

		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, nil, err
		}
	} else {
		h, ok := hostctx.FromContext(ctx)
		if !ok {
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		}
		host = h
	}

	mdmEnrolled, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "checking mdm enrollment status")
	}

	// cursor-based pagination is not supported
	opts.ListOptions.After = ""
	// custom ordering is not supported, always by name (but asc/desc is configurable)
	opts.ListOptions.OrderKey = "name"
	// always include metadata
	opts.ListOptions.IncludeMetadata = true
	opts.IncludeAvailableForInstall = includeAvailableForInstall || opts.SelfServiceOnly
	opts.IsMDMEnrolled = mdmEnrolled

	software, meta, err := svc.ds.ListHostSoftware(ctx, host, opts)
	return software, meta, ctxerr.Wrap(ctx, err, "list host software")
}
