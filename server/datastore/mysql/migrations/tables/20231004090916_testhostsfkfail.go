package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231004090916, Down_20231004090916)
}

func Up_20231004090916(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE foobar (
			id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			host_id INT(10) UNSIGNED NOT NULL,

			FOREIGN KEY (host_id) REFERENCES hosts(id)
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table foobar: %w", err)
	}

	return nil
}

func Down_20231004090916(tx *sql.Tx) error {
	return nil
}
