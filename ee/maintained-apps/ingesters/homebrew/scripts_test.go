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
	// A failed move-aside must not fall through to a copy that merges stale files into the old app.
	require.Contains(t, script, `sudo mv "$APPDIR/Foo.app" "$TMPDIR/Foo.app.bkp" || exit $?`)
	// On copy failure the partial copy is removed (even on a fresh install with
	// no backup), the previous version is restored, and the script exits non-zero.
	require.Contains(t, script, `if ! sudo cp -R "$TMPDIR/Foo.app" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/Foo.app"
	if [ -d "$TMPDIR/Foo.app.bkp" ]; then
		sudo mv "$TMPDIR/Foo.app.bkp" "$APPDIR/Foo.app"
	fi
	exit 1
fi`)
}

func TestShellDoubleQuoteEscape(t *testing.T) {
	// Values without shell metacharacters (the common case) must pass through
	// unchanged so the generated scripts stay byte-identical to before.
	require.Empty(t, shellDoubleQuoteEscape(""))
	require.Equal(t, `plain`, shellDoubleQuoteEscape("plain"))
	require.Equal(t, `Foo-1.0.pkg`, shellDoubleQuoteEscape("Foo-1.0.pkg"))
	require.Equal(t, `Adobe Photoshop (2024).app`, shellDoubleQuoteEscape("Adobe Photoshop (2024).app"))
	// Apostrophes are literal inside double quotes and must NOT be escaped.
	require.Equal(t, `Cycling '74`, shellDoubleQuoteEscape("Cycling '74"))
	// The four characters special inside double quotes are backslash-escaped.
	require.Equal(t, `\$(id)`, shellDoubleQuoteEscape("$(id)"))
	require.Equal(t, `\$HOME`, shellDoubleQuoteEscape("$HOME"))
	require.Equal(t, `a\"b`, shellDoubleQuoteEscape(`a"b`))
	require.Equal(t, `a\\b`, shellDoubleQuoteEscape(`a\b`))
	require.Equal(t, "a\\`id\\`b", shellDoubleQuoteEscape("a`id`b"))
	// Backslash is escaped first, so an already-escaped dollar isn't doubled.
	require.Equal(t, `\$`, shellDoubleQuoteEscape("$"))
}

// TestInstallScriptNeutralizesCommandSubstitutionInAppPath guards against the
// homebrew-ingester instance of the "cask metadata -> generated script -> root
// RCE" class (GHSA-fv3p-h7wv-2jgm): a cask-controlled app path reaching the
// double-quoted mv/cp/rm sinks must appear as literal text, not command
// substitution.
func TestInstallScriptNeutralizesCommandSubstitutionInAppPath(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{App: []optjson.StringOr[*brewAppTarget]{{String: `Foo$(touch /tmp/pwned).app`}}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "dmg",
	}, cask)
	require.NoError(t, err)
	// The payload survives verbatim but backslash-escaped, so the shell treats it
	// as a filename rather than running `touch`.
	require.Contains(t, script, `sudo mv "$APPDIR/Foo\$(touch /tmp/pwned).app" "$TMPDIR/Foo\$(touch /tmp/pwned).app.bkp" || exit $?`)
	require.Contains(t, script, `sudo rm -rf "$APPDIR/Foo\$(touch /tmp/pwned).app"`)
	// The unescaped (executable) form must not appear anywhere.
	require.NotContains(t, script, `"$APPDIR/Foo$(touch /tmp/pwned).app"`)
}

// TestInstallScriptNeutralizesCommandSubstitutionInPkg is the pkg-filename
// variant of the above.
func TestInstallScriptNeutralizesCommandSubstitutionInPkg(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Pkg: []optjson.StringOr[*brewPkgChoices]{{String: `Foo$(id).pkg`}}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `sudo installer -pkg "$TMPDIR/Foo\$(id).pkg" -target / || exit $?`)
	require.NotContains(t, script, `"$TMPDIR/Foo$(id).pkg"`)
}

// TestInstallScriptChoiceXMLIsNotShellExpanded guards against command
// substitution and heredoc-terminator smuggling through cask-controlled pkg
// choice metadata. The XML is written with a single-quoted printf argument, not
// a heredoc, so neither $(...) in the body nor a newline-injected delimiter can
// execute.
func TestInstallScriptChoiceXMLIsNotShellExpanded(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Pkg: []optjson.StringOr[*brewPkgChoices]{
				{String: "Foo-1.0.pkg"},
				{IsOther: true, Other: &brewPkgChoices{Choices: []brewPkgConfig{
					{ChoiceIdentifier: "$(touch /tmp/pwned)", ChoiceAttribute: "selected", AttributeSetting: 1},
				}}},
			}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "pkg",
	}, cask)
	require.NoError(t, err)
	// Written via printf with a single-quoted argument, never a heredoc whose body
	// or delimiter the shell could interpret.
	require.Contains(t, script, `printf '%s\n' '`)
	require.NotContains(t, script, "cat << EOF")
	require.NotContains(t, script, "\nEOF\n")
	// The payload lands in the choices file as data, single-quoted.
	require.Contains(t, script, "$(touch /tmp/pwned)")
	// Install still runs after the file is written.
	require.Contains(t, script, `-applyChoiceChangesXML "$CHOICE_XML" || exit $?`)
}

// TestInstallScriptNeutralizesCommandSubstitutionInBinaryTarget covers the
// symlink (binary artifact) sinks: both the mkdir and ln arguments are
// cask-controlled.
func TestInstallScriptNeutralizesCommandSubstitutionInBinaryTarget(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Binary: []optjson.StringOr[*brewBinaryTarget]{
				{String: "bin/foo"},
				{IsOther: true, Other: &brewBinaryTarget{Target: "/usr/local/bin/foo$(id)"}},
			}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "zip",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `mkdir -p "/usr/local/bin"`)
	require.Contains(t, script, `/bin/ln -h -f -s -- "bin/foo" "/usr/local/bin/foo\$(id)"`)
	require.NotContains(t, script, `"/usr/local/bin/foo$(id)"`)
}

// TestUninstallScriptNeutralizesCommandSubstitutionInAppPath is the uninstall
// counterpart: the cask-controlled app path reaches a double-quoted `sudo rm
// -rf`.
func TestUninstallScriptNeutralizesCommandSubstitutionInAppPath(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{App: []optjson.StringOr[*brewAppTarget]{{String: `Foo$(touch /tmp/pwned).app`}}},
		},
	}

	script := uninstallScriptForApp(cask)
	require.Contains(t, script, `sudo rm -rf "$APPDIR/Foo\$(touch /tmp/pwned).app"`)
	require.NotContains(t, script, `"$APPDIR/Foo$(touch /tmp/pwned).app"`)
}

// TestUninstallScriptScriptDirectiveEscapesApostrophe guards the uninstall
// `script` directive against an apostrophe breaking out of the single-quoted
// command.
func TestUninstallScriptScriptDirectiveEscapesApostrophe(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						Script: optjson.StringOr[map[string]any]{
							String: `/opt/x'; touch /tmp/pwned; '`,
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	require.Contains(t, script, `'\''`)
	require.NotContains(t, script, `/opt/x'; touch /tmp/pwned; '`)
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
