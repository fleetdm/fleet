package homebrew

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
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

// TestInstallScriptStopsWhenAppWontQuit runs generated install scripts in bash
// with osascript and friends stubbed out. An app that's still running after the
// quit timeout must not be replaced: fleetd deletes $TMPDIR after the install,
// taking the moved-aside bundle the app is still running from.
func TestInstallScriptStopsWhenAppWontQuit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	if os.Geteuid() == 0 {
		t.Skip("as root the relaunch goes through /bin/launchctl asuser, which can't be stubbed")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	installers := map[string]struct {
		format   string
		artifact *brewArtifact
		replace  string
	}{
		"app": {"zip", &brewArtifact{App: []optjson.StringOr[*brewAppTarget]{{String: "Foo.app"}}}, `sudo cp -R`},
		"pkg": {"pkg", &brewArtifact{Pkg: []optjson.StringOr[*brewPkgChoices]{{String: "Foo-1.0.pkg"}}}, `sudo installer -pkg`},
	}

	// mode is how the app responds: "not-running", "quits", "ignores" (accepts
	// the quit event but keeps running), or "cancels" (the user cancels a save
	// prompt, so the quit event fails). sleep advances SECONDS so the timeout
	// loop doesn't take real time.
	run := func(t *testing.T, script, mode string) (string, error) {
		stubs := `mode=` + mode + `
running=true
[[ "$mode" == not-running ]] && running=false
stat() { echo alice; }
realpath() { echo /tmp/fleet-test/installer; }
unzip() { :; }
sleep() { SECONDS=$((SECONDS + $1)); }
sudo() { echo "sudo $*"; }
open() { :; }
osascript() {
  case "$2" in
    *" to quit")
      [[ "$mode" == quits ]] && running=false
      [[ "$mode" == cancels ]] && return 1
      return 0
      ;;
    *" is running")
      echo "$running"
      ;;
  esac
}
`
		out, err := exec.Command(bash, "-c", stubs+script).CombinedOutput()
		return string(out), err
	}

	for name, installer := range installers {
		t.Run(name, func(t *testing.T) {
			script, err := installScriptForApp(inputApp{
				Token:            "foo",
				UniqueIdentifier: "com.example.Foo",
				InstallerFormat:  installer.format,
			}, &brewCask{Artifacts: []*brewArtifact{installer.artifact}})
			require.NoError(t, err)

			out, err := run(t, script, "not-running")
			require.NoError(t, err, out)
			require.Contains(t, out, installer.replace)
			require.NotContains(t, out, "Quitting application")
			require.NotContains(t, out, "Relaunching application")

			out, err = run(t, script, "quits")
			require.NoError(t, err, out)
			require.Contains(t, out, "Application 'com.example.Foo' quit successfully.")
			require.Contains(t, out, installer.replace)
			require.Contains(t, out, "Relaunching application 'com.example.Foo'...")

			for _, mode := range []string{"ignores", "cancels"} {
				out, err = run(t, script, mode)
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, out)
				require.Equal(t, 1, exitErr.ExitCode(), out)
				require.Contains(t, out, "Application 'com.example.Foo' did not quit.")
				require.NotContains(t, out, installer.replace)
				require.NotContains(t, out, "Relaunching application")
			}
		})
	}
}

// TestCustomInstallScriptsStopWhenAppWontQuit checks the quit helper copies in
// custom install scripts. A copy that confirms the quit with pgrep reports
// success while the app keeps running, since pgrep -f never matches the bundle
// id. A call that carries on after a failed quit must be listed here with the
// reason it's safe.
func TestCustomInstallScriptsStopWhenAppWontQuit(t *testing.T) {
	continuesAnyway := map[string]string{
		"docker_desktop_install.sh": "runs under set -e",
		"expressvpn-install.sh":     "vpn-installer.sh kills the app's processes before replacing it",
		"harmony-sase-install.sh":   "the pkg force-quits the app",
		"logitune-install.sh":       "the pkg's preinstall force-kills the app",
		"qgis-ltr-install.sh":       "quit_app_at_path force-kills the app if it won't close",
	}
	quitCall := regexp.MustCompile(`^\s*quit_\w+(\s|$)`)
	// kicad-install.sh's abort_install relaunches the apps and exits 1.
	stopsOnFailure := regexp.MustCompile(`\|\|\s*(exit(\s+[1-9][0-9]*)?|abort_install)\s*$`)

	paths, err := filepath.Glob("../../inputs/homebrew/scripts/*.sh")
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		name := filepath.Base(path)
		if strings.Contains(name, "uninstall") {
			continue
		}
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		for line := range strings.SplitSeq(string(b), "\n") {
			require.NotContains(t, line, `pgrep -f "$bundle_id"`, "%s confirms the quit with pgrep", name)
			if !quitCall.MatchString(line) || stopsOnFailure.MatchString(line) {
				continue
			}
			_, ok := continuesAnyway[name]
			require.True(t, ok, "%s: %q carries on when the app won't quit", name, strings.TrimSpace(line))
		}
	}
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

	script := uninstallScriptForApp(inputApp{}, cask)
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

	script := uninstallScriptForApp(inputApp{}, cask)
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

	script := uninstallScriptForApp(inputApp{}, cask)
	require.Contains(t, script, `trash "$LOGGED_IN_USER" '~/Library/Application Support/Cycling '\''74'`)
	require.NotContains(t, script, `Cycling '74'`)
}

