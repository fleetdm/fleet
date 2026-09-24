package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260610172952, Down_20260610172952)
}

func Up_20260610172952(tx *sql.Tx) error {
	// has_acme_payload lets the RemoveProfile CertificateList trigger detect an
	// ACME profile without re-reading the by-then-deleted config profile.
	// Backfill from still-present config profiles; preserve updated_at so the
	// backfill doesn't bump the ON UPDATE timestamp.
	return withSteps([]migrationStep{
		basicMigrationStep(
			`ALTER TABLE host_mdm_apple_profiles ADD COLUMN has_acme_payload TINYINT(1) NOT NULL DEFAULT 0`,
			"adding has_acme_payload to host_mdm_apple_profiles",
		),
		basicMigrationStep(
			`UPDATE host_mdm_apple_profiles hmap
				JOIN mdm_apple_configuration_profiles mac ON mac.profile_uuid = hmap.profile_uuid
				SET hmap.has_acme_payload = 1, hmap.updated_at = hmap.updated_at
				WHERE LOCATE('com.apple.security.acme', mac.mobileconfig) > 0`,
			"backfilling has_acme_payload from config profiles",
		),
	}, tx)
}

func Down_20260610172952(tx *sql.Tx) error {
	return nil
}
