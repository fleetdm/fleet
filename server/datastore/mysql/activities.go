package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

var (
	automationActivityAuthor = "Fleet"
	deleteIDsBatchSize       = 1000
)

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
	var fleetInitiated bool
	if user != nil {
		// To support creating activities with users that were deleted. This can happen
		// for automatically installed software which uses the author of the upload as the author of
		// the installation.
		if user.ID != 0 && !user.Deleted {
			userID = &user.ID
		}
		userName = &user.Name
		userEmail = &user.Email
	}
	if automatableActivity, ok := activity.(fleet.AutomatableActivity); ok && automatableActivity.WasFromAutomation() {
		userName = &automationActivityAuthor
		fleetInitiated = true
	}

	cols := []string{"fleet_initiated", "user_id", "user_name", "activity_type", "details", "created_at"}
	args := []any{
		fleetInitiated,
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

	vppPtrAct, okPtr := activity.(*fleet.ActivityInstalledAppStoreApp)
	vppAct, ok := activity.(fleet.ActivityInstalledAppStoreApp)
	if okPtr || ok {
		hostID := vppAct.HostID
		cmdUUID := vppAct.CommandUUID
		if okPtr {
			cmdUUID = vppPtrAct.CommandUUID
			hostID = vppPtrAct.HostID
		}

		activateNext := vppAct.Status != string(fleet.SoftwareInstalled)
		if vppPtrAct != nil {
			activateNext = vppPtrAct.Status != string(fleet.SoftwareInstalled)
		}

		if activateNext {
			// NOTE: ideally this would be called in the same transaction as storing
			// the nanomdm command results, but the current design doesn't allow for
			// that with the nano store being a distinct entity to our datastore (we
			// should get rid of that distinction eventually, we've broken it already
			// in some places and it doesn't bring much benefit anymore).
			//
			// Instead, this gets called from CommandAndReportResults, which is
			// executed after the results have been saved in nano, but we already
			// accept this non-transactional fact for many other states we manage in
			// Fleet (wipe, lock results, setup experience results, etc. - see all
			// critical data that gets updated in CommandAndReportResults) so there's
			// no reason to treat the unified queue differently.
			//
			// This place here is a bit hacky but perfect for VPP apps as the activity
			// gets created only when the MDM command status is in a final state
			// (success or failure), which is exactly when we want to activate the next
			// activity.
			if _, err := ds.activateNextUpcomingActivity(ctx, ds.writer(ctx), hostID, cmdUUID); err != nil {
				return ctxerr.Wrap(ctx, err, "activate next activity from VPP app install")
			}
		}
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
			a.user_email,
			a.fleet_initiated
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
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, ua.payload->>'$.user.name')) as name,
			u.id as user_id,
			COALESCE(u.gravatar_url, ua.payload->>'$.user.gravatar_url') as gravatar_url,
			COALESCE(u.email, ua.payload->>'$.user.email') as user_email,
			:ran_script_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'script_name', COALESCE(ses.name, scr.name, ''),
				'script_execution_id', ua.execution_id,
				'async', NOT ua.payload->'$.sync_request',
				'policy_id', sua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated
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
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, ua.payload->>'$.user.name')) AS name,
			ua.user_id as user_id,
			COALESCE(u.gravatar_url, ua.payload->>'$.user.gravatar_url') as gravatar_url,
			COALESCE(u.email, ua.payload->>'$.user.email') as user_email,
			:installed_software_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, ua.payload->>'$.software_title_name', ''),
				'software_package', COALESCE(si.filename, ua.payload->>'$.installer_filename', ''),
				'install_uuid', ua.execution_id,
				'status', 'pending_install',
				'self_service', ua.payload->'$.self_service' IS TRUE,
				'policy_id', siua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated
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
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, ua.payload->>'$.user.name')) AS name,
			ua.user_id as user_id,
			COALESCE(u.gravatar_url, ua.payload->>'$.user.gravatar_url') as gravatar_url,
			COALESCE(u.email, ua.payload->>'$.user.email') as user_email,
			:uninstalled_software_type as activity_type,
			ua.created_at as created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', COALESCE(hdn.display_name, ''),
				'software_title', COALESCE(st.name, ua.payload->>'$.software_title_name', ''),
				'script_execution_id', ua.execution_id,
				'status', 'pending_uninstall',
				'self_service', COALESCE(ua.payload->'$.self_service', FALSE) IS TRUE,
				'policy_id', siua.policy_id,
				'policy_name', p.name
			) as details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated
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
			IF(ua.fleet_initiated, 'Fleet', COALESCE(u.name, ua.payload->>'$.user.name')) AS name,
			u.id AS user_id,
			COALESCE(u.gravatar_url, ua.payload->>'$.user.gravatar_url') as gravatar_url,
			COALESCE(u.email, ua.payload->>'$.user.email') as user_email,
			:installed_app_store_app_type AS activity_type,
			ua.created_at AS created_at,
			JSON_OBJECT(
				'host_id', ua.host_id,
				'host_display_name', hdn.display_name,
				'software_title', st.name,
				'app_store_id', vaua.adam_id,
				'command_uuid', ua.execution_id,
				'self_service', ua.payload->'$.self_service' IS TRUE,
				'status', 'pending_install'
			) AS details,
			IF(ua.activated_at IS NULL, 0, 1) as topmost,
			ua.priority as priority,
			ua.fleet_initiated as fleet_initiated
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
			fleet_initiated
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
		u.id as user_id,
		a.fleet_initiated as fleet_initiated
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

	// `activities` and `queries` are not tied because the activity itself holds
	// the query SQL so they don't need to be executed on the same transaction.
	//
	// All expired live queries are deleted in batch sizes of
	// `deleteIDsBatchSize` to ensure the table size is kept in check
	// with high volumes of live queries (zero-trust workflows). This differs
	// from the `activities` cleanup which uses maxCount as a limit to the
	// number of activities to delete.

	const selectUnsavedQueryIDs = `
		SELECT id
		FROM queries
		WHERE NOT saved
		AND created_at < DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY id`
	var allUnsavedQueryIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allUnsavedQueryIDs, selectUnsavedQueryIDs, expiredWindowDays); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired unsaved query IDs")
	}

	unsavedQueryIter := slices.Chunk(allUnsavedQueryIDs, deleteIDsBatchSize)

	for unsavedQueryIDs := range unsavedQueryIter {
		const deleteStmt = `DELETE FROM queries WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, unsavedQueryIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating query to delete unsaved queries")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired unsaved queries")
		}
	}

	// Cleanup orphaned distributed campaigns that reference non-existing queries.
	const selectCampaignIDs = `
		SELECT id
		FROM distributed_query_campaigns dqc
		WHERE NOT EXISTS (
			SELECT 1
			FROM queries q
			WHERE q.id = dqc.query_id
		)
		ORDER BY id`
	var allCampaignIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allCampaignIDs, selectCampaignIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired distributed query campaigns")
	}

	campaignIter := slices.Chunk(allCampaignIDs, deleteIDsBatchSize)

	for campaignIDs := range campaignIter {
		const deleteStmt = `DELETE FROM distributed_query_campaigns WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, campaignIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating delete expired distributed query campaigns stmt")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired distributed query campaigns")
		}
	}

	// Cleanup orphaned distributed campaign targets that reference non-existing distributed campaigns.
	const selectCampaignTargets = `
		SELECT id
		FROM distributed_query_campaign_targets dqct
		WHERE NOT EXISTS (
			SELECT 1
			FROM distributed_query_campaigns dqc
			WHERE dqc.id = dqct.distributed_query_campaign_id
		)
		ORDER BY id`
	var allCampaignTargetIDs []uint

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &allCampaignTargetIDs, selectCampaignTargets); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting expired distributed query campaign targets")
	}

	campaignTargetIter := slices.Chunk(allCampaignTargetIDs, deleteIDsBatchSize)

	for campaignTargetIDs := range campaignTargetIter {
		const deleteStmt = `DELETE FROM distributed_query_campaign_targets WHERE id IN (?)`
		deleteQuery, args, err := sqlx.In(deleteStmt, campaignTargetIDs)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating query to delete expired query campaign targets")
		}
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteQuery, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting expired distributed query campaign targets")
		}
	}

	return nil
}

