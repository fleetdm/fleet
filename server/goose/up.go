package goose

import (
	"database/sql"
	"log"
)

// Up applies every known migration that is not yet applied, in ascending
// version order.
//
// A missing migration numbered below the newest applied version (typically
// one shipped in a patch release after later-numbered migrations were already
// applied) is applied out of order, but only if its version is at or above
// MinOutOfOrderVersion. This keeps long-standing gaps on old databases from
// suddenly being applied against a much newer schema. When
// MinOutOfOrderVersion is 0, out-of-order application is disabled and only
// migrations above the newest applied version run (legacy behavior).
func (c *Client) Up(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	applied, err := c.AppliedVersions(db)
	if err != nil {
		return err
	}

	var maxApplied int64
	for v := range applied {
		if v > maxApplied {
			maxApplied = v
		}
	}

	for _, m := range migrations {
		if applied[m.Version] {
			continue
		}
		if m.Version < maxApplied && (c.MinOutOfOrderVersion == 0 || m.Version < c.MinOutOfOrderVersion) {
			log.Printf("goose: skipping missing migration %d: older than the newest applied migration and below the out-of-order minimum", m.Version)
			continue
		}
		if err := c.runMigration(db, m, migrateUp); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) UpByOne(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := c.GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		return err
	}

	if err = c.runMigration(db, next, migrateUp); err != nil {
		return err
	}

	return nil
}
