package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240710152744, Down_20240710152744)
}

func Up_20240710152744(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE mdm_idp_accounts
		ADD INDEX idx_mdm_idp_accounts_host_uuid (host_uuid)`,
	)
	if err != nil {
		return fmt.Errorf("failed to add idx_mdm_idp_accounts_host_uuid: %w", err)
	}
	return nil
}

func Down_20240710152744(tx *sql.Tx) error {
	return nil
}
