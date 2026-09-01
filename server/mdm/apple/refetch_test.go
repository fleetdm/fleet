package apple_mdm

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/mock"
	mdmmock "github.com/fleetdm/fleet/v4/server/mock/mdm"
	"github.com/stretchr/testify/require"
)

type fakePusher struct {
	resp map[string]*push.Response
	err  error
}

func (f *fakePusher) Push(_ context.Context, _ []string) (map[string]*push.Response, error) {
	return f.resp, f.err
}

func TestEnqueueAndNotifyStageErrors(t *testing.T) {
	ctx := t.Context()
	mdmStorage := &mdmmock.MDMAppleStore{}
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, nil
	}

	var notifErr *NotificationFailedError
	var apnsErr *APNSDeliveryError

	// a pusher-level failure happens after the command was enqueued, so it is
	// reported as a NotificationFailedError
	cmdr := NewMDMAppleCommander(mdmStorage, &fakePusher{err: errors.New("apns down")})
	err := cmdr.DeviceInformation(ctx, []string{"A"}, fleet.RefetchDeviceCommandUUIDPrefix+"u1", false)
	require.ErrorAs(t, err, &notifErr)

	// per-device push failures keep the APNSDeliveryError detectable through
	// the NotificationFailedError wrapper
	cmdr = NewMDMAppleCommander(mdmStorage, &fakePusher{resp: map[string]*push.Response{"A": {Err: errors.New("boom")}}})
	err = cmdr.DeviceInformation(ctx, []string{"A"}, fleet.RefetchDeviceCommandUUIDPrefix+"u2", false)
	require.ErrorAs(t, err, &notifErr)
	require.ErrorAs(t, err, &apnsErr)

	// an enqueue failure means nothing was queued and must NOT be reported as
	// a NotificationFailedError
	mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		return nil, errors.New("db down")
	}
	cmdr = NewMDMAppleCommander(mdmStorage, &fakePusher{})
	err = cmdr.DeviceInformation(ctx, []string{"A"}, fleet.RefetchDeviceCommandUUIDPrefix+"u3", false)
	require.Error(t, err)
	require.NotErrorAs(t, err, &notifErr)
}

type refetchTestEnv struct {
	ds         *mock.Store
	mdmStorage *mdmmock.MDMAppleStore
	commander  *MDMAppleCommander
	pusher     *fakePusher
	events     []string
}

func refetchCommandType(commandUUID string) string {
	for _, prefix := range []string{
		fleet.RefetchAppsCommandUUIDPrefix,
		fleet.RefetchCertsCommandUUIDPrefix,
		fleet.RefetchDeviceCommandUUIDPrefix,
	} {
		if strings.HasPrefix(commandUUID, prefix) {
			return prefix
		}
	}
	return commandUUID
}

func setupRefetchTest(t *testing.T) *refetchTestEnv {
	env := &refetchTestEnv{
		ds:         new(mock.Store),
		mdmStorage: &mdmmock.MDMAppleStore{},
		pusher:     &fakePusher{},
	}
	env.commander = NewMDMAppleCommander(env.mdmStorage, env.pusher)

	env.ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}, nil
	}
	env.ds.ListIOSAndIPadOSToRefetchFunc = func(ctx context.Context, refetchInterval time.Duration) ([]fleet.AppleDevicesToRefetch, error) {
		return []fleet.AppleDevicesToRefetch{
			{HostID: 1, UUID: "device-1", InstalledFromDEP: true},
		}, nil
	}
	env.ds.AddHostMDMCommandsFunc = func(ctx context.Context, commands []fleet.HostMDMCommand) error {
		for _, cmd := range commands {
			require.Equal(t, uint(1), cmd.HostID)
			env.events = append(env.events, "add:"+cmd.CommandType)
		}
		return nil
	}
	env.ds.RemoveHostMDMCommandsFunc = func(ctx context.Context, hostIDs []uint, commandType string) error {
		require.Equal(t, []uint{1}, hostIDs)
		env.events = append(env.events, "remove:"+commandType)
		return nil
	}
	env.mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
		env.events = append(env.events, "enqueue:"+refetchCommandType(cmd.CommandUUID))
		return nil, nil
	}

	return env
}