func (ds *Datastore) CancelHostUpcomingActivity(ctx context.Context, hostID uint, executionID string) (fleet.ActivityDetails, error) {
	var details fleet.ActivityDetails
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		activityDetails, err := ds.cancelHostUpcomingActivity(ctx, tx, hostID, executionID)
		details = activityDetails
		return err
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "cancel upcoming activity transaction")
	}

	return details, nil
}

func (ds *Datastore) cancelHostUpcomingActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, executionID string) (fleet.ActivityDetails, error) {
	const (
		loadScriptActivityStmt = `
	SELECT
		ua.activity_type,
		ua.host_id,
		COALESCE(hdn.display_name, '') as host_display_name,
		COALESCE(ses.name, scr.name, '') as canceled_name, -- script name in this case
		NULL as canceled_id, -- no ID for scripts in the canceled activity
		IF(ua.activated_at IS NULL, 0, 1) as activated
	FROM
		upcoming_activities ua
	INNER JOIN
		script_upcoming_activities sua ON sua.upcoming_activity_id = ua.id
	LEFT OUTER JOIN
		host_display_names hdn ON hdn.host_id = ua.host_id
	LEFT OUTER JOIN
		scripts scr ON scr.id = sua.script_id
	LEFT OUTER JOIN
		setup_experience_scripts ses ON ses.id = sua.setup_experience_script_id
	WHERE
		ua.host_id = :host_id AND
		ua.execution_id = :execution_id AND
		ua.activity_type = 'script'
`

		loadSoftwareInstallActivityStmt = `
	SELECT
		ua.activity_type,
		ua.host_id,
		COALESCE(hdn.display_name, '') as host_display_name,
		COALESCE(st.name, ua.payload->>'$.software_title_name', '') as canceled_name, -- software title name in this case
		st.id as canceled_id,
		IF(ua.activated_at IS NULL, 0, 1) as activated
	FROM
		upcoming_activities ua
	INNER JOIN
		software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
	LEFT OUTER JOIN
		software_installers si ON si.id = siua.software_installer_id
	LEFT OUTER JOIN
		software_titles st ON st.id = si.title_id
	LEFT OUTER JOIN
		host_display_names hdn ON hdn.host_id = ua.host_id
	WHERE
		ua.host_id = :host_id AND
		ua.execution_id = :execution_id AND
		ua.activity_type = 'software_install'
`

		loadSoftwareUninstallActivityStmt = `
	SELECT
		ua.activity_type,
		ua.host_id,
		COALESCE(hdn.display_name, '') as host_display_name,
		COALESCE(st.name, ua.payload->>'$.software_title_name', '') as canceled_name, -- software title name in this case
		st.id as canceled_id,
		IF(ua.activated_at IS NULL, 0, 1) as activated
	FROM
		upcoming_activities ua
	INNER JOIN
		software_install_upcoming_activities siua ON siua.upcoming_activity_id = ua.id
	LEFT OUTER JOIN
		software_installers si ON si.id = siua.software_installer_id
	LEFT OUTER JOIN
		software_titles st ON st.id = si.title_id
	LEFT OUTER JOIN
		host_display_names hdn ON hdn.host_id = ua.host_id
	WHERE
		ua.host_id = :host_id AND
		ua.execution_id = :execution_id AND
		activity_type = 'software_uninstall'
`

		loadVPPAppInstallActivityStmt = `
	SELECT
		ua.activity_type,
		ua.host_id,
		COALESCE(hdn.display_name, '') as host_display_name,
		COALESCE(st.name, '') as canceled_name, -- software title name in this case
		st.id as canceled_id,
		IF(ua.activated_at IS NULL, 0, 1) as activated
	FROM
		upcoming_activities ua
	INNER JOIN
		vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
	LEFT OUTER JOIN
		host_display_names hdn ON hdn.host_id = ua.host_id
	LEFT OUTER JOIN
		vpp_apps vpa ON vaua.adam_id = vpa.adam_id AND vaua.platform = vpa.platform
	LEFT OUTER JOIN
		software_titles st ON st.id = vpa.title_id
	WHERE
		ua.host_id = :host_id AND
		ua.execution_id = :execution_id AND
		ua.activity_type = 'vpp_app_install'
`
	)

	type activityToCancel struct {
		ActivityType    string `db:"activity_type"`
		HostID          uint   `db:"host_id"`
		HostDisplayName string `db:"host_display_name"`
		CanceledName    string `db:"canceled_name"`
		CanceledID      *uint  `db:"canceled_id"`
		Activated       bool   `db:"activated"`
	}

	var act activityToCancel
	var pastAct fleet.ActivityDetails
	// read the activity along with the required information to create the
	// "canceled" past activity, and check if the activity was activated or
	// not.
	stmt := strings.Join([]string{
		loadScriptActivityStmt, loadSoftwareInstallActivityStmt,
		loadSoftwareUninstallActivityStmt, loadVPPAppInstallActivityStmt,
	}, " UNION ALL ")
	stmt, args, err := sqlx.Named(stmt, map[string]any{"host_id": hostID, "execution_id": executionID})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build load upcoming activity to cancel statement")
	}

	if err := sqlx.GetContext(ctx, tx, &act, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("UpcomingActivity").WithName(executionID))
		}
		return nil, ctxerr.Wrap(ctx, err, "load upcoming activity to cancel")
	}

	// in all cases, we must delete the row from upcoming_activities
	const delStmt = `DELETE FROM upcoming_activities WHERE host_id = ? AND execution_id = ?`
	if _, err := tx.ExecContext(ctx, delStmt, hostID, executionID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "delete upcoming activity")
	}

	// if the activity is related to lock/wipe actions, clear the status for that
	// action as it was canceled (note that lock/wipe is prevented at the service
	// layer from being canceled if it was already activated).
	if err := clearLockWipeForCanceledActivity(ctx, tx, hostID, executionID); err != nil {
		return nil, err
	}

	// must get the host uuid for the setup experience and nano table updates
	const getHostUUIDStmt = `SELECT uuid FROM hosts WHERE id = ?`
	var hostUUID string
	if err := sqlx.GetContext(ctx, tx, &hostUUID, getHostUUIDStmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host uuid")
	}

	switch act.ActivityType {
	case "script":
		// if the script was part of the setup experience, then it must be marked
		// as "failed" for that setup experience flow (regardless of whether or
		// not it was activated).
		const failSetupExpStmt = `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND script_execution_id = ?`
		if _, err := tx.ExecContext(ctx, failSetupExpStmt, fleet.SetupExperienceStatusFailure, hostUUID, executionID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update setup_experience_status_results as failed")
		}

		if act.Activated {
			const updStmt = `UPDATE host_script_results SET canceled = 1 WHERE execution_id = ?`
			if _, err := tx.ExecContext(ctx, updStmt, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update host_script_results as canceled")
			}
		}

		pastAct = fleet.ActivityTypeCanceledRunScript{
			HostID:          act.HostID,
			HostDisplayName: act.HostDisplayName,
			ScriptName:      act.CanceledName,
		}

	case "software_install":
		// if the install was part of the setup experience, then it must be
		// marked as "failed" for that setup experience flow (regardless of
		// whether or not it was activated).
		const failSetupExpStmt = `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND host_software_installs_execution_id = ?`
		if _, err := tx.ExecContext(ctx, failSetupExpStmt, fleet.SetupExperienceStatusFailure, hostUUID, executionID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update setup_experience_status_results as failed")
		}

		if act.Activated {
			const updStmt = `UPDATE host_software_installs SET canceled = 1 WHERE execution_id = ?`
			if _, err := tx.ExecContext(ctx, updStmt, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update host_software_installs as canceled")
			}
		}

		var titleID uint
		if act.CanceledID != nil {
			titleID = *act.CanceledID
		}
		pastAct = fleet.ActivityTypeCanceledInstallSoftware{
			HostID:          act.HostID,
			HostDisplayName: act.HostDisplayName,
			SoftwareTitle:   act.CanceledName,
			SoftwareTitleID: titleID,
		}

	case "software_uninstall":
		// uninstall cannot be part of setup experience, so there's no update for
		// that in this case.

		if act.Activated {
			// uninstall is a combination of software install and script result,
			// with the same execution id.
			const updSoftwareStmt = `UPDATE host_software_installs SET canceled = 1 WHERE execution_id = ?`
			if _, err := tx.ExecContext(ctx, updSoftwareStmt, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update host_software_installs as canceled")
			}

			const updScriptStmt = `UPDATE host_script_results SET canceled = 1 WHERE execution_id = ?`
			if _, err := tx.ExecContext(ctx, updScriptStmt, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update host_script_results as canceled")
			}
		}

		var titleID uint
		if act.CanceledID != nil {
			titleID = *act.CanceledID
		}
		pastAct = fleet.ActivityTypeCanceledUninstallSoftware{
			HostID:          act.HostID,
			HostDisplayName: act.HostDisplayName,
			SoftwareTitle:   act.CanceledName,
			SoftwareTitleID: titleID,
		}

	case "vpp_app_install":
		// if the VPP install was part of the setup experience, then it must be
		// marked as "failed" for that setup experience flow (regardless of
		// whether or not it was activated).
		const failSetupExpStmt = `UPDATE setup_experience_status_results SET status = ? WHERE host_uuid = ? AND nano_command_uuid = ?`
		if _, err := tx.ExecContext(ctx, failSetupExpStmt, fleet.SetupExperienceStatusFailure, hostUUID, executionID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "update setup_experience_status_results as failed")
		}

		if act.Activated {
			const updVPPStmt = `UPDATE host_vpp_software_installs SET canceled = 1 WHERE command_uuid = ?`
			if _, err := tx.ExecContext(ctx, updVPPStmt, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update host_vpp_software_installs as canceled")
			}

			const updNanoStmt = `UPDATE nano_enrollment_queue SET active = 0 WHERE id = ? AND command_uuid = ?`
			if _, err := tx.ExecContext(ctx, updNanoStmt, hostUUID, executionID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "update nano_enrollment_queue as canceled")
			}
		}

		var titleID uint
		if act.CanceledID != nil {
			titleID = *act.CanceledID
		}
		pastAct = fleet.ActivityTypeCanceledInstallAppStoreApp{
			HostID:          act.HostID,
			HostDisplayName: act.HostDisplayName,
			SoftwareTitle:   act.CanceledName,
			SoftwareTitleID: titleID,
		}

	default:
		// cannot happen since activity type comes from the UNION query above,
		// but can be useful to detect a missing case in tests
		panic(fmt.Sprintf("unexpected activity type %q", act.ActivityType))
	}

	// must activate the next activity, if any (this should be required only if
	// the canceled activity was already "activated", but there's no harm in
	// doing it if it wasn't, and it makes sure there's always progress even in
	// unsuspected scenarios)
	if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "activate next upcoming activity")
	}

	// creating the canceled activity must be done via svc.NewActivity (not
	// ds.NewActivity), so we return the ready-to-insert activity struct to the
	// caller and let svc do the rest.
	return pastAct, nil
}

func clearLockWipeForCanceledActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, executionID string) error {
	const clearLockStmt = `DELETE FROM host_mdm_actions WHERE host_id = ? AND lock_ref = ?`
	resLock, err := tx.ExecContext(ctx, clearLockStmt, hostID, executionID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_mdm_actions for lock")
	}

	const clearWipeStmt = `DELETE FROM host_mdm_actions WHERE host_id = ? AND wipe_ref = ?`
	resWipe, err := tx.ExecContext(ctx, clearWipeStmt, hostID, executionID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete host_mdm_actions for wipe")
	}

	lockCnt, _ := resLock.RowsAffected()
	wipeCnt, _ := resWipe.RowsAffected()
	if lockCnt > 0 || wipeCnt > 0 {
		// if it did delete host_mdm_actions, then it was a lock or wipe activity,
		// we need to delete the "past" activity that gets created immediately
		// when that command is queued.
		actType := fleet.ActivityTypeLockedHost{}.ActivityName()
		if wipeCnt > 0 {
			actType = fleet.ActivityTypeWipedHost{}.ActivityName()
		}

		const findActStmt = `SELECT
				id
			FROM
				activities
				INNER JOIN host_activities ON (host_activities.activity_id = activities.id)
			WHERE
				host_activities.host_id = ? AND
				activities.activity_type = ?
			ORDER BY
				activities.created_at DESC
			LIMIT 1
`
		var activityID uint
		if err := sqlx.GetContext(ctx, tx, &activityID, findActStmt, hostID, actType); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// no activity to delete, nothing to do
				return nil
			}
			return ctxerr.Wrap(ctx, err, "find past activity for lock/wipe")
		}

		const delStmt = `DELETE FROM activities WHERE id = ?`
		if _, err := tx.ExecContext(ctx, delStmt, activityID); err != nil {
			return ctxerr.Wrap(ctx, err, "delete past activity for lock/wipe")
		}
	}
	return nil
}

