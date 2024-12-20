package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241220114903, Down_20241220114903)
}

func Up_20241220114903(tx *sql.Tx) error {
	_, err := tx.Exec(`	
ALTER TABLE mdm_apple_declarations
    CHANGE raw_json raw_json MEDIUMTEXT COLLATE utf8mb4_unicode_ci NOT NULL -- 16MB max size`)
	if err != nil {
		return fmt.Errorf("failed to change mdm_apple_declarations.raw_json column; is there a very large DDM profile?: %w", err)
	}

	return nil
}

func Down_20241220114903(tx *sql.Tx) error {
	return nil
}
