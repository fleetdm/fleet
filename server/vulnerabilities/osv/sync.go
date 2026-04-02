package osv

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// OSVFilePrefix is the prefix for OSV artifact files
	OSVFilePrefix = "osv-ubuntu-"
)

// Refresh checks all local OSV artifacts contained in 'vulnPath' deleting the old and downloading
func Refresh(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
	now time.Time,
) ([]string, error) {
	release, err := getLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	neededVersions := getNeededUbuntuVersions(versions)
	if len(neededVersions) == 0 {
		return nil, nil
	}

	syncResult, err := SyncOSV(vulnPath, neededVersions, now, release)
	if err != nil {
		return nil, fmt.Errorf("syncing OSV artifacts: %w", err)
	}

	err = removeOldOSVArtifacts(now, vulnPath)
	if err != nil {
		return syncResult.Downloaded, fmt.Errorf("warning: failed to clean up old OSV artifacts: %w", err)
	}

	return syncResult.Downloaded, nil
}

// removeOldOSVArtifacts walks 'path' removing any old OSV artifacts that don't match today's date
func removeOldOSVArtifacts(date time.Time, rootPath string) error {
	dateSuffix := fmt.Sprintf("-%d-%02d-%02d.json.gz", date.Year(), date.Month(), date.Day())

	return filepath.WalkDir(rootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		baseName := d.Name()
		if strings.HasPrefix(baseName, OSVFilePrefix) && strings.HasSuffix(baseName, ".json.gz") {
			if !strings.HasSuffix(baseName, dateSuffix) {
				parentDir := filepath.Dir(path)
				if err := os.Remove(filepath.Join(parentDir, baseName)); err != nil {
					return fmt.Errorf("removing old OSV artifact %s: %w", baseName, err)
				}
			}
		}
		return nil
	})
}

// getNeededUbuntuVersions extracts unique Ubuntu versions from OS versions
func getNeededUbuntuVersions(osVers *fleet.OSVersions) []string {
	seen := make(map[string]struct{})
	var needed []string

	for _, os := range osVers.OSVersions {
		if !IsPlatformSupported(os.Platform) {
			continue
		}

		// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
		ubuntuVer := extractUbuntuVersion(os.Version)
		if ubuntuVer == "" {
			continue
		}

		if _, exists := seen[ubuntuVer]; !exists {
			seen[ubuntuVer] = struct{}{}
			needed = append(needed, ubuntuVer)
		}
	}

	return needed
}

// osvFilename generates the OSV artifact filename for a given Ubuntu version and date
// Format: osv-ubuntu-2204-2026-03-30.json.gz
func osvFilename(ubuntuVersion string, date time.Time) string {
	return fmt.Sprintf("osv-ubuntu-%s-%d-%02d-%02d.json.gz",
		ubuntuVersion, date.Year(), date.Month(), date.Day())
}
