// Package shellpath resolves the PATH to hand to spawned child processes.
//
// A macOS app launched from Finder/Dock inherits only the bare
// /usr/bin:/bin:/usr/sbin:/sbin — NOT the user's shell PATH. That omits
// Homebrew (/opt/homebrew/bin), Go (/usr/local/go/bin), nvm, ~/.fleetctl,
// etc., so bare-name spawns the app relies on (git, go, docker, ngrok,
// python3, ...) fail with "No such file or directory" in the packaged app,
// even though they work under `wails3 dev` (which inherits the terminal's
// PATH).
//
// We capture the user's real login-shell PATH once at startup and apply it
// to every child spawn. Ported from src-tauri/src/shellpath.rs.
package shellpath

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	mu     sync.Mutex
	cached string
)

// ShellPath returns the PATH string to set on spawned children. Computed
// lazily on the first call and cached; later calls return the cached value.
func ShellPath() string {
	mu.Lock()
	defer mu.Unlock()
	if cached == "" {
		cached = resolve()
	}
	return cached
}

// Warm eagerly populates the cache (e.g. at startup) so the first real
// spawn doesn't pay the shell-probe latency. Safe to call repeatedly.
func Warm() { ShellPath() }

// Refresh re-probes and overwrites the cache, returning the new value.
// The dep-check screen calls this on "Recheck" so tools the user just
// installed (which modified .zprofile) become visible to every subsequent
// spawn, not just the next Recheck.
func Refresh() string {
	p := resolve()
	mu.Lock()
	cached = p
	mu.Unlock()
	slog.Debug("re-probed login-shell PATH", "entries", strings.Count(p, ":")+1)
	return p
}

// Env returns the current process environment with PATH overridden to the
// resolved login-shell PATH — suitable for exec.Cmd.Env so spawned tools
// (git, fleetctl, docker, ...) resolve when launched from Finder.
func Env() []string {
	return envWith(os.Environ(), ShellPath())
}

// MergeEnv layers extra KEY=VALUE pairs over an environment slice, replacing
// existing keys in place (so the child's getenv sees the override, not a
// duplicate) and appending new ones. Used by the spawn paths to apply
// caller-supplied env on top of the inherited+PATH-overridden environment.
func MergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	out := append([]string(nil), base...)
	idx := map[string]int{}
	for i, e := range out {
		if eq := strings.IndexByte(e, '='); eq >= 0 {
			idx[e[:eq]] = i
		}
	}
	for k, v := range extra {
		if i, ok := idx[k]; ok {
			out[i] = k + "=" + v
		} else {
			idx[k] = len(out)
			out = append(out, k+"="+v)
		}
	}
	return out
}

func envWith(base []string, path string) []string {
	out := make([]string, 0, len(base)+1)
	replaced := false
	for _, e := range base {
		if strings.HasPrefix(e, "PATH=") {
			out = append(out, "PATH="+path)
			replaced = true
		} else {
			out = append(out, e)
		}
	}
	if !replaced {
		out = append(out, "PATH="+path)
	}
	return out
}

func resolve() string {
	if p := probeLoginShell(); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return augmentInherited(home, os.Getenv("PATH"))
}

// probeLoginShell asks the user's login shell for its PATH. "-ilc" so the
// rc/profile files that actually set PATH (Homebrew shellenv, nvm, custom
// exports) are sourced. Returns "" on any failure.
func probeLoginShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	out, err := exec.Command(shell, "-ilc", `printf %s "$PATH"`).Output()
	if err != nil {
		return ""
	}
	p := lastNonEmptyLine(string(out))
	// Sanity-check: a real PATH has separators and absolute entries.
	if !strings.Contains(p, "/") {
		return ""
	}
	return p
}

// lastNonEmptyLine returns the last non-empty, trimmed line of s. Some
// interactive shells (zsh with session restoration) print a banner like
// "Restored session: ..." before our command runs; that banner always
// precedes our printf, so the PATH is the last line.
func lastNonEmptyLine(s string) string {
	var last string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			last = t
		}
	}
	return last
}

// augmentInherited is the fallback when the shell probe fails: prepend the
// common tool locations to the inherited PATH, de-duplicated. Order is
// preserved; the prepended dirs win over inherited ones.
func augmentInherited(home, inherited string) string {
	var parts []string
	seen := map[string]bool{}
	push := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			parts = append(parts, p)
		}
	}

	push("/opt/homebrew/bin")
	push("/opt/homebrew/sbin")
	push("/usr/local/bin")
	push("/usr/local/go/bin")
	if home != "" {
		push(filepath.Join(home, "go/bin"))
		push(filepath.Join(home, ".local/bin"))
		push(filepath.Join(home, ".fleetctl"))
	}
	for _, e := range strings.Split(inherited, ":") {
		push(e)
	}
	return strings.Join(parts, ":")
}
