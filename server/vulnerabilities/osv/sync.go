package osv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
)

const (
	// OSVFilePrefix is the prefix for Ubuntu OSV artifact files
	OSVFilePrefix = "osv-ubuntu-"
	// OSVRHELFilePrefix is the prefix for RHEL OSV artifact files
	OSVRHELFilePrefix = "osv-rhel-"
	// OSVAndroidFilePrefix is the prefix for Android OSV artifact files
	OSVAndroidFilePrefix = "osv-android-"
)

// osvFilePrefixes lists every OSV artifact family Fleet mirrors.
var osvFilePrefixes = []string{OSVFilePrefix, OSVRHELFilePrefix, OSVAndroidFilePrefix}

// Refresh checks all local Ubuntu OSV artifacts contained in 'vulnPath', deleting outdated
// artifacts and downloading the latest required ones.
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

	release, err := vulnrepo.LatestRelease(ctx, isOSVReleaseAsset)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	return refreshArtifacts(ctx, vulnPath, OSVFilePrefix, neededVersions, now, release)
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

	release, err := vulnrepo.LatestRelease(ctx, isOSVReleaseAsset)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	return refreshArtifacts(ctx, vulnPath, OSVRHELFilePrefix, neededVersions, now, release)
}

// RefreshAndroid checks local Android OSV artifacts, deleting outdated ones and downloading the latest.
func RefreshAndroid(
	ctx context.Context,
	oses []fleet.OperatingSystem,
	vulnPath string,
) ([]string, error) {
	neededVersions := getNeededAndroidVersions(oses)
	if len(neededVersions) == 0 {
		return nil, nil
	}

	release, err := vulnrepo.LatestRelease(ctx, isOSVReleaseAsset)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	// Artifact filenames encode the release date (all assets in a release share
	// it), so we derive the date from the release rather than the cron execution
	// time. Using "now" would build filenames that don't match the release assets
	// on any day the cron runs after the release was cut.
	releaseDate, ok := releaseDateFromAssets(release)
	if !ok {
		return nil, fmt.Errorf("no OSV artifacts found in latest release %q", release.TagName)
	}

	return refreshArtifacts(ctx, vulnPath, OSVAndroidFilePrefix, neededVersions, releaseDate, release)
}

// refreshArtifacts downloads the artifacts of one OSV family for versions, then removes the
// older artifacts of every version that is now up to date. Versions whose download failed or
// that were not in the release keep their last-known-good artifact.
func refreshArtifacts(
	ctx context.Context,
	vulnPath string,
	prefix string,
	versions []string,
	date time.Time,
	release *vulnrepo.Release,
) ([]string, error) {
	syncResult, err := syncArtifacts(ctx, vulnPath, prefix, versions, date, release)
	if err != nil {
		return nil, fmt.Errorf("syncing %s* artifacts: %w", prefix, err)
	}

	upToDateVersions := make([]string, 0, len(syncResult.Downloaded)+len(syncResult.Skipped))
	upToDateVersions = append(upToDateVersions, syncResult.Downloaded...)
	upToDateVersions = append(upToDateVersions, syncResult.Skipped...)
	if err := removeOldArtifacts(prefix, date, vulnPath, upToDateVersions); err != nil {
		return syncResult.Downloaded, fmt.Errorf("warning: failed to clean up old %s* artifacts: %w", prefix, err)
	}

	return syncResult.Downloaded, nil
}

// RefreshAll downloads every OSV artifact present in the latest release, without
// filtering by host inventory. It is intended for use by tools that pre-seed a
// vulnerability directory without DB access (e.g. `fleetctl vulnerability-data-stream`).
// Unlike Refresh / RefreshRHEL / RefreshAndroid, it does not delete older artifacts —
// that responsibility stays with the server's vulnerability cron once the directory
// is in use.
func RefreshAll(ctx context.Context, vulnPath string) ([]string, error) {
	release, err := vulnrepo.LatestRelease(ctx, isOSVReleaseAsset)
	if err != nil {
		return nil, fmt.Errorf("getting latest release: %w", err)
	}

	releaseDate, ok := releaseDateFromAssets(release)
	if !ok {
		return nil, fmt.Errorf("no OSV artifacts found in latest release %q", release.TagName)
	}

	versions := versionsFromRelease(release)

	var downloaded []string
	for _, prefix := range osvFilePrefixes {
		if len(versions[prefix]) == 0 {
			continue
		}
		result, err := syncArtifacts(ctx, vulnPath, prefix, versions[prefix], releaseDate, release)
		if err != nil {
			return downloaded, fmt.Errorf("syncing %s* artifacts: %w", prefix, err)
		}
		downloaded = append(downloaded, result.Downloaded...)
		if len(result.Failed) > 0 {
			return downloaded, fmt.Errorf("failed to download %s* artifacts for versions: %v", prefix, result.Failed)
		}
	}

	return downloaded, nil
}

// versionsFromRelease returns the versions present in a release's OSV assets, keyed by artifact
// prefix. Asset names look like `osv-ubuntu-2204-2026-04-27.json.gz`,
// `osv-rhel-9-2026-04-27.json.gz`, or `osv-android-16-2026-07-14.json.gz`.
func versionsFromRelease(release *vulnrepo.Release) map[string][]string {
	versions := make(map[string][]string)
	for assetName := range release.Assets {
		for _, prefix := range osvFilePrefixes {
			if v := versionFromAssetName(assetName, prefix); v != "" {
				versions[prefix] = append(versions[prefix], v)
				break
			}
		}
	}
	return versions
}

// neededVersions returns the distinct, non-empty results of version over items, in first-seen
// order. version returns "" for an item that does not belong to the family.
func neededVersions[T any](items []T, version func(T) string) []string {
	seen := make(map[string]struct{})
	var needed []string
	for _, item := range items {
		v := version(item)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		needed = append(needed, v)
	}
	return needed
}

// getNeededUbuntuVersions extracts unique Ubuntu versions from OS versions
func getNeededUbuntuVersions(osVers *fleet.OSVersions) []string {
	return neededVersions(osVers.OSVersions, func(os fleet.OSVersion) string {
		if strings.ToLower(os.Platform) != "ubuntu" {
			return ""
		}
		// Extract Ubuntu version (e.g., "22.04.8 LTS" -> "2204")
		return extractUbuntuVersion(os.Version)
	})
}

// getNeededRHELVersions extracts unique RHEL major versions from OS versions.
func getNeededRHELVersions(osVers *fleet.OSVersions) []string {
	return neededVersions(osVers.OSVersions, func(os fleet.OSVersion) string {
		if strings.ToLower(os.Platform) != "rhel" {
			return ""
		}
		// Fedora reports platform "rhel" but Red Hat OSV data does not cover Fedora.
		// Fedora hosts will continue using OVAL for vulnerability scanning.
		if strings.Contains(os.Name, "Fedora") {
			return ""
		}
		return extractRHELMajorVersion(os.Version)
	})
}

func getNeededAndroidVersions(oses []fleet.OperatingSystem) []string {
	return neededVersions(oses, func(os fleet.OperatingSystem) string {
		if os.Platform != "android" {
			return ""
		}
		return extractAndroidMajorVersion(os.Version)
	})
}

func extractAndroidMajorVersion(version string) string {
	if version == "" {
		return ""
	}
	// "16 (2026-05-01)" -> "16"
	// "16" -> "16"
	if idx := strings.Index(version, " "); idx > 0 {
		return version[:idx]
	}
	return version
}
