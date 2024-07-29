package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240729120947, Down_20240729120947)
}

func Up_20240729120947(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE host_software_installs ADD COLUMN host_deleted_at timestamp NULL DEFAULT NULL")
	if err != nil {
		return fmt.Errorf("failed to create host_deleted_at column on host_software_installs table: %w", err)
	}
	_, err = tx.Exec(`
UPDATE
    host_software_installs i
LEFT JOIN
    hosts h
    ON i.host_id = h.id
SET
    i.host_deleted_at = NOW()
WHERE
    i.host_deleted_at IS NULL
AND
    h.id IS NULL
`)
	if err != nil {
		return fmt.Errorf("failed to update host_software_installs.host_deleted_at for hosts that no longer exist")
	}
	return nil
}

func Down_20240729120947(tx *sql.Tx) error {
	return nil
}
