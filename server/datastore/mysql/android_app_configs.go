package mysql

import (
	"context"

	// "github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// TODO(JK): decide if we want to use the appconfig struct or just validate then pass the raw data
// to be a litte more efficient
func (ds *Datastore) UpsertAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string, cfg fleet.AndroidAppConfig) error {
	return nil
}

func (ds *Datastore) DeleteAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) error {
	return nil
}

func (ds *Datastore) GetAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) (fleet.AndroidAppConfig, error) {
	return fleet.AndroidAppConfig{}, nil
}
