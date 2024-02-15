package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

var freeValidVulnSortColumns = []string{
	"cve",
	"hosts_count",
	"host_count_updated_at",
	"created_at",
}

type listVulnerabilitiesRequest struct {
	fleet.VulnListOptions
}

type listVulnerabilitiesResponse struct {
	Vulnerabilities []fleet.VulnerabilityWithMetadata `json:"vulnerabilities"`
	Count           uint                              `json:"count"`
	Meta            *fleet.PaginationMetadata         `json:"meta,omitempty"`
	Err             error                             `json:"error,omitempty"`
}

func (r listVulnerabilitiesResponse) error() error { return r.Err }

func listVulnerabilitiesEndpoint(ctx context.Context, req interface{}, svc fleet.Service) (errorer, error) {
	request := req.(*listVulnerabilitiesRequest)
	vulns, meta, err := svc.ListVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return listVulnerabilitiesResponse{Err: err}, nil
	}

	count, err := svc.CountVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return listVulnerabilitiesResponse{Err: err}, nil
	}

	return listVulnerabilitiesResponse{
		Vulnerabilities: vulns,
		Meta:            meta,
		Count:           count,
	}, nil
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: &opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	if len(opt.ValidSortColumns) == 0 {
		opt.ValidSortColumns = freeValidVulnSortColumns
	}

	if !opt.HasValidSortColumn() {
		return nil, nil, badRequest("invalid order key")
	}

	if opt.KnownExploit && !opt.IsEE {
		return nil, nil, fleet.ErrMissingLicense
	}

	vulns, meta, err := svc.ds.ListVulnerabilities(ctx, opt)
	if err != nil {
		return nil, nil, err
	}

	for i, vuln := range vulns {
		if vuln.Source == fleet.MSRCSource {
			vulns[i].DetailsLink = fmt.Sprintf("https://msrc.microsoft.com/update-guide/en-US/vulnerability/%s", vuln.CVE)
		} else {
			vulns[i].DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE)
		}
	}

	return vulns, meta, nil
}

func (svc *Service) CountVulnerabilities(ctx context.Context, opts fleet.VulnListOptions) (uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: &opts.TeamID,
	}, fleet.ActionRead); err != nil {
		return 0, err
	}

	return svc.ds.CountVulnerabilities(ctx, opts)
}

type getVulnerabilityRequest struct {
	CVE    string `url:"cve"`
	TeamID *uint  `query:"team_id,optional"`
}

type getVulnerabilityResponse struct {
	Vulnerability *fleet.VulnerabilityWithMetadata `json:"vulnerability"`
	OSVersions    []*fleet.VulnerableOS            `json:"os_versions"`
	Software      []*fleet.VulnerableSoftware      `json:"software"`
	Err           error                            `json:"error,omitempty"`
}

func (r getVulnerabilityResponse) error() error { return r.Err }

func getVulnerabilityEndpoint(ctx context.Context, req interface{}, svc fleet.Service) (errorer, error) {
	request := req.(*getVulnerabilityRequest)

	vuln, err := svc.Vulnerability(ctx, request.CVE, request.TeamID, false)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	if vuln.Source == fleet.MSRCSource {
		vuln.DetailsLink = fmt.Sprintf("https://msrc.microsoft.com/update-guide/en-US/vulnerability/%s", vuln.CVE)
	} else {
		vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE)
	}

	osVersions, _, err := svc.ListOSVersionsByCVE(ctx, vuln.CVE, request.TeamID)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	software, _, err := svc.ListSoftwareByCVE(ctx, vuln.CVE, request.TeamID)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	return getVulnerabilityResponse{
		Vulnerability: vuln,
		OSVersions:    osVersions,
		Software:      software,
	}, nil
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (*fleet.VulnerabilityWithMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	if teamID != nil {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "checking if team exists")
		} else if !exists {
			return nil, authz.ForbiddenWithInternal("team does not exist", nil, nil, nil)
		}
	}

	vuln, err := svc.ds.Vulnerability(ctx, cve, teamID, useCVSScores)
	if err != nil {
		return nil, err
	}

	return vuln, nil
}

func (svc *Service) ListOSVersionsByCVE(ctx context.Context, cve string, teamID *uint) (result []*fleet.VulnerableOS, updatedAt time.Time, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, updatedAt, err
	}
	return svc.ds.OSVersionsByCVE(ctx, cve, teamID)
}

func (svc *Service) ListSoftwareByCVE(ctx context.Context, cve string, teamID *uint) (result []*fleet.VulnerableSoftware, updatedAt time.Time, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, updatedAt, err
	}
	return svc.ds.SoftwareByCVE(ctx, cve, teamID)
}
