package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

var automationActivityAuthor = "Fleet"

// NewActivity stores an activity item that the user performed
func (ds *Datastore) NewActivity(
	ctx context.Context, user *fleet.User, activity fleet.ActivityDetails, details []byte, createdAt time.Time,
) error {
	// Sanity check to ensure we processed activity webhook before storing the activity
	processed, _ := ctx.Value(fleet.ActivityWebhookContextKey).(bool)
	if !processed {
		return ctxerr.New(
			ctx, "activity webhook not processed. Please use svc.NewActivity instead of ds.NewActivity. This is a Fleet server bug.",
		)
	}

	var userID *uint
	var userName *string
	var userEmail *string
	if user != nil {
		// To support creating activities with users that were deleted. This can happen
		// for automatically installed software which uses the author of the upload as the author of
		// the installation.
		if user.ID != 0 {
			userID = &user.ID
		}
		userName = &user.Name
		userEmail = &user.Email
	} else if automatableActivity, ok := activity.(fleet.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
		userName = &automationActivityAuthor
	}

	cols := []string{"user_id", "user_name", "activity_type", "details", "created_at"}
	args := []any{
		userID,
		userName,
		activity.ActivityName(),
		details,
		createdAt,
	}
	if userEmail != nil {
		args = append(args, userEmail)
		cols = append(cols, "user_email")
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		const insertActStmt = `INSERT INTO activities (%s) VALUES (%s)`
		sql := fmt.Sprintf(insertActStmt, strings.Join(cols, ","), strings.Repeat("?,", len(cols)-1)+"?")
		res, err := tx.ExecContext(ctx, sql, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "new activity")
		}

		// this supposes a reasonable amount of hosts per activity, to revisit if we
		// get in the 10K+.
		if ah, ok := activity.(fleet.ActivityHosts); ok {
			const insertActHostStmt = `INSERT INTO host_activities (host_id, activity_id) VALUES `

			var sb strings.Builder
			if hostIDs := ah.HostIDs(); len(hostIDs) > 0 {
				sb.WriteString(insertActHostStmt)
				actID, _ := res.LastInsertId()
				for _, hid := range hostIDs {
					sb.WriteString(fmt.Sprintf("(%d, %d),", hid, actID))
				}

				stmt := strings.TrimSuffix(sb.String(), ",")
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return ctxerr.Wrap(ctx, err, "insert host activity")
				}
			}
		}
		return nil
	})
}

// ListActivities returns a slice of activities performed across the organization
func (ds *Datastore) ListActivities(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	// Fetch activities

	activities := []*fleet.Activity{}
	activitiesQ := `
		SELECT
			a.id,
			a.user_id,
			a.created_at,
			a.activity_type,
			a.user_name as name,
			a.streamed,
			a.user_email
		FROM activities a
		WHERE true`

	var args []interface{}
	if opt.Streamed != nil {
		activitiesQ += " AND a.streamed = ?"
		args = append(args, *opt.Streamed)
	}
	opt.ListOptions.IncludeMetadata = !(opt.ListOptions.UsesCursorPagination())

	activitiesQ, args = appendListOptionsWithCursorToSQL(activitiesQ, args, &opt.ListOptions)

	err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, activitiesQ, args...)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	if len(activities) > 0 {
		// Fetch details as a separate query due to sort buffer issue triggered by large JSON details entries. Issue last reproduced on MySQL 8.0.36
		// https://stackoverflow.com/questions/29575835/error-1038-out-of-sort-memory-consider-increasing-sort-buffer-size/67266529
		IDs := make([]uint, 0, len(activities))
		for _, a := range activities {
			IDs = append(IDs, a.ID)
		}
		detailsStmt, detailsArgs, err := sqlx.In("SELECT id, details FROM activities WHERE id IN (?)", IDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "Error binding activity IDs")
		}
		type activityDetails struct {
			ID      uint             `db:"id"`
			Details *json.RawMessage `db:"details"`
		}
		var details []activityDetails
		err = sqlx.SelectContext(ctx, ds.reader(ctx), &details, detailsStmt, detailsArgs...)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "select activities details")
		}
		detailsLookup := make(map[uint]*json.RawMessage, len(details))
		for _, d := range details {
			detailsLookup[d.ID] = d.Details
		}
		for _, a := range activities {
			det, ok := detailsLookup[a.ID]
			if !ok {
				level.Warn(ds.logger).Log("msg", "Activity details not found", "activity_id", a.ID)
				continue
			}
			a.Details = det
		}
	}

	// Fetch users as a stand-alone query (because of performance reasons)

	lookup := make(map[uint][]int)
	for idx, a := range activities {
		if a.ActorID != nil {
			lookup[*a.ActorID] = append(lookup[*a.ActorID], idx)
		}
	}

	if len(lookup) != 0 {
		usersQ := `
			SELECT u.id, u.name, u.gravatar_url, u.email
			FROM users u
			WHERE id IN (?)
		`
		userIDs := make([]uint, 0, len(lookup))
		for k := range lookup {
			userIDs = append(userIDs, k)
		}

		usersQ, usersArgs, err := sqlx.In(usersQ, userIDs)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "Error binding usersIDs")
		}

		var usersR []struct {
			ID          uint   `db:"id"`
			Name        string `db:"name"`
			GravatarUrl string `db:"gravatar_url"`
			Email       string `db:"email"`
		}

		err = sqlx.SelectContext(ctx, ds.reader(ctx), &usersR, usersQ, usersArgs...)
		if err != nil && err != sql.ErrNoRows {
			return nil, nil, ctxerr.Wrap(ctx, err, "selecting users")
		}

		for _, r := range usersR {
			entries, ok := lookup[r.ID]
			if !ok {
				continue
			}

			email := r.Email
			gravatar := r.GravatarUrl
			name := r.Name

			for _, idx := range entries {
				activities[idx].ActorEmail = &email
				activities[idx].ActorGravatar = &gravatar
				activities[idx].ActorFullName = &name
			}
		}
	}

	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(activities) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, metaData, nil
}

