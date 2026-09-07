package homebrew

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/micromdm/plist"
)

func installScriptForApp(app inputApp, cask *brewCask) (string, error) {
	sb := newScriptBuilder()

	sb.AddVariable("TMPDIR", `$(dirname "$(realpath "$INSTALLER_PATH")")`)
	sb.AddVariable("APPDIR", `"/Applications/"`)

	sb.Extract(app.InstallerFormat)

	// Add quit/relaunch functions if we have App or Pkg artifacts
	var needsQuitRelaunch bool
	for _, artifact := range cask.Artifacts {
		if len(artifact.App) > 0 || len(artifact.Pkg) > 0 {
			needsQuitRelaunch = true
			break
		}
	}

	if needsQuitRelaunch {
		sb.AddFunction("quit_and_track_application", quitAndTrackApplicationFunc)
		sb.AddFunction("relaunch_application", relaunchApplicationFunc)
	}

	for _, artifact := range cask.Artifacts {
		switch {
		case len(artifact.App) > 0:
			sb.Write("# copy to the applications folder")
			// Quit the app before installing if it's running, and track state for relaunch
			sb.Writef("quit_and_track_application '%s'", app.UniqueIdentifier)
			for _, appItem := range artifact.App {
				// Only process string values (skip objects with target, those are handled by custom scripts)
				if appItem.String == "" {
					continue
				}
				// appPath is cask-controlled and interpolated inside double-quoted
				// strings alongside "$APPDIR"/"$TMPDIR"; escape it so a payload like
				// $(...) can't reach the root-privileged mv/cp/rm below.
				appPath := shellDoubleQuoteEscape(appItem.String)
				sb.Writef(`if [ -d "$APPDIR/%[1]s" ]; then
	sudo mv "$APPDIR/%[1]s" "$TMPDIR/%[1]s.bkp" || exit $?
fi`, appPath)
				sb.Writef(`if ! sudo cp -R "$TMPDIR/%[1]s" "$APPDIR"; then
	# remove the partial copy so a failed install isn't inventoried as the new
	# version, then restore the previous version if there was one
	sudo rm -rf "$APPDIR/%[1]s"
	if [ -d "$TMPDIR/%[1]s.bkp" ]; then
		sudo mv "$TMPDIR/%[1]s.bkp" "$APPDIR/%[1]s"
	fi
	exit 1
fi`, appPath)
			}
			// Relaunch the app if it was running before installation
			sb.Writef("relaunch_application '%s'", app.UniqueIdentifier)

		case len(artifact.Pkg) > 0:
			sb.Write("# install pkg files")
			// Quit the app before installing if it's running, and track state for relaunch
			sb.Writef("quit_and_track_application '%s'", app.UniqueIdentifier)
			switch len(artifact.Pkg) {
			case 1:
				if err := sb.InstallPkg(artifact.Pkg[0].String); err != nil {
					return "", fmt.Errorf("building statement to install pkg: %w", err)
				}
			case 2:
				if err := sb.InstallPkg(artifact.Pkg[0].String, artifact.Pkg[1].Other.Choices); err != nil {
					return "", fmt.Errorf("building statement to install pkg with choices: %w", err)
				}
			default:
				return "", fmt.Errorf("application %s has unknown directive format for pkg", app.Token)
			}
			// Relaunch the app if it was running before installation
			sb.Writef("relaunch_application '%s'", app.UniqueIdentifier)

		case len(artifact.Binary) > 0:
			if len(artifact.Binary) == 2 {
				source := artifact.Binary[0].String
				target := artifact.Binary[1].Other.Target

				if !strings.Contains(target, "$HOMEBREW_PREFIX") &&
					!strings.Contains(source, "$HOMEBREW_PREFIX") {
					sb.Symlink(source, target)
				}
			}
		}
	}

	return sb.String(), nil
}

