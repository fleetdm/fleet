package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261002140527, Down_20261002140527)
}

func Up_20261002140527(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS mdm_apple_dep_enrollment_challenges (
	id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	challenge        VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
	idp_account_uuid VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	hardware_serial  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	host_uuid        VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	expires_at       DATETIME(6) NOT NULL,
	used_at          DATETIME(6) DEFAULT NULL,
	created_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at       DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (id),
	UNIQUE KEY idx_mdm_apple_dep_enrollment_challenges_challenge (challenge),
	KEY idx_mdm_apple_dep_enrollment_challenges_expires_at (expires_at),
	CONSTRAINT fk_mdm_apple_dep_enrollment_challenges_idp_account
		FOREIGN KEY (idp_account_uuid) REFERENCES mdm_idp_accounts (uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create mdm_apple_dep_enrollment_challenges table: %w", err)
	}

	if !columnExists(tx, "mdm_apple_enrollment_profiles", "previous_token") {
		if _, err := tx.Exec(`
ALTER TABLE mdm_apple_enrollment_profiles
	ADD COLUMN previous_token VARCHAR(36) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
	ADD COLUMN previous_token_expires_at DATETIME(6) DEFAULT NULL`); err != nil {
			return fmt.Errorf("add previous token columns to mdm_apple_enrollment_profiles: %w", err)
		}
	}

	return nil
}

func Down_20261002140527(tx *sql.Tx) error {
	return nil
}
