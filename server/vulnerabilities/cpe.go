package vulnerabilities

import (
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func SyncCPEDatabase(ds fleet.Datastore) error {
	config, err := ds.AppConfig()
	if err != nil {
		return err
	}

	_ = config

	//curl \
	//  -H "Accept: application/vnd.github.v3+json" \
	//  https://api.github.com/repos/fleetdm/fleet/releases/latest

	// see https://www.sqlite.org/sqldiff.html#vtab for differentials

	// if db is not there, then download
	// if release.published_at > file stat, then download

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

func CPEFromSoftware(ds fleet.Datastore, software *fleet.Software) (string, error) {
	config, err := ds.AppConfig()
	if err != nil {
		return "", err
	}

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

	db, err := CPEDB(*config.VulnerabilityDatabasesPath)
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
