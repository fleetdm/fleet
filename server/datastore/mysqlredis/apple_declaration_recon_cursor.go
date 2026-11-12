package mysqlredis

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/redis"
	redigo "github.com/gomodule/redigo/redis"
)

// appleDeclarationReconCursorKey is the Redis key holding the host_uuid
// cursor for the batched DDM reconciler. Kept separate from the profile
// cursor so the two passes advance independently.
const appleDeclarationReconCursorKey = "mdm:apple:declaration_recon_cursor"

func (d *Datastore) GetMDMAppleDeclarationReconcileCursor(ctx context.Context) (string, error) {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	cursor, err := redigo.String(conn.Do("GET", appleDeclarationReconCursorKey))
	switch {
	case err == nil:
		return cursor, nil
	case errors.Is(err, redigo.ErrNil):
		return "", nil
	default:
		return "", ctxerr.Wrap(ctx, err, "get apple MDM declaration reconcile cursor")
	}
}

func (d *Datastore) SetMDMAppleDeclarationReconcileCursor(ctx context.Context, cursor string) error {
	conn := redis.ConfigureDoer(d.pool, d.pool.Get())
	defer conn.Close()

	if _, err := conn.Do("SET", appleDeclarationReconCursorKey, cursor); err != nil {
		return ctxerr.Wrap(ctx, err, "set apple MDM declaration reconcile cursor")
	}
	return nil
}
