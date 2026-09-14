package android

import (
	"context"

	licensectx "github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
)

var _ android.Service = (*Service)(nil)

// Service wraps a core android.Service with premium feature gating.
type Service struct {
	android.Service
}

func NewService(svc android.Service) *Service {
	return &Service{Service: svc}
}

func (svc *Service) GetZeroTouchConfiguration(ctx context.Context, teamID *uint) (*android.ZeroTouchConfigurationResponse, error) {
	if !licensectx.IsPremium(ctx) {
		return nil, fleet.ErrMissingLicense
	}
	return svc.Service.GetZeroTouchConfiguration(ctx, teamID)
}
