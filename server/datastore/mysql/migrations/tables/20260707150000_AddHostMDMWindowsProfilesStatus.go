package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260707150000, Down_20260707150000)
}

// Up_20260707150000 creates host_mdm_windows_profiles_status, a per-host rollup of the aggregate
// Windows configuration-profile delivery status. It materializes exactly one bucket per host
// ('failed'|'pending'|'verifying'|'verified'|”), which is the value the Windows profiles summary
// previously recomputed on every request via a correlated aggregation over the whole
// host_mdm_windows_profiles table (O(hosts x profiles-per-host)). Reading the rollup instead makes
// GET /configuration_profiles/summary O(hosts). See issue #48340.
//
// The bucket priority (failed > pending > verifying > verified), the reserved-profile exclusion, the
// install-only filter for verifying/verified, and the NULL-as-pending rule are kept byte-for-byte in
// sync with windowsHostProfileStatusSubquery (server/datastore/mysql/microsoft_mdm.go). The enum
// strings and the single reserved Windows profile name ("Windows OS Updates") are hardcoded here so
// this migration stays self-contained; the reconciler cron and the in-transaction write-path
// maintenance keep the table correct going forward, so any later divergence self-heals.
func Up_20260707150000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE host_mdm_windows_profiles_status (
	host_uuid  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
	status     VARCHAR(20)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	updated_at DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (host_uuid),
	KEY idx_host_mdm_windows_profiles_status_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating host_mdm_windows_profiles_status table: %w", err)
	}

	// Backfill one row per host. GROUP BY host_uuid is a leftmost prefix of the clustered PRIMARY KEY
	// (host_uuid, profile_uuid), so MySQL streams the aggregation in index order without a temp table,
	// which keeps this bounded even on multi-million-row installs. Rows that resolve to '' (host has only
	// reserved profiles) are stored as '' and ignored by the summary read, matching prior behavior.
	if _, err := tx.Exec(`
INSERT INTO host_mdm_windows_profiles_status (host_uuid, status)
SELECT
	hmwp.host_uuid,
	CASE
		WHEN SUM(CASE WHEN hmwp.status = 'failed' AND hmwp.profile_name NOT IN ('Windows OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'failed'
		WHEN SUM(CASE WHEN (hmwp.status IS NULL OR hmwp.status = 'pending') AND hmwp.profile_name NOT IN ('Windows OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'pending'
		WHEN SUM(CASE WHEN hmwp.operation_type = 'install' AND hmwp.status = 'verifying' AND hmwp.profile_name NOT IN ('Windows OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'verifying'
		WHEN SUM(CASE WHEN hmwp.operation_type = 'install' AND hmwp.status = 'verified' AND hmwp.profile_name NOT IN ('Windows OS Updates') THEN 1 ELSE 0 END) > 0
			THEN 'verified'
		ELSE ''
	END AS status
FROM host_mdm_windows_profiles hmwp
GROUP BY hmwp.host_uuid`); err != nil {
		return fmt.Errorf("backfilling host_mdm_windows_profiles_status: %w", err)
	}

	return nil
}

func Down_20260707150000(tx *sql.Tx) error {
	return nil
}
