package inmem

import "github.com/fleetdm/fleet/v4/server/fleet"

// NewActivity stores an activity item that the user performed
func (d *Datastore) NewActivity(user *fleet.User, activityType string, details *map[string]interface{}) error {
	return nil
}
