package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReapStuckMDMInstalls(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	const (
		hostID      = uint(7)
		hostUUID    = "host-uuid-7"
		commandUUID = "install-cmd-uuid"
	)

	setupMockDS := func() *mock.DataStore {
		ds := new(mock.DataStore)
		ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(_ context.Context, _, _ string, _ fleet.SetupExperienceStatusResultStatus) (bool, error) {
			return false, nil
		}
		return ds
	}

	appStoreReaped := func() []fleet.ReapedMDMInstall {
		return []fleet.ReapedMDMInstall{{
			HostID:           hostID,
			HostUUID:         hostUUID,
			CommandUUID:      commandUUID,
			User:             &fleet.User{ID: 3},
			AppStoreActivity: &fleet.ActivityInstalledAppStoreApp{HostID: hostID, CommandUUID: commandUUID},
		}}
	}

	t.Run("emits the failed install activity", func(t *testing.T) {
		ds := setupMockDS()
		ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, olderThan time.Duration, maxHosts int) ([]fleet.ReapedMDMInstall, error) {
			assert.Equal(t, 24*time.Hour, olderThan)
			assert.Equal(t, 500, maxHosts)
			return appStoreReaped(), nil
		}

		var gotUser *fleet.User
		var gotActivity fleet.ActivityDetails
		newActivityFn := func(_ context.Context, user *fleet.User, act fleet.ActivityDetails) error {
			gotUser, gotActivity = user, act
			return nil
		}

		require.NoError(t, ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, 24*time.Hour, 500))
		require.NotNil(t, gotActivity)
		require.NotNil(t, gotUser)
		assert.Equal(t, uint(3), gotUser.ID)
		act, ok := gotActivity.(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.Equal(t, commandUUID, act.CommandUUID)
		assert.False(t, act.FromSetupExperience)
	})

	t.Run("fails the setup experience step and flags the activity", func(t *testing.T) {
		ds := setupMockDS()
		ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, _ time.Duration, _ int) ([]fleet.ReapedMDMInstall, error) {
			return appStoreReaped(), nil
		}
		// Without this the step stays running for good and, on macOS, the rest of the setup
		// experience is never cancelled.
		var gotHostUUID, gotCmdUUID string
		var gotStatus fleet.SetupExperienceStatusResultStatus
		ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(_ context.Context, hUUID, cmdUUID string,
			status fleet.SetupExperienceStatusResultStatus,
		) (bool, error) {
			gotHostUUID, gotCmdUUID, gotStatus = hUUID, cmdUUID, status
			return true, nil
		}
		ds.HostByIdentifierFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: hostUUID, Platform: "ios"}, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(_ context.Context, _ string, _ uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}

		var gotActivity fleet.ActivityDetails
		newActivityFn := func(_ context.Context, _ *fleet.User, act fleet.ActivityDetails) error {
			gotActivity = act
			return nil
		}

		require.NoError(t, ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, 24*time.Hour, 500))
		assert.Equal(t, hostUUID, gotHostUUID)
		assert.Equal(t, commandUUID, gotCmdUUID)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, gotStatus)

		act, ok := gotActivity.(*fleet.ActivityInstalledAppStoreApp)
		require.True(t, ok)
		assert.True(t, act.FromSetupExperience)
	})

	t.Run("in-house installs fail the setup experience step and flag the activity", func(t *testing.T) {
		ds := setupMockDS()
		ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, _ time.Duration, _ int) ([]fleet.ReapedMDMInstall, error) {
			return []fleet.ReapedMDMInstall{{
				HostID:          hostID,
				HostUUID:        hostUUID,
				CommandUUID:     commandUUID,
				InHouseActivity: &fleet.ActivityTypeInstalledSoftware{HostID: hostID, CommandUUID: commandUUID},
			}}, nil
		}
		// In-house apps install during iOS/iPadOS setup experience; a reaped one must fail its
		// step or the row stays running for good (the queue row is gone, so no command result
		// will ever flip it).
		var gotHostUUID, gotCmdUUID string
		var gotStatus fleet.SetupExperienceStatusResultStatus
		ds.MaybeUpdateSetupExperienceVPPStatusFunc = func(_ context.Context, hUUID, cmdUUID string,
			status fleet.SetupExperienceStatusResultStatus,
		) (bool, error) {
			gotHostUUID, gotCmdUUID, gotStatus = hUUID, cmdUUID, status
			return true, nil
		}
		ds.HostByIdentifierFunc = func(_ context.Context, _ string) (*fleet.Host, error) {
			return &fleet.Host{ID: hostID, UUID: hostUUID, Platform: "ios"}, nil
		}
		ds.ListSetupExperienceResultsByHostUUIDFunc = func(_ context.Context, _ string, _ uint) ([]*fleet.SetupExperienceStatusResult, error) {
			return nil, nil
		}

		var gotActivity fleet.ActivityDetails
		newActivityFn := func(_ context.Context, _ *fleet.User, act fleet.ActivityDetails) error {
			gotActivity = act
			return nil
		}

		require.NoError(t, ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, 24*time.Hour, 500))
		assert.Equal(t, hostUUID, gotHostUUID)
		assert.Equal(t, commandUUID, gotCmdUUID)
		assert.Equal(t, fleet.SetupExperienceStatusFailure, gotStatus)

		act, ok := gotActivity.(*fleet.ActivityTypeInstalledSoftware)
		require.True(t, ok)
		assert.True(t, act.FromSetupExperience)
	})

	t.Run("records the installs a partially failed run did reap", func(t *testing.T) {
		ds := setupMockDS()
		// A host that errors must not cost the hosts that succeeded their activities: the
		// datastore has already failed those installs.
		ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, _ time.Duration, _ int) ([]fleet.ReapedMDMInstall, error) {
			return appStoreReaped(), errors.New("one host blew up")
		}

		var activities int
		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			activities++
			return nil
		}

		err := ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, 24*time.Hour, 500)
		require.Error(t, err)
		require.ErrorContains(t, err, "one host blew up")
		assert.Equal(t, 1, activities)
	})

	t.Run("a non-positive timeout reaps nothing at all", func(t *testing.T) {
		// Reaching the query with 0 would make every activated install on the fleet older than
		// the threshold, so this must not call the datastore at all.
		for _, olderThan := range []time.Duration{0, -1 * time.Hour} {
			ds := setupMockDS()
			ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, _ time.Duration, _ int) ([]fleet.ReapedMDMInstall, error) {
				t.Fatalf("the datastore must not be reached with olderThan=%s", olderThan)
				return nil, nil
			}
			newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
				t.Fatal("no activity should be emitted")
				return nil
			}
			require.NoError(t, ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, olderThan, 500))
			require.False(t, ds.ReapStuckActivatedMDMInstallsFuncInvoked)
		}
	})

	t.Run("nothing to reap does nothing", func(t *testing.T) {
		ds := setupMockDS()
		ds.ReapStuckActivatedMDMInstallsFunc = func(_ context.Context, _ time.Duration, _ int) ([]fleet.ReapedMDMInstall, error) {
			return nil, nil
		}
		newActivityFn := func(_ context.Context, _ *fleet.User, _ fleet.ActivityDetails) error {
			t.Fatal("no activity should be emitted")
			return nil
		}
		require.NoError(t, ReapStuckMDMInstalls(ctx, ds, logger, newActivityFn, 24*time.Hour, 500))
	})
}
