package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20231004123931, Down_20231004123931)
}

func Up_20231004123931(tx *sql.Tx) error {
	stmt := `
		CREATE TABLE foo (
			id BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
			host_ID INT(10) UNSIGNED NOT NULL,
			FOREIGN KEY (host_ID) REFERENCES hosts (id) ON DELETE CASCADE,
			`
	_, err := tx.Exec(stmt)
	if err != nil {
		return err
	}
	return nil
}

func Down_20231004123931(tx *sql.Tx) error {
	return nil
}