// TestUninstallScriptTrashWithoutGUIUser runs generated trash calls in bash with
// the console user stubbed. Fleet also runs uninstalls at the login window,
// where there's no user Trash: moving into /Users/loginwindow/.Trash left system
// paths behind, and an empty user shifted the arguments so nothing was removed.
func TestUninstallScriptTrashWithoutGUIUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	type paths struct {
		dir, helper1, helper2 string
	}
	run := func(t *testing.T, consoleUser string, rmFails bool) (string, paths) {
		tmp := t.TempDir()
		p := paths{
			dir:     filepath.Join(tmp, "Shared", "Foo"),
			helper1: filepath.Join(tmp, "Library", "Foo Helper 1"),
			helper2: filepath.Join(tmp, "Library", "Foo Helper 2"),
		}
		require.NoError(t, os.MkdirAll(filepath.Join(p.dir, "data"), 0o755))
		require.NoError(t, os.MkdirAll(filepath.Dir(p.helper1), 0o755))
		require.NoError(t, os.WriteFile(p.helper1, nil, 0o644))
		require.NoError(t, os.WriteFile(p.helper2, nil, 0o644))

		cask := &brewCask{
			Artifacts: []*brewArtifact{
				{
					Zap: []*brewUninstall{
						{
							Trash: optjson.StringOr[[]string]{
								IsOther: true,
								Other: []string{
									"~/Library/Preferences/com.example.foo.plist",
									"~/Library/Caches/com.example.foo*",
									filepath.Join(tmp, "Library", "Foo Helper*"),
									p.dir,
								},
							},
						},
					},
				},
			},
		}
		// mv is stubbed so the logged-in case doesn't touch /Users; rm is real
		// unless it's made to fail.
		stubs := `console_user=` + shellSingleQuote(consoleUser) + `
rm_fails=` + strconv.FormatBool(rmFails) + `
scutil() { [[ -z "$console_user" ]] || echo "  Name : $console_user"; }
jot() { echo 42; }
mv() { echo "mv $*"; }
rm() {
  if [[ "$rm_fails" == true ]]; then
    echo "rm: $2: Operation not permitted" >&2
    return 1
  fi
  command rm "$@"
}
`
		out, err := exec.Command(bash, "-c", stubs+uninstallScriptForApp(inputApp{}, cask)).CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out), p
	}

	for name, user := range map[string]string{"loginwindow": "loginwindow", "empty": "", "root": "root"} {
		t.Run(name, func(t *testing.T) {
			out, p := run(t, user, false)
			require.Contains(t, out, "Not logged into a non-root GUI; skipping ~/Library/Preferences/com.example.foo.plist.\n")
			require.Contains(t, out, "Not logged into a non-root GUI; skipping ~/Library/Caches/com.example.foo*.\n")
			require.Contains(t, out, "removing "+p.dir+".\n")
			require.NoDirExists(t, p.dir)
			require.NoFileExists(t, p.helper1)
			require.NoFileExists(t, p.helper2)
			require.NotContains(t, out, "mv ")
		})
	}

	t.Run("normal user", func(t *testing.T) {
		out, p := run(t, "alice", false)
		require.Contains(t, out, "/Users/alice/Library/Preferences/com.example.foo.plist doesn't exist.\n")
		require.Contains(t, out, "/Users/alice/Library/Caches/com.example.foo* doesn't exist.\n")
		require.Contains(t, out, "mv -f "+p.dir+" /Users/alice/.Trash/Foo_")
		require.Contains(t, out, "mv -f "+p.helper1+" /Users/alice/.Trash/Foo Helper 1_")
		require.Contains(t, out, "mv -f "+p.helper2+" /Users/alice/.Trash/Foo Helper 2_")
		require.NotContains(t, out, "Not logged into")
		require.DirExists(t, p.dir)
	})

	// Cleanup is best-effort: a path that can't be removed is reported but
	// doesn't fail the uninstall, even when it's the script's last command.
	t.Run("removal fails", func(t *testing.T) {
		out, p := run(t, "loginwindow", true)
		require.Contains(t, out, "Failed to remove "+p.helper1+".\n")
		require.True(t, strings.HasSuffix(out, "Failed to remove "+p.dir+".\n"), out)
		require.DirExists(t, p.dir)
	})
}

