package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20220708152154, Down_20220708152154)
}

func Up_20220708152154(tx *sql.Tx) error {
	_, err := tx.Exec(
		`ALTER TABLE host_batteries MODIFY health VARCHAR(20) NOT NULL;`,
	)
	return err
}

func Down_20220708152154(tx *sql.Tx) error {
	return nil
}
