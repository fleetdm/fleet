package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260304120000, Down_20260304120000)
}

func Up_20260304120000(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE windows_enforcement_profiles (
    profile_uuid VARCHAR(37) NOT NULL,
    team_id INT UNSIGNED NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL COLLATE utf8mb4_unicode_ci,
    raw_policy MEDIUMBLOB NOT NULL,
    checksum BINARY(16) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (profile_uuid),
    UNIQUE KEY idx_enforcement_team_name (team_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("create windows_enforcement_profiles table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE host_windows_enforcement (
    host_uuid VARCHAR(255) NOT NULL COLLATE utf8mb4_unicode_ci,
    profile_uuid VARCHAR(37) NOT NULL,
    status VARCHAR(20) DEFAULT NULL,
    operation_type VARCHAR(20) DEFAULT NULL,
    detail TEXT COLLATE utf8mb4_unicode_ci,
    retries TINYINT UNSIGNED NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (host_uuid, profile_uuid),
    FOREIGN KEY (status) REFERENCES mdm_delivery_status(status) ON UPDATE CASCADE,
    FOREIGN KEY (operation_type) REFERENCES mdm_operation_types(operation_type) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("create host_windows_enforcement table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE windows_enforcement_profile_labels (
    profile_uuid VARCHAR(37) NOT NULL,
    label_name VARCHAR(255) NOT NULL COLLATE utf8mb4_unicode_ci,
    label_id INT UNSIGNED DEFAULT NULL,
    exclude TINYINT(1) NOT NULL DEFAULT 0,
    require_all TINYINT(1) NOT NULL DEFAULT 0,

    PRIMARY KEY (profile_uuid, label_name),
    FOREIGN KEY (profile_uuid) REFERENCES windows_enforcement_profiles(profile_uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`)
	if err != nil {
		return fmt.Errorf("create windows_enforcement_profile_labels table: %w", err)
	}

	return nil
}

func Down_20260304120000(tx *sql.Tx) error {
	return nil
}
