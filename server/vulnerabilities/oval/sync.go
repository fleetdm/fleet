package oval

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"

	"github.com/google/go-github/v37/github"
)

func ghNvdFileGetter() func(string) (io.ReadCloser, error) {
	ghClient := fleethttp.NewGithubClient()
	return func(file string) (io.ReadCloser, error) {
		src, r, err := github.NewClient(ghClient).Repositories.DownloadContents(
			context.Background(), "fleetdm", "nvd", file, nil)
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

func whatToDownload(osVers *fleet.OSVersions, existing map[string]bool, date time.Time) []Platform {
	var r []Platform
	for _, os := range osVers.OSVersions {
		platform := NewPlatform(os.Platform, os.Name)
		_, ok := existing[platform.ToFilename(date, "json")]
		if !ok && platform.IsSupported() {
			r = append(r, platform)
		}
	}
	return r
}

// removeOldDefs walks 'path' removing any old oval definitions, returns a set containing
// definitions that are up to date according to 'date'
func removeOldDefs(date time.Time, path string) (map[string]bool, error) {
	dateSuffix := fmt.Sprintf("-%d_%02d_%02d.json", date.Year(), date.Month(), date.Day())
	upToDate := make(map[string]bool)

	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if strings.HasPrefix(filepath.Base(path), OvalFilePrefix) {
			if strings.HasSuffix(path, dateSuffix) {
				upToDate[filepath.Base(path)] = true
			} else {
				err := os.Remove(path)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return upToDate, nil
}

// Sync syncs the oval definitions for one or more platforms.
// If 'platforms' is nil, then all supported platforms will be synched.
func Sync(dstDir string, platforms []Platform) error {
	sources, err := getOvalSources(ghNvdFileGetter())
	if err != nil {
		return fmt.Errorf("getOvalSources: %w", err)
	}

	if platforms == nil {
		for s := range sources {
			platforms = append(platforms, s)
		}
	}

	client := fleethttp.NewClient()
	dwn := downloadDecompressed(client)
	for _, platform := range platforms {
		defFile, err := downloadDefinitions(sources, platform, dwn)
		if err != nil {
			return fmt.Errorf("downloadDefinitions: %w", err)
		}

		dstFile := strings.Replace(filepath.Base(defFile), ".xml", ".json", 1)
		dstPath := filepath.Join(dstDir, dstFile)
		err = parseDefinitions(platform, defFile, dstPath)
		if err != nil {
			return fmt.Errorf("parseDefinitions: %w", err)
		}

		err = os.Remove(defFile)
		if err != nil {
			return fmt.Errorf("removing %s: %w", defFile, err)
		}
	}
	return nil
}

// Refresh checks all local OVAL artifacts contained in 'vulnPath' deleting the old and downloading
// any missing definitions based on today's date and all the hosts' platforms/os versions contained in 'osVersions'.
// Returns a slice of Platforms of the newly downloaded OVAL files.
func Refresh(
	ctx context.Context,
	versions *fleet.OSVersions,
	vulnPath string,
) ([]Platform, error) {
	now := time.Now()

	existing, err := removeOldDefs(now, vulnPath)
	if err != nil {
		return nil, err
	}

	toDownload := whatToDownload(versions, existing, now)
	if len(toDownload) > 0 {
		err = Sync(vulnPath, toDownload)
		if err != nil {
			return nil, err
		}
	}

	return toDownload, nil
}
