package osv

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
)

// isOSVReleaseAsset reports whether a release asset name is a non-delta OSV artifact Fleet
// consumes. New OSV ecosystems must be added to osvFilePrefixes or their assets are silently
// dropped from the release listing.
func isOSVReleaseAsset(name string) bool {
	for _, prefix := range osvFilePrefixes {
		if strings.HasPrefix(name, prefix) {
			return isArtifact(name, prefix)
		}
	}
	return false
}

type SyncResult struct {
	// Downloaded versions were fetched from the release and saved to disk.
	Downloaded []string
	// Skipped versions already had a local file with a matching checksum.
	Skipped []string
	// NotInRelease versions had no matching asset in the release (likely caused by a date-boundary).
	NotInRelease []string
	// Failed versions had an asset in the release but the download or
	// checksum verification failed.
	Failed []string
}

// downloadFunc is vulnrepo.DownloadFunc; the alias keeps the existing test seam readable.
type downloadFunc = vulnrepo.DownloadFunc

// syncArtifacts downloads the artifacts of one OSV family (identified by prefix) for the given
// versions and date from release.
func syncArtifacts(ctx context.Context, dstDir, prefix string, versions []string, date time.Time, release *vulnrepo.Release) (*SyncResult, error) {
	return syncOSVWithDownloader(ctx, dstDir, prefix, versions, date, release, vulnrepo.DownloadAsset)
}

// syncOSVWithDownloader is the internal implementation that accepts a custom download function for testing
func syncOSVWithDownloader(ctx context.Context, dstDir, prefix string, versions []string, date time.Time, release *vulnrepo.Release, download downloadFunc) (*SyncResult, error) {
	result := &SyncResult{
		Downloaded:   make([]string, 0),
		Skipped:      make([]string, 0),
		NotInRelease: make([]string, 0),
		Failed:       make([]string, 0),
	}

	for _, version := range versions {
		filename := artifactFilename(prefix, version, date)
		dstPath := filepath.Join(dstDir, filename)

		assetInfo, ok := release.Assets[filename]
		if !ok {
			result.NotInRelease = append(result.NotInRelease, version)
			continue
		}

		downloaded, err := vulnrepo.EnsureAsset(ctx, assetInfo, dstPath, download)
		switch {
		case err != nil:
			result.Failed = append(result.Failed, version)
		case downloaded:
			result.Downloaded = append(result.Downloaded, version)
		default:
			result.Skipped = append(result.Skipped, version)
		}
	}

	if len(result.Failed) > 0 && len(result.Downloaded) == 0 && len(result.Skipped) == 0 {
		return result, errors.New("all OSV artifact downloads failed")
	}

	return result, nil
}
