// Package vulnrepo downloads vulnerability data that Fleet mirrors as release assets in the
// fleetdm/vulnerabilities repository.
//
// Fleet servers never fetch upstream vulnerability databases directly. A publisher job in that
// repository pulls each upstream source and attaches the result to a release, which keeps a
// single egress point for restricted deployments. Each data source owns its own asset naming
// and sync policy; this package only knows how to find a release's assets, download one, and
// verify it.
package vulnrepo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/google/go-github/v37/github"
)

const (
	githubOwner = "fleetdm"
	githubRepo  = "vulnerabilities"
)

// Asset is a release asset Fleet can download.
type Asset struct {
	Name string
	ID   int64
	// Digest is the asset's checksum as GitHub reports it, prefixed with its algorithm
	// ("sha256:..."). It is empty for assets uploaded before GitHub began reporting one.
	Digest string
}

// Release is a release and the subset of its assets a caller asked to keep, by asset name.
type Release struct {
	TagName string
	Assets  map[string]*Asset
}

// rawRelease and rawAsset mirror the GitHub API's release payload. The release listing is
// decoded here rather than through the go-github client because the vendored version (v37)
// predates the `digest` field on release assets, and that field is what makes it possible to
// tell a complete download from a truncated one.
type rawAsset struct {
	Name   string `json:"name"`
	ID     int64  `json:"id"`
	Digest string `json:"digest"`
}

type rawRelease struct {
	TagName string     `json:"tag_name"`
	Assets  []rawAsset `json:"assets"`
}

// LatestRelease returns the repository's latest release, keeping only the assets keep reports
// true for. A release carries assets for every data source Fleet mirrors, so a caller that
// does not filter will see all of them.
func LatestRelease(ctx context.Context, keep func(assetName string) bool) (*Release, error) {
	httpClient := fleethttp.NewClient()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", githubOwner, githubRepo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching latest release")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ctxerr.Errorf(ctx, "github http status error: %d", resp.StatusCode)
	}

	var raw rawRelease
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding release")
	}
	if raw.TagName == "" {
		return nil, ctxerr.New(ctx, "release tag name is empty")
	}

	assets := make(map[string]*Asset)
	for _, asset := range raw.Assets {
		if !isPlainFileName(asset.Name) {
			continue
		}
		if keep != nil && !keep(asset.Name) {
			continue
		}
		assets[asset.Name] = &Asset{Name: asset.Name, ID: asset.ID, Digest: asset.Digest}
	}

	return &Release{TagName: raw.TagName, Assets: assets}, nil
}

// isPlainFileName reports whether name is usable as a file name on its own. Callers join an
// asset name onto their databases path, so a name carrying a path separator would write
// outside it, and names are whatever the release publisher uploaded.
func isPlainFileName(name string) bool {
	// filepath.Base leaves "." and ".." untouched, so they pass the round-trip check and
	// have to be rejected on their own.
	if name == "" || name == "." || name == ".." {
		return false
	}
	return name == filepath.Base(name)
}

// DownloadAsset writes a release asset to dstPath, following the redirect GitHub answers asset
// downloads with. The caller is responsible for removing dstPath if the download fails.
func DownloadAsset(ctx context.Context, assetID int64, dstPath string) error {
	client := github.NewClient(fleethttp.NewGithubClient())

	httpClient := fleethttp.NewClient()
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(ctx, githubOwner, githubRepo, assetID, httpClient)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading release asset")
	}

	if redirectURL != "" {
		if rc != nil {
			rc.Close()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, redirectURL, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating redirect request")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "downloading from redirect URL")
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return ctxerr.Errorf(ctx, "download http status error: %d", resp.StatusCode)
		}
		rc = resp.Body
	}

	if rc == nil {
		return ctxerr.New(ctx, "release asset download returned no content")
	}
	defer rc.Close()

	// #nosec G304 -- dstPath is built by the caller from a Fleet-controlled vuln directory
	out, err := os.Create(dstPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating destination file")
	}
	defer out.Close()

	if _, err := io.Copy(out, rc); err != nil {
		return ctxerr.Wrap(ctx, err, "writing to destination file")
	}

	return nil
}

// DownloadFunc writes the release asset with the given ID to dstPath. DownloadAsset is the
// real implementation; the indirection exists so callers can substitute a fake in tests.
type DownloadFunc func(ctx context.Context, assetID int64, dstPath string) error

// EnsureAsset makes dstPath match asset, downloading it if it does not already, and reports
// whether a download happened. A failed or corrupted download leaves no file behind: a
// truncated artifact reads as a database with fewer entries, which analyzers interpret as
// entries having been remediated.
//
// An asset with no digest is always re-downloaded, because a matching file name alone is no
// evidence the local copy is complete.
func EnsureAsset(ctx context.Context, asset *Asset, dstPath string, download DownloadFunc) (bool, error) {
	if _, err := os.Stat(dstPath); err == nil && asset.Digest != "" {
		local, err := FileSHA256(dstPath)
		if err == nil && local == asset.Digest {
			return false, nil
		}
	}

	if err := download(ctx, asset.ID, dstPath); err != nil {
		os.Remove(dstPath)
		return false, ctxerr.Wrapf(ctx, err, "downloading %s", asset.Name)
	}

	if asset.Digest != "" {
		local, err := FileSHA256(dstPath)
		if err != nil {
			os.Remove(dstPath)
			return false, ctxerr.Wrapf(ctx, err, "checksumming %s", asset.Name)
		}
		if local != asset.Digest {
			os.Remove(dstPath)
			return false, ctxerr.Errorf(ctx, "checksum mismatch for %s", asset.Name)
		}
	}

	return true, nil
}

// FileSHA256 returns a file's digest in the same format Asset.Digest carries, so the two can
// be compared directly.
func FileSHA256(path string) (string, error) {
	// #nosec G304 -- path is built by the caller from a Fleet-controlled vuln directory
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

// NewestLocalAsset returns the path of the newest regular file in dir whose name match
// accepts, or "" when there is none. Newest is by name: mirrored assets carry their release
// date as YYYY-MM-DD, so lexical order is chronological, whereas a re-download of an older
// asset would make modification time lie.
func NewestLocalAsset(dir string, match func(name string) bool) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", dir, err)
	}

	var newest string
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !match(entry.Name()) || entry.Name() <= newest {
			continue
		}
		newest = entry.Name()
	}
	if newest == "" {
		return "", nil
	}
	return filepath.Join(dir, newest), nil
}

// RemoveLocalAssets deletes every regular file in dir whose name match accepts. Every data
// source shares the directory, so match has to be specific to the caller's own assets.
func RemoveLocalAssets(dir string, match func(name string) bool) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	for _, entry := range entries {
		if !entry.Type().IsRegular() || !match(entry.Name()) {
			continue
		}
		// #nosec G122 -- name comes from ReadDir in a Fleet-controlled vuln directory, checked IsRegular above
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing %s: %w", entry.Name(), err)
		}
	}
	return nil
}
