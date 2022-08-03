package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/apple/scep/scep_mysql"
	nanodep_mysql "github.com/micromdm/nanodep/storage/mysql"
	nanomdm_mysql "github.com/micromdm/nanomdm/storage/mysql"
)

func init() {
	MigrationClient.AddMigration(Up_20220725165245, Down_20220725165245)
}

func Up_20220725165245(tx *sql.Tx) error {
	// (1) Apply MDM SCEP schema.
	_, err := tx.Exec(scep_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM SCEP schema: %w", err)
	}

	// (2) Apply MDM Core schema.
	_, err = tx.Exec(nanomdm_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM core schema: %w", err)
	}

	// (3) Apply MDM Core extra tables.
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS mdm_apple_current_push_topic(
		topic VARCHAR(255) NOT NULL,

		-- The unique_value column and its constraint enforces a one-row table.
		-- TODO(lucas): Discuss other alternatives.
		unique_value ENUM('unique') NOT NULL,
		UNIQUE (unique_value),

		CHECK (topic != '')
	);`)
	if err != nil {
		return fmt.Errorf("failed to create mdm_apple_current_push_topic: %w", err)
	}

	// (4) Apply MDM DEP schema.
	_, err = tx.Exec(nanodep_mysql.Schema)
	if err != nil {
		return fmt.Errorf("failed to apply MDM core schema: %w", err)
	}

	return nil
}

func Down_20220725165245(tx *sql.Tx) error {
	return nil
}
