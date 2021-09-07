package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210903132338, Down_20210903132338)
}

func Up_20210903132338(tx *sql.Tx) error {
	_, err := tx.Exec(`alter table host_users drop column id, drop primary key, add primary key(host_id, uid, username);`)
	if err != nil {
		return errors.Wrap(err, "dropping id from host_users")
	}
	return nil
}

func Down_20210903132338(tx *sql.Tx) error {
	return nil
}
