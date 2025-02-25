package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestNanoMDMStorage(t *testing.T) {
	ds := CreateMySQLDS(t)
	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestEnqueueDeviceLockCommand", testEnqueueDeviceLockCommand},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testEnqueueDeviceLockCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host1-name",
		OsqueryHostID: ptr.String("1337"),
		NodeKey:       ptr.String("1337"),
		UUID:          "test-uuid-1",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	// no commands yet
	res, err := ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Empty(t, res)

	cmd := &mdm.Command{}
	cmd.CommandUUID = "cmd-uuid"
	cmd.Command.RequestType = "DeviceLock"
	cmd.Raw = []byte("<?xml")

	err = ns.EnqueueDeviceLockCommand(ctx, host, cmd, "123456")
	require.NoError(t, err)

	// command has no results yet, so the status is empty
	res, err = ds.ListMDMAppleCommands(ctx, fleet.TeamFilter{User: test.UserAdmin}, &fleet.MDMCommandListOptions{})
	require.NoError(t, err)
	require.Len(t, res, 1)

	require.NotZero(t, res[0].UpdatedAt)
	res[0].UpdatedAt = time.Time{}

	require.ElementsMatch(t, []*fleet.MDMAppleCommand{
		{
			DeviceID:    host.UUID,
			CommandUUID: "cmd-uuid",
			Status:      "Pending",
			RequestType: "DeviceLock",
			Hostname:    host.Hostname,
			TeamID:      nil,
		},
	}, res)

	status, err := ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	require.Equal(t, "cmd-uuid", status.LockMDMCommand.CommandUUID)
	require.Equal(t, "123456", status.UnlockPIN)
}
