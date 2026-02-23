package homebrew

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/micromdm/plist"
)

func installScriptForApp(app inputApp, cask *brewCask) (string, error) {
	sb := newScriptBuilder()

	sb.AddVariable("TMPDIR", `$(dirname "$(realpath $INSTALLER_PATH)")`)
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
				appPath := appItem.String
				sb.Writef(`if [ -d "$APPDIR/%[1]s" ]; then
	sudo mv "$APPDIR/%[1]s" "$TMPDIR/%[1]s.bkp"
fi`, appPath)
				sb.Copy(appPath, "$APPDIR")
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
			// Remove all collected app paths
			for _, appPath := range appPathsToRemove {
				sb.RemoveFile(fmt.Sprintf(`"$APPDIR/%s"`, appPath))
			}
		case len(artifact.Binary) > 0:
			if len(artifact.Binary) == 2 {
				target := artifact.Binary[1].Other.Target
				if !strings.Contains(target, "$HOMEBREW_PREFIX") {
					sb.RemoveFile(fmt.Sprintf(`'%s'`, target))
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
		sb.Writef("remove_launchctl_service '%s'", lc)
	})

	process(u.Quit, func(appName string) {
		sb.AddFunction("quit_application", quitApplicationFunc)
		sb.Writef("quit_application '%s'", appName)
		if appName == "com.docker.docker" {
			sb.Writef("quit_application 'com.electron.dockerdesktop'")
		}
	})

	// per the spec, signals can't have a different format. In the homebrew
	// source code an error is raised when the format is different.
	if u.Signal.IsOther && len(u.Signal.Other) == 2 {
		addUserVar()
		sb.AddFunction("send_signal", sendSignalFunc)
		sb.Writef(`send_signal '%s' '%s' "$LOGGED_IN_USER"`, u.Signal.Other[0], u.Signal.Other[1])
	}

	if u.Script.IsOther {
		// for supported FMAs, this is a map with "executable" as the script path,
		// optional "args" array, optional "sudo" boolean, and optional "must_succeed" boolean
		addUserVar()
		executable, ok := u.Script.Other["executable"].(string)
		if !ok {
			panic("executable not found or not a string in script")
		}

		// Build the command with arguments if present
		var cmdParts []string
		cmdParts = append(cmdParts, fmt.Sprintf("'%s'", executable))

		// Handle args if present
		if argsVal, hasArgs := u.Script.Other["args"]; hasArgs {
			args, ok := argsVal.([]interface{})
			if !ok {
				panic("args must be an array in script")
			}
			for _, arg := range args {
				argStr, ok := arg.(string)
				if !ok {
					panic("all args must be strings")
				}
				cmdParts = append(cmdParts, fmt.Sprintf("'%s'", argStr))
			}
		}

		cmd := strings.Join(cmdParts, " ")

		// Handle must_succeed - if false, we can ignore errors
		mustSucceed := true
		if mustSucceedVal, hasMustSucceed := u.Script.Other["must_succeed"]; hasMustSucceed {
			if ms, ok := mustSucceedVal.(bool); ok {
				mustSucceed = ms
			}
		}

		// Handle sudo - check if sudo is required (defaults to false if not specified)
		needsSudo := false
		if sudoVal, hasSudo := u.Script.Other["sudo"]; hasSudo {
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
	} else if len(u.Script.String) > 0 {
		addUserVar()
		sb.Writef(`(cd /Users/$LOGGED_IN_USER && sudo -u "$LOGGED_IN_USER" '%s')`, u.Script.String)
	}

	process(u.PkgUtil, func(pkgID string) {
		sb.AddFunction("expand_pkgid_and_map", expandWildcardPkgs)
		sb.AddFunction("remove_pkg_files", removePkgFiles)
		sb.AddFunction("forget_pkg", forgetPkgFunc)
		sb.Writef("remove_pkg_files '%s'", pkgID)
		sb.Writef("forget_pkg '%s'", pkgID)
	})

	process(u.Delete, func(path string) {
		sb.RemoveFile(fmt.Sprintf("'%s'", path))
	})

	process(u.RmDir, func(dir string) {
		sb.Writef("sudo rmdir '%s'", dir)
	})

	process(u.Trash, func(path string) {
		addUserVar()
		sb.AddFunction("trash", trashFunc)
		sb.Writef("trash $LOGGED_IN_USER '%s'", path)
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
		s.Write(`MOUNT_POINT=$(mktemp -d /tmp/dmg_mount_XXXXXX)
hdiutil attach -plist -nobrowse -readonly -mountpoint "$MOUNT_POINT" "$INSTALLER_PATH"
sudo cp -R "$MOUNT_POINT"/* "$TMPDIR"
hdiutil detach "$MOUNT_POINT"`)

	case "zip":
		s.Write("# extract contents")
		s.Write(`unzip "$INSTALLER_PATH" -d "$TMPDIR"`)
	}
}

// Copy writes a command to copy a file from the temporary directory to a
// destination.
func (s *scriptBuilder) Copy(file, dest string) {
	s.Writef(`sudo cp -R "$TMPDIR/%s" "%s"`, file, dest)
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
	if len(choices) == 0 {
		s.Writef(`sudo installer -pkg "$TMPDIR/%s" -target /`, pkg)
		return nil
	}

	choiceXML, err := plist.MarshalIndent(choices[0], "  ")
	if err != nil {
		return err
	}

	s.Writef(`
CHOICE_XML=$(mktemp /tmp/choice_xml_XXX)

cat << EOF > "$CHOICE_XML"
%s
EOF

sudo installer -pkg "$TMPDIR"/%s -target / -applyChoiceChangesXML "$CHOICE_XML"
`, choiceXML, pkg)

	return nil
}

// Symlink writes a command to create a symbolic link from 'source' to 'target'.
func (s *scriptBuilder) Symlink(source, target string) {
	pathname := filepath.Dir(target)
	if _, ok := s.pathsCreated[pathname]; !ok {
		s.Writef("mkdir -p %s", pathname)
		s.pathsCreated[pathname] = struct{}{}
	}
	s.Writef(`/bin/ln -h -f -s -- "%s" "%s"`, source, target)
}

// String generates the final script as a string.
//
// It includes the shebang, any variables, functions, and statements in the
// correct order.
func (s *scriptBuilder) String() string {
	var script strings.Builder
	script.WriteString("#!/bin/sh\n\n")

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

  for should_sudo in "${booleans[@]}"; do
    plist_status=$(launchctl list "${service}" 2>/dev/null)

    if [[ $plist_status == \{* ]]; then
      if [[ $should_sudo == "true" ]]; then
        sudo launchctl remove "${service}"
      else
        launchctl remove "${service}"
      fi
      sleep 1
    fi

    paths=(
      "/Library/LaunchAgents/${service}.plist"
      "/Library/LaunchDaemons/${service}.plist"
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
}`

// quitApplicationFunc quits a running application. It's a direct port of the
// homebrew implementation
// https://github.com/Homebrew/brew/blob/e1ff668957dd8a66304c0290dfa66083e6c7444e/Library/Homebrew/cask/artifact/abstract_uninstall.rb#L192
const quitApplicationFunc = `quit_application() {
  local bundle_id="$1"
  local timeout_duration=10

  # check if the application is running
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
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
  if ! osascript -e "application id \"$bundle_id\" is running" 2>/dev/null; then
    eval "export $var_name=0"
    return
  fi

  local console_user
  console_user=$(stat -f "%Su" /dev/console)
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
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
  if [[ $EUID -eq 0 && "$console_user" == "root" ]]; then
    echo "Not logged into a non-root GUI; skipping relaunching application ID '$bundle_id'."
    return
  fi

  echo "Relaunching application '$bundle_id'..."

  # Try to launch the application
  if osascript -e "tell application id \"$bundle_id\" to activate" >/dev/null 2>&1; then
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
