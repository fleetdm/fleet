package osv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/google/go-github/v37/github"
)

const (
	// GitHub repository information
	osvGithubOwner = "fleetdm"
	osvGithubRepo  = "vulnerabilities"
)

// ghOSVFileGetter returns a function that can fetch files from the GitHub vulnerabilities repository
func ghOSVFileGetter() func(string) (io.ReadCloser, error) {
	ghClient := fleethttp.NewGithubClient()
	return func(file string) (io.ReadCloser, error) {
		src, r, err := github.NewClient(ghClient).Repositories.DownloadContents(
			context.Background(), osvGithubOwner, osvGithubRepo, file, nil)
		if err != nil {
			return nil, err
		}

		// Even if err is nil, the request can fail
		if r.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("github http status error: %d", r.StatusCode)
		}

		return src, nil
	}
}

// getLatestReleaseTag fetches the latest release tag from the vulnerabilities repository
func getLatestReleaseTag() (string, error) {
	ghClient := fleethttp.NewGithubClient()
	client := github.NewClient(ghClient)

	// Get the latest release
	release, resp, err := client.Repositories.GetLatestRelease(
		context.Background(),
		osvGithubOwner,
		osvGithubRepo,
	)
	if err != nil {
		return "", fmt.Errorf("getting latest release: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github http status error: %d", resp.StatusCode)
	}

	if release.TagName == nil {
		return "", fmt.Errorf("release tag name is nil")
	}

	return *release.TagName, nil
}

// downloadOSVArtifact downloads a specific OSV artifact from the latest GitHub release
func downloadOSVArtifact(ubuntuVersion string, date time.Time, dstPath string) error {
	// Get the latest release tag
	releaseTag, err := getLatestReleaseTag()
	if err != nil {
		return fmt.Errorf("getting latest release tag: %w", err)
	}

	ghClient := fleethttp.NewGithubClient()
	client := github.NewClient(ghClient)

	// Get the release by tag
	release, resp, err := client.Repositories.GetReleaseByTag(
		context.Background(),
		osvGithubOwner,
		osvGithubRepo,
		releaseTag,
	)
	if err != nil {
		return fmt.Errorf("getting release by tag %s: %w", releaseTag, err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github http status error: %d", resp.StatusCode)
	}

	// Find the asset for this Ubuntu version
	assetName := osvFilename(ubuntuVersion, date)
	var assetID int64

	for _, asset := range release.Assets {
		if asset.Name != nil && *asset.Name == assetName {
			if asset.ID != nil {
				assetID = *asset.ID
				break
			}
		}
	}

	if assetID == 0 {
		return fmt.Errorf("OSV artifact not found in release: %s", assetName)
	}

	// Download the asset
	httpClient := fleethttp.NewClient()
	rc, redirectURL, err := client.Repositories.DownloadReleaseAsset(
		context.Background(),
		osvGithubOwner,
		osvGithubRepo,
		assetID,
		httpClient,
	)
	if err != nil {
		return fmt.Errorf("downloading release asset: %w", err)
	}

	// If we got a redirect URL, follow it
	if redirectURL != "" {
		resp, err := httpClient.Get(redirectURL)
		if err != nil {
			return fmt.Errorf("downloading from redirect URL: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
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

// SyncOSV downloads OSV artifacts for the specified Ubuntu versions
func SyncOSV(dstDir string, ubuntuVersions []string, date time.Time) error {
	for _, ubuntuVersion := range ubuntuVersions {
		filename := osvFilename(ubuntuVersion, date)
		dstPath := filepath.Join(dstDir, filename)

		err := downloadOSVArtifact(ubuntuVersion, date, dstPath)
		if err != nil {
			return fmt.Errorf("downloading OSV artifact for Ubuntu %s: %w", ubuntuVersion, err)
		}
	}

	return nil
}
