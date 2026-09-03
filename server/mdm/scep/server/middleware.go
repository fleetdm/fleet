package scepserver

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

// acmeOnlyEnrollmentMiddleware refuses all SCEP issuance when only
// ACME-attested Apple Business enrollment is allowed. Fleet never solicits
// SCEP renewals in that mode, so any PKIOperation is stale or hostile.
func ACMEOnlyEnrollmentMiddleware(appCfgGetter fleet.GetsAppConfig) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request any) (response any, err error) {
			cfg, err := appCfgGetter.AppConfig(ctx)
			if err != nil {
				return nil, fmt.Errorf("loading app config for SCEP enrollment policy: %w", err)
			}
			if cfg.MDM.IsAppleMDMSCEPBlocked() {
				return nil, &fleet.ABOnlyEnrollmentForbiddenError{}
			}
			return next(ctx, request)
		}
	}
}

// EndpointLoggingMiddleware returns an endpoint middleware that logs the
// duration of each invocation, and the resulting error, if any.
func EndpointLoggingMiddleware(logger *slog.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request any) (response any, err error) {
			var attrs []slog.Attr
			if oper, ok := request.(interface {
				scepOperation() string
			}); ok {
				attrs = append(attrs, slog.String("op", oper.scepOperation()))
			}
			defer func(begin time.Time) {
				logErr := err
				if logErr == nil {
					if resp, ok := response.(SCEPResponse); ok {
						logErr = resp.Err
					}
				}
				attrs = append(attrs, slog.Any("error", logErr), slog.Duration("took", time.Since(begin)))
				logger.LogAttrs(ctx, slog.LevelInfo, "scep endpoint", attrs...)
			}(time.Now())
			return next(ctx, request)
		}
	}
}
