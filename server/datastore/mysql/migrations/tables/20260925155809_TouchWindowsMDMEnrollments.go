package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260925155809, Down_20260925155809)
}

// Up_20260925155809 gives every Windows MDM enrollment a fresh retention
// window when the stale-enrollment cleanup first ships, so its first run
// after upgrade does not reap devices whose host was deleted recently.
func Up_20260925155809(tx *sql.Tx) error {
	touch := incrementalMigrationStep(
		func(tx *sql.Tx) (uint64, error) {
			var total uint64
			err := tx.QueryRow(`SELECT COUNT(*) FROM mdm_windows_enrollments`).Scan(&total)
			return total, err
		},
		touchWindowsMDMEnrollments,
	)
	if err := touch(tx); err != nil {
		return fmt.Errorf("touching windows mdm enrollments: %w", err)
	}
	return nil
}

// touchWindowsMDMEnrollmentsBatchSize is a var so tests can force several
// batches.
var touchWindowsMDMEnrollmentsBatchSize = 5000

// touchWindowsMDMEnrollments walks the table in id-keyed batches so each
// UPDATE is bounded, the way Fleet migrations on host-scaled tables do.
func touchWindowsMDMEnrollments(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	batchSize := touchWindowsMDMEnrollmentsBatchSize
	var lastID uint64
	for {
		var ids []uint64
		if err := txx.Select(&ids,
			`SELECT id FROM mdm_windows_enrollments WHERE id > ? ORDER BY id LIMIT ?`,
			lastID, batchSize); err != nil {
			return fmt.Errorf("selecting batch starting after id %d: %w", lastID, err)
		}
		if len(ids) == 0 {
			return nil
		}

		batchLast := ids[len(ids)-1]
		if _, err := txx.Exec(`
			UPDATE mdm_windows_enrollments SET updated_at = CURRENT_TIMESTAMP
			WHERE id > ? AND id <= ?`,
			lastID, batchLast); err != nil {
			return fmt.Errorf("touching batch after id %d: %w", lastID, err)
		}
		for range ids {
			increment()
		}
		lastID = batchLast
	}
}

func Down_20260925155809(tx *sql.Tx) error {
	return nil
}
