package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230629140530, Down_20230629140530)
}

func Up_20230629140530(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE mdm_windows_enrollments (
            id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
			mdm_device_id VARCHAR(255) NOT NULL,
			mdm_hardware_id VARCHAR(255) NOT NULL,
			device_state VARCHAR(255) NOT NULL,
			device_type VARCHAR(255) NOT NULL,
			device_name VARCHAR(255) NOT NULL,
			enroll_type VARCHAR(255) NOT NULL,
			enroll_user_id VARCHAR(255) NOT NULL,
			enroll_proto_version VARCHAR(255) NOT NULL,
			enroll_client_version VARCHAR(255) NOT NULL,
			not_in_oobe TINYINT(1) NOT NULL DEFAULT FALSE,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY idx_type (mdm_hardware_id)
        ) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_windows_enrollments table: %w", err)
	}

	return nil
}

func Down_20230629140530(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `mdm_windows_enrollments`;")
	return err
}
