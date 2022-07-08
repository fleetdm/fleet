package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220708095046, Down_20220708095046)
}

func Up_20220708095046(tx *sql.Tx) error {
	fmt.Println("Adding unique constraint on (cve, software_id) to software_cve table...")
	_, err := tx.Exec(`
	ALTER TABLE software_cve ADD CONSTRAINT unq_software_id_cve UNIQUE (software_id, cve), ALGORITHM=INPLACE, LOCK=NONE;
`)
	if err != nil {
		return errors.Wrapf(err, "adding uniq constraint to software_id on software_cve")
	}
	fmt.Println("Done Adding unique constraint on (cve, software_id) to software_cve table...")

	return nil
}

func Down_20220708095046(tx *sql.Tx) error {
	return nil
}
