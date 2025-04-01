package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250331042354, Down_20250331042354)
}

func Up_20250331042354(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS scim_users (
	    id int UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	    external_id VARCHAR(255) NULL,
	    user_name VARCHAR(255) NOT NULL,
	    given_name VARCHAR(255) NULL,
	    family_name VARCHAR(255) NULL,
	    active TINYINT(1) NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
	    UNIQUE KEY idx_scim_users_user_name (user_name),
	    KEY idx_scim_users_external_id (external_id)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

	CREATE TABLE IF NOT EXISTS host_scim_user (
	    host_id INT UNSIGNED NOT NULL,
	    scim_user_id INT UNSIGNED NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    PRIMARY KEY (host_id, scim_user_id),
        CONSTRAINT fk_host_scim_scim_user_id FOREIGN KEY (scim_user_id) REFERENCES scim_users (id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	
	CREATE TABLE if NOT EXISTS scim_user_emails (
	    -- Using BIGINT because we clear and repopulate the emails frequently (during user update)
	    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	    scim_user_id INT UNSIGNED NOT NULL,
	    email VARCHAR(255) NOT NULL,
	    ` + "`primary`" + ` TINYINT(1) NULL,
	    type VARCHAR(31) NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
	    KEY idx_scim_user_emails_email_type(type, email),
        CONSTRAINT fk_scim_user_emails_scim_user_id FOREIGN KEY (scim_user_id) REFERENCES scim_users (id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;

	CREATE TABLE IF NOT EXISTS scim_groups (
	    id int UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	    external_id VARCHAR(255) NULL,
	    display_name VARCHAR(255) NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6),
	    KEY idx_scim_groups_external_id (external_id),
	    -- Entra ID requires a unique display name
	    UNIQUE KEY idx_scim_groups_display_name (display_name)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	
	CREATE TABLE IF NOT EXISTS scim_user_group (
	    scim_user_id INT UNSIGNED NOT NULL,
	    group_id INT UNSIGNED NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    PRIMARY KEY (scim_user_id, group_id),
        CONSTRAINT fk_scim_user_group_scim_user_id FOREIGN KEY (scim_user_id) REFERENCES scim_users (id) ON DELETE CASCADE,
        CONSTRAINT fk_scim_user_group_group_id FOREIGN KEY (group_id) REFERENCES scim_groups (id) ON DELETE CASCADE
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)

	if err != nil {
		return fmt.Errorf("failed to create scim tables: %s", err)
	}

	return nil
}

func Down_20250331042354(tx *sql.Tx) error {
	return nil
}
