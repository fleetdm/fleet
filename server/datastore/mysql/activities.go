package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
			a.details,
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
	if err == sql.ErrNoRows {
		return nil, nil, ctxerr.Wrap(ctx, notFound("Activity"))
	} else if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
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

func (ds *Datastore) ListHostUpcomingActivities(ctx context.Context, hostID uint, opt fleet.ListOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	const listStmt = `
		SELECT
			hsr.execution_id as uuid,
			u.name as name,
			u.id as user_id,
			u.gravatar_url as gravatar_url,
			u.email as user_email,
			? as activity_type,
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
			hsr.host_id = ? AND
			hsr.exit_code IS NULL
`

	args := []any{fleet.ActivityTypeRanScript{}.ActivityName(), hostID}
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
