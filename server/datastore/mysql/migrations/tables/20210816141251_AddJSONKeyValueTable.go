package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210816141251, Down_20210816141251)
}

func Up_20210816141251(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS kv_json (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			json_key varchar(255) NOT NULL,
			json_value JSON NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY unique_kv_json_key(json_key) 
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create kv_json")
	}
	return nil
}

func Down_20210816141251(tx *sql.Tx) error {
	return nil
}
