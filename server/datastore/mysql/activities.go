package mysql

import (
	"context"
	"database/sql"
	"encoding/json"

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
	if user != nil {
		userID = &user.ID
		userName = &user.Name
	}

	_, err = ds.writer.ExecContext(ctx,
		`INSERT INTO activities (user_id, user_name, activity_type, details) VALUES(?,?,?,?)`,
		userID,
		userName,
		activity.ActivityName(),
		detailsBytes,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "new activity")
	}
	return nil
}

// ListActivities returns a slice of activities performed across the organization
func (ds *Datastore) ListActivities(ctx context.Context, opt fleet.ListActivitiesOptions) ([]*fleet.Activity, *fleet.PaginationMetadata, error) {
	activities := []*fleet.Activity{}
	query := `
SELECT
	a.id,
	a.user_id,
	a.created_at,
	a.activity_type,
	a.details,
	coalesce(u.name, a.user_name) as name,
	u.gravatar_url,
	u.email,
	a.streamed
FROM activities a
LEFT JOIN users u ON (a.user_id=u.id)
WHERE true`

	var args []interface{}
	if opt.Streamed != nil {
		query += " AND a.streamed = ?"
		args = append(args, *opt.Streamed)
	}

	if !(opt.ListOptions.UsesCursorPagination()) {
		opt.ListOptions.IncludeMetadata = true
	}

	query, args = appendListOptionsWithCursorToSQL(query, args, &opt.ListOptions)

	err := sqlx.SelectContext(ctx, ds.reader, &activities, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil, ctxerr.Wrap(ctx, notFound("Activity"))
	} else if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select activities")
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
	if _, err := ds.writer.ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "exec mark activities as streamed")
	}
	return nil
}
