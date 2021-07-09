package mysql

import (
	"database/sql"
	"encoding/json"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (d *Datastore) NewActivity(user *fleet.User, activityType string, details *map[string]interface{}) error {
	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return errors.Wrap(err, "marshaling activity details")
	}
	_, err = d.db.Exec(
		`INSERT INTO activities (user_id, activity_type, details) VALUES(?,?,?)`,
		user.ID,
		activityType,
		detailsBytes,
	)
	if err != nil {
		return errors.Wrap(err, "new activity")
	}
	return nil
}

func (d *Datastore) ListActivities(opt fleet.ListOptions) ([]*fleet.Activity, error) {
	activities := []*fleet.Activity{}
	query := "SELECT a.*, u.name as name FROM activities a JOIN users u ON (a.user_id=u.id) WHERE true"
	query = appendListOptionsToSQL(query, opt)

	err := d.db.Select(&activities, query)
	if err == sql.ErrNoRows {
		return nil, notFound("Activity")
	} else if err != nil {
		return nil, errors.Wrap(err, "select activities")
	}

	return activities, nil
}
