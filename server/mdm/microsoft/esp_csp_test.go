package microsoft_mdm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertAppInstallationState checks that the SyncML output contains a Replace
// block associating the given app name with the expected InstallationState value.
func assertAppInstallationState(t *testing.T, raw, appName string, expectedStatus uint) {
	t.Helper()
	// The template produces a <Replace> block where the LocURI containing the
	// app name's InstallationState is followed by a <Data> element with the status.
	needle := fmt.Sprintf("Apps/%s/InstallationState</LocURI>", appName)
	idx := strings.Index(raw, needle)
	require.NotEqual(t, -1, idx, "expected InstallationState entry for %q", appName)

	// Find the <Data>N</Data> that follows within the same <Replace> block.
	after := raw[idx:]
	dataNeedle := fmt.Sprintf("<Data>%d</Data>", expectedStatus)
	endNeedle := "</Replace>"
	endIdx := strings.Index(after, endNeedle)
	require.NotEqual(t, -1, endIdx, "expected closing </Replace> after InstallationState for %q", appName)

	block := after[:endIdx]
	assert.Contains(t, block, dataNeedle,
		"app %q should have InstallationState=%d", appName, expectedStatus)
}

func TestESPInitialCommand(t *testing.T) {
	t.Parallel()

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
		// Should have timeout, block, and skip-user settings (int format per DMClient CSP spec)
		assert.Contains(t, raw, "TimeOutUntilSyncFailure")
		assert.Contains(t, raw, fmt.Sprintf("<Data>%d</Data>", ESPTimeoutSeconds))
		assert.Contains(t, raw, "BlockInStatusPage")
		assert.Contains(t, raw, "SkipUserStatusPage")
		assert.Contains(t, raw, "<Format>int</Format>")
		assert.Contains(t, raw, "<Data>1</Data>")
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
					IsSCEP:      false,
				},
				{
					ProfileUUID: "prof-2",
					TopLocURI:   "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP/cert1",
					IsSCEP:      true, // SCEP profiles tracked under Certificates, not Security policies
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

		// prof-1 (non-SCEP) tracked as security policy
		assert.Contains(t, raw, "ExpectedPolicies/./Device/Vendor/MSFT/Policy/Config/WiFi")
		assert.Contains(t, raw, "TrackingPolicies/prof-1")

		// prof-2 (SCEP) tracked under Certificates, not Security policies
		assert.NotContains(t, raw, "ExpectedPolicies/./Device/Vendor/MSFT/ClientCertificateInstall")
		assert.NotContains(t, raw, "TrackingPolicies/prof-2")
		assert.Contains(t, raw, "ExpectedSCEPCerts/prof-2")
		assert.NotContains(t, raw, "ExpectedSCEPCerts/prof-1")

		// Software tracking -- verify each app has the correct InstallationState
		assertAppInstallationState(t, raw, "Fleet osquery", ESPItemStatusNotInstalled)
		assertAppInstallationState(t, raw, "Chrome", ESPItemStatusNotInstalled)
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
		assert.Contains(t, raw, "App &lt;with&gt; &amp; &#34;special&#34; chars")
	})
}

func TestESPStatusUpdateCommand(t *testing.T) {
	t.Parallel()

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

		// Verify each app is associated with the correct status value
		assertAppInstallationState(t, raw, "Fleet osquery", ESPItemStatusCompleted)
		assertAppInstallationState(t, raw, "Chrome", ESPItemStatusError)
		assertAppInstallationState(t, raw, "Slack", ESPItemStatusNotInstalled)
	})
}

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