func uninstallScriptForApp(cask *brewCask) string {
	sb := newScriptBuilder()

	for _, artifact := range cask.Artifacts {
		switch {
		case len(artifact.App) > 0:
			sb.AddVariable("APPDIR", `"/Applications/"`)
			// Collect app paths to remove, prioritizing target names (what actually gets installed)
			var appPathsToRemove []string
			var hasTarget bool
			for _, appItem := range artifact.App {
				if appItem.Other != nil {
					appPathsToRemove = append(appPathsToRemove, appItem.Other.Target)
					hasTarget = true
				}
			}
			// Only use string values if no target was found (target takes precedence)
			if !hasTarget {
				for _, appItem := range artifact.App {
					if appItem.String != "" {
						appPathsToRemove = append(appPathsToRemove, appItem.String)
					}
				}
			}
			// Remove all collected app paths. appPath is cask-controlled and lands
			// inside a double-quoted `sudo rm -rf` argument, so escape it to stop
			// $(...)/backtick command substitution.
			for _, appPath := range appPathsToRemove {
				sb.RemoveFile(fmt.Sprintf(`"$APPDIR/%s"`, shellDoubleQuoteEscape(appPath)))
			}
		case len(artifact.Binary) > 0:
			if len(artifact.Binary) == 2 {
				target := artifact.Binary[1].Other.Target
				if !strings.Contains(target, "$HOMEBREW_PREFIX") {
					sb.RemoveFile(shellSingleQuote(target))
				}
			}
		case len(artifact.Uninstall) > 0:
			sortUninstall(artifact.Uninstall)
			if len(cask.PreUninstallScripts) > 0 {
				sb.Write(strings.Join(cask.PreUninstallScripts, "\n"))
			}
			for _, u := range artifact.Uninstall {
				processUninstallArtifact(u, sb)
			}
			if len(cask.PostUninstallScripts) > 0 {
				sb.Write(strings.Join(cask.PostUninstallScripts, "\n"))
			}
		case len(artifact.Zap) > 0:
			sortUninstall(artifact.Zap)
			for _, z := range artifact.Zap {
				processUninstallArtifact(z, sb)
			}
		}
	}

	return sb.String()
}

// priority of uninstall directives is defined by homebrew here:
// https://github.com/Homebrew/brew/blob/e1ff668957dd8a66304c0290dfa66083e6c7444e/Library/Homebrew/cask/artifact/abstract_uninstall.rb#L18-L30
const (
	PriorityEarlyScript = iota
	PriorityLaunchctl
	PriorityQuit
	PrioritySignal
	PriorityLoginItem
	PriorityKext
	PriorityScript
	PriorityPkgutil
	PriorityDelete
	PriorityTrash
	PriorityRmdir
)

// uninstallArtifactOrder returns an integer representing the priority of the
// artifact based on the uninstall directives it contains. Lower number means
// higher priority
func uninstallArtifactOrder(artifact *brewUninstall) int {
	switch {
	case len(artifact.EarlyScript.String)+len(artifact.EarlyScript.Other) > 0:
		return PriorityEarlyScript
	case len(artifact.LaunchCtl.String)+len(artifact.LaunchCtl.Other) > 0:
		return PriorityLaunchctl
	case len(artifact.Quit.String)+len(artifact.Quit.Other) > 0:
		return PriorityQuit
	case len(artifact.Signal.String)+len(artifact.Signal.Other) > 0:
		return PrioritySignal
	case len(artifact.LoginItem.String)+len(artifact.LoginItem.Other) > 0:
		return PriorityLoginItem
	case len(artifact.Kext.String)+len(artifact.Kext.Other) > 0:
		return PriorityKext
	case len(artifact.Script.String)+len(artifact.Script.Other) > 0:
		return PriorityScript
	case len(artifact.PkgUtil.String)+len(artifact.PkgUtil.Other) > 0:
		return PriorityPkgutil
	case len(artifact.Delete.String)+len(artifact.Delete.Other) > 0:
		return PriorityDelete
	case len(artifact.Trash.String)+len(artifact.Trash.Other) > 0:
		return PriorityTrash
	case len(artifact.RmDir.String)+len(artifact.RmDir.Other) > 0:
		return PriorityRmdir
	default:
		return 999
	}
}

func sortUninstall(artifacts []*brewUninstall) {
	slices.SortFunc(artifacts, func(a, b *brewUninstall) int {
		return uninstallArtifactOrder(a) - uninstallArtifactOrder(b)
	})
}

// shellSingleQuote wraps s in single quotes for safe use as a single shell
// token, escaping any embedded single quote by closing the quote, emitting an
// escaped quote, and reopening (the standard POSIX idiom). Cask-supplied paths
// can contain apostrophes (e.g. "Cycling '74"), which would otherwise
// prematurely close the quote and produce a syntax error.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// dqEscaper backslash-escapes the only four characters that keep a special
// meaning inside a double-quoted shell string: backslash, dollar, backtick, and
// double quote. strings.Replacer makes a single pass and never rescans the
// replacement text it inserts, so the escapes it introduces are not themselves
// re-escaped (regardless of pair order).
var dqEscaper = strings.NewReplacer(
	`\`, `\\`,
	`$`, `\$`,
	"`", "\\`",
	`"`, `\"`,
)