// GetHostUpcomingActivityMeta returns metadata for an upcoming activity,
// such as whether it is activated or not, if the activity corresponds to a
// lock/wipe/unlock command, etc.
func (ds *Datastore) GetHostUpcomingActivityMeta(ctx context.Context, hostID uint, executionID string) (*fleet.UpcomingActivityMeta, error) {
	const getStmt = `
	SELECT
		ua.execution_id,
		ua.activated_at,
		ua.activity_type,
		CASE
			WHEN hma.lock_ref = :execution_id THEN :lock_action
			WHEN hma.unlock_ref = :execution_id THEN :unlock_action
			WHEN hma.wipe_ref = :execution_id THEN :wipe_action
			ELSE :none_action
		END AS well_known_action
	FROM
		upcoming_activities ua
		LEFT JOIN host_mdm_actions hma ON hma.host_id = ua.host_id
	WHERE
		ua.host_id = :host_id AND
		ua.execution_id = :execution_id
`

	namedArgs := map[string]any{
		"host_id":       hostID,
		"execution_id":  executionID,
		"lock_action":   fleet.WellKnownActionLock,
		"unlock_action": fleet.WellKnownActionUnlock,
		"wipe_action":   fleet.WellKnownActionWipe,
		"none_action":   fleet.WellKnownActionNone,
	}
	stmt, args, err := sqlx.Named(getStmt, namedArgs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build named query for upcoming activity meta")
	}

	var actMeta fleet.UpcomingActivityMeta
	err = sqlx.GetContext(ctx, ds.reader(ctx), &actMeta, stmt, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, notFound("UpcomingActivity").WithName(executionID)
		}
		return nil, ctxerr.Wrap(ctx, err, "lookup upcoming activity meta")
	}
	return &actMeta, nil
}

