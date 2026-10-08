package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261007202302, Down_20261007202302)
}

// Up_20261007202302 adds software.ai_tool. The (ai_tool, title_id) index serves lookups of the
// titles that have a flagged version without walking every version of a title.
func Up_20261007202302(tx *sql.Tx) error {
	if !columnExists(tx, "software", "ai_tool") {
		if _, err := tx.Exec(`ALTER TABLE software ADD COLUMN ai_tool TINYINT(1) NOT NULL DEFAULT 0, ALGORITHM=INSTANT`); err != nil {
			return fmt.Errorf("adding ai_tool to software: %w", err)
		}
	}
	if !indexExistsTx(tx, "software", "idx_software_ai_tool_title_id") {
		if _, err := tx.Exec(`ALTER TABLE software ADD INDEX idx_software_ai_tool_title_id (ai_tool, title_id), ALGORITHM=INPLACE, LOCK=NONE`); err != nil {
			return fmt.Errorf("adding idx_software_ai_tool_title_id to software: %w", err)
		}
	}
	return nil
}

func Down_20261007202302(tx *sql.Tx) error {
	return nil
}
