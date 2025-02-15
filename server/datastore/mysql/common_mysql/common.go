package common_mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

func WithTxx(ctx context.Context, db *sqlx.DB, fn TxFn, logger log.Logger) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				logger.Log("err", err, "msg", "error encountered during transaction panic rollback")
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil && rbErr != sql.ErrTxDone {
			return ctxerr.Wrapf(ctx, err, "got err '%s' rolling back after err", rbErr.Error())
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ctxerr.Wrap(ctx, err, "commit transaction")
	}

	return nil
}
