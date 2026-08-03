//go:build darwin

package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/fsutil"
	"github.com/fleetdm/fleet/v4/orbit/pkg/table/ai_tools/internal/homes"
	"howett.net/plist"
)

type infoPlist struct {
	BundleName    string `plist:"CFBundleName"`
	BundleID      string `plist:"CFBundleIdentifier"`
	ShortVersion  string `plist:"CFBundleShortVersionString"`
	BundleVersion string `plist:"CFBundleVersion"`
	Executable    string `plist:"CFBundleExecutable"`
}

func scanApps(homesList []homes.Home) []App {
	seen := map[string]struct{}{}
	var out []App

	scanDir := func(dir, scope string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".app") {
				continue
			}
			appPath := filepath.Join(dir, e.Name())
			info := readInfoPlist(appPath)
			k, ok := matchKnown(e.Name(), info.BundleName, info.BundleID)
			if _, dup := seen[k.name]; !ok || dup {
				continue
			}
			seen[k.name] = struct{}{}
			// info.Executable comes from the bundle's user-writable Info.plist; a
			// value with a path separator or ".." would escape the bundle when
			// joined (and later be hashed as root). Only accept a bare filename.
			exec := ""
			if info.Executable != "" && !strings.ContainsAny(info.Executable, `/\`) && !strings.Contains(info.Executable, "..") {
				exec = filepath.Join(appPath, "Contents", "MacOS", info.Executable)
			}
			out = append(out, App{
				Name:           k.name,
				BundleID:       info.BundleID,
				Version:        firstNonEmpty(info.ShortVersion, info.BundleVersion),
				Path:           appPath,
				PlatformSource: "applications",
				Scope:          scope,
				execPath:       exec,
			})
		}
	}

	scanDir("/Applications", "system")
	scanDir("/Applications/Utilities", "system")
	for _, h := range homesList {
		scanDir(filepath.Join(h.Dir, "Applications"), "user")
	}
	return out
}

func readInfoPlist(appPath string) infoPlist {
	var info infoPlist
	b, err := fsutil.ReadFileBounded(filepath.Join(appPath, "Contents", "Info.plist"))
	if err != nil {
		return info
	}
	_, _ = plist.Unmarshal(b, &info)
	return info
}
