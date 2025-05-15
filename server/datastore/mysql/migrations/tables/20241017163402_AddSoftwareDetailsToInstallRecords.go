package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241017163402, Down_20241017163402)
}

func Up_20241017163402(tx *sql.Tx) error {
	_, err := tx.Exec("ALTER TABLE host_software_installs ADD COLUMN installer_filename VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[deleted installer]'")
	if err != nil {
		return fmt.Errorf("failed to create installer_filename column on host_software_installs table: %w", err)
	}

	_, err = tx.Exec("ALTER TABLE host_software_installs ADD COLUMN version VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'unknown'")
	if err != nil {
		return fmt.Errorf("failed to create version column on host_software_installs table: %w", err)
	}

	_, err = tx.Exec("ALTER TABLE host_software_installs ADD COLUMN software_title_id INT UNSIGNED DEFAULT NULL")
	if err != nil {
		return fmt.Errorf("failed to create software_title_id column on host_software_installs table: %w", err)
	}

	_, err = tx.Exec("ALTER TABLE host_software_installs ADD COLUMN software_title_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '[deleted title]'")
	if err != nil {
		return fmt.Errorf("failed to create software_title_name column on host_software_installs table: %w", err)
	}

	_, err = tx.Exec("ALTER TABLE host_software_installs CHANGE COLUMN software_installer_id software_installer_id INT UNSIGNED DEFAULT NULL")
	if err != nil {
		return fmt.Errorf("failed to allow nullability on software_installer_id column in host_software_installs table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_software_installs DROP CONSTRAINT fk_host_software_installs_installer_id`)
	if err != nil {
		return fmt.Errorf("failed to switch on-delete behavior of software installer foreign key from host_software_installs table (constraint drop): %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_software_installs ADD CONSTRAINT fk_host_software_installs_installer_id
    	FOREIGN KEY (software_installer_id) REFERENCES software_installers (id) ON DELETE SET NULL ON UPDATE CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to switch on-delete behavior of software installer foreign key from host_software_installs table (constraint re-add): %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_software_installs ADD CONSTRAINT fk_host_software_installs_software_title_id
    	FOREIGN KEY (software_title_id) REFERENCES software_titles (id) ON DELETE SET NULL ON UPDATE CASCADE`)
	if err != nil {
		return fmt.Errorf("failed to add foreign key for software_title_id to host_software_installs table: %w", err)
	}

	_, err = tx.Exec(`
UPDATE host_software_installs i
JOIN software_installers si ON si.id = i.software_installer_id
LEFT JOIN software_titles st ON st.id = si.title_id
SET
    i.software_title_id = st.id,
    i.software_title_name = COALESCE(st.name, "[deleted title]"),
    i.installer_filename = IF(i.uninstall, "", si.filename),
    i.version = IF(i.uninstall = 0 AND i.created_at >= si.uploaded_at, si.version, "unknown"),
    i.updated_at = i.updated_at
`) // only one left join because prior to this migration software_installer_id wasn't nullable on host_software_installs
	if err != nil {
		return fmt.Errorf("failed to propagate software title and installer information into host_software_installs: %w", err)
	}

	return nil
}

func Down_20241017163402(tx *sql.Tx) error {
	return nil
}
