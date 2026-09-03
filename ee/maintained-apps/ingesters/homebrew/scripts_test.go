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
	// Input is raw data, never "already escaped": a backslash-dollar sequence has
	// each byte escaped independently, and the shell reads the result back as the
	// literal input bytes.
	require.Equal(t, `\\\$foo`, shellDoubleQuoteEscape(`\$foo`))
}

func TestEscapeCaskPath(t *testing.T) {
	// A leading $APPDIR is the legitimate expansion every real binary-artifact
	// source uses; it must survive unescaped, and inert paths must come back
	// byte-identical.
	require.Equal(t, `$APPDIR`, escapeCaskPath("$APPDIR"))
	require.Equal(t, `$APPDIR/Zed.app/Contents/MacOS/cli`, escapeCaskPath("$APPDIR/Zed.app/Contents/MacOS/cli"))
	require.Equal(t, `$APPDIR/Surge Dashboard.app`, escapeCaskPath("$APPDIR/Surge Dashboard.app"))
	require.Equal(t, `zed`, escapeCaskPath("zed"))
	// Everything after the prefix is data.
	require.Equal(t, `$APPDIR/Foo\$(id).app`, escapeCaskPath("$APPDIR/Foo$(id).app"))
	// A prefix that isn't exactly "$APPDIR/" (the shell would expand the longer
	// variable name $APPDIRx instead) is escaped whole.
	require.Equal(t, `\$APPDIRx/foo`, escapeCaskPath("$APPDIRx/foo"))
	require.Equal(t, `\$(id)`, escapeCaskPath("$(id)"))
}

func TestInertShellWord(t *testing.T) {
	for _, ok := range []string{".", "/usr/local/bin", "$APPDIR", "$APPDIR/Some", "a/b.c-d_e+f"} {
		require.True(t, inertShellWord(ok), "input: %q", ok)
	}
	for _, bad := range []string{"", "-v", "--", "a b", "$(id)", "`id`", "$HOME", "$APPDIR/$(id)", "$APPDIR/a b", "~", `a\b`, `a"b`, "a'b", "a;b", "a\nb"} {
		require.False(t, inertShellWord(bad), "input: %q", bad)
	}
}

