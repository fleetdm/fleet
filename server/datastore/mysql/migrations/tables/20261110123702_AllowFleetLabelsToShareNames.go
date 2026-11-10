package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261110123702, Down_20261110123702)
}

func Up_20261110123702(tx *sql.Tx) error {
	// NULL team_ids never collide in a unique index, so global labels are keyed as 0 (no fleet has that ID) to
	// stay unique among themselves.
	if !indexExistsTx(tx, "labels", "idx_label_unique_name_team") {
		if _, err := tx.Exec("ALTER TABLE labels ADD UNIQUE KEY idx_label_unique_name_team (name, (COALESCE(team_id, 0)))"); err != nil {
			return fmt.Errorf("adding labels name and fleet unique index: %w", err)
		}
	}
	if indexExistsTx(tx, "labels", "idx_label_unique_name") {
		if _, err := tx.Exec("ALTER TABLE labels DROP INDEX idx_label_unique_name"); err != nil {
			return fmt.Errorf("dropping labels name unique index: %w", err)
		}
	}
	return nil
}

func Down_20261110123702(tx *sql.Tx) error {
	return nil
}
