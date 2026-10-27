package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// androidAppReconCursorKey is the Redis key holding the host id cursor for the Android app reconciler.
const androidAppReconCursorKey = "android:apps:recon_cursor"

func (d *Datastore) GetAndroidAppReconcileCursor(ctx context.Context) (uint, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.Uint64(conn.Do("GET", androidAppReconCursorKey))
	switch {
	case err == nil:
		return uint(cursor), nil
	case errors.Is(err, redigo.ErrNil):
		return 0, nil
	default:
		return 0, ctxerr.Wrap(ctx, err, "get android app reconcile cursor")
	}
}

func (d *Datastore) SetAndroidAppReconcileCursor(ctx context.Context, cursor uint) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	_, err := conn.Do("SET", androidAppReconCursorKey, cursor)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set android app reconcile cursor")
	}
	return nil
}
