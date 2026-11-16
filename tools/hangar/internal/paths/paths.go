// Package paths resolves the per-application macOS directories (settings,
// data, logs) and provides the small path-safety helpers that the Tauri
// build had scattered across settings.rs: shell-tilde expansion, the
// "stay under $HOME" guard, and an extension allowlist check.
//
// macOS-only by design (parity with the Rust app's macOS-first scope).
// Directory layout mirrors what Tauri's path API resolved from the bundle
// identifier so an eventual cutover keeps reading the same files:
//
//	app_config_dir / app_data_dir -> ~/Library/Application Support/<BundleID>
//	app_log_dir                   -> ~/Library/Logs/<BundleID>
//
// On macOS Tauri's config and data dirs are the same Application Support
// folder, so settings.json, perf-configs.json and running.json all live
// together there.
//
// HANGAR_DATA_DIR replaces that folder (logs go to its logs/ subdir), so a dev
// Hangar can run beside the real one without sharing state.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// BundleID is the macOS bundle identifier that scopes this app's data and
// log directories. It must match the .app bundle's Info.plist identifier.
//
// This is the canonical Hangar identifier: settings written by the original
// Rust/Tauri Hangar under the same ID carry over untouched. (During the
// Rust->Go build-alongside phase this was temporarily the "-go" variant so a
// dev build couldn't corrupt the real app's state; that phase is over now
// that the Go port is the canonical Hangar.)
const BundleID = "com.fleetdm.fleet-hangar"

func appSupportDir(home string) string {
	return filepath.Join(home, "Library", "Application Support", BundleID)
}

func appLogDir(home string) string {
	return filepath.Join(home, "Library", "Logs", BundleID)
}

const (
	dataDirEnv      = "HANGAR_DATA_DIR"
	reservationsEnv = "HANGAR_RESERVATIONS_FILE"
)

// ConfigDir returns the app's config directory (Application Support),
// creating it if needed. Settings and perf-configs live here.
func ConfigDir() (string, error) { return ensureDir(appSupportDir, "") }

// DataDir returns the app's data directory. Same folder as ConfigDir on
// macOS (Tauri parity); running.json lives here.
func DataDir() (string, error) { return ensureDir(appSupportDir, "") }

// LogDir returns the app's log directory (~/Library/Logs/<BundleID>, or
// $HANGAR_DATA_DIR/logs), creating it if needed.
func LogDir() (string, error) { return ensureDir(appLogDir, "logs") }

// ReservationsFile returns the slot reservations path:
// $HANGAR_RESERVATIONS_FILE, else reservations.json in the data dir.
func ReservationsFile() (string, error) {
	if f := os.Getenv(reservationsEnv); f != "" {
		if !filepath.IsAbs(f) {
			return "", fmt.Errorf("%s must be an absolute path, got %q", reservationsEnv, f)
		}
		return f, nil
	}
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "reservations.json"), nil
}

// ensureDir resolves pick(home), or sub under $HANGAR_DATA_DIR when set, and
// creates it 0700.
func ensureDir(pick func(home string) string, sub string) (string, error) {
	var dir string
	if d := os.Getenv(dataDirEnv); d != "" {
		if !filepath.IsAbs(d) {
			return "", fmt.Errorf("%s must be an absolute path, got %q", dataDirEnv, d)
		}
		dir = filepath.Join(d, sub)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home dir: %w", err)
		}
		dir = pick(home)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}
	return dir, nil
}

// LockDataDir takes an exclusive flock on <dir>/hangar.lock for the life of
// the process, so a second Hangar on the same data dir can't reap the first
// one's children or overwrite its state. It errors if the lock is held.
// The returned release closes the lock file; keep it reachable until exit,
// since the file's finalizer would otherwise close it and drop the lock.
func LockDataDir(dir string) (release func(), err error) {
	path := filepath.Join(dir, "hangar.lock")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, fmt.Errorf("%s is held by another Hangar", path)
		}
		return nil, fmt.Errorf("locking %s: %w", path, err)
	}
	return func() { f.Close() }, nil
}

// Expand expands a leading "~" or "~/" to the user's home directory.
// Anything else is returned unchanged. Mirrors the Rust shellexpand:
// only a leading tilde is special — no $VAR or mid-path ~ handling.
func Expand(p string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return expand(p, home)
}

func expand(p, home string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(home, rest)
	}
	if p == "~" {
		return home
	}
	return p
}

// UnderHome reports an error if the (already shell-expanded) path is not
// inside $HOME or contains any ".." segment. This is the guard the generic
// file/open commands use so a compromised webview can't read or overwrite
// arbitrary files. Not a full sandbox — start_process and the capture
// runner are general-purpose by design.
func UnderHome(p string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.New("no home dir")
	}
	return underHome(p, home)
}

func underHome(p, home string) error {
	if !HasPathPrefix(p, home) {
		return fmt.Errorf("path must be under %s", home)
	}
	// Reject any literal ".." segment (checked on the raw path, before any
	// cleaning, matching the Rust component scan).
	for _, seg := range strings.Split(p, string(os.PathSeparator)) {
		if seg == ".." {
			return errors.New("path contains `..` segments")
		}
	}
	return nil
}

// HasPathPrefix reports whether p is at or under prefix, comparing whole
// path components — so "/a/bc" is NOT under "/a/b" (a plain string prefix
// would wrongly say yes). Compares raw segments without cleaning, matching
// Rust's Path::starts_with over components.
func HasPathPrefix(p, prefix string) bool {
	ps := strings.Split(p, string(os.PathSeparator))
	pre := strings.Split(prefix, string(os.PathSeparator))
	if len(ps) < len(pre) {
		return false
	}
	for i := range pre {
		if ps[i] != pre[i] {
			return false
		}
	}
	return true
}

// HasExt reports whether p's extension (case-insensitive, without the dot)
// is one of allowed.
func HasExt(p string, allowed ...string) bool {
	ext := strings.TrimPrefix(filepath.Ext(p), ".")
	if ext == "" {
		return false
	}
	for _, a := range allowed {
		if strings.EqualFold(ext, a) {
			return true
		}
	}
	return false
}
