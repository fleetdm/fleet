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

// Refresh checks all local OSV artifacts contained in 'vulnPath', deleting outdated artifacts and downloading the latest required ones.
func Refresh(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
	now time.Time,
) ([]string, error) {
	neededVersions := getNeededUbuntuVersions(versions)
	if len(neededVersions) == 0 {
		return nil, nil
	}

	release, err := getLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	syncResult, err := SyncOSV(ctx, vulnPath, neededVersions, now, release)
	if err != nil {
		return nil, fmt.Errorf("syncing OSV artifacts: %w", err)
	}

	upToDateVersions := make([]string, 0, len(syncResult.Downloaded)+len(syncResult.Skipped))
	upToDateVersions = append(upToDateVersions, syncResult.Downloaded...)
	upToDateVersions = append(upToDateVersions, syncResult.Skipped...)
	err = removeOldOSVArtifacts(now, vulnPath, upToDateVersions)
	if err != nil {
		return syncResult.Downloaded, fmt.Errorf("warning: failed to clean up old OSV artifacts: %w", err)
	}

	return syncResult.Downloaded, nil
}

// removeOldOSVArtifacts removes old OSV artifacts that don't match today's date
func removeOldOSVArtifacts(date time.Time, rootPath string, successfulVersions []string) error {
	dateSuffix := fmt.Sprintf("-%d-%02d-%02d.json.gz", date.Year(), date.Month(), date.Day())

	successfulSet := make(map[string]struct{}, len(successfulVersions))
	for _, v := range successfulVersions {
		successfulSet[v] = struct{}{}
	}

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", rootPath, err)
	}

	for _, entry := range entries {
		// Skip directories and non-regular files
		if entry.IsDir() || !entry.Type().IsRegular() {
			continue
		}

		baseName := entry.Name()

		// Skip non-OSV files early for performance
		if !strings.HasPrefix(baseName, OSVFilePrefix) {
			continue
		}

		// Check if it's a non-delta OSV artifact
		if strings.HasSuffix(baseName, ".json.gz") && !strings.Contains(baseName, "delta") {
			if !strings.HasSuffix(baseName, dateSuffix) {
				versionStart := len(OSVFilePrefix)
				versionEnd := strings.Index(baseName[versionStart:], "-")
				if versionEnd == -1 {
					continue
				}
				ubuntuVersion := baseName[versionStart : versionStart+versionEnd]

				if _, ok := successfulSet[ubuntuVersion]; ok {
					filePath := filepath.Join(rootPath, baseName)
					// #nosec G122 -- path is from ReadDir in Fleet-controlled vuln directory, checked IsRegular above
					if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("removing old OSV artifact %s: %w", baseName, err)
					}
				}
			}
		}
	}

	return nil
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
	return fmt.Sprintf("%s%s-%d-%02d-%02d.json.gz",
		OSVFilePrefix, ubuntuVersion, date.Year(), date.Month(), date.Day())
}
