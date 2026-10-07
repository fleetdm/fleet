package tables

import (
	"database/sql"
	"fmt"
)

// init registers the Microsoft Graph cloud migration.
func init() {
	MigrationClient.AddMigration(Up_20261007130000, Down_20261007130000)
}

// Up_20261007130000 adds a cloud column defaulting to Global and safely skips it on retries.
func Up_20261007130000(tx *sql.Tx) error {
	if !columnExists(tx, "mdm_microsoft_graph_credentials", "cloud") {
		if _, err := tx.Exec(`ALTER TABLE mdm_microsoft_graph_credentials
			ADD COLUMN cloud VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'global' AFTER client_id`); err != nil {
			return fmt.Errorf("adding microsoft graph cloud: %w", err)
		}
	}
	return nil
}

// Down_20261007130000 leaves stored cloud metadata intact on rollback.
func Down_20261007130000(tx *sql.Tx) error {
	return nil
}
