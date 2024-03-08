package goose

import (
	"database/sql"
)

func (c *Client) Up(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		current, err := c.GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				return nil
			}
			return err
		}

		if err = c.runMigration(db, next, migrateUp); err != nil {
			return err
		}
	}
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
