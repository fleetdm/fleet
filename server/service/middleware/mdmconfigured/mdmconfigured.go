// Package mdmconfigured implements a middleware that ensures that MDM is
// configured.
package mdmconfigured

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type Middleware struct {
	svc fleet.Service
}

func NewAppleMiddleware(svc fleet.Service) *Middleware {
	return &Middleware{svc: svc}
}

func (m *Middleware) Verify() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (interface{}, error) {

			if err := m.svc.VerifyMDMAppleConfigured(ctx); err != nil {
				return nil, err
			}

			return next(ctx, req)
		}
	}
}
