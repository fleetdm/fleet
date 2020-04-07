package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20200407120000, Down20200407120000)
}

func Up20200407120000(tx *sql.Tx) error {
	if _, err := tx.Exec(
		"ALTER TABLE `labels` " +
			"ADD COLUMN `label_membership_type` int(10) unsigned NOT NULL default '0'",
	); err != nil {
		return errors.Wrap(err, "add label_membership_type column ")
	}

	// All hosts should now be the only "manual" label
	if _, err := tx.Exec(
		"UPDATE `labels` " +
			"SET `label_membership_type` = 1 " +
			"WHERE `name` = 'All Hosts' AND `label_type` = 1",
	); err != nil {
		return errors.Wrap(err, "drop label_query_executions")
	}

	return nil
}

func Down20200407120000(tx *sql.Tx) error {
	return nil
}
