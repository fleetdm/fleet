package service

import (
	"context"
	"fmt"
	"regexp"
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
	Vulnerabilities    []fleet.VulnerabilityWithMetadata `json:"vulnerabilities"`
	Count              uint                              `json:"count"`
	CountsUpdatedAt    time.Time                         `json:"counts_updated_at"`
	Meta               *fleet.PaginationMetadata         `json:"meta,omitempty"`
	Err                error                             `json:"error,omitempty"`
	KnownVulnerability *bool                             `json:"known_vulnerability,omitempty"`
}

// Allow formats like: CVE-2017-12345, cve-2017-12345 or 2017-12345
var cveRegex = regexp.MustCompile(`(?i)^(CVE-)?\d{4}-\d{4}\d*$`)

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

	updatedAt := time.Now()
	for _, vuln := range vulns {
		if vuln.HostsCountUpdatedAt.Before(updatedAt) {
			updatedAt = vuln.HostsCountUpdatedAt
		}
	}

	// Check whether the query was for a vulnerability known to fleet
	var knownVulnerability *bool
	if len(request.ListOptions.MatchQuery) > 0 {
		query := request.ListOptions.MatchQuery
		matches := cveRegex.FindStringSubmatch(query)
		if matches != nil {
			const cvePrefix = "CVE-"
			if len(matches) > 1 && matches[1] == "" {
				// If CVE prefix was missing, we add it
				query = cvePrefix + query
			}
			// As an optimization, we first check if the CVE was one of the ones returned
			// by the query. If it was, we already know it's known to Fleet.
			var known bool
			for _, vuln := range vulns {
				if vuln.CVE.CVE == query {
					known = true
					break
				}
			}
			if !known {
				known, err = svc.IsCVEKnownToFleet(ctx, query)
				if err != nil {
					return listVulnerabilitiesResponse{Err: err}, nil
				}
			}
			knownVulnerability = &known
		}
	}

	return listVulnerabilitiesResponse{
		Vulnerabilities:    vulns,
		Meta:               meta,
		Count:              count,
		CountsUpdatedAt:    updatedAt,
		KnownVulnerability: knownVulnerability,
	}, nil
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
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
		vulns[i].DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE.CVE)
	}

	return vulns, meta, nil
}

func (svc *Service) CountVulnerabilities(ctx context.Context, opts fleet.VulnListOptions) (uint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opts.TeamID,
	}, fleet.ActionRead); err != nil {
		return 0, err
	}

	return svc.ds.CountVulnerabilities(ctx, opts)
}

func (svc *Service) IsCVEKnownToFleet(ctx context.Context, cve string) (bool, error) {
	return svc.ds.IsCVEKnownToFleet(ctx, cve)
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

	vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE.CVE)

	osVersions, _, err := svc.ListOSVersionsByCVE(ctx, vuln.CVE.CVE, request.TeamID)
	if err != nil {
		return getVulnerabilityResponse{Err: err}, nil
	}

	software, _, err := svc.ListSoftwareByCVE(ctx, vuln.CVE.CVE, request.TeamID)
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

	if teamID != nil && *teamID != 0 {
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
