package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210802135933, Down_20210802135933)
}

func Up_20210802135933(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS software_cve (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			cpe_id int(10) unsigned,
			cve varchar(255) NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			FOREIGN KEY fk_software_cve_cpe_id (cpe_id) REFERENCES software_cpe(id) ON DELETE CASCADE,
			UNIQUE KEY unique_cpe_cve(cpe_id, cve) 
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create cve")
	}
	return nil
}

func Down_20210802135933(tx *sql.Tx) error {
	return nil
}
