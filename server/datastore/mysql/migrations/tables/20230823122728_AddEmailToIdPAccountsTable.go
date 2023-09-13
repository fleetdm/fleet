package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230823122728, Down_20230823122728)
}

func Up_20230823122728(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE mdm_idp_accounts
          ADD COLUMN email varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add email to mdm_idp_accounts: %w", err)
	}
	return nil
}

func Down_20230823122728(tx *sql.Tx) error {
	return nil
}
