package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// windowsReconCursorKey is the Redis key holding the host_uuid cursor for
// the mdm_windows_profile_manager cron's batched reconciliation.
const windowsReconCursorKey = "mdm:windows:recon_cursor"

// GetMDMWindowsReconcileCursor returns the persisted host_uuid cursor used
// by the Windows MDM reconciliation cron to bound per-tick work.
//
// Returns "" if the key is unset (fresh deployment, Redis flushed, or full
// pass complete). Loss of this key is harmless: the cron resumes from the
// beginning, and the listing predicates filter out hosts whose state
// already matches the desired state, so re-processing converges quickly.
func (d *Datastore) GetMDMWindowsReconcileCursor(ctx context.Context) (string, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.String(conn.Do("GET", windowsReconCursorKey))
	switch {
	case err == nil:
		return cursor, nil
	case errors.Is(err, redigo.ErrNil):
		return "", nil
	default:
		return "", ctxerr.Wrap(ctx, err, "get windows MDM reconcile cursor")
	}
}

// SetMDMWindowsReconcileCursor persists the host_uuid cursor used by the
// Windows MDM reconciliation cron. An empty string indicates a full pass
// has completed; the next tick will start from the beginning.
//
// No TTL: the cron writes this on every tick, so eviction by Redis would
// only delay the next tick's read by the time it takes the cron to fire
// again, and the worst-case effect is one redundant pass.
func (d *Datastore) SetMDMWindowsReconcileCursor(ctx context.Context, cursor string) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", windowsReconCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set windows MDM reconcile cursor")
	}
	return nil
}
