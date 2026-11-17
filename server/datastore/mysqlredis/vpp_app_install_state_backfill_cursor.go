package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// vppAppInstallStateBackfillCursorKey is the Redis key holding the cursor of the cron that rebuilds
// host_vpp_app_installs.
const vppAppInstallStateBackfillCursorKey = "vpp:app_install_state_backfill_cursor"

// GetVPPAppInstallStateBackfillCursor returns the persisted cursor, or "" when none is stored.
//
// Loss of this key is harmless: the cron restarts the walk, and refreshing a key is idempotent.
func (d *Datastore) GetVPPAppInstallStateBackfillCursor(ctx context.Context) (string, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.String(conn.Do("GET", vppAppInstallStateBackfillCursorKey))
	switch {
	case err == nil:
		return cursor, nil
	case errors.Is(err, redigo.ErrNil):
		return "", nil
	default:
		return "", ctxerr.Wrap(ctx, err, "get vpp app install state backfill cursor")
	}
}

// SetVPPAppInstallStateBackfillCursor persists the cursor. An empty string means a full pass completed.
func (d *Datastore) SetVPPAppInstallStateBackfillCursor(ctx context.Context, cursor string) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", vppAppInstallStateBackfillCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set vpp app install state backfill cursor")
	}
	return nil
}
