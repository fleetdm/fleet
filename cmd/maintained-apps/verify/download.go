package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	mdm_maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

const downloadTimeout = 5 * time.Minute

// downloader fetches installers to disk and memoizes results by URL so
// re-runs and same-URL entries don't download twice.
type downloader struct {
	client *http.Client
	tmpDir string
	cache  map[string]*downloaded
}

type downloaded struct {
	path   string
	sha256 string
	err    error
}

func newDownloader(tmpDir string) *downloader {
	return &downloader{
		client: fleethttp.NewClient(fleethttp.WithTimeout(downloadTimeout)),
		tmpDir: tmpDir,
		cache:  make(map[string]*downloaded),
	}
}

// fetch downloads the installer at url (through the same
// maintainedapps.DownloadInstaller path production and the validator use),
// saves it under the downloader's temp dir, and returns the on-disk path and
// the SHA256 of the downloaded bytes.
func (d *downloader) fetch(ctx context.Context, url string) (string, string, error) {
	if cached, ok := d.cache[url]; ok {
		return cached.path, cached.sha256, cached.err
	}

	path, sha, err := d.download(ctx, url)
	d.cache[url] = &downloaded{path: path, sha256: sha, err: err}
	return path, sha, err
}

func (d *downloader) download(ctx context.Context, url string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()

	installerTFR, filename, err := mdm_maintained_apps.DownloadInstaller(ctx, url, d.client)
	if err != nil {
		return "", "", fmt.Errorf("downloading installer: %w", err)
	}
	defer installerTFR.Close()

	// Keep the real filename (in a unique subdirectory to avoid collisions):
	// signature-verification tooling keys off the file extension.
	cleanFilename := filepath.Base(filename)
	if cleanFilename == "." || cleanFilename == ".." || cleanFilename == string(filepath.Separator) {
		cleanFilename = "installer"
	}
	dir, err := os.MkdirTemp(d.tmpDir, "app-")
	if err != nil {
		return "", "", fmt.Errorf("creating download directory: %w", err)
	}
	filePath := filepath.Join(dir, cleanFilename)

	out, err := os.Create(filePath)
	if err != nil {
		return "", "", fmt.Errorf("creating file: %w", err)
	}
	defer out.Close()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, h), installerTFR); err != nil {
		out.Close()
		os.RemoveAll(dir) // don't leave a partial download on disk
		return "", "", fmt.Errorf("saving installer: %w", err)
	}

	return filePath, hex.EncodeToString(h.Sum(nil)), nil
}

// evict drops a URL's cache entry and removes its downloaded file. Used by
// full-catalog runs to bound disk usage.
func (d *downloader) evict(url string) {
	cached, ok := d.cache[url]
	if !ok {
		return
	}
	delete(d.cache, url)
	if cached.path != "" {
		os.RemoveAll(filepath.Dir(cached.path))
	}
}
