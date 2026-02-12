package execuser

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

	args, env, err := getConfigForCommand(opts.user, path)
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

func getConfigForCommand(user string, path string) (args []string, env []string, err error) {
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

	args = []string{"-n", "-i", "-u", user, "-H"}
	env = make([]string, 0)

	if userDisplaySession.Type == userpkg.GuiSessionTypeWayland {
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

	return args, env, nil
}

// getUserWaylandDisplay returns the value to set on WAYLAND_DISPLAY for the given user.
func getUserWaylandDisplay(uid string) (string, error) {
	matches, err := filepath.Glob("/run/user/" + uid + "/wayland-*")
	if err != nil {
		return "", fmt.Errorf("list wayland socket files: %w", err)
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i] < matches[j]
	})
	for _, match := range matches {
		if strings.HasSuffix(match, ".lock") {
			continue
		}
		return filepath.Base(match), nil
	}
	return "", errors.New("wayland socket not found")
}

// getUserX11Display returns the value to set on DISPLAY for the given user.
// It scans /proc to find a process owned by the user that has DISPLAY set
// in its environment.
func getUserX11Display(userID string) (string, error) {
	uid, err := strconv.ParseUint(userID, 10, 32)
	if err != nil {
		return "", fmt.Errorf("parse user ID %q: %w", userID, err)
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return "", fmt.Errorf("read /proc: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip non-PID directories.
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		// Check if the process belongs to our target user.
		info, err := entry.Info()
		if err != nil {
			continue
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || stat.Uid != uint32(uid) {
			continue
		}

		// Try to read DISPLAY from this process's environment.
		display, err := readEnvFromProc(entry.Name(), "DISPLAY")
		if err != nil || display == "" {
			continue
		}

		log.Debug().Msgf("found DISPLAY variable in %q", entry.Name())
		return display, nil
	}

	return "", fmt.Errorf("DISPLAY not found in any process for user %s", userID)
}

// readEnvFromProc reads a specific environment variable from /proc/<pid>/environ.
func readEnvFromProc(pid string, envVar string) (string, error) {
	return readEnvFromProcFile(fmt.Sprintf("/proc/%s/environ", pid), envVar)
}

// readEnvFromProcFile reads a specific environment variable from a /proc environ file.
// The file contains null-byte separated KEY=VALUE entries.
func readEnvFromProcFile(path string, envVar string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	prefix := envVar + "="
	for _, entry := range bytes.Split(data, []byte{0}) {
		if s := string(entry); strings.HasPrefix(s, prefix) {
			return s[len(prefix):], nil
		}
	}
	return "", nil
}