// UnblockHostsUpcomingActivityQueue checks for hosts that have upcoming
// activities but none is "activated", meaning that the queue is blocked
// (cannot make progress anymore), possibly due to a failure when activating
// the next activity, or to a missing call to activateNextUpcomingActivity. It
// unblocks up to maxHosts found in this situation (by activating the next
// activity for each host).
func (ds *Datastore) UnblockHostsUpcomingActivityQueue(ctx context.Context, maxHosts int) (int, error) {
	const findBlockedHostsStmt = `
		SELECT
			DISTINCT inactive_ua.host_id
		FROM
			upcoming_activities inactive_ua
			LEFT OUTER JOIN upcoming_activities active_ua ON
				active_ua.host_id = inactive_ua.host_id AND
				active_ua.activated_at IS NOT NULL
		WHERE
			active_ua.host_id IS NULL AND
			inactive_ua.activated_at IS NULL
		LIMIT ?`

	var blockedHostIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &blockedHostIDs, findBlockedHostsStmt, maxHosts); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "select blocked hosts")
	}
	return len(blockedHostIDs), ds.activateNextUpcomingActivityForBatchOfHosts(ctx, blockedHostIDs)
}

func (ds *Datastore) activateNextUpcomingActivityForBatchOfHosts(ctx context.Context, hostIDs []uint) error {
	const maxHostIDsPerBatch = 500

	slices.Sort(hostIDs)              // sorting can help avoid deadlocks
	hostIDs = slices.Compact(hostIDs) // dedupe IDs (must be sorted first)

	var errs []error
	for batch := range slices.Chunk(hostIDs, maxHostIDsPerBatch) {
		err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			for _, hostID := range batch {
				if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
					return ctxerr.Wrap(ctx, err, "activate next activity")
				}
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// This function activates the next upcoming activity, if any, for the specified host.
// It does a few things to achieve this:
//   - If there was an activity already marked as activated (activated_at is
//     not NULL) and fromCompletedExecID is provided, it deletes it, as calling
//     this function means that this activated activity is now completed (in a
//     final state, either success or failure).
//   - If no other activity is still activated and there is an upcoming
//     activity to activate next, it does so, respecting the priority and enqueue
//     order. Activation consists of inserting the activity in its respective
//     table, e.g. `host_script_results` for scripts, `host_software_installs` for
//     software installs, `host_vpp_software_installs` and nano command queue for
//     VPP installs; and setting the activated_at timestamp in the
//     `upcoming_activities` table.
//   - As an optimization for MDM, if the activity type is `vpp_app_install`
//     and the next few upcoming activities are all of this type, they are
//     batch-activated together (up to a limit) to reduce the processing
//     latency and number of push notifications to send to this host.
//
// When called after receiving results for an activity, the fromCompletedExecID
// argument identifies that completed activity.
func (ds *Datastore) activateNextUpcomingActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, fromCompletedExecID string) (activatedExecIDs []string, err error) {
	const maxMDMCommandActivations = 5

	const deleteCompletedStmt = `
DELETE FROM upcoming_activities
WHERE
	host_id = ? AND
	activated_at IS NOT NULL AND
	execution_id = ?
`

	const findNextStmt = `
SELECT
	execution_id,
	activity_type,
	activated_at,
	IF(activated_at IS NULL, 0, 1) as topmost,
	priority
FROM
	upcoming_activities
WHERE
	host_id = ?
	%s
ORDER BY topmost DESC, priority DESC, created_at ASC
LIMIT ?
`

	const findNextSpecificExecIDsClause = ` AND execution_id IN (?) `

	const markActivatedStmt = `
UPDATE upcoming_activities
SET
	activated_at = NOW()
WHERE
	host_id = ? AND
	execution_id IN (?)
`

	// first we delete the completed activity, if any
	if fromCompletedExecID != "" {
		if _, err := tx.ExecContext(ctx, deleteCompletedStmt, hostID, fromCompletedExecID); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "delete completed upcoming activity")
		}
	}

	// next we look for an upcoming activity to activate
	type nextActivity struct {
		ExecutionID  string     `db:"execution_id"`
		ActivityType string     `db:"activity_type"`
		ActivatedAt  *time.Time `db:"activated_at"`
		Topmost      bool       `db:"topmost"`
		Priority     int        `db:"priority"`
	}
	var nextActivities []nextActivity
	stmt, args := fmt.Sprintf(findNextStmt, ""), []any{hostID, maxMDMCommandActivations}
	if len(ds.testActivateSpecificNextActivities) > 0 {
		stmt, args, err = sqlx.In(fmt.Sprintf(findNextStmt, findNextSpecificExecIDsClause),
			hostID, ds.testActivateSpecificNextActivities, len(ds.testActivateSpecificNextActivities))
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "prepare find next upcoming activities statement with test execution ids")
		}
	}
	if err := sqlx.SelectContext(ctx, tx, &nextActivities, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find next upcoming activities to activate")
	}

	var toActivate []nextActivity
	for _, act := range nextActivities {
		if act.ActivatedAt != nil {
			// there are still activated activities, do not activate more
			break
		}
		if len(toActivate) > 0 {
			// we already identified one to activate, allow more only if they are a)
			// the same type, b) that type is vpp_app_install, c) the same priority.
			// The reason for that is to batch-activate MDM commands to reduce
			// latency and push notifications required, and the same priority check
			// is because we can't enforce the ordering of commands if they don't
			// share the same priority (we transfer the created_at timestamp to the
			// nano queue, which guarantees same order of processing for activities
			// with the same priority).
			if toActivate[0].ActivityType != act.ActivityType ||
				toActivate[0].ActivityType != "vpp_app_install" ||
				toActivate[0].Priority != act.Priority {
				break
			}
		}
		toActivate = append(toActivate, act)
		activatedExecIDs = append(activatedExecIDs, act.ExecutionID)
	}

	if len(toActivate) == 0 {
		return nil, nil
	}

	// activate the next activities as required for its activity type
	var fn func(context.Context, sqlx.ExtContext, uint, []string) error
	switch actType := toActivate[0].ActivityType; actType {
	case "script":
		fn = ds.activateNextScriptActivity
	case "software_install":
		fn = ds.activateNextSoftwareInstallActivity
	case "software_uninstall":
		fn = ds.activateNextSoftwareUninstallActivity
	case "vpp_app_install":
		fn = ds.activateNextVPPAppInstallActivity
	default:
		return nil, ctxerr.Errorf(ctx, "unsupported activity type %s", actType)
	}
	if err := fn(ctx, tx, hostID, activatedExecIDs); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "activate next activities")
	}

	// finally, mark the activities as activated
	stmt, args, err = sqlx.In(markActivatedStmt, hostID, activatedExecIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "prepare statement to mark upcoming activities as activated")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "mark upcoming activities as activated")
	}
	return activatedExecIDs, nil
}

