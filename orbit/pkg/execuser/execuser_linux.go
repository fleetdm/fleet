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
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// run uses sudo to run the given path as login user.
func run(path string, opts eopts) (lastLogs string, err error) {
	args, err := getUserAndDisplayArgs(path, opts)
	if err != nil {
		return "", fmt.Errorf("get args: %w", err)
	}

	args = append(args,
		// Append the packaged libayatana-appindicator3 libraries path to LD_LIBRARY_PATH.
		//
		// Fleet Desktop doesn't use libayatana-appindicator3 since 1.18.3, but we need to
		// keep this to support older versions of Fleet Desktop.
		fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", filepath.Dir(path), os.ExpandEnv("$LD_LIBRARY_PATH")),
		path,
	)

	if len(opts.args) > 0 {
		for _, arg := range opts.args {
			args = append(args, arg[0], arg[1])
		}
	}

	cmd := exec.Command("sudo", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("cmd=%s", cmd.String())

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("open path %q: %w", path, err)
	}
	return "", nil
}

// run uses sudo to run the given path as login user and waits for the process to finish.
func runWithOutput(path string, opts eopts) (output []byte, exitCode int, err error) {
	args, err := getUserAndDisplayArgs(path, opts)
	if err != nil {
		return nil, -1, fmt.Errorf("get args: %w", err)
	}

	args = append(args, path)

	if len(opts.args) > 0 {
		for _, arg := range opts.args {
			args = append(args, arg[0], arg[1])
		}
	}

	// Prefix with "timeout" and "sudo" if applicable
	var cmdArgs []string
	if opts.timeout > 0 {
		cmdArgs = append(cmdArgs, "timeout", fmt.Sprintf("%ds", int(opts.timeout.Seconds())))
	}
	cmdArgs = append(cmdArgs, "sudo")
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...) // #nosec G204

	log.Printf("cmd=%s", cmd.String())

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
	args, err := getUserAndDisplayArgs(path, opts)
	if err != nil {
		return nil, fmt.Errorf("get args: %w", err)
	}

	args = append(args, path)

	if len(opts.args) > 0 {
		for _, arg := range opts.args {
			args = append(args, arg[0], arg[1])
		}
	}

	cmd := exec.Command("sudo", args...)
	log.Printf("cmd=%s", cmd.String())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("open path %q: %w", path, err)
	}

	return stdin, nil
}

func getUserAndDisplayArgs(path string, opts eopts) ([]string, error) {
	user, err := getLoginUID()
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	// TODO(lucas): Default to display :0 if user DISPLAY environment variable
	// could not be found, revisit when working on multi-user/multi-session support.
	// This assumes there's only one desktop session and belongs to the
	// user returned in `getLoginUID'.
	defaultDisplay := ":0"

	log.Info().
		Str("user", user.name).
		Int64("id", user.id).
		Msg("attempting to get user's DISPLAY")

	display, err := getUserDisplay(user.name, opts)
	if err != nil {
		log.Error().
			Str("user", user.name).
			Int64("id", user.id).
			Err(err).
			Msgf("failed to get user's DISPLAY, using default %s", defaultDisplay)
		display = defaultDisplay
	} else if display == "" {
		log.Warn().
			Str("user", user.name).
			Int64("id", user.id).
			Msgf("user's DISPLAY not found, using default %s", defaultDisplay)
		display = defaultDisplay
	}

	log.Info().
		Str("path", path).
		Str("user", user.name).
		Int64("id", user.id).
		Str("display", display).
		Msg("running sudo")

	args := argsForSudo(user, opts)

	args = append(args,
		"DISPLAY="+display,
		// DBUS_SESSION_BUS_ADDRESS sets the location of the user login session bus.
		// Required by the libayatana-appindicator3 library to display a tray icon
		// on the desktop session.
		//
		// This is required for Ubuntu 18, and not required for Ubuntu 21/22
		// (because it's already part of the user).
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%d/bus", user.id),
	)

	return args, nil
}

type user struct {
	name string
	id   int64
}

func argsForSudo(u *user, opts eopts) []string {
	// -H: "[...] to set HOME environment to what's specified in the target's user password database entry."
	// -i: needed to run the command with the user's context, from `man sudo`:
	// "The command is run with an environment similar to the one a user would receive at log in"
	// -u: "[..]Run the command as a user other than the default target user (usually root)."
	args := []string{"-i", "-u", u.name, "-H"}
	for _, nv := range opts.env {
		args = append(args, fmt.Sprintf("%s=%s", nv[0], nv[1]))
	}
	return args
}

// getLoginUID returns the name and uid of the first login user
// as reported by the `users' command.
//
// NOTE(lucas): It is always picking first login user as returned
// by `users', revisit when working on multi-user/multi-session support.
func getLoginUID() (*user, error) {
	out, err := exec.Command("users").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("users exec failed: %w", err)
	}
	usernames := parseUsersOutput(string(out))
	username := usernames[0]
	if username == "" {
		return nil, errors.New("no user session found")
	}
	out, err = exec.Command("id", "-u", username).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("id exec failed: %w", err)
	}
	uid, err := parseIDOutput(string(out))
	if err != nil {
		return nil, err
	}
	return &user{
		name: username,
		id:   uid,
	}, nil
}

// parseUsersOutput parses the output of the `users' command.
//
//	`users' command prints on a single line a blank-separated list of user names of
//	users currently logged in to the current host. Each user name
//	corresponds to a login session, so if a user has more than one login
//	session, that user's name will appear the same number of times in the
//	output.
//
// Returns the list of usernames.
func parseUsersOutput(s string) []string {
	var users []string
	users = append(users, strings.Split(strings.TrimSpace(s), " ")...)
	return users
}

// parseIDOutput parses the output of the `id' command.
//
// Returns the parsed uid.
func parseIDOutput(s string) (int64, error) {
	uid, err := strconv.ParseInt(strings.TrimSpace(s), 10, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to parse uid: %w", err)
	}
	return uid, nil
}

var whoLineRegexp = regexp.MustCompile(`(\w+)\s+(:\d+)\s+`)

func getUserDisplay(user string, opts eopts) (string, error) {
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
	return "", nil
}
