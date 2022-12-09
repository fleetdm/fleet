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
func (ds *Datastore) NewActivity(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) (*fleet.Activity, error) {
	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "marshaling activity details")
	}
	res, err := ds.writer.ExecContext(ctx,
		`INSERT INTO activities (user_id, user_name, activity_type, details) VALUES(?,?,?,?)`,
		user.ID,
		user.Name,
		activityType,
		detailsBytes,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new activity")
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last id after inserting activity")
	}
	return activityDB(ctx, ds.writer, uint(id))
}

var activityStmt = `
SELECT 
	a.id,
	a.user_id,
	a.created_at,
	a.activity_type,
	a.details,
	coalesce(u.name, a.user_name) as name,
	u.gravatar_url,
	u.email
FROM activities a 
LEFT JOIN users u ON (a.user_id=u.id)`

func activityDB(ctx context.Context, q sqlx.QueryerContext, id uint) (*fleet.Activity, error) {
	stmt := activityStmt + " WHERE a.id = ?"
	var activity fleet.Activity
	if err := sqlx.GetContext(ctx, q, &activity, stmt, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Activity").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "select activity")
	}
	return &activity, nil
}

// ListActivities returns a slice of activities performed across the organization
func (ds *Datastore) ListActivities(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Activity, error) {
	activities := []*fleet.Activity{}
	stmt := activityStmt + " WHERE true"
	stmt = appendListOptionsToSQL(stmt, opt)

	err := sqlx.SelectContext(ctx, ds.reader, &activities, stmt)
	if err == sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, notFound("Activity"))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	return activities, nil
}
