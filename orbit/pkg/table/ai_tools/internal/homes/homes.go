// Package homes enumerates real user home directories on the host.
//
// A Fleet-deployed osquery extension runs as root (macOS/Linux) or
// SYSTEM/Administrator (Windows), so every table must inventory ALL users'
// homes — not just the daemon account's. Editor plugins, MCP configs and AI
// tools all live under per-user home directories.
package homes

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
)

// Home is a single user's account and home directory.
type Home struct {
	UID      string
	Username string
	Dir      string
}

// All returns every real (non-system) user home directory discoverable on the
// host. Results are de-duplicated and stat-verified to be directories.
func All() []Home {
	seen := map[string]struct{}{}
	var out []Home

	add := func(dir string) {
		if dir == "" {
			return
		}
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			return
		}
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			return
		}
		seen[dir] = struct{}{}
		uid, username := owner(dir, fi)
		out = append(out, Home{UID: uid, Username: username, Dir: dir})
	}

	switch runtime.GOOS {
	case "darwin":
		listChildren("/Users", []string{"shared", ".localized", "guest"}, add)
	case "windows":
		drive := os.Getenv("SystemDrive")
		if drive == "" {
			drive = "C:"
		}
		listChildren(filepath.Join(drive+`\`, "Users"),
			[]string{"public", "default", "default user", "all users", "defaultapppool"}, add)
	default: // linux and other unix
		listChildren("/home", nil, add)
		add("/root")
	}

	// Always include the running user's home as a fallback (covers non-standard
	// layouts and the case where the extension runs unprivileged).
	if h, err := os.UserHomeDir(); err == nil {
		add(h)
	}
	return out
}

// owner returns the uid/username to attribute rows under dir to. It derives
// them from the directory's actual owner (via stat) rather than the directory
// name, so a low-privilege user who creates a directory named after another
// account (e.g. /Users/root) cannot forge the uid/username attribution columns.
// When ownership can't be read from the OS (e.g. Windows), it falls back to the
// directory name.
func owner(dir string, fi os.FileInfo) (uid, username string) {
	if ownerUID, ok := statOwnerUID(fi); ok {
		if u, err := user.LookupId(ownerUID); err == nil {
			return ownerUID, u.Username
		}
		// The owner uid is authoritative even when it maps to no named account;
		// do not guess a username from the (attacker-influenced) directory name.
		return ownerUID, ""
	}
	username = filepath.Base(dir)
	if u, err := user.Lookup(username); err == nil {
		return u.Uid, username
	}
	return "", username
}

func listChildren(parent string, skipLower []string, add func(string)) {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return
	}
	skip := map[string]struct{}{}
	for _, s := range skipLower {
		skip[s] = struct{}{}
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if _, ok := skip[strings.ToLower(name)]; ok {
			continue
		}
		add(filepath.Join(parent, name))
	}
}
