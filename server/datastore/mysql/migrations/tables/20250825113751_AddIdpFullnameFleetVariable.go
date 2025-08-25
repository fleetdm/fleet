package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250825113058, Down_20250825113058)
}

func Up_20250825113058(tx *sql.Tx) error {
	if _, err := tx.Exec("INSERT INTO fleet_variables (name, is_prefix) VALUES ('FLEET_VAR_HOST_END_USER_IDP_FULL_NAME', 0)"); err != nil {
		return fmt.Errorf("inserting fullname idp fleet variable: %w", err)
	}
	return nil
}

func Down_20250825113058(tx *sql.Tx) error {
	return nil
}
