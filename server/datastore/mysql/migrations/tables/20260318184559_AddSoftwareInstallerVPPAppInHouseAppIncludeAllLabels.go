package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260318184559, Down_20260318184559)
}

func Up_20260318184559(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE in_house_app_labels ADD COLUMN require_all BOOL NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add require_all to in_house_app_labels: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_installer_labels ADD COLUMN require_all BOOL NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add require_all to software_installer_labels: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE vpp_app_team_labels ADD COLUMN require_all BOOL NOT NULL DEFAULT false`)
	if err != nil {
		return fmt.Errorf("failed to add require_all to vpp_app_team_labels: %w", err)
	}

	return nil
}

func Down_20260318184559(tx *sql.Tx) error {
	return nil
}
