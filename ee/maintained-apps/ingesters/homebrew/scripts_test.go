package homebrew

import (
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

// TestInstallScriptPkgPropagatesInstallerExitCode guards against a regression
// where a failing `installer -pkg` was reported as a successful install because
// its exit code was discarded and the script ended on `relaunch_application`,
// which always exits 0.
func TestInstallScriptPkgPropagatesInstallerExitCode(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Pkg: []optjson.StringOr[*brewPkgChoices]{{String: "Foo-1.0.pkg"}}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `sudo installer -pkg "$TMPDIR/Foo-1.0.pkg" -target / || exit $?`)
	// relaunch still runs, but only after the install command's exit code is checked.
	require.Contains(t, script, "relaunch_application 'com.example.Foo'")
}

// TestInstallScriptPkgWithChoicesPropagatesExitCode is the choices variant of the
// above (e.g. Microsoft apps), which installs via -applyChoiceChangesXML.
func TestInstallScriptPkgWithChoicesPropagatesExitCode(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Pkg: []optjson.StringOr[*brewPkgChoices]{
				{String: "Foo-1.0.pkg"},
				{IsOther: true, Other: &brewPkgChoices{Choices: []brewPkgConfig{}}},
			}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `-applyChoiceChangesXML "$CHOICE_XML" || exit $?`)
	// The pkg filename must stay inside the quotes so filenames with spaces don't word-split.
	require.Contains(t, script, `sudo installer -pkg "$TMPDIR/Foo-1.0.pkg" -target /`)
}

// TestInstallScriptAppCopyPropagatesAndRestores is the cp -R equivalent: a
// failing copy must exit non-zero and restore the app it moved aside, so a
// failed install neither reports success nor leaves the host without a working
// app.
func TestInstallScriptAppCopyPropagatesAndRestores(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{App: []optjson.StringOr[*brewAppTarget]{{String: "Foo.app"}}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "dmg",
	}, cask)
	require.NoError(t, err)
	// A failed move-aside must not fall through to a copy that nests into the old app.
	require.Contains(t, script, `sudo mv "$APPDIR/Foo.app" "$TMPDIR/Foo.app.bkp" || exit $?`)
	require.Contains(t, script, `if ! sudo cp -R "$TMPDIR/Foo.app" "$APPDIR"; then`)
	require.Contains(t, script, `sudo mv "$TMPDIR/Foo.app.bkp" "$APPDIR/Foo.app"`)
	require.Contains(t, script, "exit 1")
}

func TestShellSingleQuote(t *testing.T) {
	for in, want := range map[string]string{
		"":                      `''`,
		"plain":                 `'plain'`,
		"~/Library/Caches/X":    `'~/Library/Caches/X'`,
		"Cycling '74":           `'Cycling '\''74'`,
		"a'b'c":                 `'a'\''b'\''c'`,
		"~/Documents/Max [0-9]": `'~/Documents/Max [0-9]'`,
	} {
		require.Equal(t, want, shellSingleQuote(in), "input: %q", in)
	}
}

// TestUninstallScriptEscapesApostrophe guards against a regression where a
// cask zap/trash path containing a single quote (e.g. Ableton Live Suite's
// bundled "Cycling '74" directory) produced an unterminated single-quoted
// string and a bash syntax error.
func TestUninstallScriptEscapesApostrophe(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Zap: []*brewUninstall{
					{
						Trash: optjson.StringOr[[]string]{
							IsOther: true,
							Other:   []string{"~/Library/Application Support/Cycling '74"},
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	require.Contains(t, script, `trash $LOGGED_IN_USER '~/Library/Application Support/Cycling '\''74'`)
	require.NotContains(t, script, `Cycling '74'`)
}

// TestUninstallScriptExpandsLaunchctlWildcard guards against a regression where
// a cask zap/uninstall launchctl label containing a wildcard (e.g.
// "com.elgato.StreamDeck*") was passed straight to `launchctl list` and used as
// a plist filename, neither of which supports wildcards, so the matching
// launchd job and plist were never removed. The generated helper must expand
// the wildcard against the loaded services before removing.
func TestUninstallScriptExpandsLaunchctlWildcard(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Zap: []*brewUninstall{
					{
						LaunchCtl: optjson.StringOr[[]string]{
							String: "com.elgato.StreamDeck*",
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	require.Contains(t, script, `remove_launchctl_service 'com.elgato.StreamDeck*'`)
	// The helper must expand a wildcard label before touching launchctl.
	require.Contains(t, script, `if [[ "$service" == *"*"* ]]; then`)
	require.Contains(t, script, `regex=$(printf '%s' "$service" | sed -e 's/[][(){}.^$+?|\\]/\\&/g' -e 's/\*/.*/g')`)
	require.Contains(t, script, `[[ "$id" =~ $regex ]] && services+=("$id")`)
	// The regex must be anchored so a wildcard label matches the full label and
	// not a substring (e.g. "ai.krisp.krispMac*" must not match
	// "x.ai.krisp.krispMac.helper").
	require.Contains(t, script, `regex="^${regex}$"`)
	// launchctl list reports loaded-but-not-running jobs with a "-" (or 0) PID,
	// so the helper must match on the label regardless of PID. Guard against a
	// regression that filters those jobs out.
	require.Contains(t, script, `while read -r _ _ id; do`)
	require.NotContains(t, script, `[[ "$pid" =~ ^[0-9]+$ ]]`)
	require.NotContains(t, script, `(( pid != 0 ))`)
}
