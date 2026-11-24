package govulndb

import (
	"compress/gzip"
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/vulnrepo"
)

// Refresh downloads the latest mirrored Go vulnerability database artifact into vulnPath and
// removes the ones it supersedes. It returns the artifact's name, or "" when the latest release
// carries none: the publisher may not have run yet, which is not an error.
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
		return "", ctxerr.Wrapf(ctx, err, "ensuring Go vulnerability database asset %s", asset.Name)
	}

	if err := vulnrepo.RemoveLocalAssets(vulnPath, func(name string) bool {
		return name != asset.Name && isArtifactAsset(name)
	}); err != nil {
		return asset.Name, ctxerr.Wrap(ctx, err, "cleaning up superseded artifacts")
	}

	return asset.Name, nil
}

// LoadArtifact reads the newest mirrored artifact in vulnPath, or nil if the directory holds
// none.
func LoadArtifact(ctx context.Context, vulnPath string) (*Artifact, error) {
	path, err := vulnrepo.NewestLocalAsset(vulnPath, isArtifactAsset)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "finding newest Go vulnerability database artifact")
	}
	if path == "" {
		return nil, nil
	}

	// #nosec G304 -- path is built from a Fleet-controlled vuln directory listing
	f, err := os.Open(path)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "opening %s", path)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "gunzipping %s", path)
	}
	defer gz.Close()

	var artifact Artifact
	if err := json.UnmarshalRead(gz, &artifact); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "decoding %s", path)
	}

	// Matching nothing would mark every stored Go vulnerability as remediated, and its toolchain
	// rules would take toolchain installs away from NVD, so refuse an unknown schema or an empty
	// module list (a corrupted or partial download).
	if artifact.SchemaVersion != SchemaVersion {
		return nil, ctxerr.Errorf(ctx, "%s has schema version %q, want %q", path, artifact.SchemaVersion, SchemaVersion)
	}
	if len(artifact.Modules) == 0 {
		return nil, ctxerr.Errorf(ctx, "%s contains no modules (possible corrupted feed)", path)
	}

	return &artifact, nil
}

// isArtifactAsset reports whether a release asset is a Go vulnerability database artifact; a
// release carries assets for every mirrored data source.
func isArtifactAsset(name string) bool {
	return strings.HasPrefix(name, FilePrefix) && strings.HasSuffix(name, FileExt)
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
