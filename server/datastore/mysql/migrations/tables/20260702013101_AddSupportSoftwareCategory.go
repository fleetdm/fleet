package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260702013101, Down_20260702013101)
}

func Up_20260702013101(tx *sql.Tx) error {
	// Add the new "🛟 Support" default self-service software category so it
	// matches fleet.DefaultSelfServiceCategoryNames, which is used when seeding
	// categories for newly created fleets.

	// Global (team_id=0) default row. Pin both timestamps to the same constant
	// the previous category migration used so the generated schema stays
	// deterministic across repeated runs of make dump-test-schema.
	if _, err := tx.Exec(`
INSERT IGNORE INTO software_categories (name, team_id, created_at, updated_at)
VALUES ('🛟 Support', 0, '2026-05-29 00:00:00', '2026-05-29 00:00:00')
`); err != nil {
		return errors.Wrap(err, "inserting default Support software category")
	}

	// Give every existing fleet its own copy of the new default, matching the
	// per-fleet backfill done when the categories were first scoped by team.
	if _, err := tx.Exec(`
INSERT IGNORE INTO software_categories (name, team_id)
SELECT '🛟 Support', t.id FROM teams t
`); err != nil {
		return errors.Wrap(err, "backfilling Support category per fleet")
	}

	return nil
}

func Down_20260702013101(tx *sql.Tx) error {
	return nil
}