// Custom uninstall scripts carry their own copy of trash().
func TestCustomUninstallScriptsUseCurrentTrash(t *testing.T) {
	paths, err := filepath.Glob("../../inputs/homebrew/scripts/*.sh")
	require.NoError(t, err)
	require.NotEmpty(t, paths)

	for _, path := range paths {
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		script := string(b)
		if !strings.Contains(script, "\ntrash() {") {
			continue
		}
		require.Contains(t, script, trashFunc, "%s has a stale copy of trash()", filepath.Base(path))
		require.NotContains(t, script, "trash $LOGGED_IN_USER", "%s passes $LOGGED_IN_USER to trash unquoted", filepath.Base(path))
	}
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

	script := uninstallScriptForApp(inputApp{}, cask)
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

// TestUninstallScriptExpandsQuitWildcard guards against a regression where a
// cask quit ID containing a wildcard (e.g. "com.elgato.WaveLink*") was passed
// straight to quit_application, whose osascript calls reject wildcard IDs, so
// the app was never quit before its bundle was removed.
func TestUninstallScriptExpandsQuitWildcard(t *testing.T) {
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						Quit: optjson.StringOr[[]string]{
							IsOther: true,
							Other:   []string{"com.elgato.WaveLink*", "com.elgato.StreamDeck"},
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(inputApp{}, cask)
	require.Contains(t, script, `quit_matching_applications 'com.elgato.WaveLink*'`)
	require.NotContains(t, script, `quit_application 'com.elgato.WaveLink*'`)
	require.Contains(t, script, quitMatchingApplicationsFunc)
	// quit_matching_applications quits each match through quit_application.
	require.Contains(t, script, quitApplicationFunc)
	require.Contains(t, script, `quit_application 'com.elgato.StreamDeck'`)

	// A cask without wildcard quit IDs must not pick up the new helper, so its
	// uninstall script (and script ref) stays unchanged.
	cask.Artifacts[0].Uninstall[0].Quit = optjson.StringOr[[]string]{String: "com.elgato.StreamDeck"}
	script = uninstallScriptForApp(inputApp{}, cask)
	require.NotContains(t, script, "quit_matching_applications")
	require.Contains(t, script, `quit_application 'com.elgato.StreamDeck'`)
}

// TestUninstallScriptQuitsAppByBundleIDWhenCaskListsNames covers casks whose
// quit stanza lists app names: osascript's application id only resolves bundle
// IDs, so those entries never quit the app.
func TestUninstallScriptQuitsAppByBundleIDWhenCaskListsNames(t *testing.T) {
	app := inputApp{UniqueIdentifier: "video.fathom.electron"}
	cask := &brewCask{
		Artifacts: []*brewArtifact{
			{
				Uninstall: []*brewUninstall{
					{
						Quit: optjson.StringOr[[]string]{
							IsOther: true,
							Other:   []string{"Fathom Helper", "Fathom", "video.fathom.electron", "com.example.agent"},
						},
					},
				},
			},
		},
	}

	script := uninstallScriptForApp(app, cask)
	require.Equal(t, 1, strings.Count(script, `quit_application 'video.fathom.electron'`), script)
	require.Contains(t, script, `quit_application 'com.example.agent'`)
	require.NotContains(t, script, `quit_application 'Fathom`)

	// A cask that quits the app only by name already stops it, so the app isn't
	// quit a second time before it's removed.
	cask.Artifacts[0].Uninstall[0].Quit = optjson.StringOr[[]string]{String: "Fathom"}
	script = uninstallScriptForApp(app, cask)
	require.Equal(t, 1, strings.Count(script, `quit_application 'video.fathom.electron' || exit 1`), script)

	// A dotless wildcard is a pattern, not an app name, so it still expands
	// against running apps instead of collapsing to the main bundle ID.
	cask.Artifacts[0].Uninstall[0].Quit = optjson.StringOr[[]string]{String: "*Fathom*"}
	script = uninstallScriptForApp(app, cask)
	require.Contains(t, script, `quit_matching_applications '*Fathom*'`)
	require.NotContains(t, script, `quit_application 'video.fathom.electron'`)
}

// TestQuitMatchingApplicationsRunsMatches runs the generated helper in bash with
// osascript and friends stubbed out, checking which running apps it quits.
func TestQuitMatchingApplicationsRunsMatches(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	if os.Geteuid() == 0 {
		t.Skip("as root the helper lists apps through /bin/launchctl asuser, which can't be stubbed")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	run := func(t *testing.T, consoleUser string, listFails bool, quitIDs ...string) string {
		cask := &brewCask{
			Artifacts: []*brewArtifact{
				{
					Uninstall: []*brewUninstall{
						{Quit: optjson.StringOr[[]string]{IsOther: true, Other: quitIDs}},
					},
				},
			},
		}
		// Shell functions take precedence over the real commands. The running
		// list includes a duplicate, a differently-cased match, IDs that only
		// match as a substring or not at all, and a matching ID that carries
		// AppleScript (a running app can report any bundle ID).
		stubs := `list_fails=` + strconv.FormatBool(listFails) + `
quit=" "
stat() { echo ` + consoleUser + `; }
sleep() { :; }
osascript() {
  if [[ "$1" == "-l" ]]; then
    if [[ "$list_fails" == true ]]; then
      return 1
    fi
    printf '%s\n' com.elgato.WaveLink com.elgato.wavelink.Helper com.elgato.WaveLink x.com.elgato.WaveLink com.elgato.StreamDeck \
      'com.elgato.WaveLink" is running or (do shell script "id") is "" or application id "com.elgato.WaveLink'
    return
  fi
  local id=${2#*\"}
  id=${id%%\"*}
  case "$2" in
    *" to quit") quit+="$id " ;;
    *" is running") if [[ "$quit" == *" $id "* ]]; then echo false; else echo true; fi ;;
  esac
}
`
		out, err := exec.Command(bash, "-c", stubs+uninstallScriptForApp(inputApp{}, cask)).CombinedOutput()
		require.NoError(t, err, string(out))
		return string(out)
	}

	t.Run("quits each match once", func(t *testing.T) {
		out := run(t, "alice", false, "com.elgato.WaveLink*")
		require.Equal(t, 1, strings.Count(out, "Quitting application 'com.elgato.WaveLink'..."), out)
		require.Equal(t, 1, strings.Count(out, "Quitting application 'com.elgato.wavelink.Helper'..."), out)
		require.NotContains(t, out, "x.com.elgato.WaveLink")
		require.NotContains(t, out, "com.elgato.StreamDeck")
		require.NotContains(t, out, "do shell script")
	})

	t.Run("no match", func(t *testing.T) {
		out := run(t, "alice", false, "com.example.Missing*")
		require.Equal(t, "No running application matches 'com.example.Missing*'.\n", out)
	})

	t.Run("regex metacharacters are literal", func(t *testing.T) {
		out := run(t, "alice", false, "com.elgato.Wave+Link*")
		require.Equal(t, "No running application matches 'com.elgato.Wave+Link*'.\n", out)
	})

	t.Run("no GUI user", func(t *testing.T) {
		out := run(t, "root", false, "com.elgato.WaveLink*")
		require.Equal(t, "Not logged into a non-root GUI; skipping quitting applications matching 'com.elgato.WaveLink*'.\n", out)
	})

	t.Run("listing fails", func(t *testing.T) {
		out := run(t, "alice", true, "com.elgato.WaveLink*")
		require.Equal(t, "Failed to list running applications; skipping quitting applications matching 'com.elgato.WaveLink*'.\n", out)
	})
}

func TestQuitTargetsApp(t *testing.T) {
	for _, tc := range []struct {
		quitID, bundleID string
		want             bool
	}{
		{"com.example.Foo", "com.example.Foo", true},
		{"COM.EXAMPLE.foo", "com.example.Foo", true},
		{"com.example.Foo*", "com.example.Foo", true},
		{"com.example.F*o", "com.example.Foo", true},
		{"com.example.Foo.*", "com.example.Foo", false},
		{"com.example.Foo.helper", "com.example.Foo", false},
		{"com.microsoft.autoupdate2", "com.microsoft.Word", false},
		{"com.example.Fo.", "com.example.Foo", false},
		{"com.example.Fo+", "com.example.Foo", false},
		{"*", "", false},
	} {
		require.Equal(t, tc.want, quitTargetsApp(tc.quitID, tc.bundleID), "%q vs %q", tc.quitID, tc.bundleID)
	}

	cask := &brewCask{Artifacts: []*brewArtifact{{Uninstall: []*brewUninstall{{
		Quit: optjson.StringOr[[]string]{IsOther: true, Other: []string{"com.docker.docker", "com.docker.helper"}},
	}}}}}
	script := uninstallScriptForApp(inputApp{UniqueIdentifier: "com.docker.docker"}, cask)
	require.Contains(t, script, "quit_application 'com.docker.docker' || exit 1\nquit_application 'com.electron.dockerdesktop'\nquit_application 'com.docker.helper'\n")
}

// runStubbedUninstallScript runs a generated uninstall script in bash with
// osascript and friends stubbed out. running lists the running apps; stuck ones
// accept the quit event but keep running, and canceled ones fail it, as when
// the user cancels a save prompt. sleep advances SECONDS so the timeout loop
// doesn't take real time.
func runStubbedUninstallScript(t *testing.T, script, running, stuck, canceled string) (string, int) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	stubs := `running=" ` + running + ` "
stuck=" ` + stuck + ` "
canceled=" ` + canceled + ` "
stat() { echo alice; }
sleep() { SECONDS=$((SECONDS + $1)); }
sudo() { echo "sudo $*"; }
osascript() {
  if [[ "$1" == "-l" ]]; then
    printf '%s\n' $running
    return
  fi
  local id=${2#*\"}
  id=${id%%\"*}
  case "$2" in
    *" to quit")
      [[ "$canceled" == *" $id "* ]] && return 1
      [[ "$stuck" == *" $id "* ]] || running=${running/ $id / }
      return 0
      ;;
    *" is running")
      if [[ "$running" == *" $id "* ]]; then echo true; else echo false; fi
      ;;
  esac
}
`
	out, err := exec.Command(bash, "-c", stubs+script).CombinedOutput()
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return string(out), exitErr.ExitCode()
	}
	require.NoError(t, err, string(out))
	return string(out), 0
}

// TestUninstallScriptStopsWhenAppWontQuit runs generated uninstall scripts with
// runStubbedUninstallScript. The app being uninstalled must not be removed
// while it's still running. Other quit IDs (helpers, shared updaters) only log
// that they didn't quit, as in Homebrew.
func TestUninstallScriptStopsWhenAppWontQuit(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("as root the wildcard helper lists apps through /bin/launchctl asuser, which can't be stubbed")
	}
	const remove = "sudo rm -rf /Applications//Foo.app"

	run := func(t *testing.T, quitIDs []string, running, stuck, canceled string) (string, int) {
		cask := &brewCask{Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{Quit: optjson.StringOr[[]string]{IsOther: true, Other: quitIDs}}}},
			{App: []optjson.StringOr[*brewAppTarget]{{String: "Foo.app"}}},
		}}
		script := uninstallScriptForApp(inputApp{UniqueIdentifier: "com.example.Foo"}, cask)
		return runStubbedUninstallScript(t, script, running, stuck, canceled)
	}

	const all = "com.example.Foo com.example.Foo.helper com.example.Updater"
	appAndHelper := []string{"com.example.Foo", "com.example.Foo.helper"}

	out, code := run(t, appAndHelper, "", "", "")
	require.Zero(t, code, out)
	require.NotContains(t, out, "Quitting application")
	require.Contains(t, out, remove)

	out, code = run(t, appAndHelper, all, "", "")
	require.Zero(t, code, out)
	require.Contains(t, out, "Application 'com.example.Foo' quit successfully.")
	require.Contains(t, out, "Application 'com.example.Foo.helper' quit successfully.")
	require.Contains(t, out, remove)

	for _, quitIDs := range [][]string{appAndHelper, {"com.example.Foo*"}} {
		out, code = run(t, quitIDs, all, "com.example.Foo", "")
		require.Equal(t, 1, code, out)
		require.Contains(t, out, "Application 'com.example.Foo' did not quit.")
		require.NotContains(t, out, remove)

		out, code = run(t, quitIDs, all, "", "com.example.Foo")
		require.Equal(t, 1, code, out)
		require.Contains(t, out, "Application 'com.example.Foo' did not quit.")
		require.NotContains(t, out, remove)
	}

	for _, quitIDs := range [][]string{appAndHelper, {"com.example.Foo.*"}} {
		out, code = run(t, quitIDs, all, "com.example.Foo.helper", "")
		require.Zero(t, code, out)
		require.Contains(t, out, "Application 'com.example.Foo.helper' did not quit.")
		require.Contains(t, out, remove)
	}

	out, code = run(t, []string{"com.example.Updater"}, all, "", "com.example.Updater")
	require.Zero(t, code, out)
	require.Contains(t, out, "Application 'com.example.Updater' did not quit.")
	require.Contains(t, out, remove)
}

