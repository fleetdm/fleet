package goval_dictionary

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/utils"
	"strings"
)

func NewDB(db *sql.DB, platform oval.Platform) *Database {
	return &Database{sqlite: db, platform: platform}
}

type Database struct {
	sqlite   *sql.DB
	platform oval.Platform
}

// Eval evaluates the current goval_dictionary database against an OS version and a list of installed software,
// returns all software vulnerabilities found.
func (db Database) Eval(software []fleet.Software) ([]fleet.SoftwareVulnerability, error) {
	searchStmt := `SELECT packages.version, cves.cve_id 
		FROM packages join definitions on definitions.id = packages.definition_id
		JOIN advisories ON advisories.definition_id = definitions.id JOIN cves ON cves.advisory_id = advisories.id
		WHERE packages.name = ? AND packages.arch = ?`
	vulnerabilities := make([]fleet.SoftwareVulnerability, 0)

	for _, swItem := range software {
		affectedSoftwareRows, err := db.sqlite.Query(searchStmt, swItem.Name, swItem.Arch)
		if errors.Is(err, sql.ErrNoRows) {
			continue // No vulns for this package-OS combo
		}
		if err != nil {
			return nil, fmt.Errorf("could not query goval_dictionary database for package %s", swItem.Name)
		}
		defer affectedSoftwareRows.Close()
		for affectedSoftwareRows.Next() {
			var fixedVersionWithEpochPrefix, cve string
			if err := affectedSoftwareRows.Scan(&fixedVersionWithEpochPrefix, &cve); err != nil {
				return nil, fmt.Errorf("could not read package vulnerability result %s", swItem.Name)
			} else if affectedSoftwareRows.Err() != nil {
				return nil, affectedSoftwareRows.Err()
			}

			var currentVersion string
			if swItem.Release != "" {
				currentVersion = fmt.Sprintf("%s-%s", swItem.Version, swItem.Release)
			} else {
				currentVersion = swItem.Version
			}
			fixedVersion := strings.Split(fixedVersionWithEpochPrefix, ":")[1]

			if utils.Rpmvercmp(currentVersion, fixedVersion) < 0 {
				vulnerabilities = append(vulnerabilities, fleet.SoftwareVulnerability{
					SoftwareID:        swItem.ID,
					CVE:               cve,
					ResolvedInVersion: &fixedVersion,
				})
			}
		}
	}

	return vulnerabilities, nil
}
