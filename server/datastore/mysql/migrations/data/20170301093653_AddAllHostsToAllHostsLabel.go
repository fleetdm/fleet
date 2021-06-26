package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20170301093653, Down_20170301093653)
}

func Up_20170301093653(tx *sql.Tx) error {
	// Insert any host not currently in 'All Hosts' label into the label
	_, err := tx.Exec(`
		INSERT IGNORE INTO label_membership (
                        host_id,
                        label_id
                ) SELECT
                id as host_id,
                (SELECT id as label_id FROM labels WHERE name = 'All Hosts' AND label_type = ?)
                FROM hosts
`,
		fleet.LabelTypeBuiltIn)
	if err != nil {
		return errors.Wrap(err, "adding hosts to 'All Hosts'")
	}

	return nil
}

func Down_20170301093653(tx *sql.Tx) error {
	// This operation not reversible
	return nil
}
