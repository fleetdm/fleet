package mysql

import (
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