// shellDoubleQuoteEscape escapes s for safe interpolation inside an existing
// double-quoted shell string. Cask metadata is attacker-influenced (it flows
// verbatim from formulae.brew.sh into root-privileged generated scripts), so a
// value like `$(...)` or a backtick must not be treated as command
// substitution. Unlike shellSingleQuote this preserves a surrounding "$VAR"
// (e.g. "$APPDIR"/"$TMPDIR") that legitimately needs to expand, and it leaves
// values without shell metacharacters — the common case — byte-for-byte
// unchanged, so the generated scripts (and their downstream comparisons in the
// auto-update path) don't churn.
func shellDoubleQuoteEscape(s string) string {
	return dqEscaper.Replace(s)
}

// escapeCaskPath escapes a cask-supplied path for use inside a double-quoted
// shell string while preserving a leading "$APPDIR" — the one cask variable the
// generated scripts define and legitimately expand at runtime (every real
// binary-artifact source is "$APPDIR/Foo.app/..."). Everything after the prefix
// is data and gets escaped; paths without metacharacters come back unchanged.
func escapeCaskPath(s string) string {
	if s == "$APPDIR" {
		return s
	}
	if rest, ok := strings.CutPrefix(s, "$APPDIR/"); ok {
		return "$APPDIR/" + shellDoubleQuoteEscape(rest)
	}
	return shellDoubleQuoteEscape(s)
}

var inertBareword = regexp.MustCompile(`^[A-Za-z0-9_./+][A-Za-z0-9_./+-]*$`)

// inertShellWord reports whether s is safe to emit as a single unquoted shell
// word: an optional "$APPDIR" prefix followed only by filename-safe bytes, with
// no leading dash that a command would parse as an option. Inert values keep
// the generator's historical unquoted form: the auto-update job treats any byte
// difference between a stored script and the manifest script as an admin
// customization and pins the stored script (which names the old installer
// file), so format churn would break auto-updates for every affected app.
func inertShellWord(s string) bool {
	if s == "$APPDIR" {
		return true
	}
	if rest, ok := strings.CutPrefix(s, "$APPDIR/"); ok {
		return inertBareword.MatchString(rest)
	}
	return inertBareword.MatchString(s)
}

// heredocInert reports whether s is pure data in an unquoted << EOF heredoc
// body: nothing the shell expands there ($, backtick, backslash) and no line
// equal to the delimiter, which would terminate the heredoc early and run the
// remainder as commands.
func heredocInert(s string) bool {
	if strings.ContainsAny(s, "$`\\") {
		return false
	}
	return !slices.Contains(strings.Split(s, "\n"), "EOF")
}

// processScript writes a cask "script"/"early_script" uninstall directive. Both
// take the same shape, so they share this; the caller decides the order.
func processScript(script optjson.StringOr[map[string]any], sb *scriptBuilder, addUserVar func()) {
	if script.IsOther {
		// for supported FMAs, this is a map with "executable" as the script path,
		// optional "args" array, optional "sudo" boolean, and optional "must_succeed" boolean
		executable, ok := script.Other["executable"].(string)
		if !ok {
			panic("executable not found or not a string in script")
		}

		// Build the command with arguments if present
		var cmdParts []string
		cmdParts = append(cmdParts, shellSingleQuote(executable))

		// Handle args if present
		if argsVal, hasArgs := script.Other["args"]; hasArgs {
			args, ok := argsVal.([]interface{})
			if !ok {
				panic("args must be an array in script")
			}
			for _, arg := range args {
				argStr, ok := arg.(string)
				if !ok {
					panic("all args must be strings")
				}
				cmdParts = append(cmdParts, shellSingleQuote(argStr))
			}
		}

		// Paths under the Caskroom only exist where Homebrew installed the app, so
		// the directive can't do anything on a Fleet-managed host (same reason the
		// binary artifact skips them).
		if strings.Contains(executable, "$HOMEBREW_PREFIX") ||
			slices.ContainsFunc(cmdParts, func(p string) bool { return strings.Contains(p, "$HOMEBREW_PREFIX") }) {
			return
		}

		addUserVar()

		cmd := strings.Join(cmdParts, " ")

		// Handle must_succeed - if false, we can ignore errors
		mustSucceed := true
		if mustSucceedVal, hasMustSucceed := script.Other["must_succeed"]; hasMustSucceed {
			if ms, ok := mustSucceedVal.(bool); ok {
				mustSucceed = ms
			}
		}

		// Handle sudo - check if sudo is required (defaults to false if not specified)
		needsSudo := false
		if sudoVal, hasSudo := script.Other["sudo"]; hasSudo {
			if sudo, ok := sudoVal.(bool); ok && sudo {
				needsSudo = true
			}
		}

		// Build the command execution
		if needsSudo {
			if mustSucceed {
				sb.Writef(`(cd /Users/$LOGGED_IN_USER && sudo %s)`, cmd)
			} else {
				sb.Writef(`(cd /Users/$LOGGED_IN_USER && sudo %s) || true`, cmd)
			}
		} else {
			if mustSucceed {
				sb.Writef(`(cd /Users/$LOGGED_IN_USER && %s)`, cmd)
			} else {
				sb.Writef(`(cd /Users/$LOGGED_IN_USER && %s) || true`, cmd)
			}
		}
	} else if len(script.String) > 0 {
		if strings.Contains(script.String, "$HOMEBREW_PREFIX") {
			return
		}
		addUserVar()
		// Quote via shellSingleQuote rather than a bare '%s': the cask-controlled
		// value could otherwise close the quote with an embedded apostrophe and
		// inject commands.
		sb.Writef(`(cd /Users/$LOGGED_IN_USER && sudo -u "$LOGGED_IN_USER" %s)`, shellSingleQuote(script.String))
	}
}

