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
func (d *Datastore) NewActivity(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling activity details")
	}
	_, err = d.writer.ExecContext(ctx,
		`INSERT INTO activities (user_id, user_name, activity_type, details) VALUES(?,?,?,?)`,
		user.ID,
		user.Name,
		activityType,
		detailsBytes,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "new activity")
	}
	return nil
}

// ListActivities returns a slice of activities performed across the organization
func (d *Datastore) ListActivities(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Activity, error) {
	activities := []*fleet.Activity{}
	query := `SELECT a.id, a.user_id, a.created_at, a.activity_type, a.details, coalesce(u.name, a.user_name) as name, u.gravatar_url, u.email
	          FROM activities a LEFT JOIN users u ON (a.user_id=u.id)
			  WHERE true`
	query = appendListOptionsToSQL(query, opt)

	err := sqlx.SelectContext(ctx, d.reader, &activities, query)
	if err == sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, notFound("Activity"))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select activities")
	}

	return activities, nil
}