func (ds *Datastore) activateNextScriptActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, execIDs []string) error {
	const insStmt = `
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
	ua.host_id = ? AND
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	// sanity-check that there's something to activate
	if len(execIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(insStmt, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert to activate scripts")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert to activate scripts")
	}
	return nil
}

func (ds *Datastore) activateNextSoftwareInstallActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, execIDs []string) error {
	const insStmt = `
INSERT INTO host_software_installs
	(execution_id, host_id, software_installer_id, user_id, self_service,
		policy_id, installer_filename, version, software_title_id, software_title_name)
SELECT
	ua.execution_id,
	ua.host_id,
	siua.software_installer_id,
	ua.user_id,
	COALESCE(ua.payload->'$.self_service', 0),
	siua.policy_id,
	COALESCE(ua.payload->>'$.installer_filename', '[deleted installer]'),
	COALESCE(ua.payload->>'$.version', 'unknown'),
	siua.software_title_id,
	COALESCE(ua.payload->>'$.software_title_name', '[deleted title]')
FROM
	upcoming_activities ua
	INNER JOIN software_install_upcoming_activities siua
		ON siua.upcoming_activity_id = ua.id
WHERE
	ua.host_id = ? AND
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	// sanity-check that there's something to activate
	if len(execIDs) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(insStmt, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert to activate software installs")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert to activate software installs")
	}
	return nil
}

