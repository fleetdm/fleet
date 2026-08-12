package tables

import "database/sql"

func init() {
	MigrationClient.AddMigration(Up_20260812223244, Down_20260812223244)
}

func Up_20260812223244(tx *sql.Tx) error {
	// The Autopilot ZTDID a Windows device supplies at MDM enrollment, in the MS-MDE2 ZeroTouchProvisioning context
	// item. It is the same GUID Microsoft Graph returns as windowsAutopilotDeviceIdentity.id, so it links an
	// enrollment to a pending Autopilot host exactly, without depending on the hardware serial. Empty for devices that
	// are not Autopilot-registered.
	_, err := tx.Exec(`
		ALTER TABLE mdm_windows_enrollments
		ADD COLUMN ztd_registration_id VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''`)
	if err != nil {
		return err
	}

	// host_autopilot_devices is filtered on tenant_id by every sync pass and on autopilot_device_id by every Autopilot
	// Windows MDM enrollment. The table holds one row per registered device, which the feature sizes at 100k+, so
	// without these both are full scans.
	if _, err := tx.Exec(`
		ALTER TABLE host_autopilot_devices
		ADD KEY idx_host_autopilot_tenant_id (tenant_id),
		ADD KEY idx_host_autopilot_device_id (autopilot_device_id)`); err != nil {
		return err
	}
	return nil
}

func Down_20260812223244(tx *sql.Tx) error {
	return nil
}
