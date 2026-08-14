package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// activateNextScriptActivitiesForHosts activates the next upcoming activity for
// each host, but only where that activity is a script, in a fixed number of
// statements for the whole set rather than per host. Lives here rather than with
// the rest of the activity queue since it's specific to callers that queue a
// script for many hosts at once, not a general replacement for
// activateNextUpcomingActivityForBatchOfHosts.
//
// A host whose turn belongs to another activity type, or that already has one
// activated, is left for that other path to pick up.
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
