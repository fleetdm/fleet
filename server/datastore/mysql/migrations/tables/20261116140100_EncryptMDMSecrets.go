package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261116140100, Down_20261116140100)
}

func Up_20261116140100(tx *sql.Tx) error {
	// Change bootstrap_token_b64 from text to mediumblob to hold encrypted data.
	// unlock_token is already mediumblob, no change needed.
	if colType := columnDataType(tx, "nano_devices", "bootstrap_token_b64"); colType != "" && colType != "mediumblob" {
		if _, err := tx.Exec(`ALTER TABLE nano_devices MODIFY COLUMN bootstrap_token_b64 mediumblob`); err != nil {
			return fmt.Errorf("alter bootstrap_token_b64 to mediumblob: %w", err)
		}
	}
	return nil
}

func Down_20261116140100(tx *sql.Tx) error {
	return nil
}
