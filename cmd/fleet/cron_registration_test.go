package main

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVulnerabilityProcessingDisabled(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  config.VulnerabilitiesConfig
		want bool
	}{
		{
			name: "enabled by default",
			cfg:  config.VulnerabilitiesConfig{CurrentInstanceChecks: "auto"},
			want: false,
		},
		{
			name: "disabled via disable_schedule",
			cfg:  config.VulnerabilitiesConfig{DisableSchedule: true, CurrentInstanceChecks: "auto"},
			want: true,
		},
		{
			name: "disabled via current_instance_checks no",
			cfg:  config.VulnerabilitiesConfig{CurrentInstanceChecks: "no"},
			want: true,
		},
		{
			name: "disabled via legacy current_instance_checks 0",
			cfg:  config.VulnerabilitiesConfig{CurrentInstanceChecks: "0"},
			want: true,
		},
		{
			name: "empty current_instance_checks does not disable",
			cfg:  config.VulnerabilitiesConfig{CurrentInstanceChecks: ""},
			want: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, vulnerabilityProcessingDisabled(tc.cfg))
		})
	}
}

func TestLegacyAPNsPusherInterval(t *testing.T) {
	cases := []struct {
		name         string
		value        string
		wantInterval time.Duration
		wantEnabled  bool
		wantErr      bool
	}{
		{"unset registers the sweep", "", 0, false, false},
		{"valid interval registers the legacy pusher", "2m", 2 * time.Minute, true, false},
		{"garbage falls back to 1m with an error", "garbage", time.Minute, true, true},
		{"non-positive falls back to 1m with an error", "-5m", time.Minute, true, true},
		{"zero falls back to 1m with an error", "0", time.Minute, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			getenv := func(key string) string {
				require.Equal(t, "FLEET_MDM_APPLE_LEGACY_APNS_PUSHER_INTERVAL", key)
				return c.value
			}
			interval, enabled, err := legacyAPNsPusherInterval(getenv)
			require.Equal(t, c.wantEnabled, enabled)
			require.Equal(t, c.wantInterval, interval)
			if c.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
