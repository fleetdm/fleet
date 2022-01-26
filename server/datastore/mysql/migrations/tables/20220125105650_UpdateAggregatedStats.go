package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220125105650, Down_20220125105650)
}

func Up_20220125105650(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE aggregated_stats MODIFY id bigint(20) unsigned NOT NULL`); err != nil {
		return errors.Wrap(err, "make aggregated_stats.id bigint")
	}
	if _, err := tx.Exec("create index aggregated_stats_type_idx on aggregated_stats(`type`);"); err != nil {
		return errors.Wrap(err, "creating aggregated_stats index")
	}
	return nil
}

func Down_20220125105650(tx *sql.Tx) error {
	return nil
}
