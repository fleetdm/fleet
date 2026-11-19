// Package homes enumerates real user home directories on the host.
//
// A Fleet-deployed osquery extension runs as root (macOS/Linux) or
// SYSTEM/Administrator (Windows), so every table must inventory ALL users'
// homes — not just the daemon account's. Editor plugins, MCP configs and AI
// tools all live under per-user home directories.
package homes

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Home is a single user's account and home directory. UID is the numeric uid on
// Unix and the account SID on Windows; UID and Username are both empty when the
// home could not be attributed to a user account. Username alone may be empty
// when the owner is known but cannot be named: a uid with no passwd entry on
// Unix, a profile SID the host cannot resolve (unreachable domain controller,
// offline Entra ID) on Windows.
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

	// admit reports whether dir is a directory that exists and has not been
	// recorded yet, returning it cleaned along with its FileInfo and marking it
	// seen.
	admit := func(dir string) (string, os.FileInfo, bool) {
		if dir == "" {
			return "", nil, false
		}

		dir = filepath.Clean(dir)

		key := dir
		if runtime.GOOS == "windows" {
			key = strings.ToLower(dir)
		}

		if _, ok := seen[key]; ok {
			return "", nil, false
		}
		fi, err := os.Stat(dir)
		if err != nil || !fi.IsDir() {
			return "", nil, false
		}
		seen[key] = struct{}{}
		return dir, fi, true
	}

	// add records a home found by path alone, deriving its owner from the
	// directory itself. When ownership can't be established it stays unknown
	// ("", "")
	add := func(dir string) {
		dir, fi, ok := admit(dir)
		if !ok {
			return
		}
		uid, username, ok := statOwner(dir, fi)
		if !ok {
			uid, username = "", ""
		}
		out = append(out, Home{UID: uid, Username: username, Dir: dir})
	}

	// addKnown records a home the platform enumerated together with its owner,
	// which is authoritative and needs no derivation.
	addKnown := func(h Home) {
		dir, _, ok := admit(h.Dir)
		if !ok {
			return
		}
		h.Dir = dir
		out = append(out, h)
	}

	// Homes the OS itself records against an account come first, so that where a
	// directory is found both ways the authoritative attribution wins. This also
	// reaches profiles the directory listings below miss entirely, such as one
	// redirected off the system drive. Only Windows has such a record.
	for _, h := range platformHomes() {
		addKnown(h)
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
