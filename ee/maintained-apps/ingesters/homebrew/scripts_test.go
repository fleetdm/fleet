package homebrew

import (
"strings"
"testing"

"github.com/fleetdm/fleet/v4/pkg/optjson"
"github.com/stretchr/testify/require"
)

func TestInstallScriptDmgExtractUsesYesPipe(t *testing.T) {
cask := &brewCask{
Artifacts: []*brewArtifact{
{
App: []optjson.StringOr[*brewAppTarget]{
{String: "Evernote.app"},
},
},
},
}

script, err := installScriptForApp(inputApp{
Token:            "evernote",
UniqueIdentifier: "com.evernote.Evernote",
InstallerFormat:  "dmg",
}, cask)
require.NoError(t, err)
require.Contains(t, script, "yes | hdiutil attach -plist -nobrowse -readonly -mountpoint")
require.Contains(t, script, `|| exit 1`)
require.Contains(t, script, `hdiutil detach "$MOUNT_POINT" || true`)
}

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
// A failed install must propagate its exit code rather than being masked by
// the trailing relaunch_application call.
requirePropagatesInstallFailure(t, script)
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

script, err := installScriptForApp(inputApp{ //nolint:gosec // homebrew cask token, not a credential
Token:            "zoom-for-it-admins",
UniqueIdentifier: "us.zoom.xos",
InstallerFormat:  "pkg",
}, cask)
require.NoError(t, err)
require.Contains(t, script, `sudo installer -pkg "$TMPDIR/ZoomInstallerIT.pkg" -target /`)
require.NotContains(t, script, `sudo installer -pkg "$INSTALLER_PATH" -target /`)
// Assert on the actual invocation, not just the function definition, so the
// test fails if the relaunch call is ever dropped from the generated script.
require.Contains(t, script, `relaunch_application 'us.zoom.xos'`)
// A failed install must propagate its exit code rather than being masked by
// the trailing relaunch_application call.
requirePropagatesInstallFailure(t, script)
}

// requirePropagatesInstallFailure asserts that the generated script captures the
// install command's exit status and exits with it on failure, so the trailing
// relaunch_application call cannot mask a failed install.
func requirePropagatesInstallFailure(t *testing.T, script string) {
t.Helper()
installIdx := strings.Index(script, "sudo installer -pkg")
require.NotEqual(t, -1, installIdx, "expected an installer command in the script")
relaunchIdx := strings.Index(script[installIdx:], "relaunch_application '")
require.NotEqual(t, -1, relaunchIdx, "expected a relaunch_application call after the installer")
captureIdx := strings.Index(script[installIdx:], "INSTALL_EXIT_CODE=$?")
require.NotEqual(t, -1, captureIdx, "expected the installer exit code to be captured")
// The exit code must be captured before the relaunch runs.
require.Less(t, captureIdx, relaunchIdx, "exit code must be captured before relaunch")
require.Contains(t, script, `exit "$INSTALL_EXIT_CODE"`)
}