func processUninstallArtifact(u *brewUninstall, sb *scriptBuilder) {
	process := func(target optjson.StringOr[[]string], f func(path string)) {
		if target.IsOther {
			for _, path := range target.Other {
				f(path)
			}
		} else if len(target.String) > 0 {
			f(target.String)
		}
	}

	addUserVar := func() {
		sb.AddVariable("LOGGED_IN_USER", `$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ { print $3 }')`)
	}

	process(u.LaunchCtl, func(lc string) {
		sb.AddFunction("remove_launchctl_service", removeLaunchctlServiceFunc)
		sb.Writef("remove_launchctl_service %s", shellSingleQuote(lc))
	})

	process(u.Quit, func(appName string) {
		sb.AddFunction("quit_application", quitApplicationFunc)
		sb.Writef("quit_application %s", shellSingleQuote(appName))
		if appName == "com.docker.docker" {
			sb.Writef("quit_application 'com.electron.dockerdesktop'")
		}
	})

	// per the spec, signals can't have a different format. In the homebrew
	// source code an error is raised when the format is different.
	if u.Signal.IsOther && len(u.Signal.Other) == 2 {
		addUserVar()
		sb.AddFunction("send_signal", sendSignalFunc)
		sb.Writef(`send_signal %s %s "$LOGGED_IN_USER"`, shellSingleQuote(u.Signal.Other[0]), shellSingleQuote(u.Signal.Other[1]))
	}

	// brew runs early_script ahead of every other directive; keep that order so
	// the commands that make removal possible run before anything is removed.
	processScript(u.EarlyScript, sb, addUserVar)
	processScript(u.Script, sb, addUserVar)

	process(u.PkgUtil, func(pkgID string) {
		sb.AddFunction("expand_pkgid_and_map", expandWildcardPkgs)
		sb.AddFunction("remove_pkg_files", removePkgFiles)
		sb.AddFunction("forget_pkg", forgetPkgFunc)
		sb.Writef("remove_pkg_files %s", shellSingleQuote(pkgID))
		sb.Writef("forget_pkg %s", shellSingleQuote(pkgID))
	})

	process(u.Delete, func(path string) {
		sb.RemoveFile(shellSingleQuote(path))
	})

	process(u.RmDir, func(dir string) {
		sb.Writef("sudo rmdir %s", shellSingleQuote(dir))
	})

	process(u.Trash, func(path string) {
		addUserVar()
		sb.AddFunction("trash", trashFunc)
		sb.Writef("trash $LOGGED_IN_USER %s", shellSingleQuote(path))
	})
}

type scriptBuilder struct {
	statements   []string
	variables    map[string]string
	functions    map[string]string
	pathsCreated map[string]struct{}
}

func newScriptBuilder() *scriptBuilder {
	return &scriptBuilder{
		statements:   []string{},
		variables:    map[string]string{},
		functions:    map[string]string{},
		pathsCreated: map[string]struct{}{},
	}
}

// AddVariable adds a variable definition to the script
func (s *scriptBuilder) AddVariable(name, definition string) {
	s.variables[name] = definition
}

// AddFunction adds a shell function to the script.
func (s *scriptBuilder) AddFunction(name, definition string) {
	s.functions[name] = definition
}

