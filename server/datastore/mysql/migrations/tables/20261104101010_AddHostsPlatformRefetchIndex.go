package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20261104101010, Down_20261104101010)
}

func Up_20261104101010(tx *sql.Tx) error {
	// Lets the iOS/iPadOS refetch cron read the longest-waiting hosts in order and
	// stop at its batch limit instead of scanning every MDM enrollment each tick.
	return addIndexesTx(tx, "hosts",
		indexDef{"idx_hosts_platform_refetch_detail", "platform, refetch_requested, detail_updated_at"})
}

func Down_20261104101010(tx *sql.Tx) error {
	return nil
}
