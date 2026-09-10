package govulndb

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
)

// Refresh downloads the latest mirrored Go vulnerability database artifact into vulnPath and
// removes the ones it supersedes. It returns the name of the artifact now on disk, or an empty
// string when the latest release carries none — the publisher may not have run yet, which is
// not an error.
func Refresh(ctx context.Context, vulnPath string) (string, error) {
	release, err := vulnrepo.LatestRelease(ctx, isArtifactAsset)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting latest Go vulnerability database asset")
	}

	asset := newestAsset(release)
	if asset == nil {
		return "", nil
	}

	dstPath := filepath.Join(vulnPath, asset.Name)
	if _, err := vulnrepo.EnsureAsset(ctx, asset, dstPath, vulnrepo.DownloadAsset); err != nil {
		return "", err
	}

	if err := vulnrepo.RemoveLocalAssets(vulnPath, func(name string) bool {
		return name != asset.Name && isArtifactAsset(name)
	}); err != nil {
		return asset.Name, ctxerr.Wrap(ctx, err, "cleaning up superseded artifacts")
	}

	return asset.Name, nil
}

// isArtifactAsset reports whether a release asset name is a Go vulnerability database
// artifact. A release carries assets for every data source Fleet mirrors.
func isArtifactAsset(name string) bool {
	return strings.HasPrefix(name, filePrefix) && strings.HasSuffix(name, fileExt)
}

// newestAsset returns the most recent artifact in a release, or nil if it carries none.
// Artifacts are named govulndb-YYYY-MM-DD.json.gz, so the newest sorts last.
func newestAsset(release *vulnrepo.Release) *vulnrepo.Asset {
	var newest *vulnrepo.Asset
	for name, asset := range release.Assets {
		if newest == nil || name > newest.Name {
			newest = asset
		}
	}
	return newest
}