// Write appends a raw shell command or statement to the script.
func (s *scriptBuilder) Write(in string) {
	s.statements = append(s.statements, in)
}

// Writef formats a string according to the specified format and arguments,
// then appends it to the script.
func (s *scriptBuilder) Writef(format string, args ...any) {
	s.statements = append(s.statements, fmt.Sprintf(format, args...))
}

// Extract writes shell commands to extract the contents of an installer based
// on the given format.
//
// Supported formats are "dmg" and "zip". It adds the necessary extraction
// commands to the script.
func (s *scriptBuilder) Extract(format string) {
	switch format {
	case "dmg":
		s.Write("# extract contents")
		// Pipe yes into hdiutil to auto-accept license agreements on licensed DMGs (Homebrew
		// behavior). Harmless when the DMG has no license prompt.
		s.Write(`MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
yes | hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH" || exit 1
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT" || true`)

	case "zip":
		s.Write("# extract contents")
		s.Write(`unzip "$INSTALLER_PATH" -d "$TMPDIR"`)
	}
}

// RemoveFile writes a command to remove a file or directory with sudo
// privileges.
func (s *scriptBuilder) RemoveFile(file string) {
	s.Writef(`sudo rm -rf %s`, file)
}

// InstallPkg writes a command to install a package using the macOS `installer` utility.
// 'pkg' is the package file to install. Optionally, 'choices' can be provided to specify
// installation options.
//
// If no choices are provided, a simple install command is written.
//
// Returns an error if generating the XML for choices fails.
func (s *scriptBuilder) InstallPkg(pkg string, choices ...[]brewPkgConfig) error {
	// pkg is cask-controlled and interpolated inside a double-quoted "$TMPDIR/..."
	// argument; escape it so command substitution can't survive.
	pkg = shellDoubleQuoteEscape(pkg)
	if len(choices) == 0 {
		s.Writef(`sudo installer -pkg "$TMPDIR/%s" -target / || exit $?`, pkg)
		return nil
	}

	choiceXML, err := plist.MarshalIndent(choices[0], "  ")
	if err != nil {
		return err
	}

	// The choice XML embeds cask-controlled strings (choiceIdentifier /
	// choiceAttribute). An unquoted heredoc shell-expands $/backtick/backslash in
	// its body, and a body line equal to the delimiter ends it early, running the
	// remainder as commands. Keep the historical heredoc when the XML is inert on
	// all those counts — every real cask today, so stored scripts don't churn (see
	// inertShellWord) — and otherwise write the XML as a single-quoted printf
	// literal, which the shell can't interpret.
	if xml := string(choiceXML); heredocInert(xml) {
		s.Writef(`
CHOICE_XML=$(mktemp /tmp/choice_xml_XXX)

cat << EOF > "$CHOICE_XML"
%s
EOF

sudo installer -pkg "$TMPDIR/%s" -target / -applyChoiceChangesXML "$CHOICE_XML" || exit $?
`, xml, pkg)
	} else {
		s.Writef(`
CHOICE_XML=$(mktemp /tmp/choice_xml_XXX)

printf '%%s\n' %s > "$CHOICE_XML"

sudo installer -pkg "$TMPDIR/%s" -target / -applyChoiceChangesXML "$CHOICE_XML" || exit $?
`, shellSingleQuote(xml), pkg)
	}

	return nil
}

// Symlink writes a command to create a symbolic link from 'source' to 'target'.
func (s *scriptBuilder) Symlink(source, target string) {
	// source/target are cask-controlled: the ln arguments, though double-quoted,
	// allowed $(...) command substitution, and the mkdir argument was unquoted.
	// Inert paths keep the historical unquoted mkdir form so stored scripts don't
	// churn (see inertShellWord); `--` stops a leading dash in a hostile path from
	// being parsed as an option.
	pathname := filepath.Dir(target)
	if _, ok := s.pathsCreated[pathname]; !ok {
		if inertShellWord(pathname) {
			s.Writef("mkdir -p %s", pathname)
		} else {
			s.Writef(`mkdir -p -- "%s"`, escapeCaskPath(pathname))
		}
		s.pathsCreated[pathname] = struct{}{}
	}
	s.Writef(`/bin/ln -h -f -s -- "%s" "%s"`, escapeCaskPath(source), escapeCaskPath(target))
}

