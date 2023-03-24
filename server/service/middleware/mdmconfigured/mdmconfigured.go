// Package mdmconfigured implements a middleware that ensures that MDM is
// configured.
package mdmconfigured

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type appConfigGetter func(context.Context) (*fleet.AppConfig, error)

type Middleware struct {
	appConfigGetter
}

func NewMiddleware(cfg appConfigGetter) *Middleware {
	return &Middleware{appConfigGetter: cfg}
}

func (m *Middleware) Verify() endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, req interface{}) (response interface{}, err error) {
			res, err := next(ctx, req)
			if err != nil {
				return res, err
			}

			appCfg, err := m.appConfigGetter(ctx)
			if err != nil {
				return nil, err
			}

			if !appCfg.MDM.EnabledAndConfigured {
				return nil, fleet.MDMNotConfiguredError{}
			}

			return res, nil
		}
	}
}
