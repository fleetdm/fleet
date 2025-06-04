package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func init() {
	MigrationClient.AddMigration(Up_20241220100000, Down_20241220100000)
}

func Up_20241220100000(tx *sql.Tx) error {
	if !columnExists(tx, "nano_commands", "subtype") {
		_, err := tx.Exec(fmt.Sprintf(`	
ALTER TABLE nano_commands
ADD COLUMN subtype enum('%s','%s') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '%s'`,
			mdm.CommandSubtypeNone, mdm.CommandSubtypeProfileWithSecrets, mdm.CommandSubtypeNone))
		if err != nil {
			return fmt.Errorf("failed to create nano_commands.subtype column: %w", err)
		}
	}

	// With secret variable support, it is possible to have the whole profile as one secret ($FLEET_SECRET_PROFILE),
	// which will not be XML when stored. It is cleaner to remove the check than to add a special caveat to documentation.
	if constraintExists(tx, "nano_commands", "nano_commands_chk_3") {
		_, err := tx.Exec(`ALTER TABLE nano_commands DROP CONSTRAINT nano_commands_chk_3`)
		if err != nil {
			return fmt.Errorf("failed to drop nano_commands_chk_3 constraint: %w", err)
		}
	}

	return nil
}

func Down_20241220100000(_ *sql.Tx) error {
	return nil
}
