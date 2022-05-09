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
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"

	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v37/github"
)

func ghNvdFileGetter(client *http.Client) func(string) (io.ReadCloser, error) {
	return func(file string) (io.ReadCloser, error) {
		src, r, err := github.NewClient(client).Repositories.DownloadContents(
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
			return err
		}
		return download.Decompressed(client, *parsedUrl, dstPath)
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

// Walks 'path' removing any old oval definitions, returns a set containing
// definitions that are up to date according to 'date'
func removeOldDefs(date time.Time, path string) (map[string]bool, error) {
	dateSuffix := fmt.Sprintf("_%d-%d-%d.json", date.Year(), date.Month(), date.Day())
	upToDate := make(map[string]bool)

	err := filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if strings.HasPrefix(filepath.Base(path), OvalFilePrefix) {
			if strings.HasSuffix(path, dateSuffix) {
				upToDate[path] = true
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

// Syncs the oval definitions for one or more platforms.
// If 'platform' is nil, then all supported platforms will be synched.
func Sync(client *http.Client, dstDir string, platforms []Platform) error {
	sources, err := getOvalSources(ghNvdFileGetter(client))
	if err != nil {
		return err
	}

	if platforms == nil {
		for s := range sources {
			platforms = append(platforms, s)
		}
	}

	dwn := downloadDecompressed(client)
	for _, platform := range platforms {
		defFile, err := downloadDefinitions(sources, platform, dwn)
		if err != nil {
			return err
		}

		dstFile := strings.Replace(filepath.Base(defFile), ".xml", ".json", 1)
		dstPath := filepath.Join(dstDir, dstFile)
		err = parseDefinitions(defFile, dstPath)
		if err != nil {
			return err
		}

		err = os.Remove(defFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// Called from 'cron', updates the oval definitions if data sync is enabled.
// OVAL files will only be updated once per day.
// Running this will delete out of date oval definitions.
func AutoSync(
	ctx context.Context,
	client *http.Client,
	ds fleet.Datastore,
	logger kitlog.Logger,
	vulnPath string,
	config config.FleetConfig,
) error {
	if config.Vulnerabilities.DisableDataSync {
		return nil
	}

	today := time.Now()
	existing, err := removeOldDefs(today, vulnPath)
	if err != nil {
		return err
	}

	osVersions, err := ds.OSVersions(ctx, nil, nil)
	if err != nil {
		return err
	}
	level.Debug(logger).Log("oval-updating", "Found OS Versions", len(osVersions.OSVersions))

	toDownload := whatToDownload(osVersions, existing, today)
	level.Debug(logger).Log("oval-updating", "Downloading new definitions", len(toDownload))

	err = Sync(client, vulnPath, toDownload)
	if err != nil {
		return err
	}

	return nil
}
