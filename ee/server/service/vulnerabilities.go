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
	"published",
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	opt.ValidSortColumns = eeValidVulnSortColumns
	opt.IsEE = true
	return svc.Service.ListVulnerabilities(ctx, opt)
}

func (svc *Service) Vulnerability(ctx context.Context, cve string, teamID *uint, useCVSScores bool) (*fleet.VulnerabilityWithMetadata, error) {
	return svc.Service.Vulnerability(ctx, cve, teamID, true)
}
