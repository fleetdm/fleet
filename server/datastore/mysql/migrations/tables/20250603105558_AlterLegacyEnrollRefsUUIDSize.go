package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250603105558, Down_20250603105558)
}

func Up_20250603105558(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE legacy_host_mdm_enroll_refs
		CHANGE COLUMN host_uuid host_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;
`)
	if err != nil {
		return fmt.Errorf("failed to alter column host_uuid of legacy_host_mdm_enroll_refs: %w", err)
	}

	_, err = tx.Exec(`
	ALTER TABLE legacy_host_mdm_idp_accounts
		CHANGE COLUMN host_uuid host_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;
`)
	if err != nil {
		return fmt.Errorf("failed to alter column host_uuid of legacy_host_mdm_idp_accounts: %w", err)
	}

	_, err = tx.Exec(`
	ALTER TABLE host_mdm_idp_accounts
		CHANGE COLUMN host_uuid host_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;
`)
	if err != nil {
		return fmt.Errorf("failed to alter column host_uuid of host_mdm_idp_accounts: %w", err)
	}

	return nil
}

func Down_20250603105558(tx *sql.Tx) error {
	return nil
}
