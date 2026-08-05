package execuser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"

	userpkg "github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/rs/zerolog/log"
)

// base command to setup an exec.Cmd using `runuser`
func baserun(path string, opts eopts) (cmd *exec.Cmd, err error) {
	if opts.user == "" {
		return nil, errors.New("missing user")
	}

	args, env, err := getConfigForCommand(path, opts)
	if err != nil {
		return nil, fmt.Errorf("get args: %w", err)
	}

	env = append(env,
		// Append the packaged libayatana-appindicator3 libraries path to LD_LIBRARY_PATH.
		//
		// Fleet Desktop doesn't use libayatana-appindicator3 since 1.18.3, but we need to
		// keep this to support older versions of Fleet Desktop.
		fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", filepath.Dir(path), os.ExpandEnv("$LD_LIBRARY_PATH")),
	)

	for _, nv := range opts.env {
		env = append(env, fmt.Sprintf("%s=%s", nv[0], nv[1]))
	}

	// Hold any command line arguments to pass to the command.
	cmdArgs := make([]string, 0, len(opts.args)*2)
	if len(opts.args) > 0 {
		for _, arg := range opts.args {
			cmdArgs = append(cmdArgs, arg[0])
			if arg[1] != "" {
				cmdArgs = append(cmdArgs, arg[1])
			}
		}
	}

	// Run `env` to setup the environment.
	args = append(args, "env")
	args = append(args, env...)
	// Pass the command and its arguments.
	args = append(args, path)
	args = append(args, cmdArgs...)

	// Use sudo to run the command as the login user.
	args = append([]string{"sudo"}, args...)

	// If a timeout is set, prefix the command with "timeout".
	if opts.timeout > 0 {
		args = append([]string{"timeout", fmt.Sprintf("%ds", int(opts.timeout.Seconds()))}, args...)
	}

	cmd = exec.Command(args[0], args[1:]...) // #nosec G204
	return
}

// run a command, passing its output to stdout and stderr.
func run(path string, opts eopts) (lastLogs string, err error) {
	cmd, err := baserun(path, opts)
	if err != nil {
		return "", err
	}

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Info().Str("cmd", cmd.String()).Msg("running command")

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("open path %q: %w", path, err)
	}
	// Reap the child process in a background goroutine. Orbit runs as root and
	// only calls Start here (it monitors the desktop process separately, so this
	// function must return after starting it). Without a corresponding Wait, every
	// `sudo`/`timeout` child that exits becomes a zombie. When the desktop fails to
	// start and Orbit respawns it in a loop, these zombies accumulate by the
	// thousands. See https://github.com/fleetdm/fleet/issues/41796.
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Debug().Err(err).Msg("run cmd wait")
		}
	}()
	return "", nil
}

// runWithOutput runs a command and return its output and exit code.
func runWithOutput(path string, opts eopts) (output []byte, exitCode int, err error) {
	cmd, err := baserun(path, opts)
	if err != nil {
		return nil, -1, err
	}

	output, err = cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			return output, exitCode, fmt.Errorf("%q exited with code %d: %w", path, exitCode, err)
		}
		return output, -1, fmt.Errorf("%q error: %w", path, err)
	}

	return output, exitCode, nil
}

func getUserID(user string) (string, error) {
	uid_, err := exec.Command("id", "-u", user).Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute id command for %q: %w", user, err)
	}
	uid := strings.TrimSpace(string(uid_))
	if uid == "" {
		return "", errors.New("failed to get uid")
	}
	return uid, nil
}

func getDisplayVariableForSession(userID string, displaySessionType userpkg.GuiSessionType) string {
	if displaySessionType == userpkg.GuiSessionTypeX11 {
		x11Display, err := getUserX11Display(userID)
		if err != nil {
			log.Error().Err(err).Msg("failed to get X11 display, using default :0")
			// TODO(lucas): Revisit when working on multi-user/multi-session support.
			// Default to display ':0' if user display could not be found.
			// This assumes there's only one desktop session and belongs to the
			// user returned in `getLoginUID'.
			return ":0"
		}
		return x11Display
	}

	waylandDisplay, err := getUserWaylandDisplay(userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get wayland display, using default wayland-0")
		// TODO(lucas): Revisit when working on multi-user/multi-session support.
		// Default to display 'wayland-0' if user display could not be found.
		// This assumes there's only one desktop session and belongs to the
		// user returned in `getLoginUID'.
		return "wayland-0"
	}
	return waylandDisplay
}

