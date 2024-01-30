package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240129162819, Down_20240129162819)
}

func Up_20240129162819(tx *sql.Tx) error {
	stmt := `
		UPDATE
			mdm_windows_configuration_profiles mwcp
		SET
			profile_uuid = CONCAT("w", mwcp.profile_uuid),
			updated_at = mwcp.updated_at  
		WHERE
			profile_uuid NOT LIKE "w%";
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add w prefix to windows updates profiles uuid: %w", err)
	}

	stmt = `
		UPDATE
			host_mdm_windows_profiles
		SET
			profile_uuid = CONCAT("w", profile_uuid)
		WHERE
			profile_uuid NOT LIKE "w%";
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add w prefix to windows updates profiles uuid in host mapping: %w", err)
	}

	return nil
}

func Down_20240129162819(tx *sql.Tx) error {
	return nil
}
