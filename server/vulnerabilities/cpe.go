package vulnerabilities

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/download"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/v37/github"
	"github.com/jmoiron/sqlx"
)

const (
	owner = "fleetdm"
	repo  = "nvd"
)

type NVDRelease struct {
	Etag      string
	CreatedAt time.Time
	CPEURL    string
}

var cpeSqliteRegex = regexp.MustCompile(`^cpe-.*\.sqlite\.gz$`)

func GetLatestNVDRelease(client *http.Client) (*NVDRelease, error) {
	ghclient := github.NewClient(client)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 10})
	if err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, nil
	}

	cpeURL := ""

	// TODO: get not draft release

	for _, asset := range releases[0].Assets {
		if asset != nil {
			matched := cpeSqliteRegex.MatchString(asset.GetName())
			if !matched {
				continue
			}
			cpeURL = asset.GetBrowserDownloadURL()
		}
	}

	return &NVDRelease{
		Etag:      releases[0].GetName(),
		CreatedAt: releases[0].GetCreatedAt().Time,
		CPEURL:    cpeURL,
	}, nil
}

type syncOpts struct {
	url string
}

type CPESyncOption func(*syncOpts)

func WithCPEURL(url string) CPESyncOption {
	return func(o *syncOpts) {
		o.url = url
	}
}

const cpeDatabaseFilename = "cpe.sqlite"

// DownloadCPEDatabase downloads the CPE database from the
// latest release of github.com/fleetdm/nvd to the given dbPath.
// An alternative URL can be set via the WithCPEURL option.
//
// It won't download the database if the database has already been downloaded and
// has an mtime after the release date.
func DownloadCPEDatabase(
	vulnPath string,
	client *http.Client,
	opts ...CPESyncOption,
) error {
	var o syncOpts
	for _, fn := range opts {
		fn(&o)
	}

	dbPath := filepath.Join(vulnPath, cpeDatabaseFilename)

	if o.url == "" {
		nvdRelease, err := GetLatestNVDRelease(client)
		if err != nil {
			return err
		}
		stat, err := os.Stat(dbPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		} else if !nvdRelease.CreatedAt.After(stat.ModTime()) {
			return nil
		}
		o.url = nvdRelease.CPEURL
	}

	u, err := url.Parse(o.url)
	if err != nil {
		return err
	}
	if err := download.DownloadAndExtract(client, u, dbPath); err != nil {
		return err
	}

	return nil
}

type IndexedCPEItem struct {
	ID         int     `json:"id" db:"rowid"`
	Title      string  `json:"title" db:"title"`
	Version    *string `json:"version" db:"version"`
	TargetSW   *string `json:"target_sw" db:"target_sw"`
	CPE23      string  `json:"cpe23" db:"cpe23"`
	Deprecated bool    `json:"deprecated" db:"deprecated"`
}

func cleanAppName(appName string) string {
	return strings.TrimSuffix(appName, ".app")
}

var onlyAlphaNumeric = regexp.MustCompile("[^a-zA-Z0-9]+")

func CPEFromSoftware(db *sqlx.DB, software *fleet.Software) (string, error) {
	targetSW := ""
	switch software.Source {
	case "apps":
		targetSW = "macos"
	case "python_packages":
		targetSW = "python"
	case "chrome_extensions":
		targetSW = "chrome"
	case "firefox_addons":
		targetSW = "firefox"
	case "safari_extensions":
		targetSW = "safari"
	case "deb_packages":
	case "portage_packages":
	case "rpm_packages":
	case "npm_packages":
		targetSW = `"node.js"`
	case "atom_packages":
	case "programs":
		targetSW = `"windows*"`
	case "ie_extensions":
	case "chocolatey_packages":
	}

	checkTargetSW := ""
	args := []interface{}{onlyAlphaNumeric.ReplaceAllString(cleanAppName(software.Name), " ")}
	if targetSW != "" {
		checkTargetSW = " AND target_sw MATCH ?"
		args = append(args, targetSW)
	}
	args = append(args, software.Version)

	query := fmt.Sprintf(
		`SELECT rowid, * FROM cpe WHERE rowid in (
				  SELECT rowid FROM cpe_search WHERE title MATCH ?%s
				) and version=? order by deprecated asc`,
		checkTargetSW,
	)
	var indexedCPEs []IndexedCPEItem
	err := db.Select(&indexedCPEs, query, args...)
	if err != nil {
		return "", fmt.Errorf("getting cpes for: %s: %w", cleanAppName(software.Name), err)
	}

	for _, item := range indexedCPEs {
		if !item.Deprecated {
			return item.CPE23, nil
		}

		deprecatedItem := item
		for {
			var deprecation IndexedCPEItem

			err = db.Get(
				&deprecation,
				`SELECT rowid, * FROM cpe c WHERE cpe23 in (
						SELECT cpe23 from deprecated_by d where d.cpe_id=?
					)`,
				deprecatedItem.ID,
			)
			if err != nil {
				return "", fmt.Errorf("getting deprecation: %w", err)
			}
			if deprecation.Deprecated {
				deprecatedItem = deprecation
				continue
			}

			return deprecation.CPE23, nil
		}
	}

	return "", nil
}

func TranslateSoftwareToCPE(
	ctx context.Context,
	ds fleet.Datastore,
	vulnPath string,
	logger kitlog.Logger,
) error {
	dbPath := filepath.Join(vulnPath, cpeDatabaseFilename)

	iterator, err := ds.AllSoftwareWithoutCPEIterator(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "all software iterator")
	}
	defer iterator.Close()

	db, err := sqliteDB(dbPath)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "opening the cpe db")
	}
	defer db.Close()

	for iterator.Next() {
		software, err := iterator.Value()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting value from iterator")
		}
		cpe, err := CPEFromSoftware(db, software)
		if err != nil {
			level.Error(logger).Log("software->cpe", "error translating to CPE, skipping...", "err", err)
			continue
		}
		if cpe == "" {
			// The schema for storing CVEs requires that a CPE for every software exists,
			// having that constraint in place works fine when the only source for vulnerabilities
			// is the NVD dataset but breaks down when we look at other sources for vulnerabilities (like OVAL) - this is
			// why we set a default value for CPEs.
			cpe = fmt.Sprintf("none:%d", software.ID)
		}
		err = ds.AddCPEForSoftware(ctx, *software, cpe)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting cpe")
		}
	}

	return nil
}
