package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000003, Down_20210601000003)
}

func Up_20210601000003(tx *sql.Tx) error {
	// Old admins become global admins
	query := `
		UPDATE users
			SET global_role = 'admin'
			WHERE admin = TRUE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "update admins")
	}

	query = `
		UPDATE invites
			SET global_role = 'admin'
			WHERE admin = TRUE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "update admin invites")
	}

	// Old non-admins become global maintainers
	query = `
		UPDATE users
			SET global_role = 'maintainer'
			WHERE admin = FALSE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "update maintainers")
	}

	query = `
		UPDATE invites
			SET global_role = 'maintainer'
			WHERE admin = FALSE
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "update maintainer invites")
	}

	// Drop the old admin column
	query = `
		ALTER TABLE users
			DROP COLUMN admin
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "drop users admin column")
	}

	query = `
		ALTER TABLE invites
			DROP COLUMN admin
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "drop invites admin column")
	}

	return nil
}

func Down_20210601000003(tx *sql.Tx) error {
	return nil
}
