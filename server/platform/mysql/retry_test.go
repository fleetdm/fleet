package mysql

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-kit/log"
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
	SetFatalErrorHandler(func(err error) {
		called.Store(true)
		capturedErr.Store(err)
	})
	t.Cleanup(func() { SetFatalErrorHandler(nil) })

	testErr := errors.New("test read-only error")
	triggerFatalError(testErr)

	assert.True(t, called.Load())
	assert.Equal(t, testErr, capturedErr.Load())
}

func TestTriggerFatalErrorPanicsWithoutHandler(t *testing.T) {
	SetFatalErrorHandler(nil)

	assert.Panics(t, func() {
		triggerFatalError(errors.New("read-only"))
	})
}

func TestTriggerFatalErrorFiresOnce(t *testing.T) {
	var callCount atomic.Int32
	SetFatalErrorHandler(func(_ error) {
		callCount.Add(1)
	})
	t.Cleanup(func() { SetFatalErrorHandler(nil) })

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			triggerFatalError(errors.New("read-only"))
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
				}, log.NewNopLogger())
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
				}, log.NewNopLogger())
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
				}, log.NewNopLogger())
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
				}, log.NewNopLogger())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var handlerCalled atomic.Bool
			SetFatalErrorHandler(func(_ error) {
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