// sudoArgs builds the sudo arguments used to run a command as the login user.
//
// Without a login shell, sudo runs the command directly, so nothing from
// /etc/profile, /etc/profile.d/*, ~/.profile or ~/.bash_logout can write to the
// command's stdout. That matters for callers that read the output back: shell
// startup output is indistinguishable from the command's own, and prepending it
// to, say, a passphrase read from a dialog silently corrupts the value.
//
// -H is set either way so HOME points at the target user; sudo's default
// env_reset already sets USER/LOGNAME/SHELL.
func sudoArgs(user string, noLoginShell bool) []string {
	if noLoginShell {
		return []string{"-n", "-u", user, "-H"}
	}
	return []string{"-n", "-i", "-u", user, "-H"}
}

func getConfigForCommand(path string, opts eopts) (args []string, env []string, err error) {
	user := opts.user

	// Get user ID
	userID, err := getUserID(user)
	if err != nil {
		return nil, nil, fmt.Errorf("get user ID: %w", err)
	}
	log.Info().Str("user", user).Str("id", userID).Msg("attempting to get user session type and display")

	// Get user's display session type.
	userDisplaySession, err := userpkg.GetUserDisplaySessionType(userID)
	if err != nil {
		// Wayland is the default for most distributions,
		// thus we assume wayland if we couldn't determine the session type.
		log.Error().Err(err).Msg("assuming wayland session")
		userDisplaySession = &userpkg.UserDisplaySession{
			Type: userpkg.GuiSessionTypeWayland,
		}
	} else if userDisplaySession.Type == userpkg.GuiSessionTypeTty {
		return nil, nil, fmt.Errorf("user %q (%s) is not running a GUI session", user, userID)
	}

	// Get user's "display" variable for the GUI session.
	display := getDisplayVariableForSession(userID, userDisplaySession.Type)

	log.Info().
		Str("path", path).
		Str("user", user).
		Str("id", userID).
		Str("display", display).
		Str("session_type", userDisplaySession.Type.String()).
		Msg("running sudo")

	// On openSUSE Leap 16+ we always drop the login shell. With -i, sudo runs the
	// target user's shell as a login shell and passes the rest of the command via
	// `bash --login -c`, which sources /etc/profile and /etc/profile.d/* and
	// shell-escapes the inline command. On Leap 16 that environment indirection
	// causes our `env KEY=val ... fleet-desktop` invocation to lose env vars, so
	// fleet-desktop exits with "missing URL environment ..." and Orbit respawns it
	// in a tight loop.
	//
	// We keep -i on every other supported distribution to preserve the previously
	// QA'd behavior, except when the command's output is read back.
	//
	// Whatever the reason for dropping it, the session environment it would have
	// supplied has to be replaced, so both decisions follow the same condition.
	noLoginShell := opts.noLoginShell || isOpenSUSELeap16Plus()
	args = sudoArgs(user, noLoginShell)
	env = make([]string, 0)

	// The variable identifying the session's display, used below to recognize the
	// processes that belong to it.
	displayVar := "DISPLAY"
	if userDisplaySession.Type == userpkg.GuiSessionTypeWayland {
		displayVar = "WAYLAND_DISPLAY"
		env = append(env, "WAYLAND_DISPLAY="+display)
		// For xdg-open to work on a Wayland session we still need to set the DISPLAY variable.
		x11Display := ":" + strings.TrimPrefix(display, "wayland-")
		env = append(env, "DISPLAY="+x11Display)
	} else {
		env = append(env, "DISPLAY="+display)
	}

	env = append(env,
		// DBUS_SESSION_BUS_ADDRESS sets the location of the user login session bus.
		// Required by the libayatana-appindicator3 library to display a tray icon
		// on the desktop session.
		//
		// This is required for Ubuntu 18, and not required for Ubuntu 21/22
		// (because it's already part of the user).
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%s/bus", userID),
	)

	if noLoginShell {
		// Without a login shell nothing carries the user's session environment
		// over, and the GUI programs we launch need parts of it, so take those
		// from the session itself. See sessionEnvPrefixes.
		env = append(env, sessionEnvForChild(userID, displayVar, display)...)
	}

	return args, env, nil
}

