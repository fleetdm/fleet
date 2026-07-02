package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260630120000, Down_20260630120000)
}

func Up_20260630120000(tx *sql.Tx) error {
	// The profile-manager cron builds Windows <Delete> commands from the content of a profile version a host still has, but that
	// content is gone from the live table once the profile is deleted (it was removed) or edited (it was overwritten).
	//
	// This single table retains that content, keyed by (profile_uuid, checksum) where checksum = md5(syncml) matching
	// host_mdm_windows_profiles.checksum. It replaces the delete-only mdm_windows_configuration_profiles_pending_delete table (which was
	// keyed by profile_uuid alone) so both the deletion path and the edit path share one mechanism:
	//   - deletion: retain the version being removed; the cron deletes all of its LocURIs not still enforced by another desired profile.
	//   - edit:     retain the version being overwritten; the cron deletes the LocURIs that version had but the new version dropped.
	// In both cases the cron computes "prior LocURIs minus still-desired LocURIs", with desired being empty for a deleted profile.
	//
	// There is intentionally no foreign key to mdm_windows_configuration_profiles: the reference-counted GC owns cleanup, so a row for a
	// deleted profile lingers harmlessly until GC rather than blocking the delete. Rows are dropped once no host_mdm_windows_profiles row
	// still has that checksum for the profile (every host has moved past that version or unenrolled).
	if _, err := tx.Exec(`
		CREATE TABLE mdm_windows_configuration_profiles_prior_content (
			profile_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			checksum     BINARY(16) NOT NULL,
			syncml       MEDIUMBLOB NOT NULL,
			created_at   DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			PRIMARY KEY (profile_uuid, checksum)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create mdm_windows_configuration_profiles_prior_content: %w", err)
	}

	// Carry over content already retained for in-flight deletions so their <Delete> commands aren't lost across the upgrade. The prior
	// table stored only the last version; key it by that version's checksum.
	if _, err := tx.Exec(`
		INSERT IGNORE INTO mdm_windows_configuration_profiles_prior_content (profile_uuid, checksum, syncml, created_at)
		SELECT profile_uuid, UNHEX(MD5(syncml)), syncml, created_at
		FROM mdm_windows_configuration_profiles_pending_delete`); err != nil {
		return fmt.Errorf("backfill mdm_windows_configuration_profiles_prior_content from pending_delete: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE mdm_windows_configuration_profiles_pending_delete`); err != nil {
		return fmt.Errorf("drop mdm_windows_configuration_profiles_pending_delete: %w", err)
	}

	// The prior-content GC and the deleted-profile host-row cleanup probe host_mdm_windows_profiles by (profile_uuid, checksum). The
	// GC's NOT EXISTS now also filters on checksum.
	if _, err := tx.Exec(`ALTER TABLE host_mdm_windows_profiles
		DROP INDEX idx_host_mdm_windows_profiles_profile_uuid,
		ADD INDEX idx_host_mdm_windows_profiles_profile_uuid_checksum (profile_uuid, checksum)`); err != nil {
		return fmt.Errorf("replace profile_uuid index with (profile_uuid, checksum) on host_mdm_windows_profiles: %w", err)
	}
	return nil
}

func Down_20260630120000(tx *sql.Tx) error {
	return nil
}
