package tables

import (
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

func init() {
	MigrationClient.AddMigration(Up_20241219104318, Down_20241219104318)
}

func Up_20241219104318(tx *sql.Tx) error {
	if columnExists(tx, "nano_commands", "subtype") {
		return nil
	}
	_, err := tx.Exec(fmt.Sprintf(`	
ALTER TABLE nano_commands
ADD COLUMN subtype enum('%s','%s') CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '%s'`,
		mdm.CommandSubtypeNone, mdm.CommandSubtypeProfileWithSecrets, mdm.CommandSubtypeNone))
	if err != nil {
		return fmt.Errorf("failed to create nano_commands.subtype column: %w", err)
	}

	return nil
}

func Down_20241219104318(_ *sql.Tx) error {
	return nil
}
