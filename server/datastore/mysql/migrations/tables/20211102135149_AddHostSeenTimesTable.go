package tables

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211102135149, Down_20211102135149)
}

func Up_20211102135149(tx *sql.Tx) error {
	// Detach seen_times from hosts to allow for bulk updates without locking the hosts table
	// See https://github.com/fleetdm/fleet/issues/2776
	hostSeenTimesTable := `
		CREATE TABLE IF NOT EXISTS host_seen_times (
			host_id int(10) UNSIGNED NOT NULL,
			seen_time timestamp NULL DEFAULT NULL,
			PRIMARY KEY (host_id),
			INDEX idx_host_seen_times_seen_time (seen_time)
		);
	`
	if _, err := tx.Exec(hostSeenTimesTable); err != nil {
		return errors.Wrap(err, "create host_seen_times table")
	}

	if _, err := tx.Exec(`INSERT IGNORE INTO host_seen_times (host_id, seen_time) SELECT DISTINCT id as host_id, seen_time FROM hosts`); err != nil {
		return errors.Wrap(err, "migrating host seen_times")
	}

	if _, err := tx.Exec(`ALTER TABLE hosts DROP COLUMN seen_time`); err != nil {
		return errors.Wrap(err, "dropping host seen_times")
	}

	referencedTables := map[string]struct{}{"hosts": {}, "software": {}}
	table := "host_software"

	constraints, err := constraintsForTable(tx, table, referencedTables)
	if err != nil {
		return err
	}

	for _, constraint := range constraints {
		_, err = tx.Exec(fmt.Sprintf(`ALTER TABLE host_software DROP FOREIGN KEY %s;`, constraint))
		if err != nil {
			if !strings.Contains(err.Error(), "check that column/key exists") {
				return errors.Wrapf(err, "dropping fk %s", constraint)
			}
		}
	}

	return nil
}

func Down_20211102135149(tx *sql.Tx) error {
	return nil
}
