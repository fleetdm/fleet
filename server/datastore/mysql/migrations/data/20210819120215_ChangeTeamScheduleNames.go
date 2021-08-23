package data

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210819120215, Down_20210819120215)
}

func Up_20210819120215(tx *sql.Tx) error {
	sql := `UPDATE packs SET name = CONCAT('Team: ', (select name from teams where id=(CAST(SUBSTRING_INDEX(pack_type,'-',-1) AS UNSIGNED)))) WHERE pack_type LIKE 'team-%'`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "update team pack names")
	}

	return nil
}

func Down_20210819120215(tx *sql.Tx) error {
	return nil
}
