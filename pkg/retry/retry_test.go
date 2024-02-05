package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var errTest = errors.New("test error")

func TestRetryDo(t *testing.T) {
	t.Run("WithMaxAttempts only performs the operation the configured number of times", func(t *testing.T) {
		count := 0
		max := 3

		err := Do(func() error {
			count++
			return errTest
		}, WithMaxAttempts(max), WithInterval(1*time.Millisecond))

		require.ErrorIs(t, errTest, err)
		require.Equal(t, max, count)
	})

	t.Run("retry backoff should only allow a 5x increase on interval") func(t *testing.T) {
	  count := 0
	  max := 10
	  maxTime := time.Duration(1*time.Millisecond)
	  expectedMaxTime := time.Duration(5*time.Millisecond)

	  err := Do(func() error {
				count++
				if (time.Duration(count) * time.Millisecond) <= expectedMaxTime {
					maxTime = time.Duration(count) * time.Millisecond
				} else {
					maxTime = expectedMaxTime
				}
				if maxTime > expectedMaxTime {
					return errTest
				}
				return nil
	  }), WithMaxAttempts(max), WithBackoff(true), WithInterval(1*time.Millisecond))
	  require.NoError(t, err)
	  require.Equal(t, maxTime, expectedMaxTime)
	})

	t.Run("operations are run an unlimited number of times by default", func(t *testing.T) {
		count := 0
		max := 10

		err := Do(func() error {
			if count++; count != max {
				return errTest
			}
			return nil
		}, WithInterval(1*time.Millisecond))

		require.NoError(t, err)
		require.Equal(t, max, count)
	})
}
