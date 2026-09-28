package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260928155250, Down_20260928155250)
}

// rotation_command_uuid holds the in-flight RotateFileVaultKey command for the
// host's key; NULL means no rotation is pending.
func Up_20260928155250(tx *sql.Tx) error {
	if !columnExists(tx, "host_disk_encryption_keys", "rotation_command_uuid") {
		if _, err := tx.Exec(`
			ALTER TABLE host_disk_encryption_keys
			ADD COLUMN rotation_command_uuid VARCHAR(127) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL
		`); err != nil {
			return fmt.Errorf("adding rotation_command_uuid to host_disk_encryption_keys: %w", err)
		}
	}
	return addIndexesTx(tx, "host_disk_encryption_keys",
		indexDef{"idx_hdek_rotation_command_uuid", "rotation_command_uuid"})
}

func Down_20260928155250(tx *sql.Tx) error {
	return nil
}
