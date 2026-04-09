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
	// OSVFilePrefix is the prefix for Ubuntu OSV artifact files
	OSVFilePrefix = "osv-ubuntu-"
	// OSVRHELFilePrefix is the prefix for RHEL OSV artifact files
	OSVRHELFilePrefix = "osv-rhel-"
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
		if strings.ToLower(os.Platform) != "ubuntu" {
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

// rhelOSVFilename generates the RHEL OSV artifact filename for a given major version and date.
// Format: osv-rhel-9-2026-04-08.json.gz
func rhelOSVFilename(rhelVersion string, date time.Time) string {
	return fmt.Sprintf("%s%s-%d-%02d-%02d.json.gz",
		OSVRHELFilePrefix, rhelVersion, date.Year(), date.Month(), date.Day())
}

// RefreshRHEL checks local RHEL OSV artifacts, deleting outdated ones and downloading the latest.
func RefreshRHEL(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
	now time.Time,
) ([]string, error) {
	neededVersions := getNeededRHELVersions(versions)
	if len(neededVersions) == 0 {
		return nil, nil
	}

	release, err := getLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	syncResult, err := syncRHELOSV(ctx, vulnPath, neededVersions, now, release)
	if err != nil {
		return nil, fmt.Errorf("syncing RHEL OSV artifacts: %w", err)
	}

	upToDateVersions := make([]string, 0, len(syncResult.Downloaded)+len(syncResult.Skipped))
	upToDateVersions = append(upToDateVersions, syncResult.Downloaded...)
	upToDateVersions = append(upToDateVersions, syncResult.Skipped...)
	if err := removeOldRHELOSVArtifacts(now, vulnPath, upToDateVersions); err != nil {
		return syncResult.Downloaded, fmt.Errorf("warning: failed to clean up old RHEL OSV artifacts: %w", err)
	}

	return syncResult.Downloaded, nil
}

// syncRHELOSV downloads RHEL OSV artifacts for the given versions.
func syncRHELOSV(
	ctx context.Context,
	dstDir string,
	rhelVersions []string,
	date time.Time,
	release *ReleaseInfo,
) (*SyncResult, error) {
	return syncOSVWithDownloader(ctx, dstDir, rhelVersions, date, release, downloadOSVArtifact, rhelOSVFilename)
}

// getNeededRHELVersions extracts unique RHEL major versions from OS versions.
func getNeededRHELVersions(osVers *fleet.OSVersions) []string {
	seen := make(map[string]struct{})
	var needed []string

	for _, osVer := range osVers.OSVersions {
		if strings.ToLower(osVer.Platform) != "rhel" {
			continue
		}

		// Fedora reports platform "rhel" but Red Hat OSV data does not cover Fedora.
		// Fedora hosts will continue using OVAL for vulnerability scanning.
		if strings.Contains(osVer.Name, "Fedora") {
			continue
		}

		rhelVer := extractRHELMajorVersion(osVer.Version)
		if rhelVer == "" {
			continue
		}

		if _, exists := seen[rhelVer]; !exists {
			seen[rhelVer] = struct{}{}
			needed = append(needed, rhelVer)
		}
	}

	return needed
}

// removeOldRHELOSVArtifacts removes old RHEL OSV artifacts that don't match today's date.
func removeOldRHELOSVArtifacts(date time.Time, rootPath string, successfulVersions []string) error {
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
		if entry.IsDir() || !entry.Type().IsRegular() {
			continue
		}

		baseName := entry.Name()

		if !strings.HasPrefix(baseName, OSVRHELFilePrefix) {
			continue
		}

		if strings.HasSuffix(baseName, ".json.gz") && !strings.Contains(baseName, "delta") {
			if !strings.HasSuffix(baseName, dateSuffix) {
				versionStart := len(OSVRHELFilePrefix)
				versionEnd := strings.Index(baseName[versionStart:], "-")
				if versionEnd == -1 {
					continue
				}
				rhelVersion := baseName[versionStart : versionStart+versionEnd]

				if _, ok := successfulSet[rhelVersion]; ok {
					filePath := filepath.Join(rootPath, baseName)
					// #nosec G122 -- path is from ReadDir in Fleet-controlled vuln directory, checked IsRegular above
					if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
						return fmt.Errorf("removing old RHEL OSV artifact %s: %w", baseName, err)
					}
				}
			}
		}
	}

	return nil
}
