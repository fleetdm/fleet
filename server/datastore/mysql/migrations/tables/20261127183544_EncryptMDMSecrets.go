package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261127183544, Down_20261127183544)
}

func Up_20261127183544(tx *sql.Tx) error {
	if colType := columnDataType(tx, "nano_devices", "bootstrap_token_b64"); colType != "" && colType != "mediumblob" {
		if _, err := tx.Exec(`ALTER TABLE nano_devices MODIFY COLUMN bootstrap_token_b64 mediumblob`); err != nil {
			return fmt.Errorf("alter bootstrap_token_b64 to mediumblob: %w", err)
		}
	}
	return nil
}

func Down_20261127183544(tx *sql.Tx) error {
	return nil
}
