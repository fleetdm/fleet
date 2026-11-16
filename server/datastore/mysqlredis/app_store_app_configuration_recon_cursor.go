package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// appStoreAppConfigurationReconCursorKey is the Redis key holding the host id cursor for the App Store app configuration
// reconciler.
const appStoreAppConfigurationReconCursorKey = "vpp:app_configuration:recon_cursor"

func (d *Datastore) GetAppStoreAppConfigurationReconcileCursor(ctx context.Context) (uint, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.Uint64(conn.Do("GET", appStoreAppConfigurationReconCursorKey))
	switch {
	case err == nil:
		return uint(cursor), nil
	case errors.Is(err, redigo.ErrNil):
		return 0, nil
	default:
		return 0, ctxerr.Wrap(ctx, err, "get app store app configuration reconcile cursor")
	}
}

func (d *Datastore) SetAppStoreAppConfigurationReconcileCursor(ctx context.Context, cursor uint) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", appStoreAppConfigurationReconCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set app store app configuration reconcile cursor")
	}
	return nil
}
