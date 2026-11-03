package mysql

import (
	"context"
	"fmt"
	"slices"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// The single-host NewInternalHostScriptExecutionRequest is a better fit for
// anything user-facing.
func (ds *Datastore) BatchNewInternalHostScriptExecutionRequests(ctx context.Context, hostIDs []uint, contents string) (map[uint]string, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}

	var executionIDByHost map[uint]string
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		contentsRes, err := insertScriptContents(ctx, tx, contents)
		if err != nil {
			return err
		}
		scriptContentID, _ := contentsRes.LastInsertId()

		executionIDByHost, err = ds.insertHostScriptExecutionsForHosts(ctx, tx, hostIDs, nil, nil, uint(scriptContentID), true) //nolint:gosec // dismiss G115
		if err != nil {
			return err
		}

		// activating in the same transaction as the inserts makes the batch all or
		// nothing, so it can't leave hosts holding a queued script that never
		// activated
		err = ds.activateNextScriptActivitiesForHosts(ctx, tx, hostIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "activate batch of queued scripts")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return executionIDByHost, nil
}

func (ds *Datastore) insertHostScriptExecutionsForHosts(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint, userID *uint, scriptID *uint,
	scriptContentID uint, isInternal bool,
) (map[uint]string, error) {
	if len(hostIDs) == 0 {
		return nil, nil
	}

	slices.Sort(hostIDs)              // sorting can help avoid deadlocks
	hostIDs = slices.Compact(hostIDs) // dedupe IDs (must be sorted first)

	// the execution IDs are generated here rather than read back, so the caller
	// gets its host to execution mapping without another query
	executionIDByHost := make(map[uint]string, len(hostIDs))
	executionIDs := make([]string, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		executionID := uuid.New().String()
		executionIDByHost[hostID] = executionID
		executionIDs = append(executionIDs, executionID)
	}

	// Set fleet_initiated for internal scripts, there's no user to attribute them to and the
	// activity feed shows "Fleet" rather than a blank actor.
	const insertActivitiesStmt = `
INSERT INTO upcoming_activities
	(host_id, priority, user_id, fleet_initiated, activity_type, execution_id, payload)
VALUES
	(:host_id, :priority, :user_id, :is_internal, 'script', :execution_id,
		JSON_OBJECT(
			'sync_request', :sync_request,
			'is_internal', :is_internal,
			'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = :user_id)
		)
	)`

	// the child rows join back on execution_id rather than on the ids the insert
	// above generated, which MySQL does not guarantee to be contiguous
	const insertScriptActivitiesStmt = `
INSERT INTO script_upcoming_activities
	(upcoming_activity_id, script_id, script_content_id)
SELECT
	ua.id, ?, ?
FROM
	upcoming_activities ua
WHERE
	ua.execution_id IN (?)`

	priority := fleet.UserInitiatedActivityPriority
	if isInternal {
		priority = 0
	}

	activityArgs := make([]map[string]any, 0, len(hostIDs))
	for _, hostID := range hostIDs {
		activityArgs = append(activityArgs, map[string]any{
			"host_id":      hostID,
			"priority":     priority,
			"user_id":      userID,
			"execution_id": executionIDByHost[hostID],
			"sync_request": false,
			"is_internal":  isInternal,
		})
	}
	_, err := sqlx.NamedExecContext(ctx, tx, insertActivitiesStmt, activityArgs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "batch insert script upcoming activities")
	}

	stmt, args, err := sqlx.In(insertScriptActivitiesStmt, scriptID, scriptContentID, executionIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build batch script upcoming activities insert")
	}
	_, err = tx.ExecContext(ctx, stmt, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "batch insert script upcoming activity details")
	}

	return executionIDByHost, nil
}

// Not a general replacement for activateNextUpcomingActivityForBatchOfHosts:
// this only activates scripts, in a fixed number of statements for the whole
// set. A host whose turn belongs to another activity type, or that already has
// one activated, is left for that other path to pick up.
func (ds *Datastore) activateNextScriptActivitiesForHosts(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	// Ranks every activity type so a script can't activate ahead of an install
	// queued earlier; only rank 1 matters since scripts never batch.
	findNextScriptsStmt := `
SELECT
	execution_id
FROM (
	SELECT
		execution_id,
		activity_type,
		activated_at,
		ROW_NUMBER() OVER (
			PARTITION BY host_id
			ORDER BY IF(activated_at IS NULL, 0, 1) DESC, priority DESC, created_at ASC
		) AS rank_in_host
	FROM
		upcoming_activities
	WHERE
		host_id IN (?)
		%s
) candidates
WHERE
	rank_in_host = 1 AND
	activity_type = 'script' AND
	activated_at IS NULL
`

	// same columns as activateNextScriptActivity, for many hosts at once
	const insertScriptResultsStmt = `
INSERT INTO
	host_script_results
(host_id, execution_id, script_content_id, output, script_id, policy_id,
	user_id, sync_request, setup_experience_script_id, is_internal)
SELECT
	ua.host_id,
	ua.execution_id,
	sua.script_content_id,
	'',
	sua.script_id,
	sua.policy_id,
	ua.user_id,
	COALESCE(ua.payload->'$.sync_request', 0),
	sua.setup_experience_script_id,
	COALESCE(ua.payload->'$.is_internal', 0)
FROM
	upcoming_activities ua
	INNER JOIN script_upcoming_activities sua
		ON sua.upcoming_activity_id = ua.id
WHERE
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	const markActivatedStmt = `
UPDATE upcoming_activities
SET
	activated_at = NOW()
WHERE
	execution_id IN (?)
`

	var (
		stmt string
		args []any
		err  error
	)
	if len(ds.testActivateSpecificNextActivities) > 0 {
		stmt, args, err = sqlx.In(fmt.Sprintf(findNextScriptsStmt, ` AND execution_id IN (?) `),
			hostIDs, ds.testActivateSpecificNextActivities)
	} else {
		stmt, args, err = sqlx.In(fmt.Sprintf(findNextScriptsStmt, ""), hostIDs)
	}
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement to find next scripts to activate")
	}

	var execIDs []string
	if err := sqlx.SelectContext(ctx, tx, &execIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "find next scripts to activate")
	}
	if len(execIDs) == 0 {
		return nil
	}

	stmt, args, err = sqlx.In(insertScriptResultsStmt, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert to activate scripts")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert to activate scripts")
	}

	stmt, args, err = sqlx.In(markActivatedStmt, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement to mark upcoming activities as activated")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark upcoming activities as activated")
	}
	return nil
}