// TestUninstallScriptQuitsAppCaskDoesNotName covers casks whose uninstall quit
// doesn't name the app, which is most of them: the script still quits the app
// before removing anything, and stops if the app won't quit.
func TestUninstallScriptQuitsAppCaskDoesNotName(t *testing.T) {
	app := inputApp{UniqueIdentifier: "com.example.Foo"}
	appArtifact := &brewArtifact{App: []optjson.StringOr[*brewAppTarget]{{String: "Foo.app"}}}
	const all = "com.example.Foo com.example.Updater"

	for name, cask := range map[string]*brewCask{
		"no uninstall stanza": {Artifacts: []*brewArtifact{appArtifact}},
		// Like the Microsoft Office casks, which only quit Microsoft AutoUpdate.
		"quits only an updater": {Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{
				Quit:   optjson.StringOr[[]string]{String: "com.example.Updater"},
				Delete: optjson.StringOr[[]string]{String: "/Applications/Foo Helper.app"},
			}}},
			appArtifact,
		}},
	} {
		t.Run(name, func(t *testing.T) {
			script := uninstallScriptForApp(app, cask)
			require.Equal(t, 1, strings.Count(script, "quit_application 'com.example.Foo' || exit 1"), script)

			out, code := runStubbedUninstallScript(t, script, all, "", "")
			require.Zero(t, code, out)
			require.Contains(t, out, "Application 'com.example.Foo' quit successfully.")
			require.Contains(t, out, "sudo rm -rf /Applications//Foo.app")

			out, code = runStubbedUninstallScript(t, script, all, "com.example.Foo", "")
			require.Equal(t, 1, code, out)
			require.Contains(t, out, "Application 'com.example.Foo' did not quit.")
			require.NotContains(t, out, "sudo rm -rf")

			out, code = runStubbedUninstallScript(t, script, all, "", "com.example.Foo")
			require.Equal(t, 1, code, out)
			require.NotContains(t, out, "sudo rm -rf")

			out, code = runStubbedUninstallScript(t, script, "", "", "")
			require.Zero(t, code, out)
			require.NotContains(t, out, "Quitting application")
			require.Contains(t, out, "sudo rm -rf /Applications//Foo.app")
		})
	}

	// The updater keeps Homebrew's warn-and-continue.
	script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
		{Uninstall: []*brewUninstall{{Quit: optjson.StringOr[[]string]{String: "com.example.Updater"}}}},
		appArtifact,
	}})
	out, code := runStubbedUninstallScript(t, script, all, "com.example.Updater", "")
	require.Zero(t, code, out)
	require.Contains(t, out, "Application 'com.example.Updater' did not quit.")
	require.Contains(t, out, "Application 'com.example.Foo' quit successfully.")
	require.Contains(t, out, "sudo rm -rf /Applications//Foo.app")
}

