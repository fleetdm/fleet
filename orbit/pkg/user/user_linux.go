//go:build linux
// +build linux

package user

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type User struct {
	Name string
	ID   int64
}

func UserLoggedInViaGui() (*string, error) {
	user, err := GetLoginUser()
	if err != nil {
		return nil, fmt.Errorf("get login user: %w", err)
	}
	// Bail out if the user is gdm or root, since they aren't GUI users.
	if user.Name == "gdm" || user.Name == "root" {
		return nil, nil
	}
	// Check if the user has a GUI session.
	displaySessionType, err := GetUserDisplaySessionType(strconv.FormatInt(user.ID, 10))
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get user display session type for user %s", user.Name)
		return nil, nil
	}
	if displaySessionType == GuiSessionTypeTty {
		log.Debug().Msgf("user %s is logged in via TTY, not GUI", user.Name)
		return nil, nil
	}

	return &user.Name, nil
}

// GetLoginUser returns the name and uid of the first login user
// as reported by the `users' command.
//
// NOTE(lucas): It is always picking first login user as returned
// by `users', revisit when working on multi-user/multi-session support.
func GetLoginUser() (*User, error) {
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
	return &User{
		Name: username,
		ID:   uid,
	}, nil
}

func GetUserContext(user *User) *string {
	out, err := exec.Command("runuser", "-l", user.Name, "id -Z").CombinedOutput()
	log.Info().Msgf("`id -Z` output: %s", string(out))
	if err != nil {
		// If SELinux is not enabled, the command will fail with a non-zero exit code.
		// We'll check for the conmon error message and log if we don't find it.
		if strings.Contains(string(out), "SELinux-enabled") {
			log.Debug().Msgf("Unexpected output from `id -Z`: %s", string(out))
		}
		return nil
	}
	context := strings.TrimSpace(string(out))
	if context == "" {
		log.Debug().Msg("Unexpected empty output from `id -Z`")
		return nil
	}
	return &context
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

// getUserDisplaySessionType returns the display session type (X11 or Wayland) of the given user.
func GetUserDisplaySessionType(uid string) (guiSessionType, error) {
	cmd := exec.Command("loginctl", "show-user", uid, "-p", "Display", "--value")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("run 'loginctl' to get user GUI session: %w", err)
	}
	guiSessionID := strings.TrimSpace(stdout.String())
	if guiSessionID == "" {
		return 0, nil
	}
	cmd = exec.Command("loginctl", "show-session", guiSessionID, "-p", "Type", "--value")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return 0, fmt.Errorf("run 'loginctl' to get user GUI session type: %w", err)
	}
	guiSessionType := strings.TrimSpace(stdout.String())
	switch guiSessionType {
	case "":
		return 0, errors.New("empty GUI session type")
	case "x11":
		return GuiSessionTypeX11, nil
	case "wayland":
		return GuiSessionTypeWayland, nil
	case "tty":
		return GuiSessionTypeTty, nil
	default:
		return 0, fmt.Errorf("unknown GUI session type: %q", guiSessionType)
	}
}

type guiSessionType int

const (
	GuiSessionTypeX11 guiSessionType = iota + 1
	GuiSessionTypeWayland
	GuiSessionTypeTty
)

func (s guiSessionType) String() string {
	if s == GuiSessionTypeX11 {
		return "x11"
	}
	if s == GuiSessionTypeTty {
		return "tty"
	}
	return "wayland"
}
