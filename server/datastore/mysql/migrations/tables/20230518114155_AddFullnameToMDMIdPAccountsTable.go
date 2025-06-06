package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230518114155, Down_20230518114155)
}

func Up_20230518114155(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE mdm_idp_accounts
	  DROP COLUMN salt,
	  DROP COLUMN entropy,
	  DROP COLUMN iterations,
          ADD COLUMN fullname varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("alter mdm_idp_accounts table: %w", err)
	}
	return nil
}

func Down_20230518114155(tx *sql.Tx) error {
	return nil
}
