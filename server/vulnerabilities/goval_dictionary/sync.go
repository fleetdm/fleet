package goval_dictionary

import (
	"fmt"
	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/nvd"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"net/http"
	"net/url"
	"path/filepath"
)

func Refresh(
	versions *fleet.OSVersions,
	vulnPath string,
	logger kitlog.Logger,
) ([]oval.Platform, error) {
	toDownload := whatToDownload(versions)
	if len(toDownload) > 0 {
		level.Debug(logger).Log("msg", "goval_dictionary-sync-downloading")
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
		if err := downloadDatabase(platform, dwn, basePath, dstDir); err != nil {
			return err
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
	if err := downloader(basePath+string(platform)+".sqlite3.xz", dstPath); err != nil {
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

		if err = download.DownloadAndExtract(client, parsedUrl, dstPath); err != nil {
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
