package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
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
	} else if ranScriptActivity, ok := activity.(fleet.ActivityTypeRanScript); ok {
		if ranScriptActivity.PolicyID != nil {
			userName = &automationActivityAuthor
		}
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
		if len(activities) > int(opt.ListOptions.PerPage) {
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
func (ds *Datastore) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	// NOTE: Be sure to update both the count (here) and list statements (below)
	// if the query condition is modified.
	countStmts := []string{
		`SELECT
			COUNT(*) c
			FROM host_script_results hsr
			LEFT OUTER JOIN
		    	host_software_installs hsi ON hsi.execution_id = hsr.execution_id
			WHERE hsr.host_id = :host_id AND
						exit_code IS NULL AND
						hsi.execution_id IS NULL AND
						(sync_request = 0 OR hsr.created_at >= DATE_SUB(NOW(), INTERVAL :max_wait_time SECOND))`,
		`SELECT
			COUNT(*) c
			FROM host_software_installs hsi
			WHERE hsi.host_id = :host_id AND
				hsi.status = :software_status_install_pending`,
		`SELECT
			COUNT(*) c
			FROM host_software_installs hsi
			WHERE hsi.host_id = :host_id AND
				hsi.status = :software_status_uninstall_pending`,
		`
		SELECT
			COUNT(*) c
			FROM nano_view_queue nvq
			JOIN host_vpp_software_installs hvsi ON nvq.command_uuid = hvsi.command_uuid
			WHERE hvsi.host_id = :host_id AND nvq.status IS NULL
		`,
	}

	var count uint
	countStmt := `SELECT SUM(c) FROM ( ` + strings.Join(countStmts, " UNION ALL ") + ` ) AS counts`

	seconds := int(scripts.MaxServerWaitTime.Seconds())
	countStmt, args, err := sqlx.Named(countStmt, map[string]any{
		"host_id":                           hostID,
		"max_wait_time":                     seconds,
		"software_status_install_pending":   fleet.SoftwareInstallPending,
		"software_status_uninstall_pending": fleet.SoftwareUninstallPending,
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build count query from named args")
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, countStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "count upcoming activities")
	}
	if count == 0 {
		return []*fleet.Activity{}, &fleet.PaginationMetadata{}, nil
	}

	// NOTE: Be sure to update both the count (above) and list statements (below)
	// if the query condition is modified.
	listStmts := []string{
		// list pending scripts
		`SELECT
			hsr.execution_id as uuid,
			IF(hsr.policy_id IS NOT NULL, 'Fleet', u.name) as name,
			u.id as user_id,
			u.gravatar_url as gravatar_url,
			u.email as user_email,
			:ran_script_type as activity_type,
			hsr.created_at as created_at,
			JSON_OBJECT(
				'host_id', hsr.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'script_name', COALESCE(scr.name, ''),
				'script_execution_id', hsr.execution_id,
				'async', NOT hsr.sync_request,
			    'policy_id', hsr.policy_id,
			    'policy_name', p.name
			) as details
		FROM
			host_script_results hsr
		LEFT OUTER JOIN
			users u ON u.id = hsr.user_id
		LEFT OUTER JOIN
			policies p ON p.id = hsr.policy_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = hsr.host_id
		LEFT OUTER JOIN
			scripts scr ON scr.id = hsr.script_id
		LEFT OUTER JOIN
		    host_software_installs hsi ON hsi.execution_id = hsr.execution_id
		WHERE
			hsr.host_id = :host_id AND
			hsr.exit_code IS NULL AND
			(
				hsr.sync_request = 0 OR
				hsr.created_at >= DATE_SUB(NOW(), INTERVAL :max_wait_time SECOND)
			) AND
			hsi.execution_id IS NULL
`,
		// list pending software installs
		fmt.Sprintf(`SELECT
			hsi.execution_id as uuid,
			-- policies with automatic installers generate a host_software_installs with (user_id=NULL,self_service=0),
			-- thus the user_id for the upcoming activity needs to be the user that uploaded the software installer.
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.name, u.name) AS name,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.id, u.id) as user_id,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.gravatar_url, u.gravatar_url) as gravatar_url,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.email, u.email) AS user_email,
			:installed_software_type as activity_type,
			hsi.created_at as created_at,
			JSON_OBJECT(
				'host_id', hsi.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, ''),
				'software_package', si.filename,
				'install_uuid', hsi.execution_id,
				'status', CAST(hsi.status AS CHAR),
				'self_service', hsi.self_service IS TRUE
			) as details
		FROM
			host_software_installs hsi
		INNER JOIN
			software_installers si ON si.id = hsi.software_installer_id
		LEFT OUTER JOIN
			software_titles st ON st.id = si.title_id
		LEFT OUTER JOIN
			users u ON u.id = hsi.user_id
		LEFT OUTER JOIN
			users u2 ON u2.id = si.user_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = hsi.host_id
		WHERE
			hsi.host_id = :host_id AND
			hsi.status = :software_status_install_pending
		`),
		// list pending software uninstalls
		fmt.Sprintf(`SELECT
			hsi.execution_id as uuid,
			-- policies with automatic installers generate a host_software_installs with (user_id=NULL,self_service=0),
			-- thus the user_id for the upcoming activity needs to be the user that uploaded the software installer.
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.name, u.name) AS name,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.id, u.id) as user_id,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.gravatar_url, u.gravatar_url) as gravatar_url,
			IF(hsi.user_id IS NULL AND NOT hsi.self_service, u2.email, u.email) AS user_email,
			:uninstalled_software_type as activity_type,
			hsi.created_at as created_at,
			JSON_OBJECT(
				'host_id', hsi.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, ''),
				'script_execution_id', hsi.execution_id,
				'status', CAST(hsi.status AS CHAR)
			) as details
		FROM
			host_software_installs hsi
		INNER JOIN
			software_installers si ON si.id = hsi.software_installer_id
		LEFT OUTER JOIN
			software_titles st ON st.id = si.title_id
		LEFT OUTER JOIN
			users u ON u.id = hsi.user_id
		LEFT OUTER JOIN
			users u2 ON u2.id = si.user_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = hsi.host_id
		WHERE
			hsi.host_id = :host_id AND
			hsi.status = :software_status_uninstall_pending
		`),
		`
SELECT
	hvsi.command_uuid AS uuid,
	u.name AS name,
	u.id AS user_id,
	u.gravatar_url as gravatar_url,
	u.email as user_email,
	:installed_app_store_app_type AS activity_type,
	hvsi.created_at AS created_at,
	JSON_OBJECT(
		'host_id', hvsi.host_id,
		'host_display_name', hdn.display_name,
		'software_title', st.name,
		'app_store_id', hvsi.adam_id,
		'command_uuid', hvsi.command_uuid,
		'self_service', hvsi.self_service IS TRUE,
		-- status is always pending because only pending MDM commands are upcoming.
		'status', :software_status_install_pending
	) AS details
FROM
	host_vpp_software_installs hvsi
INNER JOIN 
	nano_view_queue nvq ON nvq.command_uuid = hvsi.command_uuid
LEFT OUTER JOIN 
	users u ON hvsi.user_id = u.id
LEFT OUTER JOIN 
	host_display_names hdn ON hdn.host_id = hvsi.host_id
LEFT OUTER JOIN 
	vpp_apps vpa ON hvsi.adam_id = vpa.adam_id AND hvsi.platform = vpa.platform
LEFT OUTER JOIN 
	software_titles st ON st.id = vpa.title_id
WHERE
	nvq.status IS NULL
	AND hvsi.host_id = :host_id
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
			details
		FROM ( ` + strings.Join(listStmts, " UNION ALL ") + ` ) AS upcoming `
	listStmt, args, err = sqlx.Named(listStmt, map[string]any{
		"host_id":                           hostID,
		"ran_script_type":                   fleet.ActivityTypeRanScript{}.ActivityName(),
		"installed_software_type":           fleet.ActivityTypeInstalledSoftware{}.ActivityName(),
		"uninstalled_software_type":         fleet.ActivityTypeUninstalledSoftware{}.ActivityName(),
		"installed_app_store_app_type":      fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		"max_wait_time":                     seconds,
		"software_status_install_pending":   fleet.SoftwareInstallPending,
		"software_status_uninstall_pending": fleet.SoftwareUninstallPending,
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build list query from named args")
	}
	stmt, args := appendListOptionsWithCursorToSQL(listStmt, args, &opt)

	var activities []*fleet.Activity
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &activities, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select upcoming activities")
	}

	var metaData *fleet.PaginationMetadata
	metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0, TotalResults: count}
	if len(activities) > int(opt.PerPage) {
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
		if len(activities) > int(opt.PerPage) {
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
