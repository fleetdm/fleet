package retry

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewLimitedWithCooldown(t *testing.T) {
	lwc := NewLimitedWithCooldown(3, 1*time.Hour)
	require.NotNil(t, lwc)
	require.Equal(t, 3, lwc.maxRetries)
	require.Equal(t, 1*time.Hour, lwc.cooldown)
}

func TestLimitedWithCooldwonDo(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		lwc := NewLimitedWithCooldown(3, 1*time.Hour)
		hash := "foo"

		err := lwc.Do(hash, func() error {
			return nil
		})

		require.NoError(t, err)
		require.Equal(t, 0, lwc.retries[hash])
		require.True(t, lwc.wait[hash].IsZero())
	})

	t.Run("fail and retry", func(t *testing.T) {
		lwc := NewLimitedWithCooldown(3, 1*time.Hour)
		hash := "foo"

		// failures followed by a success
		for i := 0; i < 2; i++ {
			err := lwc.Do(hash, func() error {
				return errors.New("failure")
			})
			require.Error(t, err)
			require.Equal(t, i+1, lwc.retries[hash])
		}

		err := lwc.Do(hash, func() error {
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 0, lwc.retries[hash])
		require.True(t, lwc.wait[hash].IsZero())
	})

	t.Run("exceed max retries", func(t *testing.T) {
		cooldown := 1 * time.Hour
		lwc := NewLimitedWithCooldown(3, cooldown)
		hash := "foo"

		for i := 0; i < 3; i++ {
			err := lwc.Do(hash, func() error {
				return errors.New("failure")
			})
			require.Error(t, err)
		}

		err := lwc.Do(hash, func() error {
			return nil
		})
		rErr, ok := err.(*ExcessRetriesError)
		require.True(t, ok)
		require.WithinDuration(t, time.Now().Add(cooldown), time.Now().Add(rErr.nextRetry), 1*time.Minute)
	})

	t.Run("cooldown period", func(t *testing.T) {
		lwc := NewLimitedWithCooldown(3, 1*time.Millisecond)
		hash := "foo"

		for i := 0; i < 3; i++ {
			err := lwc.Do(hash, func() error {
				return errors.New("failure")
			})
			require.Error(t, err)
		}

		time.Sleep(2 * time.Millisecond)

		err := lwc.Do(hash, func() error {
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 0, lwc.retries[hash])
		require.True(t, lwc.wait[hash].IsZero())
	})

	t.Run("multiple hashes", func(t *testing.T) {
		lwc := NewLimitedWithCooldown(3, 1*time.Hour)
		hash1 := "hash1"
		hash2 := "hash2"

		// Fail for hash1
		err1 := lwc.Do(hash1, func() error {
			return errors.New("failure")
		})
		require.Error(t, err1)

		// Succeed for hash2
		err2 := lwc.Do(hash2, func() error {
			return nil
		})
		require.NoError(t, err2)

		require.Equal(t, 1, lwc.retries[hash1])
		require.Equal(t, 0, lwc.retries[hash2])
	})
}
