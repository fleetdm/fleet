package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20241209164540, Down_20241209164540)
}

func Up_20241209164540(tx *sql.Tx) error {

	if tableExists(tx, "secret_variables") {
		return nil
	}

	_, err := tx.Exec(`
		CREATE TABLE secret_variables (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		name VARCHAR(255) NOT NULL,
		value BLOB NOT NULL,
		created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		PRIMARY KEY (id),
		CONSTRAINT idx_name UNIQUE (name)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`,
	)
	return err
}

func Down_20241209164540(_ *sql.Tx) error {
	return nil
}
