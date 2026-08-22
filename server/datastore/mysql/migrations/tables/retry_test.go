package tables

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func needReprepare() error {
	return &mysql.MySQLError{Number: mysqlerr.ER_NEED_REPREPARE, Message: "Prepared statement needs to be re-prepared"}
}

// shrinkReprepareBackoff keeps the retry tests to real backoff arithmetic
// without the wall-clock waits, the same way progressInterval is overridden for
// the migration-step tests.
func shrinkReprepareBackoff(t *testing.T) {
	t.Helper()
	original := reprepareInitialInterval
	reprepareInitialInterval = time.Millisecond
	t.Cleanup(func() { reprepareInitialInterval = original })
}

func TestRetryOnNeedReprepare(t *testing.T) {
	shrinkReprepareBackoff(t)

	t.Run("success runs once", func(t *testing.T) {
		calls := 0
		require.NoError(t, retryOnNeedReprepare(func() error {
			calls++
			return nil
		}))
		require.Equal(t, 1, calls)
	})

	t.Run("reprepare then success", func(t *testing.T) {
		calls := 0
		require.NoError(t, retryOnNeedReprepare(func() error {
			calls++
			if calls == 1 {
				return needReprepare()
			}
			return nil
		}))
		require.Equal(t, 2, calls)
	})

	t.Run("wrapped reprepare is still recognised", func(t *testing.T) {
		calls := 0
		require.NoError(t, retryOnNeedReprepare(func() error {
			calls++
			if calls == 1 {
				return fmt.Errorf("storing compressed response id 7: %w", needReprepare())
			}
			return nil
		}))
		require.Equal(t, 2, calls)
	})

	t.Run("persistent reprepare gives up and returns the error", func(t *testing.T) {
		calls := 0
		err := retryOnNeedReprepare(func() error {
			calls++
			return needReprepare()
		})
		require.Error(t, err)
		require.True(t, isNeedReprepare(err), "the caller must still see the MySQL error, not a backoff wrapper")
		// One initial attempt plus reprepareMaxRetries retries, and no more:
		// a migration that hangs is worse than one that reports a clear error.
		require.Equal(t, int(1+reprepareMaxRetries), calls)
	})

	// The negative control for the narrow scope. Deadlock is retryable for
	// platform/mysql.WithRetryTxx, which restarts the whole transaction. Inside a
	// migration the transaction is already gone by the time the client sees it,
	// so retrying in place would only fail again, more slowly.
	t.Run("other mysql errors are not retried", func(t *testing.T) {
		for _, number := range []uint16{mysqlerr.ER_LOCK_DEADLOCK, mysqlerr.ER_LOCK_WAIT_TIMEOUT, mysqlerr.ER_DUP_ENTRY} {
			calls := 0
			err := retryOnNeedReprepare(func() error {
				calls++
				return &mysql.MySQLError{Number: number}
			})
			require.Error(t, err)
			require.Equal(t, 1, calls, "error %d must not be retried in place", number)
		}
	})

	t.Run("non-mysql errors are not retried", func(t *testing.T) {
		sentinel := errors.New("context canceled")
		calls := 0
		err := retryOnNeedReprepare(func() error {
			calls++
			return sentinel
		})
		require.ErrorIs(t, err, sentinel)
		require.Equal(t, 1, calls)
	})
}
