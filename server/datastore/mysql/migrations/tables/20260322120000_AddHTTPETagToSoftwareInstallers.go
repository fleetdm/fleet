package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260322120000, Down_20260322120000)
}

func Up_20260322120000(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_installers ADD COLUMN http_etag VARCHAR(512) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add http_etag column to software_installers: %w", err)
	}
	// Index prefix url(255) is a MySQL limitation for InnoDB key length.
	// URLs longer than 255 bytes are still matched correctly (full row comparison)
	// but with reduced index selectivity.
	_, err = tx.Exec(`CREATE INDEX idx_software_installers_team_url ON software_installers (global_or_team_id, url(255))`)
	if err != nil {
		return fmt.Errorf("failed to add team+url index to software_installers: %w", err)
	}
	return nil
}

func Down_20260322120000(tx *sql.Tx) error {
	return nil
}
