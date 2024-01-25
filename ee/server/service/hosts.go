package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

func (svc *Service) OSVersions(ctx context.Context, teamID *uint, platform *string, name *string, version *string, opts fleet.ListOptions, includeCVSS bool) (*fleet.OSVersions, int, *fleet.PaginationMetadata, error) {
	// reuse OSVersions, but include premium options
	return svc.Service.OSVersions(ctx, teamID, platform, name, version, opts, true)
}
