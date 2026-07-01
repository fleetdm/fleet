package main

import (
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestInitGeoIP(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	t.Run("no database path returns no-op provider", func(t *testing.T) {
		got := initGeoIP(t.Context(), config.FleetConfig{}, logger)
		assert.IsType(t, &fleet.NoOpGeoIP{}, got)
	})

	t.Run("invalid database path falls back to no-op, not fatal", func(t *testing.T) {
		cfg := config.FleetConfig{}
		cfg.GeoIP.DatabasePath = "/nonexistent/geoip.mmdb"
		got := initGeoIP(t.Context(), cfg, logger)
		assert.IsType(t, &fleet.NoOpGeoIP{}, got)
	})
}