// TestUninstallScriptQuitsAppOnce checks where the generated quit goes and when
// it's left out because the cask already stops the app.
func TestUninstallScriptQuitsAppOnce(t *testing.T) {
	app := inputApp{UniqueIdentifier: "com.example.Foo"}
	appArtifact := &brewArtifact{App: []optjson.StringOr[*brewAppTarget]{{String: "Foo.app"}}}
	const quit = "quit_application 'com.example.Foo' || exit 1"
	const removeApp = `sudo rm -rf "$APPDIR/Foo.app"`
	strs := func(s ...string) optjson.StringOr[[]string] {
		return optjson.StringOr[[]string]{IsOther: true, Other: s}
	}

	t.Run("after launchctl, before the stanza removes anything", func(t *testing.T) {
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{
				LaunchCtl: strs("com.example.Foo.agent"),
				Quit:      strs("com.example.Updater"),
				Script: optjson.StringOr[map[string]any]{IsOther: true, Other: map[string]any{
					"executable": "/Applications/Foo.app/Contents/Resources/uninstall.sh",
				}},
				PkgUtil: strs("com.example.pkg"),
			}}},
			appArtifact,
			{Zap: []*brewUninstall{{Trash: strs("~/Library/Preferences/com.example.Foo.plist")}}},
		}})
		require.Equal(t, 1, strings.Count(script, quit), script)
		order := []string{
			"remove_launchctl_service 'com.example.Foo.agent'",
			"quit_application 'com.example.Updater'\n",
			quit,
			"/Applications/Foo.app/Contents/Resources/uninstall.sh",
			"remove_pkg_files 'com.example.pkg'",
			removeApp,
			`trash "$LOGGED_IN_USER"`,
		}
		var prev string
		for _, s := range order {
			require.Contains(t, script, s)
			if prev != "" {
				require.Less(t, strings.Index(script, prev), strings.Index(script, s), "%q before %q", prev, s)
			}
			prev = s
		}
	})

	t.Run("zap quit doesn't count", func(t *testing.T) {
		// zap runs after the app is removed.
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			appArtifact,
			{Zap: []*brewUninstall{{Quit: strs("com.example.Foo")}}},
		}})
		require.Contains(t, script, quit)
		require.Less(t, strings.Index(script, quit), strings.Index(script, removeApp))
	})

	for name, uninstall := range map[string]*brewUninstall{
		"cask quits the app":                    {Quit: strs("com.example.Updater", "com.example.Foo")},
		"cask quits the app by another case":    {Quit: strs("COM.EXAMPLE.FOO")},
		"cask quits the app by wildcard":        {Quit: strs("com.example.*")},
		"cask quits the app as a single string": {Quit: optjson.StringOr[[]string]{String: "com.example.Foo"}},
	} {
		t.Run(name, func(t *testing.T) {
			script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
				{Uninstall: []*brewUninstall{uninstall}},
				appArtifact,
			}})
			require.Len(t, stoppingQuit.FindAllString(script, -1), 1, script)
		})
	}

	// Casks signal apps that don't quit cleanly, and can split the stanza
	// (perimeter81).
	t.Run("cask signals the app", func(t *testing.T) {
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{EarlyScript: optjson.StringOr[map[string]any]{IsOther: true, Other: map[string]any{
				"executable": "/usr/bin/chflags", "args": []any{"-RL", "noschg", "/Applications/Foo.app"},
			}}}}},
			{Uninstall: []*brewUninstall{{Signal: strs("TERM", "com.example.Foo")}}},
			appArtifact,
		}})
		require.NotContains(t, script, "quit_application")
		require.Contains(t, script, `send_signal 'TERM' 'com.example.Foo' "$LOGGED_IN_USER"`)
	})

	t.Run("cask signals something else", func(t *testing.T) {
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{Signal: strs("TERM", "com.example.Updater")}}},
			appArtifact,
		}})
		require.Contains(t, script, quit)
		require.Less(t, strings.Index(script, quit), strings.Index(script, "send_signal 'TERM'"))
	})

	// Like the Microsoft Office casks: a pkg install the stanza removes.
	t.Run("no app artifact", func(t *testing.T) {
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			{Uninstall: []*brewUninstall{{Quit: strs("com.example.Updater"), PkgUtil: strs("com.example.pkg")}}},
		}})
		require.Contains(t, script, quit)
		require.Less(t, strings.Index(script, quit), strings.Index(script, "remove_pkg_files 'com.example.pkg'"))
	})

	t.Run("nothing to remove", func(t *testing.T) {
		script := uninstallScriptForApp(app, &brewCask{Artifacts: []*brewArtifact{
			{Zap: []*brewUninstall{{Trash: strs("~/Library/Preferences/com.example.Foo.plist")}}},
		}})
		require.NotContains(t, script, "quit_application")
	})

	t.Run("no unique identifier", func(t *testing.T) {
		script := uninstallScriptForApp(inputApp{}, &brewCask{Artifacts: []*brewArtifact{appArtifact}})
		require.NotContains(t, script, "quit_application")
	})
}

