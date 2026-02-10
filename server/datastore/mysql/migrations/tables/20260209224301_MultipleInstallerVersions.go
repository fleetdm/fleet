package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260209224301, Down_20260209224301)
}

func Up_20260209224301(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE software_installers
			DROP INDEX idx_software_installers_team_id_title_id,
			DROP INDEX idx_software_installers_platform_title_id,
			ADD INDEX idx_software_installers_platform_title_id (global_or_team_id,platform,title_id,version) USING BTREE
		`)
	if err != nil {
		return fmt.Errorf("altering software_installers indexes: %w", err)
	}
	return nil
}

func Down_20260209224301(tx *sql.Tx) error {
	return nil
}
