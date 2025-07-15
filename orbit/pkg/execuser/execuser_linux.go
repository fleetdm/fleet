package execuser

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	userpkg "github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/rs/zerolog/log"
)

// run uses sudo to run the given path as login user.
func baserun(path string, opts eopts) (cmd *exec.Cmd, err error) {
	args, env, userContext, err := getUserAndDisplayArgs(path)
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

	// If the user has an SELinux context, we run the command with runcon so that it has the correct context.
	// Otherwise, we run the command directly.
	if userContext != nil {
		args = append(args, "-c", fmt.Sprintf("runcon \"%s\" env %s %s %s", *userContext, strings.Join(env, " "), path, strings.Join(cmdArgs, " ")))
	} else {
		args = append(args, "-c", fmt.Sprintf("%s %s %s", strings.Join(env, " "), path, strings.Join(cmdArgs, " ")))
	}

	// Use runuser to run the command as the login user.
	args = append([]string{"runuser"}, args...)

	// If a timeout is set, prefix the command with "timeout".
	if opts.timeout > 0 {
		args = append([]string{"timeout", fmt.Sprintf("%ds", int(opts.timeout.Seconds()))}, args...)
	}

	cmd = exec.Command(args[0], args[1:]...) // #nosec G204
	return
}

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

// run uses sudo to run the given path as login user and waits for the process to finish.
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

func runWithStdin(path string, opts eopts) (io.WriteCloser, error) {
	cmd, err := baserun(path, opts)
	if err != nil {
		return nil, err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("open path %q: %w", path, err)
	}

	return stdin, nil
}

func getUserAndDisplayArgs(path string) ([]string, []string, *string, error) {
	user, err := userpkg.GetLoginUser()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get user: %w", err)
	}

	log.Info().Str("user", user.Name).Int64("id", user.ID).Msg("attempting to get user session type and display")

	// Get user's display session type (x11 vs. wayland).
	uid := strconv.FormatInt(user.ID, 10)
	userDisplaySessionType, err := userpkg.GetUserDisplaySessionType(uid)
	if userDisplaySessionType == userpkg.GuiSessionTypeTty {
		return nil, nil, nil, fmt.Errorf("user %q (%d) is not running a GUI session", user.Name, user.ID)
	}
	if err != nil {
		// Wayland is the default for most distributions, thus we assume
		// wayland if we couldn't determine the session type.
		log.Error().Err(err).Msg("assuming wayland session")
		userDisplaySessionType = userpkg.GuiSessionTypeWayland
	}

	var display string
	if userDisplaySessionType == userpkg.GuiSessionTypeX11 {
		x11Display, err := getUserX11Display(user.Name)
		if err != nil {
			log.Error().Err(err).Msg("failed to get X11 display, using default :0")
			// TODO(lucas): Revisit when working on multi-user/multi-session support.
			// Default to display ':0' if user display could not be found.
			// This assumes there's only one desktop session and belongs to the
			// user returned in `getLoginUID'.
			display = ":0"
		} else {
			display = x11Display
		}
	} else {
		waylandDisplay, err := getUserWaylandDisplay(uid)
		if err != nil {
			log.Error().Err(err).Msg("failed to get wayland display, using default wayland-0")
			// TODO(lucas): Revisit when working on multi-user/multi-session support.
			// Default to display 'wayland-0' if user display could not be found.
			// This assumes there's only one desktop session and belongs to the
			// user returned in `getLoginUID'.
			display = "wayland-0"
		} else {
			display = waylandDisplay
		}
	}

	log.Info().
		Str("path", path).
		Str("user", user.Name).
		Int64("id", user.ID).
		Str("display", display).
		Str("session_type", userDisplaySessionType.String()).
		Msg("running sudo")

	args := []string{"-l", user.Name}
	env := make([]string, 0)

	if userDisplaySessionType == userpkg.GuiSessionTypeWayland {
		args = append(args, "WAYLAND_DISPLAY="+display)
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
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%d/bus", user.ID),
	)

	// Get the user's SELinux context (if any).
	context := userpkg.GetUserContext(user)

	return args, env, context, nil
}

var whoLineRegexp = regexp.MustCompile(`(\w+)\s+(:\d+)\s+`)

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
func getUserX11Display(user string) (string, error) {
	cmd := exec.Command("who")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run 'who' to get user display: %w", err)
	}
	return parseWhoOutputForDisplay(&stdout, user)
}

func parseWhoOutputForDisplay(output io.Reader, user string) (string, error) {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := scanner.Text()
		matches := whoLineRegexp.FindStringSubmatch(line)
		if len(matches) > 1 && matches[1] == user {
			return matches[2], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scanner error: %w", err)
	}
	return "", errors.New("display not found on who output")
}