// stoppingQuit matches generated quit calls that stop the script.
var stoppingQuit = regexp.MustCompile(`(?m)^quit_(application|matching_applications) '[^']*' \|\| exit 1$`)

// TestCustomUninstallScriptsUseCurrentQuitHelper checks the quit_application
// copies in custom uninstall scripts. An old copy confirmed the quit with
// pgrep, which never matches the bundle id, and never reported a failed quit.
func TestCustomUninstallScriptsUseCurrentQuitHelper(t *testing.T) {
	paths, err := filepath.Glob("../../inputs/homebrew/scripts/*uninstall*.sh")
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		script := string(b)
		if !strings.Contains(script, "\nquit_application() {\n") {
			continue
		}
		require.Contains(t, script, quitApplicationFunc, "%s has a stale copy of quit_application", filepath.Base(path))
	}
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

	script := uninstallScriptForApp(inputApp{}, cask)
	chflags := `(cd "/Users/$LOGGED_IN_USER" 2>/dev/null || cd /; '/usr/bin/chflags' '-RL' 'noschg' '/Applications/Foo.app') || true`
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

	script := uninstallScriptForApp(inputApp{}, cask)
	require.Contains(t, script, `(cd "/Users/$LOGGED_IN_USER" 2>/dev/null || cd /; sudo '/Applications/Foo.app/Contents/MacOS/Foo' '--unload-system-extension')`)
	require.NotContains(t, script, "|| true")
}

