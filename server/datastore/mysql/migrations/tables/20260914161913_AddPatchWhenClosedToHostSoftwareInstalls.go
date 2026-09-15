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
// Upgrade limitation: the backfill uses the CURRENT policies.patch_when_closed
// value as a proxy for its value at each historical install-activation time,
// which is inherently ambiguous in both directions. We accept both downsides
// rather than adding an activities.details JSON scan (heavy, and activities
// can be trimmed):
//
//   - Disable-then-upgrade: a real skip whose policy was toggled OFF (or the
//     policy row was deleted) before this migration runs keeps the column at 0
//     and renders as "Failed" going forward — a false negative.
//   - Enable-then-upgrade: an ordinary pre-install-query failure recorded while
//     patch_when_closed = 0 is flagged as a skip if an admin toggles the
//     policy ON before upgrading — a false positive.
//
// To minimize the second downside, the backfill only touches rows that could
// plausibly BE historical skips (status = 'failed_install' AND empty
// pre_install_query_output). Successful/pending rows and non-empty-output
// failures are never stamped, so their patch_when_closed stays at DEFAULT 0
// and can never enter the CTE's skipped_install classification. Any row
// created after the upgrade snapshots the correct value at activation time
// and is unaffected.
func Up_20260914161913(tx *sql.Tx) error {
	if !columnExists(tx, "host_software_installs", "patch_when_closed") {
		// ALGORITHM=INSTANT keeps the ALTER O(1) on MySQL 8.0+ (metadata-only
		// change). If the server doesn't support instant add-column, the DDL
		// fails fast instead of silently rebuilding the table under a lock on
		// a customer's large host_software_installs.
		if _, err := tx.Exec(`
			ALTER TABLE host_software_installs
			ADD COLUMN patch_when_closed TINYINT UNSIGNED NOT NULL DEFAULT 0,
			ALGORITHM=INSTANT
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

// The backfill filter — kept as a single string constant so the count and
// batch queries can't drift. Only rows that could plausibly be historical
// skips are candidates: status = 'failed_install' AND empty
// pre_install_query_output on a policy that currently has patch_when_closed = 1.
const patchWhenClosedBackfillCandidateFilter = `
	patch_when_closed = 0
	AND status = 'failed_install'
	AND pre_install_query_output = ''
	AND policy_id IN (SELECT id FROM policies WHERE patch_when_closed = 1)
`

func countHostSoftwareInstallsNeedingPatchBackfill(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM host_software_installs
		WHERE ` + patchWhenClosedBackfillCandidateFilter).Scan(&total)
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
				AND `+patchWhenClosedBackfillCandidateFilter+`
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