func (ds *Datastore) MarkActivitiesAsStreamed(ctx context.Context, activityIDs []uint) error {
	stmt := `UPDATE activities SET streamed = true WHERE id IN (?);`
	query, args, err := sqlx.In(stmt, activityIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "sqlx.In mark activities as streamed")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec mark activities as streamed")
	}
	return nil
}

// ListHostUpcomingActivities returns the list of activities pending execution
// or processing for the specific host. It is the "unified queue" of work to be
// done on the host. That queue is "virtual" in the sense that it pulls from a
// number of distinct tables that are task-specific (such as scripts to run,
// software to install, etc.) and provides a unified view of those upcoming
// tasks.
func (ds *Datastore) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.UpcomingActivity, *fleet.PaginationMetadata, error) {
	// NOTE: Be sure to update both the count (here) and list statements (below)
	// if the query condition is modified.

	const countStmt = `SELECT
	COUNT(*) c
	FROM upcoming_activities
	WHERE host_id = ?`

	var count uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, countStmt, hostID); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "count upcoming activities")
	}
	if count == 0 {
		return []*fleet.UpcomingActivity{}, &fleet.PaginationMetadata{}, nil
	}

	// NOTE: Be sure to update both the count (above) and list statements (below)
	// if the query condition is modified.

	listStmts := []string{
		// list pending scripts
		`SELECT
			ua.execution_id as uuid,
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, JSON_EXTRACT(ua.payload, '$.user.name'))) as name,
			u.id as user_id,
			COALESCE(u.gravatar_url, JSON_EXTRACT(ua.payload, '$.user.gravatar_url')) as gravatar_url,
			COALESCE(u.email, JSON_EXTRACT(ua.payload, '$.user.email')) as user_email,
			:ran_script_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'script_name', COALESCE(ses.name, scr.name, ''),
				'script_execution_id', ua.execution_id,
				'async', NOT JSON_EXTRACT(ua.payload, '$.sync_request'),
				'policy_id', sua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated,
			IF(ua.activated_at IS NULL, 1, 0) as cancellable
		FROM
			upcoming_activities ua
		INNER JOIN
			script_upcoming_activities sua ON sua.upcoming_activity_id = ua.id
		LEFT OUTER JOIN
			users u ON u.id = ua.user_id
		LEFT OUTER JOIN
			policies p ON p.id = sua.policy_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = ua.host_id
		LEFT OUTER JOIN
			scripts scr ON scr.id = sua.script_id
		LEFT OUTER JOIN
			setup_experience_scripts ses ON ses.id = sua.setup_experience_script_id
		WHERE
			ua.host_id = :host_id AND
			ua.activity_type = 'script'
`,
		// list pending software installs
		`SELECT
			ua.execution_id as uuid,
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, JSON_EXTRACT(ua.payload, '$.user.name'))) AS name,
			ua.user_id as user_id,
			COALESCE(u.gravatar_url, JSON_EXTRACT(ua.payload, '$.user.gravatar_url')) as gravatar_url,
			COALESCE(u.email, JSON_EXTRACT(ua.payload, '$.user.email')) as user_email,
			:installed_software_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, JSON_EXTRACT(ua.payload, '$.software_title_name'), ''),
				'software_package', COALESCE(si.filename, JSON_EXTRACT(ua.payload, '$.installer_filename'), ''),
				'install_uuid', ua.execution_id,
				'status', 'pending_install',
				'self_service', JSON_EXTRACT(ua.payload, '$.self_service') IS TRUE,
				'policy_id', siua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated,
			IF(ua.activated_at IS NULL, 1, 0) as cancellable
		FROM
			upcoming_activities ua
		INNER JOIN
			software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
		LEFT OUTER JOIN
			software_installers si ON si.id = siua.software_installer_id
		LEFT OUTER JOIN
			software_titles st ON st.id = si.title_id
		LEFT OUTER JOIN
			users u ON u.id = ua.user_id
		LEFT OUTER JOIN
			policies p ON p.id = siua.policy_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = ua.host_id
		WHERE
			ua.host_id = :host_id AND
			ua.activity_type = 'software_install'
		`,
		// list pending software uninstalls
		`SELECT
			ua.execution_id as uuid,
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, JSON_EXTRACT(ua.payload, '$.user.name'))) AS name,
			ua.user_id as user_id,
			COALESCE(u.gravatar_url, JSON_EXTRACT(ua.payload, '$.user.gravatar_url')) as gravatar_url,
			COALESCE(u.email, JSON_EXTRACT(ua.payload, '$.user.email')) as user_email,
			:uninstalled_software_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, JSON_EXTRACT(ua.payload, '$.software_title_name'), ''),
				'script_execution_id', ua.execution_id,
				'status', 'pending_uninstall',
				'policy_id', siua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated,
			IF(ua.activated_at IS NULL, 1, 0) as cancellable
		FROM
			upcoming_activities ua
		INNER JOIN
			software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
		LEFT OUTER JOIN
			software_installers si ON si.id = siua.software_installer_id
		LEFT OUTER JOIN
			software_titles st ON st.id = si.title_id
		LEFT OUTER JOIN
			users u ON u.id = ua.user_id
		LEFT OUTER JOIN
			policies p ON p.id = siua.policy_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = ua.host_id
		WHERE
			ua.host_id = :host_id AND
			activity_type = 'software_uninstall'
		`,
		`SELECT
			ua.execution_id AS uuid,
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, JSON_EXTRACT(ua.payload, '$.user.name'))) AS name,
			u.id AS user_id,
			COALESCE(u.gravatar_url, JSON_EXTRACT(ua.payload, '$.user.gravatar_url')) as gravatar_url,
			COALESCE(u.email, JSON_EXTRACT(ua.payload, '$.user.email')) as user_email,
			:installed_app_store_app_type AS activity_type,
			ua.created_at AS created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', hdn.display_name,
				'software_title', st.name,
				'app_store_id', vaua.adam_id,
				'command_uuid', ua.execution_id,
				'self_service', JSON_EXTRACT(ua.payload, '$.self_service') IS TRUE,
				'status', 'pending_install'
			) AS details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated,
			IF(ua.activated_at IS NULL, 1, 0) as cancellable
		FROM
			upcoming_activities ua
		INNER JOIN
			vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
		LEFT OUTER JOIN
			users u ON ua.user_id = u.id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = ua.host_id
		LEFT OUTER JOIN
			vpp_apps vpa ON vaua.adam_id = vpa.adam_id AND vaua.platform = vpa.platform
		LEFT OUTER JOIN
			software_titles st ON st.id = vpa.title_id
		WHERE
			ua.host_id = :host_id AND
			ua.activity_type = 'vpp_app_install'
		`,
	}

	listStmt := `
		SELECT
			uuid,
			name,
			user_id,
			gravatar_url,
			user_email,
			activity_type,
			created_at,
			details,
			fleet_initiated,
			cancellable
		FROM ( ` + strings.Join(listStmts, " UNION ALL ") + ` ) AS upcoming
		ORDER BY topmost DESC, priority DESC, created_at ASC`

	listStmt, args, err := sqlx.Named(listStmt, map[string]any{
		"host_id":                      hostID,
		"ran_script_type":              fleet.ActivityTypeRanScript{}.ActivityName(),
		"installed_software_type":      fleet.ActivityTypeInstalledSoftware{}.ActivityName(),
		"uninstalled_software_type":    fleet.ActivityTypeUninstalledSoftware{}.ActivityName(),
		"installed_app_store_app_type": fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build list query from named args")
	}

	// the ListOptions supported for this query are limited, only the pagination
	// OFFSET and LIMIT can be added, so it's fine to have the ORDER BY already
	// in the query before calling this (enforced at the server layer).
	stmt, args := appendListOptionsWithCursorToSQL(listStmt, args, &opt)

	var activities []*fleet.UpcomingActivity
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select upcoming activities")
	}

	// first activity (next one to execute) is always non-cancellable, per spec
	if len(activities) > 0 && opt.Page == 0 {
		activities[0].Cancellable = false
	}

	metaData := &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0, TotalResults: count}
	if len(activities) > int(opt.PerPage) { //nolint:gosec // dismiss G115
		metaData.HasNextResults = true
		activities = activities[:len(activities)-1]
	}

	return activities, metaData, nil
}