// TestUninstallScriptScriptDirectiveWithoutGUIUser runs generated script
// directives in bash with the console user stubbed. Fleet also runs uninstalls
// at the login window, where `cd /Users/loginwindow` failed and skipped the
// vendor uninstaller, and an empty user ran it with `sudo -u ""`.
func TestUninstallScriptScriptDirectiveWithoutGUIUser(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not found")
	}

	tmp := t.TempDir()
	// /Users is relocated into tmp, where only alice has a home.
	usersDir := filepath.Join(tmp, "Users")
	require.NoError(t, os.MkdirAll(filepath.Join(usersDir, "alice"), 0o755))
	resolvedUsersDir, err := filepath.EvalSymlinks(usersDir)
	require.NoError(t, err)
	aliceHome := filepath.Join(resolvedUsersDir, "alice")

	rootTool := filepath.Join(tmp, "bin", "root-tool")
	userTool := filepath.Join(tmp, "bin", "user-tool")
	toolScript := []byte(`#!/bin/bash
echo "ran ${0##*/} in $(pwd -P)"
exit "${TOOL_EXIT:-0}"
`)
	require.NoError(t, os.MkdirAll(filepath.Dir(rootTool), 0o755))
	for _, tool := range []string{rootTool, userTool} {
		require.NoError(t, os.WriteFile(tool, toolScript, 0o755)) //nolint:gosec // G306: the test executes it
	}

	rootScript := func(mustSucceed bool) *brewUninstall {
		return &brewUninstall{EarlyScript: optjson.StringOr[map[string]any]{
			IsOther: true,
			Other: map[string]any{
				"executable":   rootTool,
				"args":         []any{"--unload"},
				"sudo":         true,
				"must_succeed": mustSucceed,
			},
		}}
	}
	userScript := &brewUninstall{Script: optjson.StringOr[map[string]any]{String: userTool}}

	run := func(t *testing.T, consoleUser string, toolExit int, directives ...*brewUninstall) (string, int) {
		stubs := `console_user=` + shellSingleQuote(consoleUser) + `
users_dir=` + shellSingleQuote(usersDir) + `
scutil() { [[ -z "$console_user" ]] || echo "  Name : $console_user"; }
cd() {
  if [[ "$1" == /Users/* ]]; then
    builtin cd "$users_dir/${1#/Users/}"
  else
    builtin cd "$@"
  fi
}
sudo() {
  echo "sudo $*"
  [[ "$1" != -u ]] || shift 2
  "$@"
}
`
		cask := &brewCask{Artifacts: []*brewArtifact{{Uninstall: directives}}}
		cmd := exec.Command(bash, "-c", stubs+uninstallScriptForApp(inputApp{}, cask))
		cmd.Env = append(os.Environ(), "TOOL_EXIT="+strconv.Itoa(toolExit))
		out, err := cmd.CombinedOutput()
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return string(out), exitErr.ExitCode()
		}
		require.NoError(t, err, string(out))
		return string(out), 0
	}

	for name, user := range map[string]string{"loginwindow": "loginwindow", "empty": "", "root": "root"} {
		t.Run(name, func(t *testing.T) {
			out, code := run(t, user, 0, rootScript(true), userScript)
			require.Zero(t, code, out)
			require.Contains(t, out, "sudo "+rootTool+" --unload\nran root-tool in ")
			require.Contains(t, out, "Not logged into a non-root GUI; skipping "+userTool+".\n")
			require.NotContains(t, out, "user-tool in")
			if user != "" {
				require.Contains(t, out, "ran root-tool in /\n")
			}
		})
	}

	t.Run("normal user", func(t *testing.T) {
		out, code := run(t, "alice", 0, rootScript(true), userScript)
		require.Zero(t, code, out)
		require.Contains(t, out, "sudo "+rootTool+" --unload\nran root-tool in "+aliceHome+"\n")
		require.Contains(t, out, "sudo -u alice "+userTool+"\nran user-tool in "+aliceHome+"\n")
		require.NotContains(t, out, "Not logged into")
	})

	t.Run("must_succeed", func(t *testing.T) {
		out, code := run(t, "loginwindow", 3, rootScript(true))
		require.Contains(t, out, "ran root-tool in /\n")
		require.Equal(t, 3, code, out)

		out, code = run(t, "loginwindow", 3, rootScript(false))
		require.Contains(t, out, "ran root-tool in /\n")
		require.Zero(t, code, out)

		out, code = run(t, "alice", 3, userScript)
		require.Contains(t, out, "ran user-tool in "+aliceHome+"\n")
		require.Equal(t, 3, code, out)
	})
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

	script := uninstallScriptForApp(inputApp{}, cask)
	require.NotContains(t, script, "$HOMEBREW_PREFIX")
	require.NotContains(t, script, "/usr/sbin/installer")
}

