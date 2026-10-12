package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261012135052, Down_20261012135052)
}

func Up_20261012135052(tx *sql.Tx) error {
	// Use BIGINT so an out of range policy_id in details becomes NULL instead of failing the activity insert.
	if !columnExists(tx, "activity_past", "policy_id") {
		_, err := tx.Exec(`ALTER TABLE activity_past
			ADD COLUMN policy_id BIGINT UNSIGNED AS (JSON_VALUE(details, '$.policy_id' RETURNING UNSIGNED)) VIRTUAL,
			ALGORITHM=INSTANT`)
		if err != nil {
			return fmt.Errorf("adding policy_id to activity_past: %w", err)
		}
	}
	if !indexExistsTx(tx, "activity_past", "idx_activity_past_policy_type_created") {
		_, err := tx.Exec(`ALTER TABLE activity_past
			ADD INDEX idx_activity_past_policy_type_created (policy_id, activity_type, created_at DESC),
			ALGORITHM=INPLACE, LOCK=NONE`)
		if err != nil {
			return fmt.Errorf("adding idx_activity_past_policy_type_created to activity_past: %w", err)
		}
	}
	return nil
}

func Down_20261012135052(tx *sql.Tx) error {
	return nil
}
