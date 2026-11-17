package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20261115001714, Down_20261115001714)
}

func Up_20261115001714(tx *sql.Tx) error {
	// GetLatestCronStats runs on every cron schedule tick on every Fleet
	// server, once for stats_type = 'scheduled' and once for
	// stats_type = 'triggered'. With only (name, created_at) indexed, the
	// 'triggered' half scans every cron_stats row for the schedule name
	// (~2 days of apple_mdm_worker ticks) to return no rows, because that
	// worker is only ever scheduled. Indexing (name, stats_type,
	// created_at) lets both halves resolve via the index.
	// See https://github.com/fleetdm/fleet/issues/58777.
	//
	// GetLatestCronStats is the only query that filters cron_stats by name
	// and it always constrains stats_type too, so the composite index
	// makes idx_cron_stats_name_created_at redundant; drop it. The new
	// index is created first so the old one remains as a fallback if this
	// step fails or is interrupted.
	if err := addIndexesTx(tx, "cron_stats",
		indexDef{"idx_cron_stats_name_stats_type_created_at", "name, stats_type, created_at"}); err != nil {
		return err
	}

	if indexExistsTx(tx, "cron_stats", "idx_cron_stats_name_created_at") {
		if _, err := tx.Exec(`ALTER TABLE cron_stats DROP INDEX idx_cron_stats_name_created_at`); err != nil {
			return err
		}
	}

	return nil
}

func Down_20261115001714(tx *sql.Tx) error {
	return nil
}
