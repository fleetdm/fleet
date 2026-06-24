package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260624210253, Down_20260624210253)
}

func Up_20260624210253(tx *sql.Tx) error {
	// Keyed by host_uuid (the device's MDM enrollment UDID) rather than
	// host_id so this table correlates directly with the nanomdm tables
	// (nano_enrollments.id, host_mdm_apple_profiles.host_uuid, etc.), which
	// is how the SCEP/ACME renewal path joins to it.
	//
	// No foreign key on host_uuid: per handbook/engineering/scaling-fleet.md,
	// host FKs are avoided on host-extra-data tables because they cause
	// InnoDB locking contention. Cleanup on host deletion is handled by the
	// additionalHostRefsByUUID map in server/datastore/mysql/hosts.go.
	if _, err := tx.Exec(`
		CREATE TABLE host_mdm_apple_enrollment_permissions (
			host_uuid     VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			access_rights INT          NOT NULL DEFAULT 8191,
			delivered_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (host_uuid)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("create host_mdm_apple_enrollment_permissions: %w", err)
	}

	// Backfill existing manually-enrolled Apple hosts. Before this feature every
	// enrollment profile was delivered with AccessRights=8191 (all permissions),
	// so that is the correct starting value for all rows that existed prior to
	// this migration.
	//
	// The host_mdm table is shared across MDM platforms. Restrict the backfill
	// to Apple platforms (darwin/ios/ipados) so this Apple-specific table does
	// not accumulate rows for Windows hosts that also enroll with
	// installed_from_dep=0 (e.g. GPO/Settings-app enrollment).
	//
	// INSERT IGNORE + the non-empty guard protect against blank uuids or the
	// rare case of two host rows sharing a uuid.
	if _, err := tx.Exec(`
		INSERT IGNORE INTO host_mdm_apple_enrollment_permissions (host_uuid, access_rights)
		SELECT h.uuid, 8191
		FROM host_mdm hm
		JOIN hosts h ON h.id = hm.host_id
		WHERE hm.enrolled = 1
		  AND hm.installed_from_dep = 0
		  AND h.platform IN ('darwin', 'ios', 'ipados')
		  AND h.uuid != ''
	`); err != nil {
		return fmt.Errorf("backfill host_mdm_apple_enrollment_permissions: %w", err)
	}

	return nil
}

func Down_20260624210253(tx *sql.Tx) error {
	return nil
}
