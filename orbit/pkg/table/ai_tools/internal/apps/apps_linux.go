//go:build linux

package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
)

func scanApps(homesList []homes.Home) []App {
	seen := map[string]struct{}{}
	var out []App

	scanDir := func(dir, scope string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			name, exec := parseDesktop(filepath.Join(dir, e.Name()))
			ka, ok := matchKnown(e.Name(), name, exec)
			if _, dup := seen[ka.name]; !ok || dup {
				continue
			}
			seen[ka.name] = struct{}{}
			out = append(out, App{
				Name:           ka.name,
				Path:           firstNonEmpty(exec, e.Name()),
				PlatformSource: "desktop-file",
				Scope:          scope,
				execPath:       execBinary(exec),
			})
		}
	}

	scanDir("/usr/share/applications", "system")
	scanDir("/usr/local/share/applications", "system")
	for _, h := range homesList {
		scanDir(filepath.Join(h.Dir, ".local", "share", "applications"), "user")
	}

	// Ollama commonly installs as a service binary with no .desktop entry.
	for _, b := range []string{"/usr/local/bin/ollama", "/usr/bin/ollama"} {
		_, dup := seen["ollama"]
		if fi, err := os.Stat(b); err == nil && !fi.IsDir() && !dup {
			seen["ollama"] = struct{}{}
			out = append(out, App{Name: "ollama", Path: b, PlatformSource: "desktop-file", Scope: "system", execPath: b})
		}
	}
	return out
}

func parseDesktop(path string) (name, exec string) {
	b, err := fsutil.ReadFileBounded(path)
	if err != nil {
		return "", ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case name == "" && strings.HasPrefix(line, "Name="):
			name = strings.TrimPrefix(line, "Name=")
		case exec == "" && strings.HasPrefix(line, "Exec="):
			exec = strings.TrimPrefix(line, "Exec=")
		}
	}
	return name, exec
}

// execBinary extracts the binary path from a .desktop Exec= line (the first
// whitespace-separated token), returning it only when it is an absolute path to
// an existing file — desktop field codes like %U are discarded.
func execBinary(exec string) string {
	fields := strings.Fields(exec)
	if len(fields) == 0 {
		return ""
	}
	bin := fields[0]
	if !strings.HasPrefix(bin, "/") {
		return ""
	}
	if fi, err := os.Stat(bin); err != nil || fi.IsDir() {
		return ""
	}
	return bin
}
