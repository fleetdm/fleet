package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260928155250, Down_20260928155250)
}

// rotation_command_uuid holds the in-flight RotateFileVaultKey command for the
// host's key; NULL means no rotation is pending. rotation_requested_at is when it
// was set, so a marker whose command isn't queued yet isn't mistaken for a stale one.
func Up_20260928155250(tx *sql.Tx) error {
	var clauses []string
	if !columnExists(tx, "host_disk_encryption_keys", "rotation_command_uuid") {
		clauses = append(clauses, "ADD COLUMN rotation_command_uuid VARCHAR(127) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL")
	}
	if !columnExists(tx, "host_disk_encryption_keys", "rotation_requested_at") {
		clauses = append(clauses, "ADD COLUMN rotation_requested_at TIMESTAMP(6) NULL DEFAULT NULL")
	}
	if len(clauses) > 0 {
		if _, err := tx.Exec("ALTER TABLE host_disk_encryption_keys " + strings.Join(clauses, ", ")); err != nil {
			return fmt.Errorf("adding rotation columns to host_disk_encryption_keys: %w", err)
		}
	}
	return addIndexesTx(tx, "host_disk_encryption_keys",
		indexDef{"idx_hdek_rotation_command_uuid", "rotation_command_uuid"})
}

func Down_20260928155250(tx *sql.Tx) error {
	return nil
}
