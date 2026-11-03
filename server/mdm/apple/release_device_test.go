package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeReleaseCommander struct {
	fleet.MDMAppleCommandIssuer
	calls int
	err   error
}

func (f *fakeReleaseCommander) DeviceConfigured(context.Context, string, string) error {
	f.calls++
	return f.err
}

type notFoundErr struct{}

func (notFoundErr) Error() string    { return "not found" }
func (notFoundErr) IsNotFound() bool { return true }

func TestReleaseDevice(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	setup := func(awaiting bool, getErr, setErr error) (*mock.Store, *[]bool) {
		ds := new(mock.Store)
		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, _ string) (bool, error) {
			assert.True(t, ctxdb.IsPrimaryRequired(ctx), "flag must be read from the primary")
			return awaiting, getErr
		}
		var set []bool
		ds.SetHostAwaitingConfigurationFunc = func(_ context.Context, _ string, awaiting bool) error {
			set = append(set, awaiting)
			return setErr
		}
		return ds, &set
	}

	t.Run("sends then clears the flag", func(t *testing.T) {
		ds, set := setup(true, nil, nil)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
		assert.Equal(t, []bool{false}, *set)
	})

	t.Run("skips when the flag is already cleared", func(t *testing.T) {
		ds, set := setup(false, nil, nil)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Zero(t, cmdr.calls)
		assert.Empty(t, *set)
	})

	t.Run("skips when the host has no flag", func(t *testing.T) {
		ds, _ := setup(false, notFoundErr{}, nil)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Zero(t, cmdr.calls)
	})

	t.Run("enqueue failure keeps the flag for a retry", func(t *testing.T) {
		ds, set := setup(true, nil, nil)
		cmdr := &fakeReleaseCommander{err: errors.New("db down")}
		require.ErrorContains(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"), "db down")
		assert.Empty(t, *set)
	})

	t.Run("failing to clear the flag after sending is returned", func(t *testing.T) {
		ds, _ := setup(true, nil, errors.New("lock wait timeout"))
		cmdr := &fakeReleaseCommander{}
		require.ErrorContains(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"), "lock wait timeout")
		assert.Equal(t, 1, cmdr.calls)
	})

	t.Run("push failure after enqueue is not an error", func(t *testing.T) {
		ds, set := setup(true, nil, nil)
		cmdr := &fakeReleaseCommander{err: &NotificationFailedError{err: errors.New("apns down")}}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
		assert.Equal(t, []bool{false}, *set)
	})
}
