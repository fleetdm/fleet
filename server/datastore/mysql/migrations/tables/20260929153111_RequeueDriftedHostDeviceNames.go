package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260929153111, Down_20260929153111)
}

func Up_20260929153111(tx *sql.Tx) error {
	// Rows aren't removed when a host turns off MDM, so only re-queue hosts that are still
	// eligible (same predicate as BulkUpsertHostDeviceNameEnforcement).
	if _, err := tx.Exec(`
		UPDATE host_mdm_apple_device_names hmadn
		JOIN hosts h ON h.uuid = hmadn.host_uuid
		JOIN nano_enrollments ne ON ne.id = h.uuid
		JOIN host_mdm hm ON hm.host_id = h.id
		SET hmadn.status = NULL, hmadn.command_uuid = NULL, hmadn.retries = 0
		WHERE hmadn.status = 'failed'
			AND hmadn.detail = 'Host was renamed on the device and no longer matches the fleet''s naming template.'
			AND h.platform IN ('darwin', 'ios', 'ipados')
			AND ne.enabled = 1
			AND ne.type = 'Device'
			AND hm.enrolled = 1
			AND hm.is_personal_enrollment = 0
	`); err != nil {
		return fmt.Errorf("re-queueing drifted host device names: %w", err)
	}
	return nil
}

func Down_20260929153111(tx *sql.Tx) error {
	return nil
}
