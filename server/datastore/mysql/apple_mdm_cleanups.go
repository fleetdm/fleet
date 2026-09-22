package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// GetMDMAppleCommandCleanupState returns the Apple MDM command cleanup cron's
// persisted cursors. The bare mysql.Datastore has no place to persist them,
// so this returns nil (every scan starts from the oldest rows). The
// mysqlredis wrapper overrides this to back it with Redis.
func (ds *Datastore) GetMDMAppleCommandCleanupState(_ context.Context) (*fleet.MDMAppleCommandCleanupState, error) {
	return nil, nil
}

// SetMDMAppleCommandCleanupState persists the Apple MDM command cleanup cron's
// cursors. The bare mysql.Datastore is a no-op; the mysqlredis wrapper backs
// it with Redis.
func (ds *Datastore) SetMDMAppleCommandCleanupState(_ context.Context, _ *fleet.MDMAppleCommandCleanupState) error {
	return nil
}
