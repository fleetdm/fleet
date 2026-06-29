package main

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestShouldEnableAuditLog(t *testing.T) {
	premium := &fleet.LicenseInfo{Tier: fleet.TierPremium}
	free := &fleet.LicenseInfo{Tier: fleet.TierFree}

	for _, tc := range []struct {
		name      string
		license   *fleet.LicenseInfo
		enabled   bool
		wantAudit bool
	}{
		{name: "premium and enabled", license: premium, enabled: true, wantAudit: true},
		{name: "premium but disabled", license: premium, enabled: false, wantAudit: false},
		{name: "free but enabled is not allowed", license: free, enabled: true, wantAudit: false},
		{name: "free and disabled", license: free, enabled: false, wantAudit: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := config.FleetConfig{}
			cfg.Activity.EnableAuditLog = tc.enabled
			assert.Equal(t, tc.wantAudit, shouldEnableAuditLog(tc.license, cfg))
		})
	}
}

func TestBuildLoggingConfigMapsConfig(t *testing.T) {
	cfg := config.FleetConfig{}
	cfg.Firehose.Region = "us-east-1"
	cfg.PubSub.Project = "fleet-project"
	cfg.Nats.Server = "nats://localhost:4222"

	got := buildLoggingConfig(cfg)

	assert.Equal(t, "us-east-1", got.Firehose.Region)
	assert.Equal(t, "fleet-project", got.PubSub.Project)
	assert.Equal(t, "nats://localhost:4222", got.Nats.Server)
}
