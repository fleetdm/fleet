package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var eeValidVulnSortColumns = []string{
	"cve",
	"hosts_count",
	"created_at",
	"cvss_score",
	"epss_probability",
	"cve_published",
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	opt.ValidSortColumns = eeValidVulnSortColumns
	opt.IsEE = true
	return svc.Service.ListVulnerabilities(ctx, opt)
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (vuln *fleet.VulnerabilityWithMetadata,
	known bool, err error) {
	return svc.Service.Vulnerability(ctx, cve, teamID, true)
}
