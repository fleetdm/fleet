package tables

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
)

func init() {
	MigrationClient.AddMigration(Up_20260723181413, Down_20260723181413)
}

// Up_20260723181413 removes redundant App Store app installs that Fleet queued for itself, keeping
// the most recently queued one per host and app, or none where an install of that app has already
// been sent to the device. A host whose queue stops draining collects one copy per cycle from
// automatic updates and from policy automations alike, and installs the app once per copy on drain.
//
// An install a person asked for is never removed, so a host holding one of those and an automated
// duplicate keeps both.
func Up_20260723181413(tx *sql.Tx) error {
	step := incrementalMigrationStep(countRedundantQueuedVPPAppInstalls, deleteRedundantQueuedVPPAppInstalls)
	if err := step(tx); err != nil {
		return fmt.Errorf("deleting redundant queued VPP app installs: %w", err)
	}
	return nil
}

// Fleet queued it itself, rather than a person asking for it. A deleted policy clears policy_id,
// which fails safe by making the row look person-requested.
//
// `= 1` and not `IS TRUE`, which on a JSON value means only "not JSON 0 and not SQL NULL" and so also
// matches JSON false and null. `= 1` matches the JSON number a Go bool is stored as, failing in the
// safe direction for an irreversible delete. Same test as IsAutoUpdateVPPInstall.
const automatedVPPInstallExpr = `(
			JSON_EXTRACT(ua.payload, '$.from_auto_update') = 1
			OR vaua.policy_id IS NOT NULL
		)`

// The grouping query has to see two populations, the automated installs it may delete and any
// install of the same app already sent to the device, whoever asked for it. An install a person asked
// for occupies the slot just as an automated one does, so scoping this to automated installs would
// leave a duplicate queued behind a stuck manual install.
const vppInstallGroupScopeWhere = `
		ua.activity_type = 'vpp_app_install'
		AND (` + automatedVPPInstallExpr + ` OR ua.activated_at IS NOT NULL)`

// Deletable rows are automated installs not yet sent to a device and not holding a setup experience
// open, which a migration would strand in "running" because it bypasses the cancel path. No setup
// experience row can be automated today, so that clause is unreachable, and it stays so that widening
// the automated test cannot strand one.
//
// An expression rather than a WHERE fragment because the grouping query counts deletable rows while
// still seeing the rows it must not delete.
const deletableQueuedVPPInstallExpr = `(
			ua.activated_at IS NULL
			AND ` + automatedVPPInstallExpr + `
			AND NOT EXISTS (
				SELECT 1 FROM setup_experience_status_results sesr
				WHERE sesr.nano_command_uuid = ua.execution_id
			)
		)`

const deletableQueuedVPPInstallWhere = `
		ua.activity_type = 'vpp_app_install'
		AND ` + deletableQueuedVPPInstallExpr

// redundantQueuedVPPInstallGroupsStmt returns each host and app holding a redundant queued install,
// with the id to keep, or 0 when every queued row for that pair should go.
//
// An install already sent to the device occupies the slot, because anything queued behind it installs
// the same app a second time, which is the state the runtime filter now prevents, so the migration
// must not leave one behind either.
//
// STRAIGHT_JOIN pins upcoming_activities as the driving table so the (activity_type, host_id) index
// selects the candidates; the child table has no filter of its own and is a legal driving table.
const redundantQueuedVPPInstallGroupsStmt = `
	SELECT
		ua.host_id,
		vaua.adam_id,
		SUM(` + deletableQueuedVPPInstallExpr + `) AS queued_count,
		IF(
			SUM(ua.activated_at IS NOT NULL) > 0,
			0,
			MAX(IF(` + deletableQueuedVPPInstallExpr + `, ua.id, NULL))
		) AS keep_id
	FROM upcoming_activities ua
	STRAIGHT_JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
	WHERE ` + vppInstallGroupScopeWhere + `
	GROUP BY ua.host_id, vaua.adam_id
	HAVING queued_count > 0 AND (keep_id = 0 OR queued_count > 1)`

// countRedundantQueuedVPPAppInstalls counts the rows the pass below will delete, which is what stops
// an unaffected deployment doing any work, since incrementalMigrationStep skips a zero count. It is
// an upper bound, because the delete re-applies its predicate and so skips a row that another
// instance activated in the meantime, while progress still counts it.
func countRedundantQueuedVPPAppInstalls(tx *sql.Tx) (uint64, error) {
	var total uint64
	err := tx.QueryRow(`
		SELECT COALESCE(SUM(IF(keep_id = 0, queued_count, queued_count - 1)), 0)
		FROM (` + redundantQueuedVPPInstallGroupsStmt + `) redundant_groups`).Scan(&total)
	return total, err
}

// deleteRedundantQueuedVPPAppInstalls deletes every queued automated install except the newest one
// per host and app, or all of them where an install of that app has already been sent.
//
// The newest survives rather than the earliest, because nanoEnqueueVPPInstall copies
// upcoming_activities.created_at into the nano queue and the APNs retry cron ignores commands older
// than seven days, so keeping a months-old row would leave one that can never be re-pushed.
//
// The ids are gathered in one pass rather than per host and app. Every row deleted holds an X-lock
// until goose commits the whole migration, so the shortest pass is also the shortest window for pods
// still serving check-ins. The deletes stay batched, which is why this is not one statement.
func deleteRedundantQueuedVPPAppInstalls(tx *sql.Tx, increment incrementCountFn) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	redundantIDsStmt := `
		SELECT ua.id
		FROM upcoming_activities ua
		STRAIGHT_JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
		JOIN (` + redundantQueuedVPPInstallGroupsStmt + `) redundant_groups
			ON redundant_groups.host_id = ua.host_id AND redundant_groups.adam_id = vaua.adam_id
		WHERE ua.id <> redundant_groups.keep_id
			AND ` + deletableQueuedVPPInstallWhere + `
		ORDER BY ua.id`

	var ids []uint64
	if err := txx.Select(&ids, redundantIDsStmt); err != nil {
		return fmt.Errorf("selecting redundant queued VPP app installs: %w", err)
	}

	// The ids come from a snapshot read, while the delete is applied to the latest committed rows, so
	// it repeats the predicate. Without it a row that another instance activated in between, which a
	// rolling deploy makes possible, would be deleted along with its command already in flight.
	const deleteStmt = `
		DELETE ua FROM upcoming_activities ua
		STRAIGHT_JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
		WHERE ua.id IN (?)
			AND ` + deletableQueuedVPPInstallWhere

	const batchSize = 1000
	for len(ids) > 0 {
		batch := ids
		if len(batch) > batchSize {
			batch = batch[:batchSize]
		}
		ids = ids[len(batch):]

		// vpp_app_upcoming_activities rows go with their parent through the FK's ON DELETE CASCADE.
		query, args, err := sqlx.In(deleteStmt, batch)
		if err != nil {
			return fmt.Errorf("building delete query: %w", err)
		}
		if _, err := txx.Exec(query, args...); err != nil {
			return fmt.Errorf("deleting redundant queued VPP app installs: %w", err)
		}

		for range batch {
			increment()
		}
	}
	return nil
}

func Down_20260723181413(tx *sql.Tx) error {
	return nil
}
