package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListSoftware(ctx context.Context, opts fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
	// reuse ListSoftware, but include cve scores in premium version
	// unless without_vulnerability_details is set to true
	// including these details causes a lot of memory bloat
	if (opts.MaximumCVSS > 0 || opts.MinimumCVSS > 0 || opts.KnownExploit) || !opts.WithoutVulnerabilityDetails {
		opts.IncludeCVEScores = true
	}
	return svc.Service.ListSoftware(ctx, opts)
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint, teamID *uint, _ bool) (*fleet.Software, error) {
	// reuse SoftwareByID, but include cve scores in premium version
	return svc.Service.SoftwareByID(ctx, id, teamID, true)
}
