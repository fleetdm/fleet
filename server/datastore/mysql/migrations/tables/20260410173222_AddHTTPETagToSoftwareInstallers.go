package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260410173222, Down_20260410173222)
}

func Up_20260410173222(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_installers ADD COLUMN http_etag VARCHAR(512) COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
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

// Down_20260410173222 is a no-op. Fleet convention: down migrations return nil
// because forward-only migrations are safer than attempting rollback DDL.
func Down_20260410173222(tx *sql.Tx) error {
	return nil
}
