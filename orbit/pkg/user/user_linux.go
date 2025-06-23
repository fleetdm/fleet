//go:build linux
// +build linux

package user

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
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
	if user.Name == "gdm" || user.Name == "root" {
		return nil, nil // gdm is the default user for GDM login manager, not a real user.
	}
	return &user.Name, nil
}

// getLoginUser returns the name and uid of the first login user
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

type guiSessionType int

const (
	guiSessionTypeX11 guiSessionType = iota + 1
	guiSessionTypeWayland
)

func (s guiSessionType) String() string {
	if s == guiSessionTypeX11 {
		return "x11"
	}
	return "wayland"
}
