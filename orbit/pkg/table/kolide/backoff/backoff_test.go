package backoff

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImmediate(t *testing.T) {
	t.Parallel()

	bkoff := New()
	bkoff.delay = 0.0

	err := bkoff.Run(willSucceed)
	require.NoError(t, err)
	require.Equal(t, 1, bkoff.count)
}

func TestEventual(t *testing.T) {
	t.Parallel()
	bkoff := New()
	bkoff.delay = 0.0

	tTime := &takesTime{
		attempts: 0,
	}

	err := bkoff.Run(tTime.eventual)
	require.NoError(t, err)
	require.Equal(t, 5, bkoff.count)
	require.Equal(t, 5, tTime.attempts)

}

func TestSlowFail(t *testing.T) {
	t.Parallel()
	bkoff := New()
	bkoff.delay = 0.0

	err := bkoff.Run(willFail)
	require.Error(t, err)
	require.Equal(t, 20, bkoff.count)
}

func TestMaxAttempts(t *testing.T) {
	t.Parallel()
	bkoff := New(MaxAttempts(5))
	bkoff.delay = 0.0

	err := bkoff.Run(willFail)
	require.Error(t, err)
	require.Equal(t, 5, bkoff.count)
}

func willSucceed() error {
	return nil
}

func willFail() error {
	return errors.New("nope")
}

type takesTime struct {
	attempts int
}

func (t *takesTime) eventual() error {
	t.attempts += 1

	if t.attempts >= 5 {
		return nil
	}

	return errors.New("not yet")
}
