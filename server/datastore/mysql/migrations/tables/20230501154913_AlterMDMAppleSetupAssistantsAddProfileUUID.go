package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230501154913, Down_20230501154913)
}

func Up_20230501154913(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE mdm_apple_setup_assistants ADD COLUMN profile_uuid VARCHAR(255) NOT NULL DEFAULT '';
`)
	return errors.Wrap(err, "add profile_uuid")
}

func Down_20230501154913(tx *sql.Tx) error {
	return nil
}
