package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261006085053, Down_20261006085053)
}

func Up_20261006085053(tx *sql.Tx) error {
	if !tableExists(tx, "mdm_apple_scep_challenges") {
		if _, err := tx.Exec(`CREATE TABLE mdm_apple_scep_challenges (
			id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
			challenge          VARCHAR(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
			-- ade, ota_phase1, ota_phase2, adue, renewal
			purpose            VARCHAR(31) COLLATE utf8mb4_unicode_ci NOT NULL,
			-- device lanes: the UDID the device claimed; renewal: the enrollment's device channel ID (UDID, or EnrollmentID for ADUE)
			host_uuid          VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			hardware_serial    VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			-- ADUE only; no FK so deleting an IdP account doesn't erase rows kept for inspection
			idp_account_uuid   VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			expires_at         DATETIME(6) NOT NULL,
			consumed_at        DATETIME(6) DEFAULT NULL,
			-- identity_certificates.serial, for inspection only (the binding itself is in the certificate); no FK because identity cert rows have their own lifecycle
			issued_cert_serial BIGINT DEFAULT NULL,
			created_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at         DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_mdm_apple_scep_challenges_challenge (challenge),
			UNIQUE KEY idx_mdm_apple_scep_challenges_cert_serial (issued_cert_serial),
			-- renewal create-or-reuse lookup
			KEY idx_mdm_apple_scep_challenges_host_purpose (host_uuid, purpose),
			KEY idx_mdm_apple_scep_challenges_expires_at (expires_at),
			KEY idx_mdm_apple_scep_challenges_consumed_at (consumed_at)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
 		`); err != nil {
			return fmt.Errorf("error creating table mdm_apple_scep_challenges: %w", err)
		}
	}

	_, err := tx.Exec(`ALTER TABLE acme_enrollments
		-- acme (fresh ADE enrollment) or acme_renewal
		ADD COLUMN purpose VARCHAR(31) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'acme',
		-- acme_renewal only: the enrollment's device channel ID
		ADD COLUMN enrollment_id VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
  	`)
	if err != nil {
		return fmt.Errorf("error altering table acme_enrollments: %w", err)
	}
	return nil
}

func Down_20261006085053(tx *sql.Tx) error {
	return nil
}
