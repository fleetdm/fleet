package microsoft_mdm

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESPInitialCommand(t *testing.T) {
	t.Run("missing cmdUUID", func(t *testing.T) {
		_, err := ESPInitialCommand(ESPInitialCommandSpec{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cmdUUID is required")
	})

	t.Run("no profiles or software", func(t *testing.T) {
		cmd, err := ESPInitialCommand(ESPInitialCommandSpec{
			CmdUUID: "uuid-1",
		})
		require.NoError(t, err)
		require.NotNil(t, cmd)
		assert.Equal(t, "uuid-1", cmd.CommandUUID)

		raw := string(cmd.RawCommand)
		// Should have timeout and block settings
		assert.Contains(t, raw, "TimeOutUntilSyncFailure")
		assert.Contains(t, raw, "10800")
		assert.Contains(t, raw, "BlockInStatusPage")
		assert.Contains(t, raw, "true")
		// Should not have any profile or app entries
		assert.NotContains(t, raw, "ExpectedPolicies")
		assert.NotContains(t, raw, "TrackingPolicies")
	})

	t.Run("with profiles and software", func(t *testing.T) {
		cmd, err := ESPInitialCommand(ESPInitialCommandSpec{
			CmdUUID: "uuid-2",
			Profiles: []ESPProfileTrackingInfo{
				{
					ProfileUUID: "prof-1",
					TopLocURI:   "./Device/Vendor/MSFT/Policy/Config/WiFi",
					HasSCEP:     false,
				},
				{
					ProfileUUID: "prof-2",
					TopLocURI:   "./Device/Vendor/MSFT/Policy/Config/VPN",
					HasSCEP:     true,
				},
			},
			Software: []ESPSoftwareTrackingInfo{
				{Name: "Fleet osquery", Status: ESPItemStatusNotInstalled},
				{Name: "Chrome", Status: ESPItemStatusNotInstalled},
			},
		})
		require.NoError(t, err)

		raw := string(cmd.RawCommand)

		// Timeout and block
		assert.Contains(t, raw, "TimeOutUntilSyncFailure")
		assert.Contains(t, raw, "BlockInStatusPage")

		// Profile expected policies
		assert.Contains(t, raw, "ExpectedPolicies/./Device/Vendor/MSFT/Policy/Config/WiFi")
		assert.Contains(t, raw, "ExpectedPolicies/./Device/Vendor/MSFT/Policy/Config/VPN")

		// Profile tracking policies
		assert.Contains(t, raw, "TrackingPolicies/prof-1")
		assert.Contains(t, raw, "TrackingPolicies/prof-2")

		// SCEP only for prof-2
		assert.Contains(t, raw, "ExpectedSCEPCerts/prof-2")
		assert.NotContains(t, raw, "ExpectedSCEPCerts/prof-1")

		// Software tracking
		assert.Contains(t, raw, "TrackingPolicies/Apps/Fleet osquery")
		assert.Contains(t, raw, "TrackingPolicies/Apps/Chrome")

		// InstallationState should be 1 (NotInstalled) for both
		assert.Equal(t, 2, strings.Count(raw, "<Data>1</Data>"))
	})

	t.Run("xml escaping in names", func(t *testing.T) {
		cmd, err := ESPInitialCommand(ESPInitialCommandSpec{
			CmdUUID: "uuid-3",
			Software: []ESPSoftwareTrackingInfo{
				{Name: "App <with> & \"special\" chars", Status: ESPItemStatusNotInstalled},
			},
		})
		require.NoError(t, err)
		raw := string(cmd.RawCommand)
		assert.Contains(t, raw, "App &lt;with&gt; &amp; &quot;special&quot; chars")
	})
}

func TestESPStatusUpdateCommand(t *testing.T) {
	t.Run("missing cmdUUID", func(t *testing.T) {
		_, err := ESPStatusUpdateCommand(ESPStatusUpdateSpec{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cmdUUID is required")
	})

	t.Run("status updates", func(t *testing.T) {
		cmd, err := ESPStatusUpdateCommand(ESPStatusUpdateSpec{
			CmdUUID: "uuid-4",
			Software: []ESPSoftwareTrackingInfo{
				{Name: "Fleet osquery", Status: ESPItemStatusCompleted},
				{Name: "Chrome", Status: ESPItemStatusError},
				{Name: "Slack", Status: ESPItemStatusNotInstalled},
			},
		})
		require.NoError(t, err)

		raw := string(cmd.RawCommand)

		// Check each status value
		assert.Contains(t, raw, "Apps/Fleet osquery/InstallationState")
		assert.Contains(t, raw, "Apps/Chrome/InstallationState")
		assert.Contains(t, raw, "Apps/Slack/InstallationState")

		// Completed=3, Error=4, NotInstalled=1
		assert.Contains(t, raw, "<Data>3</Data>")
		assert.Contains(t, raw, "<Data>4</Data>")
		assert.Contains(t, raw, "<Data>1</Data>")
	})
}

func TestSetupExperienceStatusToESP(t *testing.T) {
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
