package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231109115838, Down_20231109115838)
}

func Up_20231109115838(tx *sql.Tx) error {
	stmt := `
		ALTER TABLE activities
		ADD COLUMN user_email varchar(255) NOT NULL DEFAULT '';
	`
	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("add user_email to activities: %w", err)
	}

	stmt = `
		UPDATE activities t1
		INNER JOIN users t2 ON t1.user_id = t2.id 
		SET t1.user_email = t2.email;
	`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("insert existing emails into activities: %w", err)
	}

	return nil
}

func Down_20231109115838(tx *sql.Tx) error {
	return nil
}
