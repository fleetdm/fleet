package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210927143115, Down_20210927143115)
}

func Up_20210927143115(tx *sql.Tx) error {
	_, err := tx.Exec("TRUNCATE TABLE policy_membership_history")
	if err != nil {
		return errors.Wrap(err, "clearing policy_membership_history")
	}

	_, err = tx.Exec("ALTER TABLE hosts ADD COLUMN policy_updated_at timestamp NOT NULL DEFAULT '2000-01-01 00:00:00'")
	if err != nil {
		return errors.Wrap(err, "adding policy_updated_at column")
	}

	return err
}

func Down_20210927143115(tx *sql.Tx) error {
	return nil
}
