package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// usersLogin logs one of the driver's own password users in, loads /me the way the UI does, and logs out now and
// then (most people let the session expire, so sessions pile up as they do in production). With fail, it uses a
// wrong password and expects a 401.
type usersLogin struct {
	fail   bool
	logout float64
}

func newUsersLogin(p *params) (verb, error) {
	v := &usersLogin{}
	var err error
	if v.fail, err = p.boolean("fail", false); err != nil {
		return nil, err
	}
	if v.logout, err = p.float("logout", 0.3); err != nil {
		return nil, err
	}
	if v.logout < 0 || v.logout > 1 {
		return nil, fmt.Errorf("with.logout %v is outside 0-1", v.logout)
	}
	return v, nil
}

func (v *usersLogin) setup(ctx context.Context, e *env) error {
	e.usersOnce.Do(func() { e.users, e.usersErr = ensureUserPool(ctx, e) })
	return e.usersErr
}

func (v *usersLogin) run(ctx context.Context, e *env) result {
	u := e.users.users[randIntN(len(e.users.users))]
	detail := map[string]any{"user": strings.TrimSuffix(u.Email, "@example.com")}
	pw := u.Password
	if v.fail {
		pw = "wrong-" + newPassword()
	}
	var login struct {
		Token string `json:"token"`
	}
	status, err := e.api.doAs(ctx, "", http.MethodPost, "/api/latest/fleet/login", nil,
		map[string]string{"email": u.Email, "password": pw}, &login)
	if v.fail {
		if status == http.StatusUnauthorized {
			return result{status: status, detail: detail}
		}
		if err == nil {
			err = errors.New("login with a wrong password succeeded")
		}
		return result{status: status, err: err, detail: detail}
	}
	if err != nil {
		return result{status: status, err: err, detail: detail}
	}
	if s, err := e.api.doAs(ctx, login.Token, http.MethodGet, "/api/latest/fleet/me", nil, nil, nil); err != nil {
		return result{status: s, err: err, detail: detail}
	}
	if randFloat() < v.logout {
		detail["logout"] = true
		if s, err := e.api.doAs(ctx, login.Token, http.MethodPost, "/api/latest/fleet/logout", nil, nil, nil); err != nil {
			return result{status: s, err: err, detail: detail}
		}
	}
	return result{status: status, detail: detail}
}

// userPool is the driver's own password users: SSO users can't log in on an instance without an IdP, so logins
// use these. Passwords live only in the state file (0600) and are never logged.
type userPool struct {
	users []poolUser
}

type poolUser struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ensureUserPool creates the pool's missing users. A pool user whose password the driver doesn't know (not in the
// state file, or the derived one doesn't log in) is deleted and created again: the prefix marks it as the driver's.
func ensureUserPool(ctx context.Context, e *env) (*userPool, error) {
	path := filepath.Join(e.stateDir, "users.json")
	saved := map[string]poolUser{}
	switch b, err := os.ReadFile(path); {
	case e.usersSecret != "": // derived passwords: no state file
	case err == nil:
		var list []poolUser
		if err := json.Unmarshal(b, &list); err != nil {
			return nil, fmt.Errorf("user pool state %s: %w", path, err)
		}
		for _, u := range list {
			saved[u.Email] = u
		}
	case !errors.Is(err, fs.ErrNotExist):
		return nil, fmt.Errorf("user pool state: %w", err)
	}

	var existing struct {
		Users []struct {
			ID    uint   `json:"id"`
			Email string `json:"email"`
		} `json:"users"`
	}
	q := url.Values{"query": {e.userPrefix}, "per_page": {"1000"}}
	if _, err := e.api.do(ctx, http.MethodGet, "/api/latest/fleet/users", q, nil, &existing); err != nil {
		return nil, fmt.Errorf("user pool: %w", err)
	}
	ids := make(map[string]uint, len(existing.Users))
	for _, u := range existing.Users {
		ids[u.Email] = u.ID
	}

	roles := []string{"observer", "observer", "maintainer", "admin"}
	pool := &userPool{}
	created := 0
	for i := 1; i <= e.userCount; i++ {
		email := fmt.Sprintf("%s%03d@example.com", e.userPrefix, i)
		if id, ok := ids[email]; ok {
			if s, ok := saved[email]; ok && s.ID == id {
				pool.users = append(pool.users, s)
				continue
			}
			if e.usersSecret != "" {
				u := poolUser{ID: id, Email: email, Password: derivePassword(e.usersSecret, email)}
				ok, err := canLogIn(ctx, e, u)
				if err != nil {
					return nil, fmt.Errorf("user pool: check %s: %w", email, err)
				}
				if ok {
					pool.users = append(pool.users, u)
					continue
				}
			}
			if _, err := e.api.do(ctx, http.MethodDelete, fmt.Sprintf("/api/latest/fleet/users/%d", id), nil, nil, nil); err != nil {
				return nil, fmt.Errorf("user pool: replace %s: %w", email, err)
			}
		}
		pw := newPassword()
		if e.usersSecret != "" {
			pw = derivePassword(e.usersSecret, email)
		}
		var cr struct {
			User struct {
				ID uint `json:"id"`
			} `json:"user"`
		}
		body := map[string]any{
			"name": fmt.Sprintf("Driver user %03d", i), "email": email, "password": pw,
			"global_role": roles[i%len(roles)], "admin_forced_password_reset": false,
		}
		if _, err := e.api.do(ctx, http.MethodPost, "/api/latest/fleet/users/admin", nil, body, &cr); err != nil {
			return nil, fmt.Errorf("user pool: create %s: %w", email, err)
		}
		u := poolUser{ID: cr.User.ID, Email: email, Password: pw}
		pool.users = append(pool.users, u)
		saved[email] = u
		created++
		if e.usersSecret != "" {
			continue
		}
		if err := saveUsers(path, saved); err != nil { // after each user, so a failed run loses no password
			return nil, err
		}
	}
	e.logger.InfoContext(ctx, "user pool ready", "users", len(pool.users), "created", created)
	return pool, nil
}

func saveUsers(path string, users map[string]poolUser) error {
	list := make([]poolUser, 0, len(users))
	for _, u := range users {
		list = append(list, u)
	}
	b, err := json.Marshal(list) //nolint:gosec // the pool passwords belong in this 0600 state file
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("user pool state: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("user pool state: %w", err)
	}
	return os.Rename(tmp, path)
}

// canLogIn checks a pool user's password with a login (and logs the session out).
func canLogIn(ctx context.Context, e *env, u poolUser) (bool, error) {
	var login struct {
		Token string `json:"token"`
	}
	status, err := e.api.doAs(ctx, "", http.MethodPost, "/api/latest/fleet/login", nil,
		map[string]string{"email": u.Email, "password": u.Password}, &login)
	switch {
	case status == http.StatusUnauthorized:
		return false, nil
	case err != nil:
		return false, err
	}
	_, _ = e.api.doAs(ctx, login.Token, http.MethodPost, "/api/latest/fleet/logout", nil, nil, nil)
	return true, nil
}

// derivePassword is the pool password for an email: the same on every run with the same secret, so the driver
// needs no state where its files don't survive a restart (ECS). It meets Fleet's rules like newPassword.
func derivePassword(secret, email string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(email))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:18]) + "#7"
}

// newPassword meets Fleet's rules (12-48 characters with a number and a symbol).
func newPassword() string {
	b := make([]byte, 18)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b) + "#7"
}
