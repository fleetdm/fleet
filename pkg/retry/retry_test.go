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
		maxAttempts := 3

		err := Do(func() error {
			count++
			return errTest
		}, WithMaxAttempts(maxAttempts), WithInterval(1*time.Millisecond))

		require.ErrorIs(t, errTest, err)
		require.Equal(t, maxAttempts, count)
	})

	t.Run("operations are run an unlimited number of times by default", func(t *testing.T) {
		count := 0
		maxAttempts := 10

		err := Do(func() error {
			if count++; count != maxAttempts {
				return errTest
			}
			return nil
		}, WithInterval(1*time.Millisecond))

		require.NoError(t, err)
		require.Equal(t, maxAttempts, count)
	})

	t.Run("with backoff", func(t *testing.T) {
		count := 0
		maxAttempts := 4
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
			if count != maxAttempts {
				return errTest
			}
			return nil
		},
			WithInterval(50*time.Millisecond),
			WithBackoffMultiplier(2),
			WithMaxAttempts(4),
		)

		require.NoError(t, err)
		require.Equal(t, maxAttempts, count)
	})

	t.Run("with error filter (test ignore)", func(t *testing.T) {
		count := 0
		err := Do(func() error {
			count++
			if count == 1 {
				return errors.New("normal")
			}
			if count == 2 {
				return errors.New("reset")
			}
			if count == 3 {
				return errors.New("ignore")
			}
			return nil
		},
			WithInterval(50*time.Millisecond),
			// We should actually run 3 times, but since one
			// of the errors causes a reset, we set max attempts to 2
			// to ensure that the reset logic is exercised.
			WithMaxAttempts(2),
			WithErrorFilter(func(err error) ErrorOutcome {
				if err.Error() == "normal" {
					return ErrorOutcomeNormalRetry
				}
				if err.Error() == "reset" {
					return ErrorOutcomeResetAttempts
				}
				if err.Error() == "ignore" {
					return ErrorOutcomeIgnore
				}
				return ErrorOutcomeDoNotRetry
			}),
		)

		require.NoError(t, err)
		require.Equal(t, 3, count)
	})

	t.Run("with error filter (test noretry)", func(t *testing.T) {
		count := 0
		err := Do(func() error {
			count++
			if count == 1 {
				return errors.New("normal")
			}
			if count == 2 {
				return errors.New("reset")
			}
			if count == 3 {
				return errors.New("stop")
			}
			return nil
		},
			WithInterval(50*time.Millisecond),
			// We should only actually run 3 times, setting this to 10
			// tests that the DoNotRetry logic is exercised.
			WithMaxAttempts(10),
			WithErrorFilter(func(err error) ErrorOutcome {
				if err.Error() == "normal" {
					return ErrorOutcomeNormalRetry
				}
				if err.Error() == "reset" {
					return ErrorOutcomeResetAttempts
				}
				if err.Error() == "stop" {
					return ErrorOutcomeDoNotRetry
				}
				return ErrorOutcomeNormalRetry
			}),
		)

		require.ErrorContains(t, err, "stop")
		require.Equal(t, 3, count)
	})
}
