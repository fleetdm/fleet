package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250217142401, Down_20250217142401)
}

func Up_20250217142401(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE android_enterprises
    		-- PubSub topic_id
			ADD COLUMN topic_id VARCHAR(63) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			-- Authentication token for callback endpoint to create enterprise
			ADD COLUMN signup_token VARCHAR(63) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
    		`)
	if err != nil {
		return fmt.Errorf("failed to update android_enterprise table: %w", err)
	}

	// TODO: Add host table

	return nil
}

func Down_20250217142401(_ *sql.Tx) error {
	return nil
}