// String generates the final script as a string.
//
// It includes the shebang, any variables, functions, and statements in the
// correct order.
func (s *scriptBuilder) String() string {
	var script strings.Builder
	script.WriteString("#!/bin/bash\n\n")

	if len(s.variables) > 0 {
		// write variables, order them alphabetically to produce deterministic
		// scripts.
		script.WriteString("# variables\n")
		keys := make([]string, 0, len(s.variables))
		for name := range s.variables {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			script.WriteString(fmt.Sprintf("%s=%s\n", name, s.variables[name]))
		}
	}

	if len(s.functions) > 0 {
		// write functions, order them alphabetically to produce deterministic
		// scripts.
		script.WriteString("# functions\n")
		keys := make([]string, 0, len(s.functions))
		for name := range s.functions {
			keys = append(keys, name)
		}
		sort.Strings(keys)
		for _, name := range keys {
			script.WriteString("\n")
			script.WriteString(s.functions[name])
			script.WriteString("\n")
		}
	}

	// write any statements
	if len(s.statements) > 0 {
		script.WriteString("\n")
		script.WriteString(strings.Join(s.statements, "\n"))
		script.WriteString("\n")
	}

	return script.String()
}

// removeLaunchctlServiceFunc removes a launchctl service, it's a direct port
// of the homebrew implementation
// https://github.com/Homebrew/brew/blob/e1ff668957dd8a66304c0290dfa66083e6c7444e/Library/Homebrew/cask/artifact/abstract_uninstall.rb#L92
const removeLaunchctlServiceFunc = `remove_launchctl_service() {
  local service="$1"
  local booleans=("true" "false")
  local plist_status
  local paths
  local should_sudo

  echo "Removing launchctl service ${service}"

  # A wildcard label can't be used with launchctl or as a plist name, so expand
  # it to the labels of currently loaded services that match the pattern.
  local services=("$service")
  if [[ "$service" == *"*"* ]]; then
    local regex
    # Escape regex metacharacters, turn '*' into '.*', and anchor the pattern so
    # it matches a full label rather than a substring.
    regex=$(printf '%s' "$service" | sed -e 's/[][(){}.^$+?|\\]/\\&/g' -e 's/\*/.*/g')
    regex="^${regex}$"
    services=()
    local id
    # Match every loaded job by label regardless of PID; launchctl list reports
    # loaded-but-not-running jobs with a "-" in the PID column.
    while read -r _ _ id; do
      [[ "$id" =~ $regex ]] && services+=("$id")
    done < <(launchctl list 2>/dev/null | tail -n +2)
    if [[ ${#services[@]} -eq 0 ]]; then
      echo "No loaded launchctl service matches ${service}"
      return
    fi
  fi

  local service_label
  for service_label in "${services[@]}"; do
    for should_sudo in "${booleans[@]}"; do
      plist_status=$(launchctl list "${service_label}" 2>/dev/null)

      if [[ $plist_status == \{* ]]; then
        if [[ $should_sudo == "true" ]]; then
          sudo launchctl remove "${service_label}"
        else
          launchctl remove "${service_label}"
        fi
        sleep 1
      fi

      paths=(
        "/Library/LaunchAgents/${service_label}.plist"
        "/Library/LaunchDaemons/${service_label}.plist"
      )

      # if not using sudo, prepend the home directory to the paths
      if [[ $should_sudo == "false" ]]; then
        for i in "${!paths[@]}"; do
          paths[i]="${HOME}${paths[i]}"
        done
      fi

      for path in "${paths[@]}"; do
        if [[ -e "$path" ]]; then
          if [[ $should_sudo == "true" ]]; then
            sudo rm -f -- "$path"
          else
            rm -f -- "$path"
          fi
        fi
      done
    done
  done
}`

// quitApplicationFunc quits a running application. It's a direct port of the
// homebrew implementation
// https://github.com/Homebrew/brew/blob/e1ff668957dd8a66304c0290dfa66083e6c7444e/Library/Homebrew/cask/artifact/abstract_uninstall.rb#L192
const quitApplicationFunc = `quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    return
  fi

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}
`

// quitAndTrackApplicationFunc quits a running application and tracks whether it was running
// so it can be relaunched after installation. Sets APP_WAS_RUNNING_<bundle_id> environment variable.
const quitAndTrackApplicationFunc = `quit_and_track_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local timeout_duration=10

  # check if the application is running
  local app_running
  app_running=$(osascript -e "application id \"$bundle_id\" is running" 2>/dev/null)
  if [[ "$app_running" != "true" ]]; then
    eval "export $var_name=0"
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping quitting application ID '$bundle_id'."
    eval "export $var_name=0"
    return
  fi

  # App was running, mark it for relaunch
  eval "export $var_name=1"
  echo "Application '$bundle_id' was running; will relaunch after installation."

  echo "Quitting application '$bundle_id'..."

  # try to quit the application within the timeout period
  local quit_success=false
  SECONDS=0
  while (( SECONDS < timeout_duration )); do
    if osascript -e "tell application id \"$bundle_id\" to quit" >/dev/null 2>&1; then
      if ! pgrep -f "$bundle_id" >/dev/null 2>&1; then
        echo "Application '$bundle_id' quit successfully."
        quit_success=true
        break
      fi
    fi
    sleep 1
  done

  if [[ "$quit_success" = false ]]; then
    echo "Application '$bundle_id' did not quit."
  fi
}
`

