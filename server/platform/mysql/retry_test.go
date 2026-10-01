package mysql

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/VividCortex/mysqlerr"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	gmysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readOnlyErr returns a MySQL error that simulates a read-only database (error 1792).
func readOnlyErr() error {
	return &gmysql.MySQLError{Number: 1792, Message: "Cannot execute statement in a READ ONLY transaction."}
}

func TestTriggerFatalErrorCallsHandler(t *testing.T) {
	var called atomic.Bool
	var capturedErr atomic.Value
	SetFatalErrorHandler(func(_ context.Context, err error) {
		called.Store(true)
		capturedErr.Store(err)
	})
	t.Cleanup(func() { SetFatalErrorHandler(nil) })

	testErr := errors.New("test read-only error")
	TriggerFatalError(t.Context(), testErr)

	assert.True(t, called.Load())
	assert.Equal(t, testErr, capturedErr.Load())
}

func TestTriggerFatalErrorPanicsWithoutHandler(t *testing.T) {
	SetFatalErrorHandler(nil)

	assert.Panics(t, func() {
		TriggerFatalError(t.Context(), errors.New("read-only"))
	})
}

func TestTriggerFatalErrorFiresOnce(t *testing.T) {
	var callCount atomic.Int32
	SetFatalErrorHandler(func(_ context.Context, _ error) {
		callCount.Add(1)
	})
	t.Cleanup(func() { SetFatalErrorHandler(nil) })

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			TriggerFatalError(t.Context(), errors.New("read-only"))
		})
	}
	wg.Wait()

	assert.Equal(t, int32(1), callCount.Load())
}

