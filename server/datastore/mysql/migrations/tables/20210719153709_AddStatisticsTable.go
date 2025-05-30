package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210719153709, Down_20210719153709)
}

func Up_20210719153709(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS statistics (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			anonymous_identifier varchar(255) NOT NULL,
			PRIMARY KEY (id) 
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create statistics")
	}

	return nil
}

func Down_20210719153709(tx *sql.Tx) error {
	return nil
}
