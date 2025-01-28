package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240801115359, Down_20240801115359)
}

func Up_20240801115359(tx *sql.Tx) error {
	if columnExists(tx, "host_vpp_software_installs", "platform") {
		return nil
	}

	_, err := tx.Exec(`
		ALTER TABLE host_vpp_software_installs
		ADD COLUMN platform VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL`)
	if err != nil {
		return fmt.Errorf("adding platform to host_vpp_software_installs: %w", err)
	}

	updateStmt := `
		UPDATE host_vpp_software_installs hvsi INNER JOIN hosts h ON h.id = hvsi.host_id
		SET hvsi.platform = h.platform, hvsi.updated_at = hvsi.updated_at`

	_, err = tx.Exec(updateStmt)
	if err != nil {
		return fmt.Errorf("updating platform in host_vpp_software_installs: %w", err)
	}

	// Since hosts may be missing, we need to update the platform for records that were not updated
	updateStmt2 := `
		UPDATE host_vpp_software_installs hvsi INNER JOIN vpp_apps vap ON vap.adam_id = hvsi.adam_id
		SET hvsi.platform = vap.platform, hvsi.updated_at = hvsi.updated_at
		WHERE hvsi.platform = ''`

	_, err = tx.Exec(updateStmt2)
	if err != nil {
		return fmt.Errorf("updating platform in host_vpp_software_installs part 2: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_vpp_software_installs DROP INDEX adam_id, ADD INDEX (adam_id, platform)`)
	if err != nil {
		return fmt.Errorf("updating key in host_vpp_software_installs: %w", err)
	}
	if indexExistsTx(tx, "host_vpp_software_installs", "host_vpp_software_installs_ibfk_2") {
		_, err = tx.Exec(`
		ALTER TABLE host_vpp_software_installs DROP FOREIGN KEY host_vpp_software_installs_ibfk_2`)
		if err != nil {
			return fmt.Errorf("updating foreign key in host_vpp_software_installs: %w", err)
		}
	}
	_, err = tx.Exec(`
		ALTER TABLE host_vpp_software_installs
			ADD CONSTRAINT host_vpp_software_installs_ibfk_3 FOREIGN KEY (adam_id, platform) REFERENCES vpp_apps (adam_id, platform) ON DELETE CASCADE`)
	if err != nil {
		return fmt.Errorf("updating foreign key in host_vpp_software_installs: %w", err)
	}

	return nil
}

func Down_20240801115359(tx *sql.Tx) error {
	return nil
}