func TestIOSiPadOSRefetchTracksBeforeEnqueue(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	noopActivity := func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
		return nil
	}

	t.Run("tracking rows are written before each enqueue", func(t *testing.T) {
		env := setupRefetchTest(t)

		err := IOSiPadOSRefetch(ctx, env.ds, env.commander, logger, noopActivity)
		require.NoError(t, err)
		require.Equal(t, []string{
			"add:" + fleet.RefetchAppsCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchAppsCommandUUIDPrefix,
			"add:" + fleet.RefetchCertsCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchCertsCommandUUIDPrefix,
			"add:" + fleet.RefetchDeviceCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchDeviceCommandUUIDPrefix,
		}, env.events)
	})

	t.Run("mixed devices are grouped by command flag", func(t *testing.T) {
		env := setupRefetchTest(t)
		env.ds.ListIOSAndIPadOSToRefetchFunc = func(ctx context.Context, refetchInterval time.Duration) ([]fleet.AppleDevicesToRefetch, error) {
			return []fleet.AppleDevicesToRefetch{
				{HostID: 1, UUID: "device-1", InstalledFromDEP: true},
				{HostID: 2, UUID: "device-2", InstalledFromDEP: false, IsPersonalEnrollment: true},
				{HostID: 3, UUID: "device-3", InstalledFromDEP: true, CommandsAlreadySent: []string{fleet.RefetchAppsCommandUUIDPrefix}},
			}, nil
		}
		env.ds.AddHostMDMCommandsFunc = func(ctx context.Context, commands []fleet.HostMDMCommand) error {
			return nil
		}

		type batch struct {
			commandType string
			uuids       []string
			raw         string
		}
		var batches []batch
		env.mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
			batches = append(batches, batch{refetchCommandType(cmd.CommandUUID), id, string(cmd.Raw)})
			return nil, nil
		}

		err := IOSiPadOSRefetch(ctx, env.ds, env.commander, logger, noopActivity)
		require.NoError(t, err)
		require.Len(t, batches, 5)

		// apps: the BYOD device gets a managed-only list first, then the DEP
		// device gets the full list; device-3 already received the command
		require.Equal(t, fleet.RefetchAppsCommandUUIDPrefix, batches[0].commandType)
		require.Equal(t, []string{"device-2"}, batches[0].uuids)
		require.Contains(t, batches[0].raw, "<true/>")
		require.Equal(t, fleet.RefetchAppsCommandUUIDPrefix, batches[1].commandType)
		require.Equal(t, []string{"device-1"}, batches[1].uuids)
		require.Contains(t, batches[1].raw, "<false/>")

		// certs: a single batch for all devices
		require.Equal(t, fleet.RefetchCertsCommandUUIDPrefix, batches[2].commandType)
		require.ElementsMatch(t, []string{"device-1", "device-2", "device-3"}, batches[2].uuids)

		// device info: the personal device gets the reduced query set (no
		// BatteryLevel), the others the full set
		require.Equal(t, fleet.RefetchDeviceCommandUUIDPrefix, batches[3].commandType)
		require.Equal(t, []string{"device-2"}, batches[3].uuids)
		require.NotContains(t, batches[3].raw, "BatteryLevel")
		require.Equal(t, fleet.RefetchDeviceCommandUUIDPrefix, batches[4].commandType)
		require.ElementsMatch(t, []string{"device-1", "device-3"}, batches[4].uuids)
		require.Contains(t, batches[4].raw, "BatteryLevel")
	})

	t.Run("enqueue failure untracks only the failed command type", func(t *testing.T) {
		env := setupRefetchTest(t)
		env.mdmStorage.EnqueueCommandFunc = func(ctx context.Context, id []string, cmd *mdm.CommandWithSubtype) (map[string]error, error) {
			commandType := refetchCommandType(cmd.CommandUUID)
			if commandType == fleet.RefetchCertsCommandUUIDPrefix {
				return nil, errors.New("db down")
			}
			env.events = append(env.events, "enqueue:"+commandType)
			return nil, nil
		}

		err := IOSiPadOSRefetch(ctx, env.ds, env.commander, logger, noopActivity)
		require.Error(t, err)
		require.Equal(t, []string{
			"add:" + fleet.RefetchAppsCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchAppsCommandUUIDPrefix,
			"add:" + fleet.RefetchCertsCommandUUIDPrefix,
			"remove:" + fleet.RefetchCertsCommandUUIDPrefix,
		}, env.events)
	})

	t.Run("push failure keeps the tracking rows", func(t *testing.T) {
		env := setupRefetchTest(t)
		// per-device push failures produce an APNSDeliveryError; the commands
		// are durably enqueued, so no tracking row may be removed
		env.pusher.resp = map[string]*push.Response{"device-1": {Err: errors.New("apns unavailable")}}

		err := IOSiPadOSRefetch(ctx, env.ds, env.commander, logger, noopActivity)
		require.NoError(t, err)
		require.False(t, env.ds.RemoveHostMDMCommandsFuncInvoked)
		require.Equal(t, []string{
			"add:" + fleet.RefetchAppsCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchAppsCommandUUIDPrefix,
			"add:" + fleet.RefetchCertsCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchCertsCommandUUIDPrefix,
			"add:" + fleet.RefetchDeviceCommandUUIDPrefix,
			"enqueue:" + fleet.RefetchDeviceCommandUUIDPrefix,
		}, env.events)
	})
}
