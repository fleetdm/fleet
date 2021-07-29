package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210721171531, Down_20210721171531)
}

func Up_20210721171531(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS software_cpe (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			software_id bigint unsigned,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			cpe varchar(255) NOT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY fk_software_cpe_software_id (software_id) REFERENCES software(id) ON DELETE CASCADE
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create cpe")
	}

	return nil
}

func Down_20210721171531(tx *sql.Tx) error {
	return nil
}
