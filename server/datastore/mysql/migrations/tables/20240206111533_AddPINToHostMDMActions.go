package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240206111533, Down_20240206111533)
}

func Up_20240206111533(tx *sql.Tx) error {
	// going with a VARCHAR instead of a number because the leading zeros are
	// important in a PIN. Being a VARCHAR will also make it easy to make larger
	// if needed in the future.
	stmt := `ALTER TABLE host_mdm_actions ADD COLUMN unlock_pin VARCHAR(6) NULL`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("alter table host_mdm_actions: %w", err)
	}
	return nil
}

func Down_20240206111533(tx *sql.Tx) error {
	return nil
}
