package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220704083846, Down_20220704083846)
}

func Up_20220704083846(tx *sql.Tx) error {
	fmt.Println("Adding software_id column to software_cve table...")

	_, err := tx.Exec(`
	ALTER TABLE software_cve ADD COLUMN software_id bigint(20) UNSIGNED NULL, ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding software_id to software_cve")
	}
	fmt.Println("Done adding software_id column to software_cve table...")

	return nil
}

func Down_20220704083846(tx *sql.Tx) error {
	return nil
}
