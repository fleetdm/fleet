package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260218175704, Down_20260218175704)
}

func Up_20260218175704(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE software_installers
			DROP INDEX idx_software_installers_team_id_title_id,
			DROP INDEX idx_software_installers_platform_title_id,
			ADD UNIQUE INDEX idx_software_installers_team_title_version (global_or_team_id,title_id,version),
			ADD COLUMN is_active TINYINT(1) NOT NULL DEFAULT 0
	`)
	if err != nil {
		return fmt.Errorf("altering software_installers: %w", err)
	}

	// At migration time, the 1-installer-per-title rule is still enforced,
	// so every existing installer is the active one for its title.
	_, err = tx.Exec(`UPDATE software_installers SET is_active = 1`)
	if err != nil {
		return fmt.Errorf("setting is_active for existing installers: %w", err)
	}

	return nil
}

func Down_20260218175704(tx *sql.Tx) error {
	return nil
}
