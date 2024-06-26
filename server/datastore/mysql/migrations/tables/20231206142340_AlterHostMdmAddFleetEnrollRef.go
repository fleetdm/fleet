package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231206142340, Down_20231206142340)
}

func Up_20231206142340(tx *sql.Tx) error {
	stmt := `ALTER TABLE host_mdm ADD COLUMN fleet_enroll_ref VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add fleet_enroll_ref to host_mdm: %w", err)
	}

	return nil
}

func Down_20231206142340(tx *sql.Tx) error {
	return nil
}