func (ds *Datastore) activateNextSoftwareUninstallActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, execIDs []string) error {
	const insScriptStmt = `
INSERT INTO
	host_script_results
(host_id, execution_id, script_content_id, output, user_id, is_internal)
SELECT
	ua.host_id,
	ua.execution_id,
	si.uninstall_script_content_id,
	'',
	ua.user_id,
	1
FROM
	upcoming_activities ua
	INNER JOIN software_install_upcoming_activities siua
		ON siua.upcoming_activity_id = ua.id
	INNER JOIN software_installers si
		ON si.id = siua.software_installer_id
WHERE
	ua.host_id = ? AND
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	const insSwStmt = `
INSERT INTO
	host_software_installs
(execution_id, host_id, software_installer_id, user_id, uninstall, installer_filename,
	software_title_id, software_title_name, self_service, version)
SELECT
	ua.execution_id,
	ua.host_id,
	siua.software_installer_id,
	ua.user_id,
	1,  -- uninstall
	'', -- no installer_filename for uninstalls
	siua.software_title_id,
	COALESCE(ua.payload->>'$.software_title_name', '[deleted title]'),
	COALESCE(ua.payload->>'$.self_service', FALSE),
	'unknown'
FROM
	upcoming_activities ua
	INNER JOIN software_install_upcoming_activities siua
		ON siua.upcoming_activity_id = ua.id
WHERE
	ua.host_id = ? AND
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`
	// sanity-check that there's something to activate
	if len(execIDs) == 0 {
		return nil
	}

	stmt, args, err := sqlx.In(insScriptStmt, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert script to activate software uninstalls")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert script to activate software uninstalls")
	}

	stmt, args, err = sqlx.In(insSwStmt, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert software to activate software uninstalls")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert software to activate software uninstalls")
	}
	return nil
}

func (ds *Datastore) activateNextVPPAppInstallActivity(ctx context.Context, tx sqlx.ExtContext, hostID uint, execIDs []string) error {
	const insStmt = `
INSERT INTO
	host_vpp_software_installs
(host_id, adam_id, platform, command_uuid,
	user_id, associated_event_id, self_service, policy_id)
SELECT
	ua.host_id,
	vaua.adam_id,
	vaua.platform,
	ua.execution_id,
	ua.user_id,
	ua.payload->>'$.associated_event_id',
	COALESCE(ua.payload->'$.self_service', 0),
	vaua.policy_id
FROM
	upcoming_activities ua
	INNER JOIN vpp_app_upcoming_activities vaua
		ON vaua.upcoming_activity_id = ua.id
WHERE
	ua.host_id = ? AND
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	const getHostUUIDStmt = `
SELECT
	uuid
FROM
	hosts
WHERE
	id = ?
`

	const insCmdStmt = `
INSERT INTO
	nano_commands
(command_uuid, request_type, command, subtype)
SELECT
	ua.execution_id,
	'InstallApplication',
	CONCAT(:raw_cmd_part1, vaua.adam_id, :raw_cmd_part2, ua.execution_id, :raw_cmd_part3),
	:subtype
FROM
	upcoming_activities ua
	INNER JOIN vpp_app_upcoming_activities vaua
		ON vaua.upcoming_activity_id = ua.id
WHERE
	ua.host_id = :host_id AND
	ua.execution_id IN (:execution_ids)
`

	const rawCmdPart1 = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Command</key>
    <dict>
		<key>InstallAsManaged</key>
		<true/>
        <key>ManagementFlags</key>
        <integer>0</integer>
        <key>Options</key>
        <dict>
            <key>PurchaseMethod</key>
            <integer>1</integer>
        </dict>
        <key>RequestType</key>
        <string>InstallApplication</string>
        <key>iTunesStoreID</key>
        <integer>`

	const rawCmdPart2 = `</integer>
    </dict>
    <key>CommandUUID</key>
    <string>`

	const rawCmdPart3 = `</string>
</dict>
</plist>`

	const insNanoQueueStmt = `
INSERT INTO
	nano_enrollment_queue
(id, command_uuid, created_at)
SELECT
	?,
	execution_id,
	created_at -- force same timestamp to keep ordering
FROM
	upcoming_activities
WHERE
	host_id = ? AND
	execution_id IN (?)
ORDER BY
	priority DESC, created_at ASC
`

	// sanity-check that there's something to activate
	if len(execIDs) == 0 {
		return nil
	}

	// get the host uuid, requires for the nano tables
	var hostUUID string
	if err := sqlx.GetContext(ctx, tx, &hostUUID, getHostUUIDStmt, hostID); err != nil {
		return ctxerr.Wrap(ctx, err, "get host uuid")
	}

	// insert the host vpp app row
	stmt, args, err := sqlx.In(insStmt, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert to activate vpp apps")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert to activate vpp apps")
	}

	// insert the nano command
	namedArgs := map[string]any{
		"raw_cmd_part1": rawCmdPart1,
		"raw_cmd_part2": rawCmdPart2,
		"raw_cmd_part3": rawCmdPart3,
		"subtype":       mdm.CommandSubtypeNone,
		"host_id":       hostID,
		"execution_ids": execIDs,
	}
	stmt, args, err = sqlx.Named(insCmdStmt, namedArgs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert nano commands")
	}
	stmt, args, err = sqlx.In(stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "expand IN arguments to insert nano commands")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert nano commands")
	}

	// enqueue the nano command in the nano queue
	stmt, args, err = sqlx.In(insNanoQueueStmt, hostUUID, hostID, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert nano queue")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert nano queue")
	}

	// best-effort APNs push notification to the host, not critical because we
	// have a cron job that will retry for hosts with pending MDM commands.
	if ds.pusher != nil {
		if _, err := ds.pusher.Push(ctx, []string{hostUUID}); err != nil {
			level.Error(ds.logger).Log("msg", "failed to send push notification", "err", err, "hostID", hostID, "hostUUID", hostUUID) //nolint:errcheck
		}
	}
	return nil
}
