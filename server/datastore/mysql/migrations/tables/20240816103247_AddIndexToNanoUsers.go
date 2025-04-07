package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240816103247, Down_20240816103247)
}

func Up_20240816103247(tx *sql.Tx) error {

	// This constraint is required for MySQL 8.4.2 because nano_enrollments foreign key expects nano_users.id to be unique.
	if !indexExistsTx(tx, "nano_users", "idx_unique_id") {
		_, err := tx.Exec(`ALTER TABLE nano_users ADD CONSTRAINT idx_unique_id UNIQUE (id)`)
		if err != nil {
			return fmt.Errorf("adding unique index to nano_users: %w", err)
		}
	}

	return nil
}

func Down_20240816103247(tx *sql.Tx) error {
	return nil
}
