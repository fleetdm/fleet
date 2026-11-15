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
	// idx_cron_stats_name_created_at is kept on purpose: the new index
	// cannot serve a name-only lookup ordered by created_at because
	// stats_type sits between the two columns.
	return addIndexesTx(tx, "cron_stats",
		indexDef{"idx_cron_stats_name_stats_type_created_at", "name, stats_type, created_at"})
}

func Down_20261115001714(tx *sql.Tx) error {
	return nil
}
