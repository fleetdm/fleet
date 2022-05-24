package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// func (svc *Service) GetHost(ctx context.Context, id uint, includeCVEScores bool) (*fleet.HostDetail, error) {
// 	// reuse GetHost, but include cve scores in premium version
// 	return svc.Service.GetHost(ctx, id, true)
// }

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, includeCVEScores bool) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include cve scores in premium version
	return svc.Service.HostByIdentifier(ctx, identifier, true)
}