func (ds *Datastore) ListHostPastActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	const listStmt = `
	SELECT
		ha.activity_id as id,
		a.user_email as user_email,
		a.user_name as name,
		a.activity_type as activity_type,
		a.details as details,
		u.gravatar_url as gravatar_url,
		a.created_at as created_at,
		u.id as user_id
	FROM
		host_activities ha
		JOIN activities a
			ON ha.activity_id = a.id
		LEFT OUTER JOIN
			users u ON u.id = a.user_id
	WHERE
		ha.host_id = ?
	`

	args := []any{hostID}
	stmt, args := appendListOptionsWithCursorToSQL(listStmt, args, &opt)

	var activities []*fleet.Activity
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select upcoming activities")
	}

	var metaData *fleet.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(activities) > int(opt.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			activities = activities[:len(activities)-1]
		}
	}

	return activities, metaData, nil
}

func (ds *Datastore) CleanupActivitiesAndAssociatedData(ctx context.Context, maxCount int, expiredWindowDays int) error {
	const selectActivitiesQuery = `
		SELECT a.id FROM activities a
		LEFT JOIN host_activities ha ON (a.id=ha.activity_id)
		WHERE ha.activity_id IS NULL AND a.created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY a.id ASC
		LIMIT ?;`
	var activityIDs []uint
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &activityIDs, selectActivitiesQuery, expiredWindowDays, maxCount); err != nil {
		return ctxerr.Wrap(ctx, err, "select activities for deletion")
	}
	if len(activityIDs) > 0 {
		deleteActivitiesQuery, args, err := sqlx.In(`DELETE FROM activities WHERE id IN (?);`, activityIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "build activities IN query")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteActivitiesQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired activities")
		}
	}

	//
	// `activities` and `queries` are not tied because the activity itself holds
	// the query SQL so they don't need to be executed on the same transaction.
	//
	// All expired live queries are deleted in batch sizes of `maxCount` to ensure
	// the table size is kept in check with high volumes of live queries (zero-trust workflows).
	// This differs from the `activities` cleanup which uses maxCount as a limit to
	// the number of activities to delete.
	//
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var rowsAffected int64

		// Start a new transaction for each batch of deletions.
		err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
			// Delete expired live queries (aka "not saved")
			result, err := tx.ExecContext(ctx,
				`DELETE FROM queries
			WHERE NOT saved AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
			LIMIT ?`,
				expiredWindowDays, maxCount,
			)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "delete expired non-saved queries")
			}

			rowsAffected, err = result.RowsAffected()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "retrieving rows affected from delete query")
			}

			// Cleanup orphaned distributed campaigns that reference non-existing queries.
			if _, err := tx.ExecContext(ctx,
				`DELETE distributed_query_campaigns FROM distributed_query_campaigns
			LEFT JOIN queries ON (distributed_query_campaigns.query_id=queries.id)
			WHERE queries.id IS NULL`,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "delete expired orphaned distributed_query_campaigns")
			}

			// Cleanup orphaned distributed campaign targets that reference non-existing distributed campaigns.
			if _, err := tx.ExecContext(ctx,
				`DELETE distributed_query_campaign_targets FROM distributed_query_campaign_targets
			LEFT JOIN distributed_query_campaigns ON (distributed_query_campaign_targets.distributed_query_campaign_id=distributed_query_campaigns.id)
			WHERE distributed_query_campaigns.id IS NULL`,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "delete expired orphaned distributed_query_campaign_targets")
			}

			return nil
		})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired queries in batch")
		}

		// Break the loop if no rows were deleted in the current batch.
		if rowsAffected == 0 {
			break
		}
	}

	return nil
}
