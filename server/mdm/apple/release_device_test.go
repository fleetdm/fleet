package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	"testing"

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

func TestReleaseDevice(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	setup := func(cleared bool, unsetErr error) (*mock.Store, *[]bool) {
		ds := new(mock.Store)
		ds.UnsetHostAwaitingConfigurationFunc = func(context.Context, string) (bool, error) {
			return cleared, unsetErr
		}
		var restored []bool
		ds.SetHostAwaitingConfigurationFunc = func(_ context.Context, _ string, awaiting bool) error {
			restored = append(restored, awaiting)
			return nil
		}
		return ds, &restored
	}

	t.Run("sends when this call clears the flag", func(t *testing.T) {
		ds, restored := setup(true, nil)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
		assert.Empty(t, *restored)
	})

	t.Run("skips when the flag was already cleared", func(t *testing.T) {
		ds, restored := setup(false, nil)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Zero(t, cmdr.calls)
		assert.Empty(t, *restored)
	})

	t.Run("failing to clear the flag sends nothing", func(t *testing.T) {
		ds, _ := setup(false, errors.New("lock wait timeout"))
		cmdr := &fakeReleaseCommander{}
		require.ErrorContains(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"), "lock wait timeout")
		assert.Zero(t, cmdr.calls)
	})

	t.Run("enqueue failure restores the flag", func(t *testing.T) {
		ds, restored := setup(true, nil)
		cmdr := &fakeReleaseCommander{err: errors.New("db down")}
		require.ErrorContains(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"), "db down")
		assert.Equal(t, []bool{true}, *restored)
	})

	t.Run("push failure after enqueue is not an error", func(t *testing.T) {
		ds, restored := setup(true, nil)
		cmdr := &fakeReleaseCommander{err: &NotificationFailedError{err: errors.New("apns down")}}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
		assert.Empty(t, *restored)
	})
}
