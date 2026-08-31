package main

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// initGeoIP returns the GeoIP provider for the server. GeoIP is best-effort:
// when no database path is configured, or the MaxMind database fails to load,
// it falls back to a no-op provider and logs the problem rather than aborting
// startup.
func initGeoIP(ctx context.Context, cfg config.FleetConfig, logger *slog.Logger) fleet.GeoIP {
	if cfg.GeoIP.DatabasePath == "" {
		return &fleet.NoOpGeoIP{}
	}

	maxmind, err := fleet.NewMaxMindGeoIP(logger, cfg.GeoIP.DatabasePath)
	if err != nil {
		logger.ErrorContext(ctx, "failed to initialize maxmind geoip, check database path", "database_path",
			cfg.GeoIP.DatabasePath, "error", err)
		return &fleet.NoOpGeoIP{}
	}
	return maxmind
}
