package microsoft_mdm

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
)

func TestSetupExperienceStatusToESP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    fleet.SetupExperienceStatusResultStatus
		expected uint
	}{
		{fleet.SetupExperienceStatusSuccess, ESPItemStatusCompleted},
		{fleet.SetupExperienceStatusFailure, ESPItemStatusError},
		{fleet.SetupExperienceStatusPending, ESPItemStatusNotInstalled},
		{fleet.SetupExperienceStatusRunning, ESPItemStatusNotInstalled},
		{fleet.SetupExperienceStatusCancelled, ESPItemStatusNotInstalled},
	}
	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			assert.Equal(t, tt.expected, SetupExperienceStatusToESP(tt.input))
		})
	}
}