// isOpenSUSELeap16Plus reports whether the host is running openSUSE Leap 16 or
// newer. We scope the no-login-shell sudo workaround to that distribution since
// it is the one observed to break under sudo -i; other distributions retain the
// previous (login-shell) launch path so we don't have to re-QA them.
func isOpenSUSELeap16Plus() bool {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	var id, versionID string
	for line := range strings.SplitSeq(string(data), "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		// /etc/os-release values may be quoted.
		value = strings.Trim(value, `"'`)
		switch key {
		case "ID":
			id = value
		case "VERSION_ID":
			versionID = value
		}
	}
	if id != "opensuse-leap" {
		return false
	}
	// VERSION_ID is typically "16" or "16.0"; compare the major component.
	major, _, _ := strings.Cut(versionID, ".")
	n, err := strconv.Atoi(major)
	if err != nil {
		return false
	}
	return n >= 16
}

// getUserWaylandDisplay returns the value to set on WAYLAND_DISPLAY for the given user.
func getUserWaylandDisplay(uid string) (string, error) {
	matches, err := filepath.Glob("/run/user/" + uid + "/wayland-*")
	if err != nil {
		return "", fmt.Errorf("list wayland socket files: %w", err)
	}
	slices.Sort(matches)
	for _, match := range matches {
		if strings.HasSuffix(match, ".lock") {
			continue
		}
		return filepath.Base(match), nil
	}
	return "", errors.New("wayland socket not found")
}

// xdgDataDirsDefault is the value from the XDG base directory specification,
// used when the user's session doesn't expose one.
const xdgDataDirsDefault = "/usr/local/share:/usr/share"

// sessionEnvPrefixes and sessionEnvNames select the variables carried over from
// the user's graphical session to a child we launch into it.
//
// A login shell would have set these from the user's startup files. Commands
// whose output we read don't get one (see getConfigForCommand), so we take the
// session's own values rather than hardcoding them: GTK needs XDG_DATA_DIRS to
// find its GSettings schemas and icon themes, and the locale and input-method
// variables decide which characters an entry dialog actually receives.
//
// This is an allowlist rather than a denylist so shell bookkeeping (PWD, SHLVL,
// _), sudo's own SUDO_* variables and the LD_* loader controls are never carried
// over. DISPLAY, WAYLAND_DISPLAY, DBUS_SESSION_BUS_ADDRESS, HOME and PATH are
// deliberately absent too: orbit, sudo -H and sudoers secure_path set those, and
// the session's values must not override them.
//
// sessionEnvExcluded carves names back out of the prefixes. GTK_MODULES makes
// GTK dlopen the modules it names, and we would rather not widen what gets
// loaded into a process that handles the end user's passphrase. Nothing we
// launch needs it: a missing module is a warning, not a failure.
var (
	sessionEnvPrefixes = []string{"XDG_", "GTK_", "QT_", "GDK_", "LC_"}
	sessionEnvNames    = []string{"LANG", "LANGUAGE", "XAUTHORITY", "XMODIFIERS"}
	sessionEnvExcluded = []string{"GTK_MODULES"}
)

// sessionEnvForChild returns the environment to add for a child launched into
// the given user's graphical session, as sorted KEY=VALUE entries.
//
// displayVar and display identify the session's display, so the environment is
// taken from a process actually on it.
func sessionEnvForChild(userID, displayVar, display string) []string {
	environ, err := getUserSessionEnv(userID, displayVar, display)
	if err != nil {
		log.Debug().Err(err).Msg("no graphical session environment found for user, using defaults")
		return withXDGDataDirs(nil)
	}
	return withXDGDataDirs(filterSessionEnv(environ))
}

// withXDGDataDirs guarantees XDG_DATA_DIRS is present with a value, since the
// GUI programs we launch do not open without it.
func withXDGDataDirs(env []string) []string {
	const prefix = "XDG_DATA_DIRS="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) && kv != prefix {
			return env
		}
	}
	return append(env, prefix+xdgDataDirsDefault)
}

