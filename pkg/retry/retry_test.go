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

	t.Run("with backoff", func(t *testing.T) {
		count := 0
		max := 4
		start := time.Now()
		err := Do(func() error {
			switch count {
			case 0:
				require.WithinDuration(t, start, time.Now(), 1*time.Millisecond)
			case 1:
				require.WithinDuration(t, start.Add(50*time.Millisecond), time.Now(), 10*time.Millisecond)
			case 2:
				require.WithinDuration(t, start.Add((50+100)*time.Millisecond), time.Now(), 10*time.Millisecond)
			case 3:
				require.WithinDuration(t, start.Add((50+100+200)*time.Millisecond), time.Now(), 10*time.Millisecond)
			}
			count++
			if count != max {
				return errTest
			}
			return nil
		},
			WithInterval(50*time.Millisecond),
			WithBackoffMultiplier(2),
			WithMaxAttempts(4),
		)

		require.NoError(t, err)
		require.Equal(t, max, count)
	})
}
