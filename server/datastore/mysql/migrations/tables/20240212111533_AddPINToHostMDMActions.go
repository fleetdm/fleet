package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240212111533, Down_20240212111533)
}

func Up_20240212111533(tx *sql.Tx) error {
	// going with a VARCHAR instead of a number because the leading zeros are
	// important in a PIN. Being a VARCHAR will also make it easy to make larger
	// if needed in the future.
	//
	// An unlock_ref field is also necessary for Windows/Linux where unlocking is
	// done via a script, so we need a reference to that script's execution uuid
	// as we already have for lock_ref and wipe_ref.
	stmt := `ALTER TABLE host_mdm_actions
		ADD COLUMN unlock_pin VARCHAR(6) NULL,
		ADD COLUMN unlock_ref VARCHAR(36) NULL,
		DROP COLUMN suspended
`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("alter table host_mdm_actions: %w", err)
	}
	return nil
}

func Down_20240212111533(tx *sql.Tx) error {
	return nil
}