// filterSessionEnv reduces a process environment to the variables a GUI child
// should inherit from the session, as sorted KEY=VALUE entries.
//
// Variables with an empty value are dropped: they carry no information, and an
// empty XDG_DATA_DIRS would suppress the default.
func filterSessionEnv(environ map[string]string) []string {
	var env []string
	for key, value := range environ {
		if value == "" || slices.Contains(sessionEnvExcluded, key) {
			continue
		}
		carry := slices.Contains(sessionEnvNames, key) ||
			slices.ContainsFunc(sessionEnvPrefixes, func(p string) bool {
				return strings.HasPrefix(key, p)
			})
		if carry {
			env = append(env, key+"="+value)
		}
	}
	slices.Sort(env)
	return env
}

// getUserSessionEnv returns the full environment of a process owned by the given
// user that is running on the given display.
//
// Reading a single process keeps the result coherent, and matching on the display
// keeps unrelated environments out: a user with an `ssh -X` session open has
// processes carrying a forwarded DISPLAY and an XAUTHORITY for it, which would
// misconfigure a program we launch on the local session.
func getUserSessionEnv(userID, displayVar, display string) (map[string]string, error) {
	pids, err := userProcPIDs(userID)
	if err != nil {
		return nil, err
	}

	for _, pid := range pids {
		environ, err := readAllEnvFromProcFile(procEnvironPath(pid))
		if err != nil {
			continue
		}
		if environ[displayVar] != display {
			continue
		}
		log.Debug().Msgf("found graphical session environment in %q", pid)
		return environ, nil
	}

	return nil, fmt.Errorf("no process on %s=%s found for user %s", displayVar, display, userID)
}

// getUserX11Display returns the value to set on DISPLAY for the given user.
func getUserX11Display(userID string) (string, error) {
	return getUserEnvFromProc(userID, "DISPLAY")
}

// getUserEnvFromProc scans the given user's processes for one that has envVar
// set in its environment, and returns its value.
func getUserEnvFromProc(userID string, envVar string) (string, error) {
	pids, err := userProcPIDs(userID)
	if err != nil {
		return "", err
	}

	for _, pid := range pids {
		value, err := readEnvFromProc(pid, envVar)
		if err != nil || value == "" {
			continue
		}
		log.Debug().Msgf("found %s variable in %q", envVar, pid)
		return value, nil
	}

	return "", fmt.Errorf("%s not found in any process for user %s", envVar, userID)
}

// userProcPIDs returns the PIDs of the processes owned by the given user.
func userProcPIDs(userID string) ([]string, error) {
	uid, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("parse user ID %q: %w", userID, err)
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	var pids []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip non-PID directories.
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(uid) {
			continue
		}
		pids = append(pids, entry.Name())
	}

	return pids, nil
}

// procEnvironPath returns the path of a process's environ file.
func procEnvironPath(pid string) string {
	return fmt.Sprintf("/proc/%s/environ", pid)
}

// readEnvFromProc reads a specific environment variable from /proc/<pid>/environ.
func readEnvFromProc(pid string, envVar string) (string, error) {
	return readEnvFromProcFile(procEnvironPath(pid), envVar)
}

// readAllEnvFromProcFile reads every environment variable from a /proc environ
// file. The file contains null-byte separated KEY=VALUE entries.
//
// The first occurrence of a name wins, matching getenv, so a duplicated name
// resolves the same way it does for the process itself.
func readAllEnvFromProcFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	environ := make(map[string]string)
	for entry := range bytes.SplitSeq(data, []byte{0}) {
		key, value, ok := strings.Cut(string(entry), "=")
		if !ok {
			continue
		}
		if _, seen := environ[key]; !seen {
			environ[key] = value
		}
	}
	return environ, nil
}

// readEnvFromProcFile reads a specific environment variable from a /proc environ
// file, returning an empty value when it is not set.
func readEnvFromProcFile(path string, envVar string) (string, error) {
	environ, err := readAllEnvFromProcFile(path)
	if err != nil {
		return "", err
	}
	return environ[envVar], nil
}
