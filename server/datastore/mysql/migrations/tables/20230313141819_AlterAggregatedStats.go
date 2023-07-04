package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230313141819, Down_20230313141819)
}

func Up_20230313141819(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE aggregated_stats ADD COLUMN global_stats tinyint(1) NOT NULL DEFAULT 0;" +
			"ALTER TABLE aggregated_stats DROP PRIMARY KEY, ADD PRIMARY KEY(`id`, `type`, `global_stats`)")
	if err != nil {
		return errors.Wrap(err, "add global_stats column")
	}

	// pre-existing rows with id=0 are global stats, and from now on when id=0
	// and global_stats=0 it will mean "hosts that are part of no team" instead
	// of "all teams/global"
	_, err = tx.Exec("UPDATE aggregated_stats SET global_stats=1 WHERE id=0")
	if err != nil {
		return errors.Wrap(err, "update global_stats flag")
	}

	return nil
}

func Down_20230313141819(tx *sql.Tx) error {
	return nil
}