// relaunchApplicationFunc relaunches an application if it was running before installation.
// Checks the APP_WAS_RUNNING_<bundle_id> environment variable set by quitAndTrackApplicationFunc.
// The install script runs as root, but GUI apps must be launched in the logged-in
// user's session (not root's) to appear in the user's Dock/GUI. We therefore use
// 'sudo -u "$console_user" open' rather than 'osascript ... to activate', since
// the latter fails when launching an app that isn't already running from a root
// context.
const relaunchApplicationFunc = `relaunch_application() {
  local bundle_id="$1"
  local var_name="APP_WAS_RUNNING_$(echo "$bundle_id" | tr '.-' '__')"
  local was_running

  # Check if the app was running before installation
  eval "was_running=\$$var_name"
  if [[ "$was_running" != "1" ]]; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ -z "$console_user" || "$console_user" == "root" || "$console_user" == "loginwindow" ]]; then
    echo "Not logged into a non-root GUI; skipping relaunching application ID '$bundle_id'."
    return
  fi

  echo "Relaunching application '$bundle_id'..."

  # Launch the app in the logged-in user's GUI session. Apps launched by root
  # won't register with the user's Dock/GUI, so run 'open' as the console user.
  # Use 'launchctl asuser' to bootstrap into the console user's Mach namespace
  # and GUI session — 'sudo -u' alone doesn't do this, which can cause
  # LSOpenURLsWithRole() failures even when 'open' exits 0.
  local open_status=0
  if [[ $EUID -eq 0 ]]; then
    local console_uid
    console_uid=$(id -u "$console_user")
    /bin/launchctl asuser "$console_uid" sudo -u "$console_user" open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  else
    open -b "$bundle_id" >/dev/null 2>&1 || open_status=$?
  fi

  if [[ $open_status -eq 0 ]]; then
    echo "Application '$bundle_id' relaunched successfully."
  else
    echo "Failed to relaunch application '$bundle_id'."
  fi
}
`

const trashFunc = `trash() {
  local logged_in_user="$1"
  local target_file="$2"
  local timestamp="$(date +%Y-%m-%d-%s)"
  local rand="$(jot -r 1 0 99999)"

  # replace ~ with /Users/$logged_in_user
  if [[ "$target_file" == ~* ]]; then
    target_file="/Users/$logged_in_user${target_file:1}"
  fi

  local trash="/Users/$logged_in_user/.Trash"

  # If the target contains glob characters, expand it and move each match.
  if [[ "$target_file" == *[*?[]* ]]; then
    local file file_name
    local matched=false
    local i=0
    # compgen -G expands the (quoted) pattern itself, so paths containing
    # spaces glob correctly; reading line by line keeps each match intact.
    while IFS= read -r file; do
      [[ -n "$file" ]] || continue
      [[ -e "$file" || -L "$file" ]] || continue
      matched=true
      i=$((i + 1))
      file_name="$(basename "$file")"
      echo "removing $file."
      # The per-match counter keeps matches that share a basename from
      # overwriting each other in the trash.
      mv -f "$file" "$trash/${file_name}_${timestamp}_${rand}_${i}"
    done < <(compgen -G "$target_file" 2>/dev/null)
    if [[ "$matched" == false ]]; then
      echo "$target_file doesn't exist."
    fi
    return
  fi

  local file_name="$(basename "${target_file}")"

  if [[ -e "$target_file" ]]; then
    echo "removing $target_file."
    mv -f "$target_file" "$trash/${file_name}_${timestamp}_${rand}"
  else
    echo "$target_file doesn't exist."
  fi
}`

