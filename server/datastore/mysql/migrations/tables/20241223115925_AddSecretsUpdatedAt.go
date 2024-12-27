package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241223115925, Down_20241223115925)
}

func Up_20241223115925(tx *sql.Tx) error {
	// Using DATETIME instead of TIMESTAMP for secrets_updated_at to avoid future Y2K38 issues,
	// since this date is used to detect if profile needs to be updated.

	// secrets_updated_at are updated when profile contents have not changed but secret variables in the profile have changed
	_, err := tx.Exec(`ALTER TABLE mdm_apple_configuration_profiles
    	ADD COLUMN secrets_updated_at DATETIME(6) NULL,
		MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN uploaded_at TIMESTAMP(6) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_profiles
    	ADD COLUMN secrets_updated_at DATETIME(6) NULL`)
	if err != nil {
		return fmt.Errorf("failed to add secrets_updated_at to host_mdm_apple_profiles table: %w", err)
	}

	// secrets_updated_at are updated when profile contents have not changed but secret variables in the profile have changed
	_, err = tx.Exec(`ALTER TABLE mdm_apple_declarations
    	ADD COLUMN secrets_updated_at DATETIME(6) NULL,
		MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN uploaded_at TIMESTAMP(6) NULL DEFAULT NULL,
    	-- token is used to identify if declaration needs to be re-applied
    	ADD COLUMN token BINARY(16) GENERATED ALWAYS AS
    		(UNHEX(MD5(CONCAT(raw_json, IFNULL(secrets_updated_at, ''))))) STORED NULL`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_declarations table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_declarations
		-- defaulting to NULL for backward compatibility with existing declarations
    	ADD COLUMN secrets_updated_at DATETIME(6) NULL,
    	-- token is used to identify if declaration needs to be re-applied
    	ADD COLUMN token BINARY(16) NOT NULL`)
	if err != nil {
		return fmt.Errorf("failed to alter host_mdm_apple_declarations table: %w", err)
	}

	return nil
}

func Down_20241223115925(_ *sql.Tx) error {
	return nil
}
