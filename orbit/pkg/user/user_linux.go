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

func getDisplaySessionFor(user User) *UserDisplaySession {
	// Skip system/display manager users since they aren't GUI users.
	// User gdm-greeter is active during the GUI log-in prompt (GNOME 49).
	if user.Name == "gdm" || user.Name == "root" || user.Name == "gdm-greeter" {
		return nil
	}
	// Check if the user has an active GUI session.
	userID := strconv.FormatInt(user.ID, 10)
	session, err := GetUserDisplaySessionType(userID)
	if err != nil {
		log.Debug().Err(err).Msgf("failed to get user display session for user %s", user.Name)
		return nil
	}
	if !session.Active {
		log.Debug().Msgf("user %s has an inactive display session, skipping", user.Name)
		return nil
	}
	if session.Type == GuiSessionTypeTty {
		log.Debug().Msgf("user %s is logged in via TTY, not GUI", user.Name)
		return nil
	}
	return session
}

// UserLoggedInViaGui returns the username that has an active GUI session.
// It returns nil, nil if there's no user with an active GUI session.
func UserLoggedInViaGui() (*string, error) {
	users, err := getLoginUsers()
	if err != nil {
		return nil, fmt.Errorf("get login users: %w", err)
	}

	for _, user := range users {
		if session := getDisplaySessionFor(user); session != nil {
			return &user.Name, nil
		}
	}

	// No valid user found
	return nil, nil
}

func GetLoggedInUserDisplaySession() (*UserDisplaySession, error) {
	users, err := getLoginUsers()
	if err != nil {
		return nil, fmt.Errorf("get login users: %w", err)
	}

	for _, user := range users {
		if session := getDisplaySessionFor(user); session != nil {
			return session, nil
		}
	}

	// No valid user found
	return nil, nil
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

// parseLoginctlUsersOutput parses the output of `loginctl list-users --no-legend`.
// Each line has the format: UID USERNAME
func parseLoginctlUsersOutput(s string) ([]User, error) {
	var users []User
	for line := range strings.SplitSeq(strings.TrimSpace(s), "\n") {
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
	Type    GuiSessionType
	Active  bool
	Desktop string
}

// GetUserDisplaySessionType returns the display session type (X11 or Wayland),
// active status and session env vars of the given user. Returns an error if the user doesn't have
// a Display session.
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
		return nil, errors.New("empty display session")
	}

	// Get the "Type" of session.
	cmd = exec.Command("loginctl", "show-session", guiSessionID, "-p", "Type", "--value")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run 'loginctl' to get user GUI session type: %w", err)
	}
	var sessionType GuiSessionType
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

	// Get the "Environment" property of the session.
	cmd = exec.Command("loginctl", "show-session", guiSessionID, "-p", "Desktop", "--value")
	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run 'loginctl' to get session environment: %w", err)
	}
	return &UserDisplaySession{
		Type:    sessionType,
		Active:  active,
		Desktop: strings.TrimSpace(stdout.String()),
	}, nil
}

func parseLoginctlSessionEnvironment(out string) map[string]string {
	env := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || key == "" {
			continue
		}
		env[key] = value
	}
	if len(env) == 0 {
		return nil
	}
	return env
}

type GuiSessionType int

const (
	GuiSessionTypeX11 GuiSessionType = iota + 1
	GuiSessionTypeWayland
	GuiSessionTypeTty
)

func (s GuiSessionType) String() string {
	if s == GuiSessionTypeX11 {
		return "x11"
	}
	if s == GuiSessionTypeTty {
		return "tty"
	}
	return "wayland"
}
