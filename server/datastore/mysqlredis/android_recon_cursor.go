package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// androidReconCursorKey is the Redis key holding the host_uuid cursor for
// the batched Android MDM profile reconciler.
const androidReconCursorKey = "mdm:android:recon_cursor"

// GetMDMAndroidReconcileCursor returns the persisted host_uuid cursor used by
// the batched Android MDM reconciliation cron to bound per-tick work.
//
// Returns "" if the key is unset (fresh deployment, Redis flushed, or full
// pass complete). Loss of this key is harmless: the cron resumes from the
// beginning. The desired-state diff is recomputed every tick, so re-
// processing converges naturally.
func (d *Datastore) GetMDMAndroidReconcileCursor(ctx context.Context) (string, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.String(conn.Do("GET", androidReconCursorKey))
	switch {
	case err == nil:
		return cursor, nil
	case errors.Is(err, redigo.ErrNil):
		return "", nil
	default:
		return "", ctxerr.Wrap(ctx, err, "get android MDM reconcile cursor")
	}
}

// SetMDMAndroidReconcileCursor persists the host_uuid cursor used by the
// batched Android MDM reconciliation cron. An empty string indicates a full
// pass has completed; the next tick will start from the beginning.
func (d *Datastore) SetMDMAndroidReconcileCursor(ctx context.Context, cursor string) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", androidReconCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set android MDM reconcile cursor")
	}
	return nil
}
