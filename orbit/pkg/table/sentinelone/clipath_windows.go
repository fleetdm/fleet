//go:build windows

package sentinelone

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// cliInstallRoots are the directories the SentinelOne installer creates on
// Windows. Each holds one versioned agent directory per install, e.g.
// "Sentinel Agent 25.1.4.434".
var cliInstallRoots = []string{
	`C:\Program Files\SentinelOne`,
	`C:\Program Files (x86)\SentinelOne`,
}

// ctlBinary is the sentinelctl equivalent shipped on Windows.
const ctlBinary = "SentinelCtl.exe"

// resolveCLIPath finds SentinelCtl.exe under the install roots. An upgrade can
// leave more than one versioned directory behind, so the highest version wins;
// directories whose version can't be read sort last.
func resolveCLIPath() string {
	type candidate struct {
		path    string
		dirName string
		version []int
	}
	var candidates []candidate

	for _, root := range cliInstallRoots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || !isAgentDir(e.Name()) {
				continue
			}
			path := filepath.Join(root, e.Name(), ctlBinary)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			candidates = append(candidates, candidate{
				path:    path,
				dirName: e.Name(),
				version: parseDirVersion(e.Name()),
			})
		}
	}

	if len(candidates) > 0 {
		slices.SortFunc(candidates, func(a, b candidate) int {
			// Versionless directories sort after versioned ones.
			switch {
			case len(a.version) == 0 && len(b.version) > 0:
				return 1
			case len(a.version) > 0 && len(b.version) == 0:
				return -1
			}
			if cmp := slices.Compare(b.version, a.version); cmp != 0 {
				return cmp
			}
			return strings.Compare(strings.ToLower(b.dirName), strings.ToLower(a.dirName))
		})
		return candidates[0].path
	}

	// Older installers dropped the binary directly in the root.
	for _, root := range cliInstallRoots {
		path := filepath.Join(root, ctlBinary)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// isAgentDir reports whether a directory under an install root looks like an
// agent install, e.g. "Sentinel Agent 25.1.4.434".
func isAgentDir(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "sentinel") && strings.Contains(name, "agent")
}

// parseDirVersion returns the dotted version in a directory name as its
// components, or nil when the name carries no readable version.
func parseDirVersion(name string) []int {
	start := strings.IndexFunc(name, func(r rune) bool { return r >= '0' && r <= '9' })
	if start < 0 {
		return nil
	}

	var b strings.Builder
	for _, r := range name[start:] {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
			continue
		}
		break
	}

	version := strings.Trim(b.String(), ".")
	if !strings.Contains(version, ".") {
		return nil
	}

	parts := strings.Split(version, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}