// varNameDerivation is the line both generated functions share.
const varNameDerivation = `  local var_name="APP_WAS_RUNNING_${bundle_id//[^[:alnum:]_]/_}"`

// varNameRoundTrip writes the flag via eval and reads it back, as the two
// generated functions do.
func varNameRoundTrip(t *testing.T, bundleID string) (name, wasRunning string) {
	t.Helper()

	script := `set -u
track() {
  local bundle_id="$1"
` + varNameDerivation + `
  eval "export $var_name=1"
}
read_back() {
  local bundle_id="$1"
` + varNameDerivation + `
  local was_running=
  eval "was_running=\$$var_name"
  printf '%s|%s' "$var_name" "$was_running"
}
track "$1"
read_back "$1"`

	out, err := exec.Command("bash", "-c", script, "bash", bundleID).CombinedOutput()
	require.NoError(t, err, "output: %s", out)

	name, wasRunning, ok := strings.Cut(string(out), "|")
	require.True(t, ok, "unexpected output: %q", out)
	return name, wasRunning
}

// "org.mozilla.pale moon" used to derive a name with a space in it, which split
// the eval so the read back came up empty and Pale Moon was never relaunched.
func TestQuitRelaunchVarNameSurvivesEval(t *testing.T) {
	for _, tc := range []struct {
		bundleID string
		want     string
	}{
		{"com.evernote.Evernote", "APP_WAS_RUNNING_com_evernote_Evernote"},
		{"Cisco-Systems.Spark", "APP_WAS_RUNNING_Cisco_Systems_Spark"},
		{"org.R-project.R", "APP_WAS_RUNNING_org_R_project_R"},
		{"org.mozilla.pale moon", "APP_WAS_RUNNING_org_mozilla_pale_moon"},
	} {
		t.Run(tc.bundleID, func(t *testing.T) {
			name, wasRunning := varNameRoundTrip(t, tc.bundleID)
			require.Equal(t, tc.want, name)
			require.Equal(t, "1", wasRunning)
		})
	}
}

// unique_identifier is repo-controlled, but the name still reaches a root eval.
func TestQuitRelaunchVarNameNeutralizesCommandSubstitution(t *testing.T) {
	pwned := filepath.Join(t.TempDir(), "pwned")

	name, wasRunning := varNameRoundTrip(t, "com.evil$(touch "+pwned+")")

	require.Regexp(t, `^APP_WAS_RUNNING_[A-Za-z0-9_]+$`, name, "derived name is not a bare shell identifier")
	require.Equal(t, "1", wasRunning)
	require.NoFileExists(t, pwned, "command substitution in the bundle ID was executed")
}

// If the two derivations drift, relaunch silently stops working.
func TestQuitAndRelaunchDeriveVarNameIdentically(t *testing.T) {
	require.Contains(t, quitAndTrackApplicationFunc, varNameDerivation)
	require.Contains(t, relaunchApplicationFunc, varNameDerivation)
	require.NotContains(t, quitAndTrackApplicationFunc, `tr '.-' '__'`)
	require.NotContains(t, relaunchApplicationFunc, `tr '.-' '__'`)
}

// Custom install scripts carry their own copy of these functions.
func TestCustomInstallScriptsDoNotUseOldVarNameDerivation(t *testing.T) {
	scripts, err := filepath.Glob("../../inputs/homebrew/scripts/*.sh")
	require.NoError(t, err)
	require.NotEmpty(t, scripts)

	for _, path := range scripts {
		b, err := os.ReadFile(path)
		require.NoError(t, err)
		require.NotContains(t, string(b), `tr '.-' '__'`, "%s derives APP_WAS_RUNNING_ the old way", filepath.Base(path))
	}
}
