package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000008, Down_20210601000008)
}

func Up_20210601000008(tx *sql.Tx) error {
	// Add team_id
	sql := `
		ALTER TABLE enroll_secrets
		ADD COLUMN team_id INT UNSIGNED,
		ADD FOREIGN KEY fk_team_id (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add team_id to enroll_secrets")
	}

	// Remove "active" as a concept from enroll secrets
	sql = `
		DELETE FROM enroll_secrets
		WHERE NOT active
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "remove inactive secrets")
	}

	sql = `
		ALTER TABLE enroll_secrets
		DROP COLUMN active,
		DROP COLUMN name,
		ADD UNIQUE INDEX (secret)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "alter enroll_secrets")
	}

	sql = `
		ALTER TABLE hosts
		DROP COLUMN enroll_secret_name
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "alter hosts")
	}

	return nil
}

func Down_20210601000008(tx *sql.Tx) error {
	return nil
}
