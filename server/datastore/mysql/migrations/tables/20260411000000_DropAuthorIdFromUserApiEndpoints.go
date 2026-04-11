package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20260411000000, Down_20260411000000)
}

func Up_20260411000000(tx *sql.Tx) error {
	// Find the foreign key constraint name for the author_id column specifically.
	var constraintName string
	err := tx.QueryRow(`
		SELECT CONSTRAINT_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_NAME = 'user_api_endpoints'
		  AND COLUMN_NAME = 'author_id'
		  AND CONSTRAINT_SCHEMA = DATABASE()
		  AND REFERENCED_TABLE_NAME = 'users'
	`).Scan(&constraintName)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("look up author_id foreign key: %w", err)
	}

	if constraintName != "" {
		escaped := strings.ReplaceAll(constraintName, "`", "``")
		if _, err := tx.Exec(fmt.Sprintf(
			"ALTER TABLE user_api_endpoints DROP FOREIGN KEY `%s`", escaped,
		)); err != nil {
			return fmt.Errorf("drop author_id foreign key: %w", err)
		}
	}

	_, err = tx.Exec(`ALTER TABLE user_api_endpoints DROP COLUMN author_id`)
	return err
}

func Down_20260411000000(tx *sql.Tx) error {
	return nil
}
