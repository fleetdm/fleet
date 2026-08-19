package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// appleReconCursorKey is the Redis key holding the host_uuid cursor for the
// batched Apple MDM profile reconciler.
const appleReconCursorKey = "mdm:apple:recon_cursor"

// GetMDMAppleReconcileCursor returns the persisted host_uuid cursor used by
// the batched Apple MDM reconciliation cron to bound per-tick work.
//
// Returns "" if the key is unset (fresh deployment, Redis flushed, or full
// pass complete). Loss of this key is harmless: the cron resumes from the
// beginning. The desired-state diff is recomputed every tick, so re-
// processing converges naturally.
func (d *Datastore) GetMDMAppleReconcileCursor(ctx context.Context) (string, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.String(conn.Do("GET", appleReconCursorKey))
	switch {
	case err == nil:
		return cursor, nil
	case errors.Is(err, redigo.ErrNil):
		return "", nil
	default:
		return "", ctxerr.Wrap(ctx, err, "get apple MDM reconcile cursor")
	}
}

// SetMDMAppleReconcileCursor persists the host_uuid cursor used by the
// batched Apple MDM reconciliation cron. An empty string indicates a full
// pass has completed; the next tick will start from the beginning.
func (d *Datastore) SetMDMAppleReconcileCursor(ctx context.Context, cursor string) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", appleReconCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set apple MDM reconcile cursor")
	}
	return nil
}
