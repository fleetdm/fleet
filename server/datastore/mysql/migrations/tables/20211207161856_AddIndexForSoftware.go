package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211207161856, Down_20211207161856)
}

func Up_20211207161856(tx *sql.Tx) error {
	if _, err := tx.Exec(`create index software_listing_idx on software(name, id);`); err != nil {
		return errors.Wrap(err, "creating software index")
	}
	return nil
}

func Down_20211207161856(tx *sql.Tx) error {
	return nil
}
