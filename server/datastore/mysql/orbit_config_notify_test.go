package mysql

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

type orbitConfigNotification struct {
	hostIDs []uint
	reason  string
}

type captureOrbitConfigNotifier struct {
	fleet.AgentCheckInNotifier
	mu    sync.Mutex
	calls []orbitConfigNotification
}

func (c *captureOrbitConfigNotifier) NotifyOrbitConfigHosts(ctx context.Context, hostIDs []uint, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls = append(c.calls, orbitConfigNotification{hostIDs: slices.Clone(hostIDs), reason: reason})
	return nil
}

// take returns the notifications so far and clears them.
func (c *captureOrbitConfigNotifier) take() []orbitConfigNotification {
	c.mu.Lock()
	defer c.mu.Unlock()
	calls := c.calls
	c.calls = nil
	return calls
}

func TestOrbitConfigNotify(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier)
	}{
		{"AfterCommit", testOrbitConfigNotifyAfterCommit},
		{"ScriptActivation", testOrbitConfigNotifyScriptActivation},
		{"DiskEncryption", testOrbitConfigNotifyDiskEncryption},
		{"SetupExperience", testOrbitConfigNotifySetupExperience},
		{"WindowsMDMSync", testOrbitConfigNotifyWindowsMDMSync},
		{"EnrollOsquery", testOrbitConfigNotifyEnrollOsquery},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			notifier := &captureOrbitConfigNotifier{}
			ds.WithOrbitConfigNotifier(notifier)
			defer ds.WithOrbitConfigNotifier(nil)
			c.fn(t, ds, notifier)
		})
	}
}

func testOrbitConfigNotifyAfterCommit(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		ds.notifyOrbitConfig(ctx, tx, fleet.AgentWSReasonActivity, 1, 2)
		require.Empty(t, notifier.take(), "notified before commit")
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{1, 2}, reason: fleet.AgentWSReasonActivity}}, notifier.take())

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		ds.notifyOrbitConfig(ctx, tx, fleet.AgentWSReasonActivity, 1)
		return errors.New("rollback")
	})
	require.Error(t, err)
	require.Empty(t, notifier.take(), "notified a rolled back change")

	// Outside a transaction, the write is already committed.
	ds.notifyOrbitConfig(ctx, ds.writer(ctx), fleet.AgentWSReasonActivity, 3)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{3}, reason: fleet.AgentWSReasonActivity}}, notifier.take())

	// Without a notifier, nothing happens (and nothing is resolved).
	ds.WithOrbitConfigNotifier(nil)
	require.NoError(t, ds.notifyOrbitConfigByHostUUIDs(ctx, ds.writer(ctx), fleet.AgentWSReasonMDM, "uuid"))
	ds.WithOrbitConfigNotifier(notifier)
	require.Empty(t, notifier.take())
}

func testOrbitConfigNotifyScriptActivation(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()
	host1 := test.NewHost(t, ds, "host1", "", "key1", "uuid1", time.Now())
	host2 := test.NewHost(t, ds, "host2", "", "key2", "uuid2", time.Now())
	notifier.take()

	// The first script is activated right away; only its host is notified.
	exec1, err := ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "echo 1"})
	require.NoError(t, err)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host1.ID}, reason: fleet.AgentWSReasonActivity}}, notifier.take())

	// A second script waits behind the first: no notification.
	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "echo 2"})
	require.NoError(t, err)
	require.Empty(t, notifier.take())

	// The first script's result activates the second.
	_, _, err = ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host1.ID,
		ExecutionID: exec1.ExecutionID,
		Output:      "1",
	}, nil)
	require.NoError(t, err)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host1.ID}, reason: fleet.AgentWSReasonActivity}}, notifier.take())

	_, err = ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host2.ID, ScriptContents: "echo 3"})
	require.NoError(t, err)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host2.ID}, reason: fleet.AgentWSReasonActivity}}, notifier.take())
}

func testOrbitConfigNotifyDiskEncryption(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "host1", "", "key1", "uuid1", time.Now())
	notifier.take()

	require.NoError(t, ds.QueueEscrow(ctx, host.ID))
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host.ID}, reason: fleet.AgentWSReasonDiskEncryption}}, notifier.take())

	// Only an undecryptable key makes the host rotate it.
	require.NoError(t, ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, true, time.Now()))
	require.Empty(t, notifier.take())
	require.NoError(t, ds.SetHostsDiskEncryptionKeyStatus(ctx, []uint{host.ID}, false, time.Now()))
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host.ID}, reason: fleet.AgentWSReasonDiskEncryption}}, notifier.take())
}

func testOrbitConfigNotifySetupExperience(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "host1", "", "key1", "uuid1", time.Now())
	notifier.take()

	require.NoError(t, ds.SetHostAwaitingConfiguration(ctx, host.UUID, false))
	require.Empty(t, notifier.take())
	require.NoError(t, ds.SetHostAwaitingConfiguration(ctx, host.UUID, true))
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host.ID}, reason: fleet.AgentWSReasonMDM}}, notifier.take())

	// An unknown host UUID notifies no one.
	require.NoError(t, ds.SetHostAwaitingConfiguration(ctx, "no-such-uuid", true))
	require.Empty(t, notifier.take())
}

func testOrbitConfigNotifyWindowsMDMSync(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()

	enroll := func(host *fleet.Host, syncCapable bool) uint {
		device := &fleet.MDMWindowsEnrolledDevice{
			MDMDeviceID:    uuid.NewString(),
			MDMHardwareID:  uuid.NewString(),
			MDMDeviceState: microsoft_mdm.MDMDeviceStateEnrolled,
			HostUUID:       host.UUID,
		}
		require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(ctx, device))
		require.NoError(t, ds.SetMDMWindowsEnrollmentFleetdSyncCapable(ctx, host.UUID, syncCapable))
		enrolled, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, device.MDMDeviceID)
		require.NoError(t, err)
		return enrolled.ID
	}
	capable := test.NewHost(t, ds, "host1", "", "key1", "uuid1", time.Now())
	notCapable := test.NewHost(t, ds, "host2", "", "key2", "uuid2", time.Now())
	ids := []uint{enroll(capable, true), enroll(notCapable, false)}
	notifier.take()

	require.NoError(t, ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ds.markMDMWindowsHasPendingCommandsByEnrollmentIDs(ctx, tx, ids)
	}))
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{capable.ID}, reason: fleet.AgentWSReasonMDM}}, notifier.take())
}

func testOrbitConfigNotifyEnrollOsquery(t *testing.T, ds *Datastore, notifier *captureOrbitConfigNotifier) {
	ctx := t.Context()
	host, err := ds.EnrollOsquery(ctx,
		fleet.WithEnrollOsqueryHostID("osquery-host-id"),
		fleet.WithEnrollOsqueryNodeKey("node-key"),
	)
	require.NoError(t, err)
	require.Equal(t, []orbitConfigNotification{{hostIDs: []uint{host.ID}, reason: fleet.AgentWSReasonEnroll}}, notifier.take())
}
