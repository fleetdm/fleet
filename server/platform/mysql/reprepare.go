package mysql

import (
	"context"
	"database/sql/driver"
	"errors"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/ngrok/sqlmw"
)

// re-preparing clears a 1615 on the next call, so attempts aren't spaced out
const reprepareMaxAttempts = 8

// ReprepareRetryInterceptor retries statements the server rejects with
// ER_NEED_REPREPARE (1615), raised when a concurrent DDL, FLUSH TABLES, or
// table-cache eviction invalidates a prepared statement between PREPARE and
// EXECUTE. A 1615 statement never ran, so retrying the same handle is safe;
// the server re-prepares on execute. No-arg statements have nothing to go stale.
type ReprepareRetryInterceptor struct {
	sqlmw.NullInterceptor
}

func (ReprepareRetryInterceptor) ConnPrepareContext(ctx context.Context, conn driver.ConnPrepareContext, query string) (driver.Stmt, error) {
	return retryOnNeedReprepare(func() (driver.Stmt, error) {
		return conn.PrepareContext(ctx, query)
	})
}

func (ReprepareRetryInterceptor) StmtExecContext(ctx context.Context, stmt driver.StmtExecContext, _ string, args []driver.NamedValue) (driver.Result, error) {
	return retryOnNeedReprepare(func() (driver.Result, error) {
		return stmt.ExecContext(ctx, args)
	})
}

func (ReprepareRetryInterceptor) StmtQueryContext(ctx context.Context, stmt driver.StmtQueryContext, _ string, args []driver.NamedValue) (driver.Rows, error) {
	return retryOnNeedReprepare(func() (driver.Rows, error) {
		return stmt.QueryContext(ctx, args)
	})
}

func retryOnNeedReprepare[T any](fn func() (T, error)) (T, error) {
	var (
		v   T
		err error
	)
	for range reprepareMaxAttempts {
		v, err = fn()
		if !IsNeedReprepare(err) {
			return v, err
		}
	}
	return v, err
}

// IsNeedReprepare reports whether err is MySQL's ER_NEED_REPREPARE (1615).
func IsNeedReprepare(err error) bool {
	mysqlErr, ok := errors.AsType[*mysql.MySQLError](err)
	return ok && mysqlErr.Number == mysqlerr.ER_NEED_REPREPARE
}
