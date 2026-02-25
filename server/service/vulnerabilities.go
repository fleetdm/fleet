package service

import (
	"context"
	"fmt"
	"net/http"
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

type cveNotFoundError struct{}

var _ fleet.NotFoundError = (*cveNotFoundError)(nil)

func (p cveNotFoundError) Error() string {
	return "This is not a known CVE. None of Fleet’s vulnerability sources are aware of this CVE."
}

func (p cveNotFoundError) IsNotFound() bool {
	return true
}

func (p cveNotFoundError) IsClientError() bool {
	return true
}

// Allow formats like: CVE-2017-12345, cve-2017-12345
var cveRegex = regexp.MustCompile(`(?i)^CVE-\d{4}-\d{4}\d*$`)

func listVulnerabilitiesEndpoint(ctx context.Context, req interface{}, svc fleet.Service) (fleet.Errorer, error) {
	request := req.(*fleet.ListVulnerabilitiesRequest)
	vulns, meta, err := svc.ListVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return fleet.ListVulnerabilitiesResponse{Err: err}, nil
	}

	count, err := svc.CountVulnerabilities(ctx, request.VulnListOptions)
	if err != nil {
		return fleet.ListVulnerabilitiesResponse{Err: err}, nil
	}

	updatedAt := time.Now()
	for _, vuln := range vulns {
		if vuln.HostsCountUpdatedAt.Before(updatedAt) {
			updatedAt = vuln.HostsCountUpdatedAt
		}
	}

	return fleet.ListVulnerabilitiesResponse{
		Vulnerabilities: vulns,
		Meta:            meta,
		Count:           count,
		CountsUpdatedAt: updatedAt,
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

func getVulnerabilityEndpoint(ctx context.Context, req interface{}, svc fleet.Service) (fleet.Errorer, error) {
	request := req.(*fleet.GetVulnerabilityRequest)

	vuln, known, err := svc.Vulnerability(ctx, request.CVE, request.TeamID, false)
	if err != nil {
		return fleet.GetVulnerabilityResponse{Err: err}, nil
	}
	if vuln == nil && known {
		// Return 204 status code if the vulnerability is known to Fleet but does not match any host software/OS
		return fleet.GetVulnerabilityResponse{StatusCode: http.StatusNoContent}, nil
	}

	vuln.DetailsLink = fmt.Sprintf("https://nvd.nist.gov/vuln/detail/%s", vuln.CVE.CVE)

	osVersions, _, err := svc.ListOSVersionsByCVE(ctx, vuln.CVE.CVE, request.TeamID)
	if err != nil {
		return fleet.GetVulnerabilityResponse{Err: err}, nil
	}

	software, _, err := svc.ListSoftwareByCVE(ctx, vuln.CVE.CVE, request.TeamID)
	if err != nil {
		return fleet.GetVulnerabilityResponse{Err: err}, nil
	}

	return fleet.GetVulnerabilityResponse{
		Vulnerability: vuln,
		OSVersions:    osVersions,
		Software:      software,
	}, nil
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (vuln *fleet.VulnerabilityWithMetadata,
	known bool, err error,
) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, false, err
	}

	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, false, err
	}

	if !cveRegex.Match([]byte(cve)) {
		return nil, false, badRequest("That vulnerability (CVE) is not valid. Try updating your search to use CVE format: \"CVE-YYYY-<4 or more digits>\"")
	}

	if teamID != nil && *teamID != 0 {
		exists, err := svc.ds.TeamExists(ctx, *teamID)
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "checking if team exists")
		} else if !exists {
			return nil, false, authz.ForbiddenWithInternal("team does not exist", nil, nil, nil)
		}
	}

	vuln, err = svc.ds.Vulnerability(ctx, cve, teamID, useCVSScores)
	switch {
	case fleet.IsNotFound(err):
		var errKnown error
		known, errKnown = svc.ds.IsCVEKnownToFleet(ctx, cve)
		if errKnown != nil {
			return nil, false, errKnown
		}
		if !known {
			return nil, false, cveNotFoundError{}
		}
	case err != nil:
		return nil, false, err
	default:
		known = true
	}

	return vuln, known, nil
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
