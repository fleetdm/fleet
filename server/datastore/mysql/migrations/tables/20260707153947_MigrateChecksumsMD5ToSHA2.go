package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260707153947, Down_20260707153947)
}

// Up_20260707153947 migrates all checksum columns from MD5 (BINARY(16)) to
// SHA2-256 (BINARY(32)). MySQL 9.6+ removed the MD5() and SHA1() SQL
// functions, so all hashing must use SHA2(). This migration handles:
//
//  1. Widening regular checksum columns from BINARY(16) to BINARY(32).
//  2. Recalculating stored checksums with SHA2().
//  3. Dropping and recreating generated columns that used MD5().
//  4. Updating host profile checksums to match new profile checksums.
func Up_20260707153947(tx *sql.Tx) error {
	// ---- Apple configuration profiles ----
	// Widen columns first.
	if _, err := tx.Exec(`ALTER TABLE mdm_apple_configuration_profiles MODIFY COLUMN checksum BINARY(32) NOT NULL`); err != nil {
		return fmt.Errorf("widen mdm_apple_configuration_profiles.checksum: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE host_mdm_apple_profiles MODIFY COLUMN checksum BINARY(32) NOT NULL`); err != nil {
		return fmt.Errorf("widen host_mdm_apple_profiles.checksum: %w", err)
	}
	// Update host checksums WHERE they currently match the profile's old checksum
	// (meaning the host has the current profile version). Do this BEFORE
	// recalculating profile checksums so we can compare old values.
	if _, err := tx.Exec(`
		UPDATE host_mdm_apple_profiles hmap
		JOIN mdm_apple_configuration_profiles macp ON macp.profile_uuid = hmap.profile_uuid
		SET hmap.checksum = UNHEX(SHA2(macp.mobileconfig, 256))
		WHERE hmap.checksum = macp.checksum`); err != nil {
		return fmt.Errorf("update host_mdm_apple_profiles checksums: %w", err)
	}
	// Now recalculate the profile checksums.
	if _, err := tx.Exec(`UPDATE mdm_apple_configuration_profiles SET checksum = UNHEX(SHA2(mobileconfig, 256))`); err != nil {
		return fmt.Errorf("recalculate mdm_apple_configuration_profiles checksums: %w", err)
	}

	// ---- Windows configuration profiles ----
	// Drop the generated column and recreate it with SHA2.
	if _, err := tx.Exec(`ALTER TABLE mdm_windows_configuration_profiles DROP COLUMN checksum`); err != nil {
		return fmt.Errorf("drop mdm_windows_configuration_profiles.checksum: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE mdm_windows_configuration_profiles ADD COLUMN checksum BINARY(32) AS (UNHEX(SHA2(syncml, 256))) STORED`); err != nil {
		return fmt.Errorf("recreate mdm_windows_configuration_profiles.checksum: %w", err)
	}
	// Widen host column and update matching checksums.
	if _, err := tx.Exec(`ALTER TABLE host_mdm_windows_profiles MODIFY COLUMN checksum BINARY(32) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("widen host_mdm_windows_profiles.checksum: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE host_mdm_windows_profiles hmwp
		JOIN mdm_windows_configuration_profiles mwcp ON mwcp.profile_uuid = hmwp.profile_uuid
		SET hmwp.checksum = mwcp.checksum`); err != nil {
		return fmt.Errorf("update host_mdm_windows_profiles checksums: %w", err)
	}

	// ---- Android configuration profiles ----
	// Drop and recreate generated column.
	if _, err := tx.Exec(`ALTER TABLE mdm_android_configuration_profiles DROP COLUMN checksum`); err != nil {
		return fmt.Errorf("drop mdm_android_configuration_profiles.checksum: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE mdm_android_configuration_profiles ADD COLUMN checksum BINARY(32) AS (UNHEX(SHA2(CAST(raw_json AS CHAR), 256))) STORED`); err != nil {
		return fmt.Errorf("recreate mdm_android_configuration_profiles.checksum: %w", err)
	}
	// Widen host column and update matching checksums.
	if _, err := tx.Exec(`ALTER TABLE host_mdm_android_profiles MODIFY COLUMN checksum BINARY(32) NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("widen host_mdm_android_profiles.checksum: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE host_mdm_android_profiles hmap
		JOIN mdm_android_configuration_profiles macp ON macp.profile_uuid = hmap.profile_uuid
		SET hmap.checksum = macp.checksum`); err != nil {
		return fmt.Errorf("update host_mdm_android_profiles checksums: %w", err)
	}

	// ---- Apple DDM declarations token ----
	// Drop and recreate generated column.
	if _, err := tx.Exec(`ALTER TABLE mdm_apple_declarations DROP COLUMN token`); err != nil {
		return fmt.Errorf("drop mdm_apple_declarations.token: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE mdm_apple_declarations ADD COLUMN token BINARY(32) GENERATED ALWAYS AS (UNHEX(SHA2(CONCAT(raw_json, IFNULL(secrets_updated_at, '')), 256))) STORED NULL`); err != nil {
		return fmt.Errorf("recreate mdm_apple_declarations.token: %w", err)
	}
	// Widen host column (the values will be refreshed on next DDM session).
	if _, err := tx.Exec(`ALTER TABLE host_mdm_apple_declarations MODIFY COLUMN token BINARY(32) NOT NULL`); err != nil {
		return fmt.Errorf("widen host_mdm_apple_declarations.token: %w", err)
	}

	// ---- Software checksums ----
	// Widen the column, drop the unique index (needs to be recreated after resize),
	// recalculate checksums, then recreate the index.
	if _, err := tx.Exec(`ALTER TABLE software DROP INDEX idx_software_checksum`); err != nil {
		return fmt.Errorf("drop software checksum index: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE software MODIFY COLUMN checksum BINARY(32) NOT NULL`); err != nil {
		return fmt.Errorf("widen software.checksum: %w", err)
	}
	// The hash must match Go's ComputeRawChecksum() which conditionally
	// appends application_id and upgrade_code when non-empty.
	if _, err := tx.Exec(`
		UPDATE software SET checksum = UNHEX(
			SHA2(
				CONCAT(
					CONCAT_WS(CHAR(0),
						version,
						source,
						COALESCE(bundle_identifier, ''),
						` + "`release`" + `,
						arch,
						vendor,
						extension_for,
						extension_id,
						name
					),
					IF(application_id IS NOT NULL AND application_id != '', CONCAT(CHAR(0), application_id), ''),
					IF(upgrade_code IS NOT NULL AND upgrade_code != '', CONCAT(CHAR(0), upgrade_code), '')
				),
			256)
		)`); err != nil {
		return fmt.Errorf("recalculate software checksums: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE software ADD UNIQUE INDEX idx_software_checksum (checksum)`); err != nil {
		return fmt.Errorf("recreate software checksum index: %w", err)
	}

	// ---- Policy checksums ----
	if _, err := tx.Exec(`ALTER TABLE policies DROP INDEX idx_policies_checksum`); err != nil {
		return fmt.Errorf("drop policies checksum index: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE policies MODIFY COLUMN checksum BINARY(32) NOT NULL`); err != nil {
		return fmt.Errorf("widen policies.checksum: %w", err)
	}
	if _, err := tx.Exec(`
		UPDATE policies SET checksum = UNHEX(
			SHA2(
				CONCAT_WS(CHAR(0),
					COALESCE(team_id, ''),
					name
				),
			256)
		)`); err != nil {
		return fmt.Errorf("recalculate policies checksums: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE policies ADD UNIQUE INDEX idx_policies_checksum (checksum)`); err != nil {
		return fmt.Errorf("recreate policies checksum index: %w", err)
	}

	return nil
}

func Down_20260707153947(_ *sql.Tx) error {
	return nil
}
