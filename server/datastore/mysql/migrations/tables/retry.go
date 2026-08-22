package tables

import (
	"errors"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-sql-driver/mysql"
)

// These bound how long a single statement will keep re-executing on
// ER_NEED_REPREPARE. A reprepare normally succeeds on the first retry, so they
// are deliberately much tighter than the one minute
// platform/mysql.WithRetryTxx allows for lock contention: a migration holds its
// transaction open for as long as it runs, and an upgrade that hangs is worse
// than one that reports a clear error.
//
// can override in tests
var (
	reprepareMaxRetries      uint64 = 5
	reprepareMaxElapsed             = 30 * time.Second
	reprepareInitialInterval        = backoff.DefaultInitialInterval
)

// isNeedReprepare reports whether err is, or wraps, MySQL's ER_NEED_REPREPARE
// (1615, "Prepared statement needs to be re-prepared").
func isNeedReprepare(err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == mysqlerr.ER_NEED_REPREPARE
}

// retryOnNeedReprepare re-runs fn, with exponential backoff, for as long as it
// returns ER_NEED_REPREPARE. Any other error, including nil, is returned as-is
// on the first attempt.
//
// A migration runs inside one transaction that goose owns, so it cannot use
// platform/mysql.WithRetryTxx: that helper retries by rolling the transaction
// back and beginning a new one, which from inside a migration would throw away
// the work the migration has already done. The errors WithRetryTxx covers need
// exactly that treatment -- ER_LOCK_DEADLOCK, for one, is only ever delivered
// to the client after InnoDB has already rolled the transaction back, so there
// is nothing left to retry in place.
//
// ER_NEED_REPREPARE is the exception, and it is why this helper is narrow
// rather than a second copy of RetryableError. It means the table metadata a
// prepared statement was built against changed and the server used up its
// automatic reprepare attempts; the statement never ran, and the transaction is
// still open and usable. MySQL documents it as the client's job to re-execute.
// That makes it the one error a migration can retry where it stands.
//
// Callers must therefore keep fn a single statement, or at least idempotent
// within the transaction: it can run more than once.
func retryOnNeedReprepare(fn func() error) error {
	operation := func() error {
		err := fn()
		if err != nil && !isNeedReprepare(err) {
			return backoff.Permanent(err)
		}
		return err
	}

	expBo := backoff.NewExponentialBackOff()
	expBo.InitialInterval = reprepareInitialInterval
	expBo.MaxElapsedTime = reprepareMaxElapsed
	return backoff.Retry(operation, backoff.WithMaxRetries(expBo, reprepareMaxRetries))
}
