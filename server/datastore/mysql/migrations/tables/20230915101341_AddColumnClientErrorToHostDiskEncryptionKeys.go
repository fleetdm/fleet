package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230915101341, Down_20230915101341)
}

func Up_20230915101341(tx *sql.Tx) error {
	stmt := `
          ALTER TABLE host_disk_encryption_keys
          ADD COLUMN client_error varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
 	 `
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add client_error to host_disk_encryption_keys: %w", err)
	}
	return nil
}

func Down_20230915101341(tx *sql.Tx) error {
	return nil
}
