// Package paths resolves common per-OS application-data roots derived from a
// given home directory.
//
// Deriving roots FROM the home dir (rather than from environment variables) is
// deliberate: when scanning other users' homes as root we cannot read their
// %APPDATA%/$XDG_* environment, so we reconstruct the conventional locations.
package paths

import (
	"path/filepath"
	"runtime"
)

// Roots holds resolved application-data roots for one home directory. Fields
// that don't apply to the current OS are left empty.
type Roots struct {
	Home          string // the home directory itself
	AppData       string // Windows %APPDATA% (Roaming)
	LocalAppData  string // Windows %LOCALAPPDATA% (Local)
	MacAppSupport string // macOS ~/Library/Application Support
	XDGConfig     string // ~/.config
	XDGData       string // ~/.local/share
}

// For builds the Roots for a single home directory.
func For(home string) Roots {
	r := Roots{
		Home:      home,
		XDGConfig: filepath.Join(home, ".config"),
		XDGData:   filepath.Join(home, ".local", "share"),
	}
	switch runtime.GOOS {
	case "windows":
		r.AppData = filepath.Join(home, "AppData", "Roaming")
		r.LocalAppData = filepath.Join(home, "AppData", "Local")
	case "darwin":
		r.MacAppSupport = filepath.Join(home, "Library", "Application Support")
	}
	return r
}

// NodeModulesDirs returns the global node_modules directories that may hold
// npm-installed tools for this home: per-user npm and bun prefixes, every
// nvm-managed Node version, and the system-wide prefixes. Directories are not
// checked for existence.
func NodeModulesDirs(home string) []string {
	dirs := []string{
		filepath.Join(home, ".npm-global", "lib", "node_modules"),
		filepath.Join(home, ".bun", "install", "global", "node_modules"),
	}
	if runtime.GOOS == "windows" {
		dirs = append(dirs, filepath.Join(home, "AppData", "Roaming", "npm", "node_modules"))
	} else {
		dirs = append(dirs, "/usr/local/lib/node_modules", "/opt/homebrew/lib/node_modules")
	}
	if matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "lib", "node_modules")); matches != nil {
		dirs = append(dirs, matches...)
	}
	return dirs
}
