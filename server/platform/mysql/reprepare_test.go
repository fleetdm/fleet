package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/VividCortex/mysqlerr"
	gomysql "github.com/go-sql-driver/mysql"
	"github.com/ngrok/sqlmw"
	"github.com/stretchr/testify/require"
)

// fakeStmt returns one queued error per call, nil meaning success.
type fakeStmt struct {
	errs  []error
	calls int
}

func (f *fakeStmt) next(t *testing.T) error {
	t.Helper()
	f.calls++
	require.LessOrEqual(t, f.calls, len(f.errs), "called more times than the case provides errors for")
	return f.errs[f.calls-1]
}

func TestReprepareRetryInterceptor(t *testing.T) {
	needReprepare := &gomysql.MySQLError{Number: mysqlerr.ER_NEED_REPREPARE, Message: "Prepared statement needs to be re-prepared"}
	duplicate := &gomysql.MySQLError{Number: mysqlerr.ER_DUP_ENTRY, Message: "Duplicate entry"}
	eight := []error{
		needReprepare, needReprepare, needReprepare, needReprepare,
		needReprepare, needReprepare, needReprepare, needReprepare,
	}

	for _, tc := range []struct {
		name      string
		errs      []error
		wantCalls int
		wantErr   error
	}{
		{"passes a successful statement straight through", []error{nil}, 1, nil},
		{"retries a single 1615", []error{needReprepare, nil}, 2, nil},
		{"retries a wrapped 1615", []error{fmt.Errorf("exec: %w", needReprepare), nil}, 2, nil},
		{"gives up after the attempt limit", eight, 8, needReprepare},
		{"does not retry another mysql error", []error{duplicate}, 1, duplicate},
		{"does not retry a non-mysql error", []error{io.ErrUnexpectedEOF}, 1, io.ErrUnexpectedEOF},
		{"passes driver.ErrSkip through so database/sql can fall back", []error{driver.ErrSkip}, 1, driver.ErrSkip},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var in ReprepareRetryInterceptor

			t.Run("exec", func(t *testing.T) {
				stmt := &fakeStmt{errs: tc.errs}
				_, err := in.StmtExecContext(t.Context(), stmtExecFunc(func() error { return stmt.next(t) }), "", nil)
				requireErr(t, err, tc.wantErr)
				require.Equal(t, tc.wantCalls, stmt.calls)
			})

			t.Run("query", func(t *testing.T) {
				stmt := &fakeStmt{errs: tc.errs}
				_, err := in.StmtQueryContext(t.Context(), stmtQueryFunc(func() error { return stmt.next(t) }), "", nil)
				requireErr(t, err, tc.wantErr)
				require.Equal(t, tc.wantCalls, stmt.calls)
			})

			t.Run("prepare", func(t *testing.T) {
				stmt := &fakeStmt{errs: tc.errs}
				_, err := in.ConnPrepareContext(t.Context(), connPrepareFunc(func() error { return stmt.next(t) }), "")
				requireErr(t, err, tc.wantErr)
				require.Equal(t, tc.wantCalls, stmt.calls)
			})
		})
	}
}

func requireErr(t *testing.T, got, want error) {
	t.Helper()
	if want == nil {
		require.NoError(t, got)
		return
	}
	require.ErrorIs(t, got, want)
}

// minimal driver shims, each delegating to a func that yields the next error

type stmtExecFunc func() error

func (f stmtExecFunc) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	return nil, f()
}

type stmtQueryFunc func() error

func (f stmtQueryFunc) QueryContext(context.Context, []driver.NamedValue) (driver.Rows, error) {
	return nil, f()
}

type connPrepareFunc func() error

func (f connPrepareFunc) PrepareContext(context.Context, string) (driver.Stmt, error) {
	return nil, f()
}

// TestReprepareRetryThroughDatabaseSQL drives the interceptor through
// database/sql and sqlmw with a fake driver, proving the full stack retries.
func TestReprepareRetryThroughDatabaseSQL(t *testing.T) {
	registerStackTestDriverOnce.Do(func() {
		sql.Register(stackTestDriverName, sqlmw.Driver(stackTestDriver, ReprepareRetryInterceptor{}))
	})
	db, err := sql.Open(stackTestDriverName, "unused-dsn")
	require.NoError(t, err)
	defer db.Close()

	stackTestDriver.execCalls = 0
	res, err := db.Exec("UPDATE t SET v = ? WHERE id = ?", 1, 2)
	require.NoError(t, err)
	n, err := res.RowsAffected()
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.Equal(t, 2, stackTestDriver.execCalls, "one 1615, then one successful retry")
}

const stackTestDriverName = "mysql-reprepare-stack-test"

var (
	stackTestDriver             = &fakeDriver{}
	registerStackTestDriverOnce sync.Once
)

// fakeDriver fails the first execute with a 1615, then succeeds.
type fakeDriver struct{ execCalls int }

func (d *fakeDriver) Open(string) (driver.Conn, error) { return &fakeConn{d: d}, nil }

type fakeConn struct{ d *fakeDriver }

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return &fakeExecStmt{d: c.d}, nil }
func (c *fakeConn) PrepareContext(_ context.Context, q string) (driver.Stmt, error) {
	return c.Prepare(q)
}
func (c *fakeConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	// mirror go-sql-driver with interpolateParams=false: decline the direct
	// path so database/sql falls back to prepare+exec, the flow under test
	return nil, driver.ErrSkip
}
func (c *fakeConn) Close() error              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error) { return nil, io.EOF }

type fakeExecStmt struct{ d *fakeDriver }

func (s *fakeExecStmt) Close() error  { return nil }
func (s *fakeExecStmt) NumInput() int { return -1 }
func (s *fakeExecStmt) Exec([]driver.Value) (driver.Result, error) {
	return nil, io.EOF // context variant below is the one in play
}
func (s *fakeExecStmt) ExecContext(context.Context, []driver.NamedValue) (driver.Result, error) {
	s.d.execCalls++
	if s.d.execCalls == 1 {
		return nil, &gomysql.MySQLError{Number: mysqlerr.ER_NEED_REPREPARE}
	}
	return driver.RowsAffected(1), nil
}
func (s *fakeExecStmt) Query([]driver.Value) (driver.Rows, error) { return nil, io.EOF }
