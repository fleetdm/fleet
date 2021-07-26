package vulnerabilities

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/go-github/v37/github"
	"github.com/pkg/errors"
)

const (
	owner = "chiiph"
	repo  = "nvd"
)

func GetLatestNVDRelease() (string, time.Time, error) {
	ghclient := github.NewClient(nil)
	ctx := context.Background()
	releases, _, err := ghclient.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{Page: 0, PerPage: 1})
	if err != nil {
		return "", time.Time{}, err
	}

	if len(releases) == 1 && releases[0].Name != nil {
		return *releases[0].Name, releases[0].GetCreatedAt().Time, err
	}

	return "", time.Time{}, nil
}

func SyncCPEDatabase(client *http.Client, dbPath string) error {
	etag, timestamp, err := GetLatestNVDRelease()
	if err != nil {
		return err
	}

	stat, err := os.Stat(dbPath)
	if err != nil {
		if err != os.ErrNotExist {
			return err
		}
	} else {
		if !timestamp.After(stat.ModTime()) {
			return nil
		}
	}

	url := fmt.Sprintf("%s.sqlite.gz", etag)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}

	dbFile, err := os.Open(dbPath)
	if err != nil {
		return err
	}
	defer dbFile.Close()
	_, err = io.Copy(dbFile, gr)
	if err != nil {
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

func CPEFromSoftware(dbPath string, software *fleet.Software) (string, error) {
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
		targetSW = "node.js"
	case "atom_packages":
	case "programs":
		targetSW = "windows*"
	case "ie_extensions":
	case "chocolatey_packages":
	}

	db, err := CPEDB(dbPath)
	if err != nil {
		return "", errors.Wrap(err, "opening the cpe db")
	}

	checkTargetSW := ""
	args := []interface{}{cleanAppName(software.Name)}
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
	err = db.Select(&indexedCPEs, query, args...)
	if err != nil {
		return "", errors.Wrap(err, "getting cpes")
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
				return "", errors.Wrap(err, "getting deprecation")
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
