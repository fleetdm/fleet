package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var freeValidVulnSortColumns = []string{
	"cve",
	"host_count",
	"host_count_updated_at",
	"created_at",
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, error) {
	if !opt.HasValidSortColumn() {
		return nil, badRequest("invalid order key")
	}

	return svc.ds.ListVulnerabilities(ctx, opt)
}
