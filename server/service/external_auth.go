package service

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/service/middleware/auth/externalauth"
	"github.com/throttled/throttled/v2"
)

// jwtAuthBurst lets an automation authenticate its first requests at once before the
// per-minute rate applies (a fleetctl gitops run makes many requests in a burst).
const jwtAuthBurst = 100

// NewExternalAuthenticator returns nil when no auth.jwt_issuers are configured, in which
// case JWT-shaped bearer tokens are rejected.
func NewExternalAuthenticator(ctx context.Context, cfg config.FleetConfig, users externalauth.UserLookup,
	limitStore throttled.GCRAStore, logger *slog.Logger,
) (*externalauth.Authenticator, error) {
	if len(cfg.Auth.JWTIssuers) == 0 {
		return nil, nil
	}
	perMin := cfg.Auth.JWTRateLimitPerMinute
	if perMin <= 0 {
		perMin = config.DefaultJWTRateLimitPerMinute
	}
	limiter, err := throttled.NewGCRARateLimiter(limitStore, throttled.RateQuota{
		MaxRate:  throttled.PerMin(perMin),
		MaxBurst: jwtAuthBurst,
	})
	if err != nil {
		return nil, err
	}
	return externalauth.New(ctx, cfg.Auth.JWTIssuers, users, limiter, logger.With("component", "external-jwt-auth"))
}
