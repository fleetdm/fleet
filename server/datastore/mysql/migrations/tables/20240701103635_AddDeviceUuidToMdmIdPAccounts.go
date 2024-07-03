package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240701103635, Down_20240701103635)
}

func Up_20240701103635(tx *sql.Tx) error {
	alterStmt := `
ALTER TABLE mdm_idp_accounts ADD COLUMN (
  host_uuid varchar(63) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  fleet_enroll_ref varchar(63) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
)`

	if _, err := tx.Exec(alterStmt); err != nil {
		return fmt.Errorf("failed to alter mdm_idp_accounts table: %w", err)
	}

	updateStmt := `
UPDATE
  mdm_idp_accounts mia
  LEFT JOIN host_mdm hmdm ON mia.uuid = hmdm.fleet_enroll_ref
  LEFT JOIN hosts h ON hmdm.host_id = h.id
SET
  mia.fleet_enroll_ref = mia.uuid,
  mia.host_uuid = COALESCE(h.uuid, ''),
  mia.updated_at = mia.updated_at`

	if _, err := tx.Exec(updateStmt); err != nil {
		return fmt.Errorf("failed to update data in mdm_idp_accounts: %w", err)
	}

	return nil
}

func Down_20240701103635(tx *sql.Tx) error {
	return nil
}
