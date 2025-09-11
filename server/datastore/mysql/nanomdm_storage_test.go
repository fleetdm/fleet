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
		{"TestGetPendingLockCommand", testGetPendingLockCommand},
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

func testGetPendingLockCommand(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	ns, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)

	host, err := ds.NewHost(ctx, &fleet.Host{
		Hostname:      "test-host2-name",
		OsqueryHostID: ptr.String("1338"),
		NodeKey:       ptr.String("1338"),
		UUID:          "test-uuid-2",
		TeamID:        nil,
		Platform:      "darwin",
	})
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)

	// Test 1: No pending commands should return nil
	cmd, pin, err := ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.Empty(t, pin)

	// Test 2: Enqueue a lock command
	lockCmd := &mdm.Command{}
	lockCmd.CommandUUID = "lock-cmd-uuid"
	lockCmd.Command.RequestType = "DeviceLock"
	lockCmd.Raw = []byte("<?xml")

	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd, "654321")
	require.NoError(t, err)

	// Test 3: Should find the pending command
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "lock-cmd-uuid", cmd.CommandUUID)
	require.Equal(t, "DeviceLock", cmd.Command.RequestType)
	require.Equal(t, "654321", pin)

	// Test 4: Simulate command acknowledgment
	_, err = ds.writer(ctx).ExecContext(ctx, `
		INSERT INTO nano_command_results (id, command_uuid, status, result) 
		VALUES (?, ?, 'Acknowledged', '<?xml version="1.0"?><plist></plist>')`,
		host.UUID, "lock-cmd-uuid")
	require.NoError(t, err)

	// Test 5: After acknowledgment, should not find the command
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.Nil(t, cmd)
	require.Empty(t, pin)

	// Test 6: Enqueue multiple commands, should return most recent
	lockCmd2 := &mdm.Command{}
	lockCmd2.CommandUUID = "lock-cmd-uuid-2"
	lockCmd2.Command.RequestType = "DeviceLock"
	lockCmd2.Raw = []byte("<?xml2")

	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd2, "111111")
	require.NoError(t, err)

	// Add a small delay to ensure different timestamps
	time.Sleep(10 * time.Millisecond)

	lockCmd3 := &mdm.Command{}
	lockCmd3.CommandUUID = "lock-cmd-uuid-3"
	lockCmd3.Command.RequestType = "DeviceLock"
	lockCmd3.Raw = []byte("<?xml3")

	err = ns.EnqueueDeviceLockCommand(ctx, host, lockCmd3, "222222")
	require.NoError(t, err)

	// Should return the most recent unacknowledged command
	cmd, pin, err = ns.GetPendingLockCommand(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "lock-cmd-uuid-3", cmd.CommandUUID)
	require.Equal(t, "222222", pin)
}
