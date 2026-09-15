package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

var backfillPatchWhenClosedBatchSize = 1000

func init() {
	MigrationClient.AddMigration(Up_20260914161913, Down_20260914161913)
}

// patch_when_closed snapshots the triggering policy's patch_when_closed value at
// install-activation time. Read paths that need to tell "Patch skipped" apart
// from an ordinary pre-install-query failure must use this column rather than
// joining policies, since policies.patch_when_closed can be toggled and the
// policy row itself is subject to ON DELETE SET NULL (policy_id → NULL), both
// of which would silently reclassify a historical row.
//
// Upgrade limitation: the backfill reads the CURRENT policies.patch_when_closed
// value, so historical skips whose policy was toggled off (or the policy was
// deleted) BEFORE this migration runs will keep the column at 0 and render as
// ordinary "Failed" going forward. Rebuilding those from another durable signal
// (e.g. activities.details JSON, which carries skipped_install) would require a
// scan of the activities table and is out of scope for this bug fix. The window
// is narrow — it takes toggling patch_when_closed off on a policy that already
// caused skips before the customer upgrades to this Fleet version.
func Up_20260914161913(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "patch_when_closed") {
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN patch_when_closed TINYINT UNSIGNED NOT NULL DEFAULT 0
		`); err != nil {
			return fmt.Errorf("adding patch_when_closed to host_software_installs: %w", err)
		}
	}

	step := incrementalMigrationStep(
		countHostSoftwareInstallsNeedingPatchBackfill,
		backfillPatchWhenClosedOnHostSoftwareInstalls,
	)
	if err := step(tx); err != nil {
		return fmt.Errorf("backfilling patch_when_closed on host_software_installs: %w", err)
	}
	return nil
}

func countHostSoftwareInstallsNeedingPatchBackfill(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM host_software_installs
		WHERE patch_when_closed = 0
			AND policy_id IN (SELECT id FROM policies WHERE patch_when_closed = 1)
	`).Scan(&total)
	return total, err
}

// backfillPatchWhenClosedOnHostSoftwareInstalls walks the primary key with a
// LIMIT batch size so each UPDATE only touches a bounded set of rows. Drives
// from the small side (policies with patch_when_closed = 1 is a tiny set —
// admins configure them deliberately) so the row lookup pins to
// fk_software_install_policy_id via IN() and the batch UPDATE hits rows by
// primary key. Chunking keeps per-statement work bounded on large
// host_software_installs tables and enables the incremental progress
// reporter for long-running upgrades.
func backfillPatchWhenClosedOnHostSoftwareInstalls(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	var lastID uint
	for {
		var ids []uint
		if err := txx.Select(&ids, `
			SELECT id FROM host_software_installs
			WHERE id > ?
				AND patch_when_closed = 0
				AND policy_id IN (SELECT id FROM policies WHERE patch_when_closed = 1)
			ORDER BY id
			LIMIT ?`, lastID, backfillPatchWhenClosedBatchSize); err != nil {
			return fmt.Errorf("selecting host_software_installs to backfill after id %d: %w", lastID, err)
		}
		if len(ids) == 0 {
			return nil
		}

		query, args, err := sqlx.In(
			`UPDATE host_software_installs SET patch_when_closed = 1 WHERE id IN (?)`,
			ids,
		)
		if err != nil {
			return fmt.Errorf("building patch_when_closed backfill after id %d: %w", lastID, err)
		}
		if _, err := txx.Exec(query, args...); err != nil {
			return fmt.Errorf("backfilling patch_when_closed after id %d: %w", lastID, err)
		}

		for range ids {
			increment()
		}
		lastID = ids[len(ids)-1]
	}
}

func Down_20260914161913(tx *sql.Tx) error {
	return nil
}
