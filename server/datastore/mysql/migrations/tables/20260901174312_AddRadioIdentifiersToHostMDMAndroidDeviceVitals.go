package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260901174312, Down_20260901174312)
}

// Up_20260901174312 adds the hardware radio identifiers AMAPI reports in
// networkInfo — imei for GSM devices, meid for CDMA ones — to
// host_mdm_android_device_vitals. A device reports at most one of the two, and
// neither is reported for personally-owned devices, so both default to NULL
// like the rest of the vitals columns.
func Up_20260901174312(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE host_mdm_android_device_vitals
  ADD COLUMN imei varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD COLUMN meid varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL
`)
	if err != nil {
		return fmt.Errorf("adding radio identifier columns to host_mdm_android_device_vitals table: %w", err)
	}
	return nil
}

func Down_20260901174312(tx *sql.Tx) error {
	return nil
}
