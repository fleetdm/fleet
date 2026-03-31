package homebrew

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuitAndTrackApplicationFunc_ContainsWindowCheck(t *testing.T) {
	// The quit_and_track_application function should verify the app has visible
	// windows via System Events before marking it for relaunch. This prevents
	// false-positive "app was running" detection when only background helpers
	// (e.g., VS Code's Code Helper processes) are active.
	assert.Contains(t, quitAndTrackApplicationFunc, "System Events",
		"quit_and_track_application should use System Events to verify visible windows")
	assert.Contains(t, quitAndTrackApplicationFunc, "count of windows",
		"quit_and_track_application should check window count via System Events")
	assert.Contains(t, quitAndTrackApplicationFunc, "no visible windows",
		"quit_and_track_application should log when skipping due to no visible windows")
}

func TestQuitAndTrackApplicationFunc_DoesNotUsePgrepF(t *testing.T) {
	// pgrep -f matches the full command line, which can false-match helper
	// processes (e.g., Code Helper, Code Helper (Renderer)) whose paths
	// contain the parent app's bundle ID. The quit loop should use a more
	// specific check.
	assert.NotContains(t, quitAndTrackApplicationFunc, `pgrep -f`,
		"quit_and_track_application should not use pgrep -f (matches helper processes)")
}

func TestQuitApplicationFunc_UnchangedForUninstall(t *testing.T) {
	// The uninstall quit_application function should still use pgrep -f
	// because during uninstall we want to wait for ALL processes (including
	// helpers) to terminate.
	assert.Contains(t, quitApplicationFunc, `pgrep -f`,
		"quit_application (uninstall) should still use pgrep -f to catch all processes")
}

func TestInstallScriptForApp_AppArtifact_HasQuitAndRelaunch(t *testing.T) {
	app := inputApp{
		Token:            "test-app",
		UniqueIdentifier: "com.test.App",
		InstallerFormat:  "zip",
	}

	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				App: []optjson.StringOr[*brewAppTarget]{
					{String: "Test App.app"},
				},
			},
		},
	}

	script, err := installScriptForApp(app, cask)
	require.NoError(t, err)

	assert.Contains(t, script, "quit_and_track_application 'com.test.App'",
		"install script should call quit_and_track_application with bundle ID")
	assert.Contains(t, script, "relaunch_application 'com.test.App'",
		"install script should call relaunch_application with bundle ID")
	assert.Contains(t, script, "System Events",
		"install script should contain System Events window check")
}

func TestInstallScriptForApp_BinaryOnly_NoQuitRelaunch(t *testing.T) {
	app := inputApp{
		Token:            "test-binary",
		UniqueIdentifier: "com.test.Binary",
		InstallerFormat:  "zip",
	}

	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Binary: []optjson.StringOr[*brewBinaryTarget]{
					{String: "test-binary"},
					{Other: &brewBinaryTarget{Target: "/usr/local/bin/test-binary"}},
				},
			},
		},
	}

	script, err := installScriptForApp(app, cask)
	require.NoError(t, err)

	assert.NotContains(t, script, "quit_and_track_application",
		"binary-only install should not include quit_and_track_application")
	assert.NotContains(t, script, "relaunch_application",
		"binary-only install should not include relaunch_application")
}

func TestUninstallScriptForApp_UsesQuitApplication(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				App: []optjson.StringOr[*brewAppTarget]{
					{String: "Test App.app"},
				},
			},
			{
				Uninstall: []*brewUninstall{
					{
						Quit: optjson.StringOr[[]string]{Other: []string{"com.test.App"}, IsOther: true},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)

	// Uninstall should use the simpler quit_application (not quit_and_track)
	assert.Contains(t, script, "quit_application",
		"uninstall script should use quit_application")
	_ = strings.Contains(script, "quit_and_track_application") // reference to suppress lint
	assert.NotContains(t, script, "quit_and_track_application",
		"uninstall script should NOT use quit_and_track_application")
}
