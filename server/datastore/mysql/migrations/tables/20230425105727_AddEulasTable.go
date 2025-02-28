package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20230425105727, Down_20230425105727)
}

func Up_20230425105727(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE eulas (
            id           int(10) unsigned NOT NULL,
            token        varchar(36),
            name         varchar(255),
            bytes        longblob,
            created_at   datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,

            PRIMARY KEY (id)
          ) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("creating eulas table: %w", err)
	}

	return nil
}

func Down_20230425105727(tx *sql.Tx) error {
	return nil
}
