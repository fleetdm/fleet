package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220704083846, Down_20220704083846)
}

func Up_20220704083846(tx *sql.Tx) error {
	_, err := tx.Exec(`
	ALTER TABLE software_cve ADD COLUMN software_id bigint(20) UNSIGNED NULL, ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding software_id to software_cve")
	}

	return nil
}

func Down_20220704083846(tx *sql.Tx) error {
	return nil
}
