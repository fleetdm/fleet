package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetABMTokenDefault(ctx context.Context, tokenID uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var found bool
		if err := sqlx.GetContext(ctx, tx, &found, "SELECT EXISTS(SELECT 1 FROM abm_tokens WHERE id = ?)", tokenID); err != nil {
			return ctxerr.Wrap(ctx, err, "checking if abm token exists")
		}
		if !found {
			return notFound("ABMToken").WithID(tokenID)
		}

		_, err := tx.ExecContext(ctx, "UPDATE abm_tokens SET is_default = IF(id = ?, 1, 0)", tokenID)
		return ctxerr.Wrap(ctx, err, "setting abm token default")
	})
}

func (ds *Datastore) ClearABMTokenDefault(ctx context.Context) error {
	var count int
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &count, "SELECT COUNT(*) FROM abm_tokens"); err != nil {
		return ctxerr.Wrap(ctx, err, "counting abm tokens")
	}

	if count < 2 {
		// no-op for no or 1 token, as a sole token is always the default
		return nil
	}

	_, err := ds.writer(ctx).ExecContext(ctx, "UPDATE abm_tokens SET is_default = 0")
	return ctxerr.Wrap(ctx, err, "clearing abm token default")
}

func (ds *Datastore) SetABMTokenServerUUID(ctx context.Context, tokenID uint, serverUUID string) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var found bool
		if err := sqlx.GetContext(ctx, tx, &found, "SELECT EXISTS(SELECT 1 FROM abm_tokens WHERE id = ?)", tokenID); err != nil {
			return ctxerr.Wrap(ctx, err, "checking if abm token exists")
		}
		if !found {
			return notFound("ABMToken").WithID(tokenID)
		}

		_, err := tx.ExecContext(ctx, "UPDATE abm_tokens SET server_uuid = NULLIF(?, '') WHERE id = ?", serverUUID, tokenID)
		return ctxerr.Wrap(ctx, err, "setting abm token server UUID")
	})
}