func TestHeredocInert(t *testing.T) {
	require.True(t, heredocInert("<?xml version=\"1.0\"?>\n<string>com.example</string>"))
	require.False(t, heredocInert("$(id)"))
	require.False(t, heredocInert("`id`"))
	require.False(t, heredocInert(`a\b`))
	// A body line equal to the delimiter would terminate the heredoc early.
	require.False(t, heredocInert("line1\nEOF\nline2"))
	require.True(t, heredocInert("not EOF alone on a line"))
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
// choice metadata. XML that isn't heredoc-inert is written with a single-quoted
// printf argument, not a heredoc, so neither $(...) in the body nor a
// newline-injected delimiter can execute.
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

// TestInstallScriptChoiceXMLKeepsHeredocWhenInert pins the historical heredoc
// form for metacharacter-free choice XML (every real cask today): the
// auto-update job treats any byte change to a generated script as an admin
// customization and pins the stored script, so the hardened printf form must
// only appear when the XML actually needs it.
func TestInstallScriptChoiceXMLKeepsHeredocWhenInert(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Pkg: []optjson.StringOr[*brewPkgChoices]{
				{String: "Foo-1.0.pkg"},
				{IsOther: true, Other: &brewPkgChoices{Choices: []brewPkgConfig{
					{ChoiceIdentifier: "com.microsoft.autoupdate", ChoiceAttribute: "selected", AttributeSetting: 0},
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
	require.Contains(t, script, "cat << EOF > \"$CHOICE_XML\"\n<?xml")
	// The marshaled plist ends with its own newline; the stored outputs have the
	// same blank line before EOF, so this pins the historical bytes exactly.
	require.Contains(t, script, "</array>\n</plist>\n\nEOF\n")
	require.NotContains(t, script, `printf '%s\n'`)
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
	// The parent dir is inert, so mkdir keeps its historical unquoted form.
	require.Contains(t, script, "\nmkdir -p /usr/local/bin\n")
	require.Contains(t, script, `/bin/ln -h -f -s -- "bin/foo" "/usr/local/bin/foo\$(id)"`)
	require.NotContains(t, script, `"/usr/local/bin/foo$(id)"`)
}

// TestInstallScriptBinaryArtifactByteIdenticalForRealCasks pins the exact
// historical output for the shapes every current binary-artifact cask uses
// (zed/cursor-style bare target, surge-style $APPDIR target): any byte change
// here reads as an admin customization to the auto-update job, which then pins
// the stored script with its stale installer filename.
func TestInstallScriptBinaryArtifactByteIdenticalForRealCasks(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Binary: []optjson.StringOr[*brewBinaryTarget]{
				{String: "$APPDIR/Zed.app/Contents/MacOS/cli"},
				{IsOther: true, Other: &brewBinaryTarget{Target: "zed"}},
			}},
			{Binary: []optjson.StringOr[*brewBinaryTarget]{
				{String: "$APPDIR/Surge.app/Contents/Applications/Surge Dashboard.app"},
				{IsOther: true, Other: &brewBinaryTarget{Target: "$APPDIR/Surge Dashboard.app"}},
			}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "zip",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, "\nmkdir -p .\n")
	require.Contains(t, script, "\n"+`/bin/ln -h -f -s -- "$APPDIR/Zed.app/Contents/MacOS/cli" "zed"`+"\n")
	require.Contains(t, script, "\nmkdir -p $APPDIR\n")
	require.Contains(t, script, `/bin/ln -h -f -s -- "$APPDIR/Surge.app/Contents/Applications/Surge Dashboard.app" "$APPDIR/Surge Dashboard.app"`)
}

// TestInstallScriptMkdirNotFooledByLeadingDash: a hostile target whose parent
// directory starts with a dash must be treated as a pathname, not an option, so
// the hardened branch adds `--`.
func TestInstallScriptMkdirNotFooledByLeadingDash(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{Binary: []optjson.StringOr[*brewBinaryTarget]{
				{String: "bin/foo"},
				{IsOther: true, Other: &brewBinaryTarget{Target: "-v/foo"}},
			}},
		},
	}

	script, err := installScriptForApp(inputApp{
		Token:            "foo",
		UniqueIdentifier: "com.example.Foo",
		InstallerFormat:  "zip",
	}, cask)
	require.NoError(t, err)
	require.Contains(t, script, `mkdir -p -- "-v"`)
	require.Contains(t, script, `/bin/ln -h -f -s -- "bin/foo" "-v/foo"`)
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

// TestUninstallScriptEarlyScriptDirective covers the `early_script` uninstall
// directive: it takes the same shape as `script`, and casks use it for the
// commands that make the rest of the removal possible (clearing an immutable
// flag, unloading a system extension), so it has to run before them.
func TestUninstallScriptEarlyScriptDirective(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						EarlyScript: optjson.StringOr[map[string]any]{
							IsOther: true,
							Other: map[string]any{
								"executable":   "/usr/bin/chflags",
								"args":         []any{"-RL", "noschg", "/Applications/Foo.app"},
								"must_succeed": false,
							},
						},
					},
					{
						PkgUtil: optjson.StringOr[[]string]{String: "com.example.foo"},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	chflags := `(cd /Users/$LOGGED_IN_USER && '/usr/bin/chflags' '-RL' 'noschg' '/Applications/Foo.app') || true`
	require.Contains(t, script, chflags)
	require.Less(t, strings.Index(script, chflags), strings.Index(script, `remove_pkg_files 'com.example.foo'`))
}

// TestUninstallScriptEarlyScriptHonorsSudo checks the shared script-directive
// options still apply to early_script.
func TestUninstallScriptEarlyScriptHonorsSudo(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						EarlyScript: optjson.StringOr[map[string]any]{
							IsOther: true,
							Other: map[string]any{
								"executable": "/Applications/Foo.app/Contents/MacOS/Foo",
								"args":       []any{"--unload-system-extension"},
								"sudo":       true,
							},
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	require.Contains(t, script, `(cd /Users/$LOGGED_IN_USER && sudo '/Applications/Foo.app/Contents/MacOS/Foo' '--unload-system-extension')`)
	require.NotContains(t, script, "|| true")
}

// TestUninstallScriptSkipsHomebrewPrefixScript drops script directives pointing
// into the Caskroom: that path only exists where Homebrew installed the app, so
// the command can't do anything on a Fleet-managed host.
func TestUninstallScriptSkipsHomebrewPrefixScript(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						EarlyScript: optjson.StringOr[map[string]any]{
							IsOther: true,
							Other: map[string]any{
								"executable":   "/usr/sbin/installer",
								"args":         []any{"-pkg", "$HOMEBREW_PREFIX/Caskroom/foo/1.0/Remove Foo.pkg", "-target", "/"},
								"sudo":         true,
								"must_succeed": false,
							},
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(cask)
	require.NotContains(t, script, "$HOMEBREW_PREFIX")
	require.NotContains(t, script, "/usr/sbin/installer")
}
