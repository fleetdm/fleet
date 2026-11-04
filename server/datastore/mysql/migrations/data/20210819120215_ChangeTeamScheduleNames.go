package data

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210819120215, Down_20210819120215)
}

func Up_20210819120215(tx *sql.Tx) error {
	// Data migrations run after all table migrations, so on a fresh install the
	// packs table has already been dropped (and had no rows to rename).
	var packsExists bool
	if err := tx.QueryRow(
		`SELECT COUNT(*) > 0 FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'packs'`,
	).Scan(&packsExists); err != nil {
		return errors.Wrap(err, "check packs table exists")
	}
	if !packsExists {
		return nil
	}

	sql := `UPDATE packs SET name = CONCAT('Team: ', (select name from teams where id=(CAST(SUBSTRING_INDEX(pack_type,'-',-1) AS UNSIGNED)))) WHERE pack_type LIKE 'team-%'`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "update team pack names")
	}

	return nil
}

func Down_20210819120215(tx *sql.Tx) error {
	return nil
}
