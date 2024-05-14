package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestSoftwareIterQueryOptionsIsValid(t *testing.T) {
	testCases := []struct {
		excluded   []string
		included   []string
		isNotValid bool
	}{
		{
			excluded: nil,
			included: nil,
		},
		{
			excluded: []string{"a", "b"},
			included: nil,
		},
		{
			excluded: nil,
			included: []string{"a", "b"},
		},
		{
			excluded:   []string{"a", "b"},
			included:   []string{"a"},
			isNotValid: true,
		},
		{
			excluded:   []string{"a"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
		{
			excluded:   []string{"c"},
			included:   []string{"a", "b"},
			isNotValid: true,
		},
	}

	for _, tC := range testCases {
		sut := SoftwareIterQueryOptions{
			ExcludedSources: tC.excluded,
			IncludedSources: tC.included,
		}

		if tC.isNotValid {
			require.False(t, sut.IsValid())
		} else {
			require.True(t, sut.IsValid())
		}
	}
}

func TestParseSoftwareLastOpenedAtRowValue(t *testing.T) {
	// Some macOS apps return last_opened_at=-1.0 on apps
	// that were never opened.
	lastOpenedAt, err := ParseSoftwareLastOpenedAtRowValue("-1.0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	// Our software queries hardcode to 0 if such info is not available.
	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("0")
	require.NoError(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("foobar")
	require.Error(t, err)
	require.Zero(t, lastOpenedAt)

	lastOpenedAt, err = ParseSoftwareLastOpenedAtRowValue("1694026958")
	require.NoError(t, err)
	require.NotZero(t, lastOpenedAt)
}

func TestEnhanceOutputDetails(t *testing.T) {
	tests := []struct {
		name                            string
		hsr                             HostSoftwareInstallerResult
		expectedPreInstallQueryOutput   string
		expectedOutput                  string
		expectedPostInstallScriptOutput string
	}{
		{
			name: "pending status",
			hsr: HostSoftwareInstallerResult{
				Status: SoftwareInstallerPending,
			},
			expectedPreInstallQueryOutput:   "",
			expectedOutput:                  "",
			expectedPostInstallScriptOutput: "",
		},
		{
			name: "non-pending status with empty PreInstallQueryOutput and successful install",
			hsr: HostSoftwareInstallerResult{
				Status:                SoftwareInstallerInstalled,
				InstallScriptExitCode: ptr.Int(0),
				PreInstallQueryOutput: "1",
			},
			expectedPreInstallQueryOutput:   "Query returned result\nProceeding to install...",
			expectedOutput:                  "Installing software...\nSuccess\n",
			expectedPostInstallScriptOutput: "",
		},
		{
			name: "non-pending status with empty PreInstallQueryOutput and failed install",
			hsr: HostSoftwareInstallerResult{
				Status:                SoftwareInstallerFailed,
				InstallScriptExitCode: ptr.Int(1),
				PreInstallQueryOutput: "1",
			},
			expectedPreInstallQueryOutput:   "Query returned result\nProceeding to install...",
			expectedOutput:                  "Installing software...\nFailed\n",
			expectedPostInstallScriptOutput: "",
		},
		{
			name: "non-pending status with non-empty PreInstallQueryOutput and disabled scripts",
			hsr: HostSoftwareInstallerResult{
				Status:                SoftwareInstallerFailed,
				InstallScriptExitCode: ptr.Int(-2),
				PreInstallQueryOutput: "1",
			},
			expectedPreInstallQueryOutput:   "Query returned result\nProceeding to install...",
			expectedOutput:                  "Installing software...\nError: Scripts are disabled for this host. To run scripts, deploy the fleetd agent with --scripts-enabled.",
			expectedPostInstallScriptOutput: "",
		},
		{
			name: "non-pending status with non-empty PreInstallQueryOutput and failed post install script",
			hsr: HostSoftwareInstallerResult{
				Status:                    SoftwareInstallerInstalled,
				InstallScriptExitCode:     ptr.Int(0),
				PostInstallScriptExitCode: ptr.Int(1),
				PreInstallQueryOutput:     "1",
				PostInstallScriptOutput:   "output!",
			},
			expectedPreInstallQueryOutput: "Query returned result\nProceeding to install...",
			expectedOutput:                "Installing software...\nSuccess\n",
			expectedPostInstallScriptOutput: `Running script...
Exit code: 1 (Failed)
output!
Rolling back software install...
Rolled back successfully
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.hsr.EnhanceOutputDetails()
			require.Equal(t, tt.expectedPreInstallQueryOutput, tt.hsr.PreInstallQueryOutput)
			require.Equal(t, tt.expectedOutput, tt.hsr.Output)
			require.Equal(t, tt.expectedPostInstallScriptOutput, tt.hsr.PostInstallScriptOutput)
		})
	}
}
