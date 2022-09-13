package execuser

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// run uses sudo to run the given path as login user.
func run(path string, opts eopts) error {
	user, err := getLoginUID()
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	log.Info().
		Str("user", user.name).
		Int64("id", user.id).
		Msg("running sudo")

	// Flag `-i` is needed to run the command with the user's context, from `man sudo`:
	// "The command is run with an environment similar to the one a user would receive at log in"
	arg := []string{"-i", "-u", user.name, "-H"}
	for _, nv := range opts.env {
		arg = append(arg, fmt.Sprintf("%s=%s", nv[0], nv[1]))
	}

	arg = append(arg,
		// TODO(lucas): Default to display 0, revisit when working on
		// multi-user/multi-session support. This assumes there's only
		// one desktop session and belongs to the user returned in `getLoginUID'.
		"DISPLAY=:0",
		// DBUS_SESSION_BUS_ADDRESS sets the location of the user login session bus.
		// Required by the libayatana-appindicator3 library to display a tray icon
		// on the desktop session.
		//
		// This is required for Ubuntu 18, and not required for Ubuntu 21/22
		// (because it's already part of the user).
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/%d/bus", user.id),
		// Append the packaged libayatana-appindicator3 libraries path to LD_LIBRARY_PATH.
		fmt.Sprintf("LD_LIBRARY_PATH=%s:%s", filepath.Dir(path), os.ExpandEnv("$LD_LIBRARY_PATH")),
		path,
	)

	cmd := exec.Command("sudo", arg...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	log.Printf("cmd=%s", cmd.String())

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open path %q: %w", path, err)
	}
	return nil
}

type user struct {
	name string
	id   int64
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
	for _, userCol := range strings.Split(strings.TrimSpace(s), " ") {
		users = append(users, userCol)
	}
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
