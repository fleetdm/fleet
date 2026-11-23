package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261123143955, Down_20261123143955)
}

// Up_20261123143955 creates host_mdm_apple_profiles_status and host_mdm_apple_declarations_status, per-host rollups of the
// aggregate Apple configuration-profile and declaration delivery status. Each holds exactly one bucket per host
// ('failed'|'pending'|'verifying'|'verified'|empty); the FileVault profile gets its own bucket since it's evaluated with the
// disk encryption state.
func Up_20261123143955(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_mdm_apple_profiles_status (
	host_uuid        VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	status           VARCHAR(20)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	filevault_status VARCHAR(20)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	updated_at       DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating host_mdm_apple_profiles_status table: %w", err)
	}

	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_mdm_apple_declarations_status (
	host_uuid  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	status     VARCHAR(20)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	updated_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_uuid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating host_mdm_apple_declarations_status table: %w", err)
	}

	// Backfill one row per host. GROUP BY host_uuid is a leftmost prefix of the clustered PRIMARY KEY of both source tables, so
	// MySQL streams the aggregation in index order without a temp table. The ON DUPLICATE KEY UPDATE makes a retry idempotent.
	if _, err := tx.Exec(`
INSERT INTO host_mdm_apple_profiles_status (host_uuid, status, filevault_status)
SELECT
	hmap.host_uuid,
	CASE
		WHEN SUM(CASE WHEN hmap.status = 'failed' AND hmap.profile_identifier != 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'failed'
		WHEN SUM(CASE WHEN (hmap.status IS NULL OR hmap.status = 'pending') AND hmap.profile_identifier != 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'pending'
		WHEN SUM(CASE WHEN hmap.operation_type = 'install' AND hmap.status = 'verifying' AND hmap.profile_identifier != 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'verifying'
		WHEN SUM(CASE WHEN hmap.operation_type = 'install' AND hmap.status = 'verified' AND hmap.profile_identifier != 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'verified'
		ELSE ''
	END AS status,
	CASE
		WHEN SUM(CASE WHEN hmap.status = 'failed' AND hmap.profile_identifier = 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'failed'
		WHEN SUM(CASE WHEN (hmap.status IS NULL OR hmap.status = 'pending') AND hmap.profile_identifier = 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'pending'
		WHEN SUM(CASE WHEN hmap.operation_type = 'install' AND hmap.status = 'verifying' AND hmap.profile_identifier = 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'verifying'
		WHEN SUM(CASE WHEN hmap.operation_type = 'install' AND hmap.status = 'verified' AND hmap.profile_identifier = 'com.fleetdm.fleet.mdm.filevault' THEN 1 ELSE 0 END) > 0
			THEN 'verified'
		ELSE ''
	END AS filevault_status
FROM host_mdm_apple_profiles hmap
GROUP BY hmap.host_uuid
ON DUPLICATE KEY UPDATE status = VALUES(status), filevault_status = VALUES(filevault_status)`); err != nil {
		return fmt.Errorf("backfilling host_mdm_apple_profiles_status: %w", err)
	}

	if _, err := tx.Exec(`
INSERT INTO host_mdm_apple_declarations_status (host_uuid, status)
SELECT
	hmad.host_uuid,
	CASE
		WHEN SUM(CASE WHEN hmad.status = 'failed' AND hmad.declaration_name NOT IN ('Fleet macOS OS Updates', 'Fleet iOS OS Updates', 'Fleet iPadOS OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'failed'
		WHEN SUM(CASE WHEN (hmad.status IS NULL OR hmad.status = 'pending') AND hmad.declaration_name NOT IN ('Fleet macOS OS Updates', 'Fleet iOS OS Updates', 'Fleet iPadOS OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'pending'
		WHEN SUM(CASE WHEN hmad.operation_type = 'install' AND hmad.status = 'verifying' AND hmad.declaration_name NOT IN ('Fleet macOS OS Updates', 'Fleet iOS OS Updates', 'Fleet iPadOS OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'verifying'
		WHEN SUM(CASE WHEN hmad.operation_type = 'install' AND hmad.status = 'verified' AND hmad.declaration_name NOT IN ('Fleet macOS OS Updates', 'Fleet iOS OS Updates', 'Fleet iPadOS OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'verified'
		ELSE ''
	END AS status
FROM host_mdm_apple_declarations hmad
GROUP BY hmad.host_uuid
ON DUPLICATE KEY UPDATE status = VALUES(status)`); err != nil {
		return fmt.Errorf("backfilling host_mdm_apple_declarations_status: %w", err)
	}

	return nil
}

func Down_20261123143955(tx *sql.Tx) error {
	return nil
}
