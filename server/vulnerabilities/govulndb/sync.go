package govulndb

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
)

// Refresh downloads the latest mirrored Go vulnerability database artifact into vulnPath and
// removes the ones it supersedes. It returns the name of the artifact now on disk, or an empty
// string when the latest release carries none — the publisher may not have run yet, which is
// not an error.
func Refresh(ctx context.Context, vulnPath string) (string, error) {
	asset, err := latestArtifactAsset(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting latest Go vulnerability database asset")
	}
	if asset == nil {
		return "", nil
	}

	dstPath := filepath.Join(vulnPath, asset.Name)

	// A digest match means the artifact on disk is byte-identical to the published one.
	// Without a digest to compare, re-download rather than trust the file name.
	upToDate := false
	if _, err := os.Stat(dstPath); err == nil && asset.Digest != "" {
		local, err := fileSHA256(dstPath)
		upToDate = err == nil && local == asset.Digest
	}

	if !upToDate {
		if err := downloadAsset(ctx, asset.ID, dstPath); err != nil {
			os.Remove(dstPath)
			return "", ctxerr.Wrapf(ctx, err, "downloading %s", asset.Name)
		}

		if asset.Digest != "" {
			local, err := fileSHA256(dstPath)
			if err != nil {
				os.Remove(dstPath)
				return "", ctxerr.Wrapf(ctx, err, "checksumming %s", asset.Name)
			}
			if local != asset.Digest {
				// A truncated or corrupted artifact would look like a database
				// with fewer advisories, which the analyzer would read as
				// advisories having been remediated.
				os.Remove(dstPath)
				return "", ctxerr.Errorf(ctx, "checksum mismatch for %s", asset.Name)
			}
		}
	}

	if err := removeSupersededArtifacts(vulnPath, asset.Name); err != nil {
		return asset.Name, ctxerr.Wrap(ctx, err, "cleaning up superseded artifacts")
	}

	return asset.Name, nil
}

// removeSupersededArtifacts deletes every Go vulnerability database artifact in vulnPath other
// than keep.
func removeSupersededArtifacts(vulnPath string, keep string) error {
	entries, err := os.ReadDir(vulnPath)
	if err != nil {
		return fmt.Errorf("reading directory %s: %w", vulnPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !entry.Type().IsRegular() {
			continue
		}
		name := entry.Name()
		if name == keep || !isArtifactAsset(name) {
			continue
		}
		// #nosec G122 -- name comes from ReadDir in a Fleet-controlled vuln directory,
		// checked IsRegular above
		if err := os.Remove(filepath.Join(vulnPath, name)); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing superseded artifact %s: %w", name, err)
		}
	}

	return nil
}
