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
// any missing definitions based on today's date and all the hosts' platforms/os versions contained in 'versions'.
// Returns a slice of Ubuntu versions of the newly downloaded OSV files.
func Refresh(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
) ([]string, error) {
	now := time.Now()

	// Get the set of up-to-date artifacts (matching today's date)
	existing, err := getExistingOSVArtifacts(now, vulnPath)
	if err != nil {
		return nil, fmt.Errorf("checking existing OSV artifacts: %w", err)
	}

	// Determine which Ubuntu versions need to be downloaded
	toDownload := whatToDownloadOSV(versions, existing, now)
	if len(toDownload) == 0 {
		return nil, nil
	}

	// Download missing OSV artifacts
	err = SyncOSV(vulnPath, toDownload, now)
	if err != nil {
		return nil, fmt.Errorf("syncing OSV artifacts: %w", err)
	}

	err = removeOldOSVArtifacts(now, vulnPath)
	if err != nil {
		return toDownload, fmt.Errorf("warning: failed to clean up old OSV artifacts: %w", err)
	}

	return toDownload, nil
}

// getExistingOSVArtifacts checks which OSV artifacts exist that match today's date
func getExistingOSVArtifacts(date time.Time, path string) (map[string]struct{}, error) {
	dateSuffix := fmt.Sprintf("-%d-%02d-%02d.json.gz", date.Year(), date.Month(), date.Day())
	upToDate := make(map[string]struct{})

	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, OSVFilePrefix) && strings.HasSuffix(baseName, dateSuffix) {
			upToDate[baseName] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return upToDate, nil
}

// removeOldOSVArtifacts walks 'path' removing any old OSV artifacts that don't match today's date
func removeOldOSVArtifacts(date time.Time, path string) error {
	dateSuffix := fmt.Sprintf("-%d-%02d-%02d.json.gz", date.Year(), date.Month(), date.Day())

	return filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, OSVFilePrefix) && strings.HasSuffix(baseName, ".json.gz") {
			if !strings.HasSuffix(baseName, dateSuffix) {
				if err := os.Remove(path); err != nil {
					return fmt.Errorf("removing old OSV artifact %s: %w", path, err)
				}
			}
		}
		return nil
	})
}

// whatToDownloadOSV determines which Ubuntu versions need OSV artifacts downloaded
func whatToDownloadOSV(osVers *fleet.OSVersions, existing map[string]struct{}, date time.Time) []string {
	var toDownload []string
	seen := make(map[string]struct{})

	for _, os := range osVers.OSVersions {
		if !IsPlatformSupported(os.Platform) {
			continue
		}

		// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
		ubuntuVer := extractUbuntuVersion(os.Version)
		if ubuntuVer == "" {
			continue
		}

		if _, exists := seen[ubuntuVer]; exists {
			continue
		}
		seen[ubuntuVer] = struct{}{}

		// Check if we already have an up-to-date artifact for this version
		filename := osvFilename(ubuntuVer, date)
		if _, exists := existing[filename]; !exists {
			toDownload = append(toDownload, ubuntuVer)
		}
	}

	return toDownload
}

// osvFilename generates the OSV artifact filename for a given Ubuntu version and date
// Format: osv-ubuntu-2204-2026-03-30.json.gz
func osvFilename(ubuntuVersion string, date time.Time) string {
	return fmt.Sprintf("osv-ubuntu-%s-%d-%02d-%02d.json.gz",
		ubuntuVersion, date.Year(), date.Month(), date.Day())
}
