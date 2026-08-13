package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260812223244, Down_20260812223244)
}

func Up_20260812223244(tx *sql.Tx) error {
	// The Autopilot device ID a Windows device supplies at MDM enrollment, in the MS-MDE2 ZeroTouchProvisioning
	// context item. It is the same GUID Microsoft Graph returns as windowsAutopilotDeviceIdentity.id, so it links an
	// enrollment to a pending Autopilot host exactly, without depending on the hardware serial. Empty for devices that
	// are not Autopilot-registered.
	//
	// ZTD (Zero Touch Deployment) is Microsoft's codename for Autopilot. The name follows the device's own
	// ZtdRegistrationId registry value rather than Graph, which calls the same GUID plain "id".
	if !columnExists(tx, "mdm_windows_enrollments", "ztd_registration_id") {
		if _, err := tx.Exec(`
			ALTER TABLE mdm_windows_enrollments
			ADD COLUMN ztd_registration_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add ztd_registration_id to mdm_windows_enrollments: %w", err)
		}
	}

	// autopilot_device_id is looked up on every Autopilot Windows MDM enrollment and is unique per row.
	if !indexExistsTx(tx, "host_autopilot_devices", "idx_host_autopilot_tenant_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_autopilot_devices
			ADD KEY idx_host_autopilot_tenant_id (tenant_id)`); err != nil {
			return fmt.Errorf("add idx_host_autopilot_tenant_id to host_autopilot_devices: %w", err)
		}
	}

	// Adding index for tenant_id to future-proof and because switching to a different tenant may keep some rows in this table,
	// so we can have multiple tenant_ids.
	if !indexExistsTx(tx, "host_autopilot_devices", "idx_host_autopilot_device_id") {
		if _, err := tx.Exec(`
			ALTER TABLE host_autopilot_devices
			ADD KEY idx_host_autopilot_device_id (autopilot_device_id)`); err != nil {
			return fmt.Errorf("add idx_host_autopilot_device_id to host_autopilot_devices: %w", err)
		}
	}
	return nil
}

func Down_20260812223244(tx *sql.Tx) error {
	return nil
}
