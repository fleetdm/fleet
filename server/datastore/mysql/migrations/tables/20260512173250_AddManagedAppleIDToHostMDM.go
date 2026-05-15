package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260512173250, Down_20260512173250)
}

func Up_20260512173250(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		ALTER TABLE host_mdm
		ADD COLUMN managed_apple_id VARCHAR(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER fleet_enroll_ref
	`); err != nil {
		return fmt.Errorf("adding managed_apple_id to host_mdm: %w", err)
	}

	// Backfill existing personal (User Enrollment / BYOD) hosts from the IDP
	// account email they were enrolled with. This mirrors what TokenUpdate does
	// for new enrollments, so already-enrolled hosts don't need to re-enroll to
	// participate in VPP user provisioning.
	if _, err := tx.Exec(`
		UPDATE host_mdm hm
		JOIN hosts h ON h.id = hm.host_id
		JOIN host_mdm_idp_accounts hmia ON hmia.host_uuid = h.uuid
		JOIN mdm_idp_accounts mia ON mia.uuid = hmia.account_uuid
		SET hm.managed_apple_id = mia.email
		WHERE hm.managed_apple_id IS NULL
			AND hm.is_personal_enrollment = 1
			AND mia.email <> ''
	`); err != nil {
		return fmt.Errorf("backfilling managed_apple_id on host_mdm: %w", err)
	}
	return nil
}

func Down_20260512173250(tx *sql.Tx) error {
	return nil
}
