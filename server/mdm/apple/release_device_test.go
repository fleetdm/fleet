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

	setup := func(pending, acknowledged bool) (*mock.Store, *[]bool) {
		ds := new(mock.Store)
		ds.GetHostDeviceConfiguredStatusFunc = func(context.Context, string) (bool, bool, error) {
			return pending, acknowledged, nil
		}
		var awaitingSet []bool
		ds.SetHostAwaitingConfigurationFunc = func(_ context.Context, _ string, awaiting bool) error {
			awaitingSet = append(awaitingSet, awaiting)
			return nil
		}
		return ds, &awaitingSet
	}

	t.Run("sends when none is pending", func(t *testing.T) {
		ds, awaitingSet := setup(false, false)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
		assert.Empty(t, *awaitingSet)
	})

	t.Run("skips while one is pending", func(t *testing.T) {
		ds, awaitingSet := setup(true, false)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Zero(t, cmdr.calls)
		assert.Empty(t, *awaitingSet)
	})

	t.Run("acknowledged clears the awaiting flag instead of sending", func(t *testing.T) {
		ds, awaitingSet := setup(true, true)
		cmdr := &fakeReleaseCommander{}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Zero(t, cmdr.calls)
		assert.Equal(t, []bool{false}, *awaitingSet)
	})

	t.Run("push failure after enqueue is not an error", func(t *testing.T) {
		ds, _ := setup(false, false)
		cmdr := &fakeReleaseCommander{err: &NotificationFailedError{err: errors.New("apns down")}}
		require.NoError(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"))
		assert.Equal(t, 1, cmdr.calls)
	})

	t.Run("enqueue failure is returned", func(t *testing.T) {
		ds, _ := setup(false, false)
		cmdr := &fakeReleaseCommander{err: errors.New("db down")}
		require.ErrorContains(t, ReleaseDevice(ctx, ds, cmdr, logger, "host"), "db down")
	})
}
