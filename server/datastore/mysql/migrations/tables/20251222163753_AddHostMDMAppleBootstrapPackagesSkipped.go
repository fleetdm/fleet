package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251222163753, Down_20251222163753)
}

func Up_20251222163753(tx *sql.Tx) error {
	// Add a new column and a constraint that either:
	// - skipped is true and command_uuid is NULL
	// - skipped is false and command_uuid is NOT NULL
	// Then remove the non-nullability of command_uuid
	_, err := tx.Exec(`
		ALTER TABLE host_mdm_apple_bootstrap_packages
		ADD COLUMN skipped TINYINT(1) NOT NULL DEFAULT 0,
		ADD CONSTRAINT ck_skipped_or_commanduuid CHECK ((skipped = 0) = (command_uuid IS NOT NULL)),
		MODIFY COLUMN command_uuid varchar(127) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
	`)
	if err != nil {
		return err
	}

	return nil
}

func Down_20251222163753(tx *sql.Tx) error {
	return nil
}
