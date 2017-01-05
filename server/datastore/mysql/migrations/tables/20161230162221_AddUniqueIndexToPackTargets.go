package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20161230162221, Down_20161230162221)
}

func Up_20161230162221(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `pack_targets` " +
			"ADD CONSTRAINT `constraint_pack_target_unique` " +
			"UNIQUE (`pack_id`, `target_id`, `type`);",
	)
	return err
}

func Down_20161230162221(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `pack_targets` " +
			"DROP INDEX `constraint_pack_target_unique`;",
	)
	return err
}
