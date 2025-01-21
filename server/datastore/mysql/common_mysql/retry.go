package common_mysql

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var DoRetryErr = errors.New("fleet datastore retry")

type TxFn func(tx sqlx.ExtContext) error

// WithRetryTxx provides a common way to commit/rollback a txFn wrapped in a retry with exponential backoff
func WithRetryTxx(ctx context.Context, db *sqlx.DB, fn TxFn, logger log.Logger) error {
	operation := func() error {
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
				// Consider rollback errors to be non-retryable
				return backoff.Permanent(ctxerr.Wrapf(ctx, err, "got err '%s' rolling back after err", rbErr.Error()))
			}

			if retryableError(err) {
				return err
			}

			// Consider any other errors to be non-retryable
			return backoff.Permanent(err)
		}

		if err := tx.Commit(); err != nil {
			err = ctxerr.Wrap(ctx, err, "commit transaction")

			if retryableError(err) {
				return err
			}

			return backoff.Permanent(err)
		}

		return nil
	}

	expBo := backoff.NewExponentialBackOff()
	// MySQL innodb_lock_wait_timeout default is 50 seconds, so transaction can be waiting for a lock for several seconds.
	// Setting a higher MaxElapsedTime to increase probability that transaction will be retried.
	// This will reduce the number of retryable 'Deadlock found' errors. However, with a loaded DB, we will still see
	// 'Context cancelled' errors when the server drops long-lasting connections.
	expBo.MaxElapsedTime = 1 * time.Minute
	expBo.InitialInterval = 2 * time.Second
	bo := backoff.WithMaxRetries(expBo, 5)
	return backoff.Retry(operation, bo)
}

// retryableError determines whether a MySQL error can be retried. By default
// errors are considered non-retryable. Only errors that we know have a
// possibility of succeeding on a retry should return true in this function.
func retryableError(err error) bool {
	base := ctxerr.Cause(err)
	if b, ok := base.(*mysql.MySQLError); ok {
		switch b.Number {
		// Consider lock related errors to be retryable
		case mysqlerr.ER_LOCK_DEADLOCK, mysqlerr.ER_LOCK_WAIT_TIMEOUT:
			return true
		}
	}
	if errors.Is(err, DoRetryErr) {
		return true
	}

	return false
}
