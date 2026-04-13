package osv

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/go-github/v37/github"
)

const (
	// GitHub repository information
	osvGithubOwner = "fleetdm"
	osvGithubRepo  = "vulnerabilities"
)

// AssetInfo contains metadata about a GitHub release asset
type AssetInfo struct {
	Name   string
	ID     int64
	Digest string
}

// ReleaseInfo contains metadata about a GitHub release and its assets
type ReleaseInfo struct {
	TagName string
	Assets  map[string]*AssetInfo
}

// rawAsset represents the GitHub API asset response with digest field
type rawAsset struct {
	Name   string `json:"name"`
	ID     int64  `json:"id"`
	Digest string `json:"digest"`
}

// rawRelease represents the GitHub API release response
type rawRelease struct {
	TagName string     `json:"tag_name"`
	Assets  []rawAsset `json:"assets"`
}

// getLatestRelease fetches the latest release from the vulnerabilities repository
func getLatestRelease(ctx context.Context) (*ReleaseInfo, error) {
	httpClient := fleethttp.NewClient()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", osvGithubOwner, osvGithubRepo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github http status error: %d", resp.StatusCode)
	}

	var release rawRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decoding release: %w", err)
	}

	if release.TagName == "" {
		return nil, errors.New("release tag name is empty")
	}

	assets := make(map[string]*AssetInfo)
	for _, asset := range release.Assets {
		isOSVAsset := strings.HasPrefix(asset.Name, OSVFilePrefix) || strings.HasPrefix(asset.Name, OSVRHELFilePrefix)
		if isOSVAsset && !strings.Contains(asset.Name, "delta") {
			assets[asset.Name] = &AssetInfo{
				Name:   asset.Name,
				ID:     asset.ID,
				Digest: asset.Digest,
			}
		}
	}

	return &ReleaseInfo{
		TagName: release.TagName,
		Assets:  assets,
	}, nil
}

// downloadOSVArtifact downloads a specific OSV artifact using the asset ID from ReleaseInfo
func downloadOSVArtifact(ctx context.Context, assetID int64, dstPath string) error {
	ghClient := fleethttp.NewGithubClient()
	client := github.NewClient(ghClient)

	httpClient := fleethttp.NewClient()
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(
		ctx,
		osvGithubOwner,
		osvGithubRepo,
		assetID,
		httpClient,
	)
	if err != nil {
		return fmt.Errorf("downloading release asset: %w", err)
	}

	if redirectURL != "" {
		if rc != nil {
			rc.Close()
		}

		req, err := http.NewRequestWithContext(ctx, "GET", redirectURL, nil)
		if err != nil {
			return fmt.Errorf("creating redirect request: %w", err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("downloading from redirect URL: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("download http status error: %d", resp.StatusCode)
		}

		rc = resp.Body
	}

	if rc != nil {
		defer rc.Close()

		// Write to destination file
		outFile, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("creating destination file: %w", err)
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, rc)
		if err != nil {
			return fmt.Errorf("writing to destination file: %w", err)
		}
	}

	return nil
}

// computeFileSHA256 computes the SHA256 digest of a file
func computeFileSHA256(path string) (string, error) {
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

// downloadFunc is a function that downloads an asset to a destination path
type downloadFunc func(ctx context.Context, assetID int64, dstPath string) error

// SyncOSV downloads OSV artifacts for the specified Ubuntu versions
func SyncOSV(ctx context.Context, dstDir string, versions []string, date time.Time, release *ReleaseInfo) (*SyncResult, error) {
	return syncOSVWithDownloader(ctx, dstDir, versions, date, release, downloadOSVArtifact, osvFilename)
}

type filenameFn func(version string, date time.Time) string

// syncOSVWithDownloader is the internal implementation that accepts a custom download function for testing
func syncOSVWithDownloader(ctx context.Context, dstDir string, versions []string, date time.Time, release *ReleaseInfo, download downloadFunc, nameFn filenameFn) (*SyncResult, error) {
	result := &SyncResult{
		Downloaded:   make([]string, 0),
		Skipped:      make([]string, 0),
		NotInRelease: make([]string, 0),
		Failed:       make([]string, 0),
	}

	for _, version := range versions {
		filename := nameFn(version, date)
		dstPath := filepath.Join(dstDir, filename)

		assetInfo, ok := release.Assets[filename]
		if !ok {
			result.NotInRelease = append(result.NotInRelease, version)
			continue
		}

		// Check if file exists and has matching checksum
		needsDownload := true
		if _, err := os.Stat(dstPath); err == nil {
			if assetInfo.Digest != "" {
				// If no digest available, always re-download to be safe
				localDigest, err := computeFileSHA256(dstPath)
				if err == nil && localDigest == assetInfo.Digest {
					// Checksums match, skip download
					needsDownload = false
					result.Skipped = append(result.Skipped, version)
				}
			}
		}

		if needsDownload {
			err := download(ctx, assetInfo.ID, dstPath)
			if err != nil {
				// Download failed, skip
				os.Remove(dstPath)
				result.Failed = append(result.Failed, version)
				continue
			}

			if assetInfo.Digest != "" {
				downloadedDigest, err := computeFileSHA256(dstPath)
				if err != nil {
					// Failed to compute digest, clean up and mark failed
					os.Remove(dstPath)
					result.Failed = append(result.Failed, version)
					continue
				}

				if downloadedDigest != assetInfo.Digest {
					// Checksum mismatch - corrupted download, clean up and mark failed
					os.Remove(dstPath)
					result.Failed = append(result.Failed, version)
					continue
				}
			}

			result.Downloaded = append(result.Downloaded, version)
		}
	}

	if len(result.Failed) > 0 && len(result.Downloaded) == 0 && len(result.Skipped) == 0 {
		return result, errors.New("all OSV artifact downloads failed")
	}

	return result, nil
}
