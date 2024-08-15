package goval_dictionary

import (
	"context"
	"fmt"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"net/http"
	"net/url"
	"path/filepath"
)

func Refresh(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
) ([]oval.Platform, error) {
	toDownload := whatToDownload(versions)
	if len(toDownload) > 0 {
		err := Sync(vulnPath, toDownload)
		if err != nil {
			return nil, err
		}
	}

	return toDownload, nil
}

func Sync(dstDir string, platforms []oval.Platform) error {
	client := fleethttp.NewClient()
	dwn := downloadDecompressed(client)
	basePath, err := nvd.GetGitHubCVEAssetPath()
	if err != nil {
		return err
	}

	for _, platform := range platforms {
		err := downloadDatabase(platform, dwn, basePath, dstDir)
		if err != nil {
			return fmt.Errorf("downloadDefinitions: %w", err)
		}
	}
	return nil
}

func downloadDatabase(
	platform oval.Platform,
	downloader func(string, string) error,
	basePath string,
	vulnDir string,
) error {
	dstPath := filepath.Join(vulnDir, platform.ToGovalDictionaryFilename())
	err := downloader(basePath+string(platform)+".sqlite3.xz", dstPath)
	if err != nil {
		return err
	}

	return nil
}

func downloadDecompressed(client *http.Client) func(string, string) error {
	return func(u, dstPath string) error {
		parsedUrl, err := url.Parse(u)
		if err != nil {
			return fmt.Errorf("url parse: %w", err)
		}

		err = download.DownloadAndExtract(client, parsedUrl, dstPath)
		if err != nil {
			return fmt.Errorf("download and extract url %s: %w", parsedUrl, err)
		}

		return nil
	}
}

func whatToDownload(osVers *fleet.OSVersions) []oval.Platform {
	var r []oval.Platform
	for _, os := range osVers.OSVersions {
		platform := oval.NewPlatform(os.Platform, os.Name)
		if platform.IsGovalDictionarySupported() {
			r = append(r, platform)
		}
	}

	return r
}
