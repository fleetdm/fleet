package main

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/stretchr/testify/assert"
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