func TestTransactionReadOnlyTriggersFatalError(t *testing.T) {
	cases := []struct {
		name      string
		txFunc    func(ctx *testing.T, db *sqlx.DB, mock sqlmock.Sqlmock) error
		setupMock func(mock sqlmock.Sqlmock)
	}{
		{
			name: "WithRetryTxx read-only from fn",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			txFunc: func(ctx *testing.T, db *sqlx.DB, mock sqlmock.Sqlmock) error {
				return WithRetryTxx(ctx.Context(), db, func(tx sqlx.ExtContext) error {
					return readOnlyErr()
				}, slog.New(slog.DiscardHandler))
			},
		},
		{
			name: "WithRetryTxx read-only from commit",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(readOnlyErr())
			},
			txFunc: func(ctx *testing.T, db *sqlx.DB, mock sqlmock.Sqlmock) error {
				return WithRetryTxx(ctx.Context(), db, func(tx sqlx.ExtContext) error {
					return nil
				}, slog.New(slog.DiscardHandler))
			},
		},
		{
			name: "WithTxx read-only from fn",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectRollback()
			},
			txFunc: func(ctx *testing.T, db *sqlx.DB, mock sqlmock.Sqlmock) error {
				return WithTxx(ctx.Context(), db, func(tx sqlx.ExtContext) error {
					return readOnlyErr()
				}, slog.New(slog.DiscardHandler))
			},
		},
		{
			name: "WithTxx read-only from commit",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit().WillReturnError(readOnlyErr())
			},
			txFunc: func(ctx *testing.T, db *sqlx.DB, mock sqlmock.Sqlmock) error {
				return WithTxx(ctx.Context(), db, func(tx sqlx.ExtContext) error {
					return nil
				}, slog.New(slog.DiscardHandler))
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var handlerCalled atomic.Bool
			SetFatalErrorHandler(func(_ context.Context, _ error) {
				handlerCalled.Store(true)
			})
			t.Cleanup(func() { SetFatalErrorHandler(nil) })

			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()
			sqlxDB := sqlx.NewDb(db, "sqlmock")

			tc.setupMock(mock)

			err = tc.txFunc(t, sqlxDB, mock)

			require.Error(t, err)
			assert.True(t, IsReadOnlyError(err))
			assert.True(t, handlerCalled.Load())
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// TestWithRetryTxxOnCommitHooks pins the contract the deferred APNs push in
// nanoEnqueueVPPInstall depends on: a hook registered from inside the
// transaction runs only once the commit has succeeded, so nothing outside the
// transaction is ever notified about writes that never landed.
func TestWithRetryTxxOnCommitHooks(t *testing.T) {
	discard := slog.New(slog.DiscardHandler)

	newDB := func(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() { db.Close() })
		return sqlx.NewDb(db, "sqlmock"), mock
	}

	t.Run("run after commit in registration order", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO t").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		var order []string
		err := WithRetryTxx(t.Context(), db, func(tx sqlx.ExtContext) error {
			wrapped, ok := tx.(WrappedExtContext)
			require.True(t, ok, "the tx handed to the fn must accept on-commit hooks")

			if _, err := tx.ExecContext(t.Context(), "INSERT INTO t VALUES (1)"); err != nil {
				return err
			}
			wrapped.AddOnCommitHook(func() {
				// Every expectation, COMMIT included, is already satisfied, so
				// the hook cannot be observing an in-flight transaction.
				require.NoError(t, mock.ExpectationsWereMet())
				order = append(order, "first")
			})
			wrapped.AddOnCommitHook(func() {
				order = append(order, "second")
			})
			order = append(order, "fn")
			return nil
		}, discard)

		require.NoError(t, err)
		assert.Equal(t, []string{"fn", "first", "second"}, order)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("do not run when the fn fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()

		fnErr := errors.New("boom")
		var ran bool
		err := WithRetryTxx(t.Context(), db, func(tx sqlx.ExtContext) error {
			tx.(WrappedExtContext).AddOnCommitHook(func() { ran = true })
			return fnErr
		}, discard)

		require.ErrorIs(t, err, fnErr)
		assert.False(t, ran, "hook ran for a rolled back transaction")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("do not run when the commit fails", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		var ran bool
		err := WithRetryTxx(t.Context(), db, func(tx sqlx.ExtContext) error {
			tx.(WrappedExtContext).AddOnCommitHook(func() { ran = true })
			return nil
		}, discard)

		require.Error(t, err)
		assert.False(t, ran, "hook ran for a transaction that failed to commit")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("do not carry over hooks from a retried attempt", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectRollback()
		mock.ExpectBegin()
		mock.ExpectCommit()

		var attempts, hookRuns int
		err := WithRetryTxx(t.Context(), db, func(tx sqlx.ExtContext) error {
			attempts++
			tx.(WrappedExtContext).AddOnCommitHook(func() { hookRuns++ })
			if attempts == 1 {
				return &gmysql.MySQLError{Number: mysqlerr.ER_LOCK_DEADLOCK, Message: "Deadlock found"}
			}
			return nil
		}, discard)

		require.NoError(t, err)
		require.Equal(t, 2, attempts)
		assert.Equal(t, 1, hookRuns, "a hook registered by the rolled back attempt ran anyway")
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("withTxx should always have WrappedExtContext", func(t *testing.T) {
		db, mock := newDB(t)
		mock.ExpectBegin()
		mock.ExpectCommit()

		err := WithRetryTxx(t.Context(), db, func(tx sqlx.ExtContext) error {
			_, ok := tx.(WrappedExtContext)
			require.True(t, ok, "tx is not a WrappedExtContext")
			return nil
		}, discard)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestRetryableError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		retryable bool
	}{
		{"deadlock", &gmysql.MySQLError{Number: mysqlerr.ER_LOCK_DEADLOCK}, true},
		{"lock wait timeout", &gmysql.MySQLError{Number: mysqlerr.ER_LOCK_WAIT_TIMEOUT}, true},
		{"needs reprepare", &gmysql.MySQLError{Number: mysqlerr.ER_NEED_REPREPARE}, true},
		{"needs reprepare wrapped by the datastore", ctxerr.Wrap(context.Background(), &gmysql.MySQLError{Number: mysqlerr.ER_NEED_REPREPARE}, "activate next activity"), true},
		{"read only", readOnlyErr(), false},
		{"duplicate entry", &gmysql.MySQLError{Number: mysqlerr.ER_DUP_ENTRY}, false},
		{"not a mysql error", errors.New("some other failure"), false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.retryable, RetryableError(c.err))
		})
	}
}