const sendSignalFunc = `send_signal() {
  local signal="$1"
  local bundle_id="$2"
  local logged_in_user="$3"
  local logged_in_uid pids

  if [ -z "$signal" ] || [ -z "$bundle_id" ] || [ -z "$logged_in_user" ]; then
    echo "Usage: uninstall_signal <signal> <bundle_id> <logged_in_user>"
    return 1
  fi

  logged_in_uid=$(id -u "$logged_in_user")
  if [ -z "$logged_in_uid" ]; then
    echo "Could not find UID for user '$logged_in_user'."
    return 1
  fi

  echo "Signalling '$signal' to application ID '$bundle_id' for user '$logged_in_user'"

  pids=$(/bin/launchctl asuser "$logged_in_uid" sudo -iu "$logged_in_user" /bin/launchctl list | awk -v bundle_id="$bundle_id" '
    $3 ~ bundle_id { print $1 }')

  if [ -z "$pids" ]; then
    echo "No processes found for bundle ID '$bundle_id'."
    return 0
  fi

  echo "Unix PIDs are $pids for processes with bundle identifier $bundle_id"
  for pid in $pids; do
    if kill -s "$signal" "$pid" 2>/dev/null; then
      echo "Successfully signaled PID $pid with signal $signal."
    else
      echo "Failed to kill PID $pid with signal $signal. Check permissions."
    fi
  done

  sleep 3
}`

const expandWildcardPkgs = `expand_pkgid_and_map() {
  local PKGID="$1"
  local FUNC="$2"
  if [[ "$PKGID" == *"*" ]]; then
    local prefix="${PKGID%\*}"
    echo "Expanding wildcard for PKGID: $PKGID"
    for receipt in $(pkgutil --pkgs | grep "^${prefix}"); do
      echo "Processing $receipt"
      "$FUNC" "$receipt"
    done
  else
    "$FUNC" "$PKGID"
  fi
}`

const removePkgFiles = `remove_pkg_files() {
  local PKGID="$1"
  expand_pkgid_and_map "$PKGID" remove_receipt_files
}

remove_receipt_files() {
  local PKGID="$1"
  local PKGINFO VOLUME INSTALL_LOCATION FULL_INSTALL_LOCATION

  echo "pkgutil --pkg-info-plist \"$PKGID\""
  PKGINFO=$(pkgutil --pkg-info-plist "$PKGID")
  VOLUME=$(echo "$PKGINFO" | awk '/<key>volume<\/key>/ {getline; gsub(/.*<string>|<\/string>.*/, ""); print}')
  INSTALL_LOCATION=$(echo "$PKGINFO" | awk '/<key>install-location<\/key>/ {getline; gsub(/.*<string>|<\/string>.*/, ""); print}')

  if [ -z "$INSTALL_LOCATION" ] || [ "$INSTALL_LOCATION" = "/" ]; then
    FULL_INSTALL_LOCATION="$VOLUME"
  else
    FULL_INSTALL_LOCATION="$VOLUME/$INSTALL_LOCATION"
    FULL_INSTALL_LOCATION=$(echo "$FULL_INSTALL_LOCATION" | sed 's|//|/|g')
  fi

  echo "sudo pkgutil --only-files --files \"$PKGID\" | sed \"s|^|${FULL_INSTALL_LOCATION}/|\" | tr '\\\\n' '\\\\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf"
  sudo pkgutil --only-files --files "$PKGID" | sed "s|^|/${INSTALL_LOCATION}/|" | tr '\n' '\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf

  echo "sudo pkgutil --only-dirs --files \"$PKGID\" | sed \"s|^|${FULL_INSTALL_LOCATION}/|\" | grep '\\.app$' | tr '\\\\n' '\\\\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf"
  sudo pkgutil --only-dirs --files "$PKGID" | sed "s|^|${FULL_INSTALL_LOCATION}/|" | grep '\.app$' | tr '\n' '\0' | /usr/bin/sudo -u root -E -- /usr/bin/xargs -0 -- /bin/rm -rf

  root_app_dir=$(
    sudo pkgutil --only-dirs --files "$PKGID" \
      | sed "s|^|${FULL_INSTALL_LOCATION}/|" \
      | grep 'Applications' \
      | awk '{ print length, $0 }' \
      | sort -n \
      | head -n1 \
      | cut -d' ' -f2-
  )
  if [ -n "$root_app_dir" ]; then
    echo "sudo rmdir -p \"$root_app_dir\" 2>/dev/null || :"
    sudo rmdir -p "$root_app_dir" 2>/dev/null || :
  fi
}`

const forgetPkgFunc = `forget_pkg() {
  local PKGID="$1"
  expand_pkgid_and_map "$PKGID" forget_receipt
}

forget_receipt() {
  local PKGID="$1"
  sudo pkgutil --forget "$PKGID"
}`
