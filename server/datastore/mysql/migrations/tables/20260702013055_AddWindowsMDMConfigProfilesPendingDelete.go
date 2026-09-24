package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260702013055, Down_20260702013055)
}

func Up_20260702013055(tx *sql.Tx) error {
	// When a Windows MDM configuration profile is deleted, its <Delete> commands cannot be generated until the profile-manager cron
	// reconciles the surviving host_mdm_windows_profiles rows, but the SyncML those <Delete> commands are built from lives only on the
	// now-deleted definition row. This table retains that content (profile_uuid, team_id, name, syncml) past the logical delete so the
	// cron can own removals asynchronously in its bounded 2,000-host batches, the same way it already owns team-transfer removals. That
	// makes the delete endpoints O(profiles) instead of O(profiles x hosts), fixing the large-removal timeout.
	//
	// It is a separate table rather than an in-place soft delete (a deleted_at column) because UNIQUE(team_id, name) on
	// mdm_windows_configuration_profiles would otherwise block re-adding a same-named profile while the deleted row lingered.
	//
	// Rows are garbage-collected (reference-counted) once no host_mdm_windows_profiles row still references the profile, so the
	// retained content survives exactly as long as some host still needs its <Delete> (e.g. a host that was offline when the profile
	// was deleted).
	if _, err := tx.Exec(`
		CREATE TABLE mdm_windows_configuration_profiles_pending_delete (
			profile_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			team_id      INT UNSIGNED NOT NULL DEFAULT 0,
			name         VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			syncml       MEDIUMBLOB NOT NULL,
			created_at   DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			PRIMARY KEY (profile_uuid)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create mdm_windows_configuration_profiles_pending_delete: %w", err)
	}

	// The reference-counted GC (and the deleted-profile host-row cleanup) look up host_mdm_windows_profiles by profile_uuid, which the
	// table's PRIMARY KEY (host_uuid, profile_uuid) cannot serve. Add a profile_uuid index so those become index probes rather than
	// full scans. Adding a secondary index is an in-place operation by default, so this stays fast even on large fleets.
	if _, err := tx.Exec(`ALTER TABLE host_mdm_windows_profiles
		ADD INDEX idx_host_mdm_windows_profiles_profile_uuid (profile_uuid)`); err != nil {
		return fmt.Errorf("add profile_uuid index to host_mdm_windows_profiles: %w", err)
	}
	return nil
}

func Down_20260702013055(tx *sql.Tx) error {
	return nil
}
