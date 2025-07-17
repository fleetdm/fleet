//go:build linux
// +build linux

package user

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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

// GetUserContext returns the SELinux context for the given user.
// Example: `unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023`
//
// If SELinux is not enabled, the `runcon` command is not available,
// or context cannot be determined, it returns nil.
func GetUserContext(user *User) *string {
	// If SELinux is not enabled, return nil right away.
	if _, err := os.Stat("/sys/fs/selinux/enforce"); err != nil {
		return nil
	}
	// If runcon is not available, we won't be able to switch contexts,
	// so return nil.
	if _, err := exec.LookPath("runcon"); err != nil {
		log.Warn().Msg("runcon not available, returning nil for user context since we can't switch contexts")
		return nil
	}
	// Find the first systemd process for the user and read its SELinux context.
	pidBytes, err := exec.Command("pgrep", "-u", strconv.FormatInt(user.ID, 10), "-nx", "systemd").Output() // #nosec G204
	if err != nil {
		log.Debug().Msgf("Error finding systemd process for user %s: %v", user.Name, err)
		return nil
	}
	pid := strings.TrimSpace(string(pidBytes))
	if pid == "" {
		log.Debug().Msgf("No systemd process found for user %s", user.Name)
		return nil
	}
	ctx, err := os.ReadFile("/proc/" + pid + "/attr/current")
	if err != nil {
		log.Debug().Msgf("Error reading SELinux context for user %s: %v", user.Name, err)
		return nil
	}
	context := strings.TrimSpace(string(ctx))
	// Remove any null byte at the end
	context = strings.TrimSuffix(context, "\x00")
	if context == "" {
		log.Debug().Msg("Empty SELinux context for user " + user.Name)
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
