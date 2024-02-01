package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231025120016, Down_20231025120016)
}

func Up_20231025120016(tx *sql.Tx) error {
	deleteDuplicatesStmt := `
DELETE a
FROM mdm_idp_accounts a
LEFT JOIN (
    -- MAX(uuid) is completely arbitrary as it'll compare the UUIDs
    -- lexicographically, but we don't have a better field to compare the rows.
    SELECT email, MAX(uuid) as latest_uuid
    FROM mdm_idp_accounts
    GROUP BY email
) b ON a.email = b.email AND a.uuid = b.latest_uuid
WHERE b.latest_uuid IS NULL;
`

	addIdxStmt := `
ALTER TABLE mdm_idp_accounts
ADD UNIQUE KEY unique_idp_email (email)`

	addTimestampsStmt := `
ALTER TABLE mdm_idp_accounts 
ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP`

	// delete duplicates
	if _, err := tx.Exec(deleteDuplicatesStmt); err != nil {
		return fmt.Errorf("failed to delete duplicated emails in mdm_idp_accounts table: %w", err)
	}

	// add an index to prevent further duplicates
	if _, err := tx.Exec(addIdxStmt); err != nil {
		return fmt.Errorf("failed to delete duplicated emails in mdm_idp_accounts table: %w", err)
	}

	// add missing timestamps
	if _, err := tx.Exec(addTimestampsStmt); err != nil {
		return fmt.Errorf("failed to delete duplicated emails in mdm_idp_accounts table: %w", err)
	}

	return nil
}

func Down_20231025120016(tx *sql.Tx) error {
	return nil
}
