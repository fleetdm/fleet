package homebrew

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestInstallScriptForPkgWithoutPkgArtifact(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				App: []optjson.StringOr[*brewAppTarget]{
					{String: "Slack.app"},
				},
			},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "slack",
		UniqueIdentifier: "com.tinyspeck.slackmacgap",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `quit_and_track_application 'com.tinyspeck.slackmacgap'`)
	require.Contains(t, script, `sudo installer -pkg "$INSTALLER_PATH" -target /`)
	require.Contains(t, script, `relaunch_application 'com.tinyspeck.slackmacgap'`)
	require.NotContains(t, script, "unzip")
	require.NotContains(t, script, "cp -R")
}

func TestInstallScriptForPkgWithPkgArtifact(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Pkg: []optjson.StringOr[*brewPkgChoices]{
					{String: "ZoomInstallerIT.pkg"},
				},
			},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "zoom-for-it-admins", //nolint:gosec // homebrew cask token, not a credential
		UniqueIdentifier: "us.zoom.xos",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `sudo installer -pkg "$TMPDIR/ZoomInstallerIT.pkg" -target /`)
	require.NotContains(t, script, `sudo installer -pkg "$INSTALLER_PATH" -target /`)
	require.Contains(t, script, "relaunch_application")
}
