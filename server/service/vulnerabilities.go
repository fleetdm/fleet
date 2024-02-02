package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

var validSortColums = []string{
	"cve",
	"host_count",
	"host_count_updated_at",
	"created_at",
}

func (svc *Service) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions, validSortColums []string) ([]fleet.VulnerabilityWithMetadata, error) {
	if opt.OrderKey != "" && !isValidSortColumn(opt.OrderKey) {
		return nil, badRequest("invalid order key")
	}

	return svc.ds.ListVulnerabilities(ctx, opt)
}

func isValidSortColumn(column string) bool {
	for _, c := range validSortColums {
		if c == column {
			return true
		}
	}
	return false
}
