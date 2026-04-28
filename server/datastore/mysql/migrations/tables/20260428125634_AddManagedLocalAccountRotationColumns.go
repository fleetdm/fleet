package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260428125634, Down_20260428125634)
}

func Up_20260428125634(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_managed_local_account_passwords
			ADD COLUMN account_uuid                VARCHAR(36)  COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
			ADD COLUMN auto_rotate_at              TIMESTAMP(6)                            NULL DEFAULT NULL,
			ADD COLUMN pending_encrypted_password  BLOB                                    NULL DEFAULT NULL,
			ADD COLUMN pending_command_uuid        VARCHAR(127) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
			ADD COLUMN initiated_by_fleet          TINYINT(1)                              NOT NULL DEFAULT 0,
			ADD KEY idx_hmlap_auto_rotate_at (auto_rotate_at)
	`); err != nil {
		return fmt.Errorf("adding rotation columns to host_managed_local_account_passwords: %w", err)
	}
	return nil
}

func Down_20260428125634(tx *sql.Tx) error {
	return nil
}
