package data

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20181119180000, Down20181119180000)
}

func Up20181119180000(tx *sql.Tx) error {
	sql := `DELETE FROM scheduled_queries WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete scheduled queries")
	}

	sql = `DELETE FROM queries WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete queries")
	}

	sql = `DELETE FROM labels WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete labels")
	}

	sql = `DELETE FROM distributed_query_campaigns WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete campaigns")
	}

	sql = `DELETE FROM hosts WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete hosts")
	}

	sql = `DELETE FROM invites WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete invites")
	}

	sql = `DELETE FROM users WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete users")
	}

	return nil
}

func Down20181119180000(tx *sql.Tx) error {
	return nil
}
