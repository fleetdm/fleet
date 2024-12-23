package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241223115925, Down_20241223115925)
}

func Up_20241223115925(tx *sql.Tx) error {
	// secrets_updated_at are updated when profile contents have not changed but secret variables in the profile have changed
	_, err := tx.Exec(`ALTER TABLE mdm_apple_configuration_profiles
    	ADD COLUMN secrets_updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN uploaded_at TIMESTAMP(6) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}

	// Add secrets_updated_at column to host_mdm_apple_profiles
	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_profiles
    	ADD COLUMN secrets_updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)`)
	if err != nil {
		return fmt.Errorf("failed to add secrets_updated_at to host_mdm_apple_profiles table: %w", err)
	}

	// secrets_updated_at are updated when profile contents have not changed but secret variables in the profile have changed
	_, err = tx.Exec(`ALTER TABLE mdm_apple_declarations
    	ADD COLUMN secrets_updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		MODIFY COLUMN uploaded_at TIMESTAMP(6) NULL DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_declarations table: %w", err)
	}

	// Add secrets_updated_at column to host_mdm_apple_declarations
	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_declarations
		-- defaulting to NULL for backward compatibility with existing declarations
    	ADD COLUMN secrets_updated_at TIMESTAMP(6) NULL`)
	if err != nil {
		return fmt.Errorf("failed to add secrets_updated_at to host_mdm_apple_declarations table: %w", err)
	}

	return nil
}

func Down_20241223115925(_ *sql.Tx) error {
	return nil
}
