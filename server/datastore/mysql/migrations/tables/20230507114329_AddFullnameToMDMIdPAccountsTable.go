package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230507114329, Down_20230507114329)
}

func Up_20230507114329(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE mdm_idp_accounts
	  DROP COLUMN salt,
	  DROP COLUMN entropy,
	  DROP COLUMN iterations,
          ADD COLUMN fullname varchar(256) NOT NULL DEFAULT ''
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("alter mdm_idp_accounts table: %w", err)
	}
	return nil
}

func Down_20230507114329(tx *sql.Tx) error {
	return nil
}
