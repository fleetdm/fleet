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

// RefreshAll downloads every Ubuntu and RHEL OSV artifact present in the latest
// release, without filtering by host inventory. It is intended for use by tools
// that pre-seed a vulnerability directory without DB access (e.g.
// `fleetctl vulnerability-data-stream`). Unlike Refresh / RefreshRHEL, it does
// not delete older artifacts — that responsibility stays with the server's
// vulnerability cron once the directory is in use.
func RefreshAll(ctx context.Context, vulnPath string) ([]string, error) {
	release, err := getLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	releaseDate, ok := releaseDateFromAssets(release)
	if !ok {
		return nil, fmt.Errorf("no OSV artifacts found in latest release %q", release.TagName)
	}

	ubuntuVers, rhelVers := versionsFromRelease(release)

	var downloaded []string
	if len(ubuntuVers) > 0 {
		result, err := SyncOSV(ctx, vulnPath, ubuntuVers, releaseDate, release)
		if err != nil {
			return downloaded, fmt.Errorf("syncing Ubuntu OSV artifacts: %w", err)
		}
		downloaded = append(downloaded, result.Downloaded...)
		if len(result.Failed) > 0 {
			return downloaded, fmt.Errorf("failed to download OSV for Ubuntu versions: %v", result.Failed)
		}
	}

	if len(rhelVers) > 0 {
		result, err := syncRHELOSV(ctx, vulnPath, rhelVers, releaseDate, release)
		if err != nil {
			return downloaded, fmt.Errorf("syncing RHEL OSV artifacts: %w", err)
		}
		downloaded = append(downloaded, result.Downloaded...)
		if len(result.Failed) > 0 {
			return downloaded, fmt.Errorf("failed to download OSV for RHEL versions: %v", result.Failed)
		}
	}

	return downloaded, nil
}

// versionsFromRelease returns the Ubuntu and RHEL versions present in a
// release's OSV assets. Asset names look like `osv-ubuntu-2204-2026-04-27.json.gz`
// or `osv-rhel-9-2026-04-27.json.gz`.
func versionsFromRelease(release *ReleaseInfo) (ubuntu []string, rhel []string) {
	for assetName := range release.Assets {
		switch {
		case strings.HasPrefix(assetName, OSVFilePrefix):
			if v := versionFromAssetName(assetName, OSVFilePrefix); v != "" {
				ubuntu = append(ubuntu, v)
			}
		case strings.HasPrefix(assetName, OSVRHELFilePrefix):
			if v := versionFromAssetName(assetName, OSVRHELFilePrefix); v != "" {
				rhel = append(rhel, v)
			}
		}
	}
	return ubuntu, rhel
}

// versionFromAssetName extracts the version segment from an OSV asset filename.
// e.g. ("osv-ubuntu-2204-2026-04-27.json.gz", "osv-ubuntu-") -> "2204".
func versionFromAssetName(name, prefix string) string {
	if !strings.HasPrefix(name, prefix) {
		return ""
	}
	rest := name[len(prefix):]
	idx := strings.Index(rest, "-")
	if idx <= 0 {
		return ""
	}
	return rest[:idx]
}

// releaseDateFromAssets returns the date encoded in any OSV asset filename in
// the release. All assets in a given release share the same date.
func releaseDateFromAssets(release *ReleaseInfo) (time.Time, bool) {
	for name := range release.Assets {
		if d, ok := dateFromAssetName(name); ok {
			return d, true
		}
	}
	return time.Time{}, false
}

// dateFromAssetName extracts the YYYY-MM-DD date suffix from an OSV asset filename.
func dateFromAssetName(name string) (time.Time, bool) {
	const layout = "2006-01-02"
	s := strings.TrimSuffix(name, ".json.gz")
	if len(s) < len(layout) {
		return time.Time{}, false
	}
	t, err := time.Parse(layout, s[len(s)-len(layout):])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
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
