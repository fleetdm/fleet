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

// UserLoggedInViaGui returns the username that has an active GUI session.
// It returns nil, nil if there's no user with an active GUI session.
func UserLoggedInViaGui() (*string, error) {
	users, err := getLoginUsers()
	if err != nil {
		return nil, fmt.Errorf("get login users: %w", err)
	}

	for _, user := range users {
		// Skip system/display manager users since they aren't GUI users.
		// User gdm-greeter is active during the GUI log-in prompt (GNOME 49).
		if user.Name == "gdm" || user.Name == "root" || user.Name == "gdm-greeter" {
			continue
		}
		// Check if the user has an active GUI session.
		session, err := GetUserDisplaySessionType(strconv.FormatInt(user.ID, 10))
		if err != nil {
			log.Debug().Err(err).Msgf("failed to get user display session for user %s", user.Name)
			continue
		}
		if session == nil {
			log.Debug().Msgf("no display session found for user %s", user.Name)
			continue
		}
		if !session.Active {
			log.Debug().Msgf("user %s has an inactive display session, skipping", user.Name)
			continue
		}
		if session.Type == GuiSessionTypeTty {
			log.Debug().Msgf("user %s is logged in via TTY, not GUI", user.Name)
			continue
		}
		return &user.Name, nil
	}

	return nil, nil
}

// GetLoginUser returns the first logged-in user as reported by
// `loginctl list-users`.
func GetLoginUser() (*User, error) {
	users, err := getLoginUsers()
	if err != nil {
		return nil, err
	}
	return &users[0], nil
}

// getLoginUsers returns all logged-in users as reported by
// `loginctl list-users`.
func getLoginUsers() ([]User, error) {
	out, err := exec.Command("loginctl", "list-users", "--no-legend", "--no-pager").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("loginctl list-users exec failed: %w, output: %s", err, string(out))
	}
	return parseLoginctlUsersOutput(string(out))
}

// GetSELinuxUserContext returns the SELinux context for the given user.
// Example: `unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023`
//
// If SELinux is not enabled, the `runcon` command is not available,
// or context cannot be determined, it returns nil.
func GetSELinuxUserContext(user *User) *string {
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

// parseLoginctlUsersOutput parses the output of `loginctl list-users --no-legend`.
// Each line has the format: UID USERNAME
func parseLoginctlUsersOutput(s string) ([]User, error) {
	var users []User
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		uid, err := strconv.ParseInt(fields[0], 10, 64)
		if err != nil {
			log.Debug().Err(err).Msgf("failed to parse uid from loginctl output: %q", line)
			continue
		}
		users = append(users, User{
			Name: fields[1],
			ID:   uid,
		})
	}
	if len(users) == 0 {
		return nil, errors.New("no user session found")
	}
	return users, nil
}

// UserDisplaySession holds the display session type and active status for a user.
type UserDisplaySession struct {
	Type   guiSessionType
	Active bool
}

// GetUserDisplaySessionType returns the display session type (X11 or Wayland)
// and active status of the given user. Returns nil with no error if the user
// has no display session.
func GetUserDisplaySessionType(uid string) (*UserDisplaySession, error) {
	// Get the "Display" session ID of the user.
	cmd := exec.Command("loginctl", "show-user", uid, "-p", "Display", "--value")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run 'loginctl' to get user GUI session: %w", err)
	}
	guiSessionID := strings.TrimSpace(stdout.String())
	if guiSessionID == "" {
		return nil, nil
	}
	// Get the "Type" of session.
	cmd = exec.Command("loginctl", "show-session", guiSessionID, "-p", "Type", "--value")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run 'loginctl' to get user GUI session type: %w", err)
	}
	var sessionType guiSessionType
	switch t := strings.TrimSpace(stdout.String()); t {
	case "":
		return nil, errors.New("empty GUI session type")
	case "x11":
		sessionType = GuiSessionTypeX11
	case "wayland":
		sessionType = GuiSessionTypeWayland
	case "tty":
		sessionType = GuiSessionTypeTty
	default:
		return nil, fmt.Errorf("unknown GUI session type: %q", t)
	}

	// Get the "Active" property of the session.
	cmd = exec.Command("loginctl", "show-session", guiSessionID, "-p", "Active", "--value")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run 'loginctl' to get session active status: %w", err)
	}
	active := strings.TrimSpace(stdout.String()) == "yes"
	return &UserDisplaySession{
		Type:   sessionType,
		Active: active,
	}, nil
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
