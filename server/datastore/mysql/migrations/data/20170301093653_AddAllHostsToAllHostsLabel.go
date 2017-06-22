package data

import (
	"database/sql"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20170301093653, Down_20170301093653)
}

func Up_20170301093653(tx *sql.Tx) error {
	// Insert any host not currently in 'All Hosts' label into the label
	_, err := tx.Exec(`
		INSERT IGNORE INTO label_query_executions (
                        host_id,
                        label_id,
                        matches
                ) SELECT
                id as host_id,
                (SELECT id as label_id FROM labels WHERE name = 'All Hosts' AND label_type = ?),
                true as matches
                FROM hosts
`,
		kolide.LabelTypeBuiltIn)
	if err != nil {
		return errors.Wrap(err, "adding hosts to 'All Hosts'")
	}

	return nil
}

func Down_20170301093653(tx *sql.Tx) error {
	// This operation not reversible
	return nil
}
