package data

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up20171212182459, Down20171212182459)
}

func Up20171212182459(tx *sql.Tx) error {
	sql := `DELETE FROM packs WHERE deleted = 1`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "delete packs")
	}
	return nil
}

func Down20171212182459(tx *sql.Tx) error {
	return nil
}
