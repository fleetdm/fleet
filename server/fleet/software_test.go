package fleet

import (
	"fmt"
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
		initial                         HostSoftwareInstallerResult
		expectedPreInstallQueryOutput   *string
		expectedOutput                  *string
		expectedPostInstallScriptOutput *string
	}{
		{
			name: "pending status",
			initial: HostSoftwareInstallerResult{
				Status: SoftwareInstallPending,
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with empty PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				PreInstallQueryOutput: ptr.String(""),
			},
			expectedPreInstallQueryOutput:   ptr.String(SoftwareInstallerQueryFailCopy),
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with non-empty PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				PreInstallQueryOutput: ptr.String("Some output"),
			},
			expectedPreInstallQueryOutput:   ptr.String(SoftwareInstallerQuerySuccessCopy),
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with nil PreInstallQueryOutput",
			initial: HostSoftwareInstallerResult{
				Status: SoftwareInstalled,
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  nil,
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with install scripts disabled",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				InstallScriptExitCode: ptr.Int(-2),
				Output:                ptr.String(""),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(SoftwareInstallerScriptsDisabledCopy),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with failed install script",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstallFailed,
				InstallScriptExitCode: ptr.Int(1),
				Output:                ptr.String("Some install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallFailCopy, "Some install output")),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with successful install script",
			initial: HostSoftwareInstallerResult{
				Status:                SoftwareInstalled,
				InstallScriptExitCode: ptr.Int(0),
				Output:                ptr.String("Some install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: nil,
		},
		{
			name: "non-pending status with successful post install script",
			initial: HostSoftwareInstallerResult{
				Status:                    SoftwareInstalled,
				InstallScriptExitCode:     ptr.Int(0),
				Output:                    ptr.String("Some install output"),
				PostInstallScriptExitCode: ptr.Int(0),
				PostInstallScriptOutput:   ptr.String("Some post install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallSuccessCopy, "Some post install output")),
		},
		{
			name: "non-pending status with failed post install script",
			initial: HostSoftwareInstallerResult{
				Status:                    SoftwareInstalled,
				InstallScriptExitCode:     ptr.Int(0),
				Output:                    ptr.String("Some install output"),
				PostInstallScriptExitCode: ptr.Int(1),
				PostInstallScriptOutput:   ptr.String("Some post install output"),
			},
			expectedPreInstallQueryOutput:   nil,
			expectedOutput:                  ptr.String(fmt.Sprintf(SoftwareInstallerInstallSuccessCopy, "Some install output")),
			expectedPostInstallScriptOutput: ptr.String(fmt.Sprintf(SoftwareInstallerPostInstallFailCopy, 1, "Some post install output")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.EnhanceOutputDetails()
			require.Equal(t, tt.expectedPreInstallQueryOutput, tt.initial.PreInstallQueryOutput)
			require.Equal(t, tt.expectedOutput, tt.initial.Output)
			require.Equal(t, tt.expectedPostInstallScriptOutput, tt.initial.PostInstallScriptOutput)
		})
	}
}


func TestMoveBrowserToExtFor(t *testing.T) {
	t.Run("moves when source=jetbrains_plugins and browser non-empty", func(t *testing.T) {
		src := "jetbrains_plugins"
		b := "goland"
		ext := "OLD"
		moveBrowserToExtFor(&src, &b, &ext)
		require.Equal(t, "jetbrains_plugins", src, "source unchanged")
		require.Equal(t, "", b, "browser cleared")
		require.Equal(t, "goland", ext, "extension_for set to previous browser")
	})

	t.Run("no-op when source is not jetbrains_plugins", func(t *testing.T) {
		src := "apps"
		b := "chrome"
		ext := "OLD"
		moveBrowserToExtFor(&src, &b, &ext)
		require.Equal(t, "apps", src)
		require.Equal(t, "chrome", b)
		require.Equal(t, "OLD", ext)
	})

	t.Run("no-op when browser empty", func(t *testing.T) {
		src := "jetbrains_plugins"
		b := ""
		ext := "OLD"
		moveBrowserToExtFor(&src, &b, &ext)
		require.Equal(t, "jetbrains_plugins", src)
		require.Equal(t, "", b)
		require.Equal(t, "OLD", ext)
	})

	t.Run("idempotent on second call", func(t *testing.T) {
		src := "jetbrains_plugins"
		b := "firefox"
		ext := ""
		moveBrowserToExtFor(&src, &b, &ext)

		require.Equal(t, "", b)
		require.Equal(t, "firefox", ext)

		// Call again: should be no-op now.
		moveBrowserToExtFor(&src, &b, &ext)
		require.Equal(t, "", b)
		require.Equal(t, "firefox", ext)
	})

	t.Run("overwrites existing extension_for", func(t *testing.T) {
		src := "jetbrains_plugins"
		b := "webstorm"
		ext := "previous"
		moveBrowserToExtFor(&src, &b, &ext)
		require.Equal(t, "", b)
		require.Equal(t, "webstorm", ext, "should overwrite ext_for with browser")
	})
}

func FuzzMoveBrowserToExtFor(f *testing.F) {
	f.Add("jetbrains_plugins", "x", "y")
	f.Add("apps", "x", "y")
	f.Add("jetbrains_plugins", "", "y")
	f.Fuzz(func(t *testing.T, srcIn, browserIn, extIn string) {
		src := srcIn
		b := browserIn
		ext := extIn
		moveBrowserToExtFor(&src, &b, &ext)

		if srcIn == jbPlugins && browserIn != "" {
			require.Equal(t, "", b)
			require.Equal(t, browserIn, ext)
			require.Equal(t, jbPlugins, src)
		} else {
			require.Equal(t, srcIn, src)
			require.Equal(t, browserIn, b)
			require.Equal(t, extIn, ext)
		}
	})
}