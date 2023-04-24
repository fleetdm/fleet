package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230421160713, Down_20230421160713)
}

func Up_20230421160713(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE eulas (
            token        varchar(36),
            name         varchar(255),
            bytes        longblob,
            created_at   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,

            PRIMARY KEY (token)
          )`)
	if err != nil {
		return fmt.Errorf("creating eulas table: %w", err)
	}

	return nil
}

func Down_20230421160713(tx *sql.Tx) error {
	return nil
}
