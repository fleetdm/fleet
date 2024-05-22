package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

// NewActivity stores an activity item that the user performed
func (ds *Datastore) NewActivity(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
	detailsBytes, err := json.Marshal(activity)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling activity details")
	}

	var userID *uint
	var userName *string
	var userEmail *string
	if user != nil {
		userID = &user.ID
		userName = &user.Name
		userEmail = &user.Email
	}

	cols := []string{"user_id", "user_name", "activity_type", "details"}
	args := []any{
		userID,
		userName,
		activity.ActivityName(),
		detailsBytes,
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
	countStmts := []string{
		`SELECT COUNT(*) c FROM host_script_results WHERE host_id = :host_id AND exit_code IS NULL`,
		`SELECT COUNT(*) c FROM host_software_installs WHERE host_id = :host_id AND pre_install_query_output IS NULL AND install_script_exit_code IS NULL`,
	}

	var count uint
	countStmt := `SELECT SUM(c) FROM ( ` + strings.Join(countStmts, " UNION ALL ") + ` ) AS counts`
	countStmt, args, err := sqlx.Named(countStmt, map[string]any{"host_id": hostID})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build count query from named args")
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, countStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "count upcoming activities")
	}
	if count == 0 {
		return []*fleet.Activity{}, &fleet.PaginationMetadata{}, nil
	}

	// NOTE: Be sure to update both the count and list statements if the list query is modified
	listStmts := []string{
		// list pending scripts
		`SELECT
			hsr.execution_id as uuid,
			u.name as name,
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
				'async', NOT hsr.sync_request
			) as details
		FROM
			host_script_results hsr
		LEFT OUTER JOIN
			users u ON u.id = hsr.user_id
		LEFT OUTER JOIN
			host_display_names hdn ON hdn.host_id = hsr.host_id
		LEFT OUTER JOIN
			scripts scr ON scr.id = hsr.script_id
		WHERE
			hsr.host_id = :host_id AND
			hsr.exit_code IS NULL AND
			(
				hsr.sync_request = 0 OR
				hsr.created_at >= DATE_SUB(NOW(), INTERVAL :max_wait_time SECOND)
			)
`,
		// list pending software installs
		fmt.Sprintf(`SELECT
			hsi.execution_id as uuid,
			u.name as name,
			u.id as user_id,
			u.gravatar_url as gravatar_url,
			u.email as user_email,
			:installed_software_type as activity_type,
			hsi.created_at as created_at,
			JSON_OBJECT(
				'host_id', hsi.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, ''),
				'install_uuid', hsi.execution_id,
				'status', %s
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
			host_display_names hdn ON hdn.host_id = hsi.host_id
		WHERE
			hsi.host_id = :host_id AND
			hsi.pre_install_query_output IS NULL AND
			hsi.install_script_exit_code IS NULL
		`, softwareInstallerHostStatusNamedQuery("hsi", "")),
	}

	seconds := int(scripts.MaxServerWaitTime.Seconds())
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
		"host_id":                   hostID,
		"ran_script_type":           fleet.ActivityTypeRanScript{}.ActivityName(),
		"installed_software_type":   fleet.ActivityTypeInstalledSoftware{}.ActivityName(),
		"max_wait_time":             seconds,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_installed": fleet.SoftwareInstallerInstalled,
		"software_status_pending":   fleet.SoftwareInstallerPending,
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
	if err := ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Delete temporary queries (aka "not saved").
		if _, err := tx.ExecContext(ctx,
			`DELETE FROM queries
			WHERE NOT saved AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
			LIMIT ?`,
			expiredWindowDays, maxCount,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired non-saved queries")
		}
		// Delete distributed campaigns that reference unexisting query (removed in the previous query).
		if _, err := tx.ExecContext(ctx,
			`DELETE distributed_query_campaigns FROM distributed_query_campaigns
			LEFT JOIN queries ON (distributed_query_campaigns.query_id=queries.id)
			WHERE queries.id IS NULL`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired orphaned distributed_query_campaigns")
		}
		// Delete distributed campaign targets that reference unexisting distributed campaign (removed in the previous query).
		if _, err := tx.ExecContext(ctx,
			`DELETE distributed_query_campaign_targets FROM distributed_query_campaign_targets
			LEFT JOIN distributed_query_campaigns ON (distributed_query_campaign_targets.distributed_query_campaign_id=distributed_query_campaigns.id)
			WHERE distributed_query_campaigns.id IS NULL`,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "delete expired orphaned distributed_query_campaign_targets")
		}
		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "delete expired distributed queries")
	}
	return nil
}
