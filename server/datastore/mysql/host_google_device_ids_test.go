package mysql

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHostGoogleDeviceIDs(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SetAndGet", testHostGoogleDeviceIDSetAndGet},
		{"KeptOnAppleReenrollment", testHostGoogleDeviceIDKeptOnAppleReenrollment},
		{"KeptOnAndroidReenrollment", testHostGoogleDeviceIDKeptOnAndroidReenrollment},
		{"ConcurrentDifferentValues", testHostGoogleDeviceIDConcurrentDifferentValues},
		{"ConcurrentSameValueDifferentHosts", testHostGoogleDeviceIDConcurrentSameValueDifferentHosts},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testHostGoogleDeviceIDSetAndGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "h1", "", "h1key", "h1uuid", time.Now())
	other := test.NewHost(t, ds, "h2", "", "h2key", "h2uuid", time.Now())
	names := []string{"devices/device-a/deviceUsers/1", "devices/device-a/deviceUsers/2"}
	requireID := func(h *fleet.Host, want string) {
		t.Helper()
		id, err := ds.GetHostGoogleDeviceID(ctx, h.UUID, googleTestCustomer)
		require.NoError(t, err)
		require.Equal(t, want, id)
	}

	requireID(host, "")

	changed, err := ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, "device-a", names)
	require.NoError(t, err)
	require.True(t, changed)
	requireID(host, "device-a")
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		names[0]: fleet.GoogleConditionalAccessDeviceUserPending,
		names[1]: fleet.GoogleConditionalAccessDeviceUserPending,
	}, googleDeviceUserStates(t, ds, host.UUID))

	// The ID shows when no customer is configured, and not for another customer.
	id, err := ds.GetHostGoogleDeviceID(ctx, host.UUID, "")
	require.NoError(t, err)
	require.Equal(t, "device-a", id)
	id, err = ds.GetHostGoogleDeviceID(ctx, host.UUID, "C03other")
	require.NoError(t, err)
	require.Empty(t, id)

	// same value again is idempotent, in any casing, and keeps the stored spelling and names
	for _, same := range []string{"device-a", "DEVICE-A"} {
		changed, err = ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, same, []string{"devices/device-a/deviceUsers/3"})
		require.NoError(t, err)
		require.False(t, changed)
	}
	requireID(host, "device-a")
	require.Len(t, googleDeviceUserStates(t, ds, host.UUID), 2)

	// a different value is refused and the stored one is kept
	_, err = ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, "device-b", []string{"devices/device-b/deviceUsers/1"})
	var conflictErr *fleet.HostGoogleDeviceIDConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Equal(t, "device-a", conflictErr.StoredID) //nolint:nilaway // cannot be nil due to require.ErrorAs above
	var statusErr interface{ Status() int }
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, http.StatusConflict, statusErr.Status()) //nolint:nilaway // cannot be nil due to require.ErrorAs above
	var invalidErr interface{ Invalid() []map[string]string }
	require.ErrorAs(t, err, &invalidErr)
	require.Equal(t, []map[string]string{{
		"name":   "google_device_id",
		"reason": "This host already has a different Google device ID.",
	}}, invalidErr.Invalid()) //nolint:nilaway // cannot be nil due to require.ErrorAs above
	requireID(host, "device-a")
	require.Len(t, googleDeviceUserStates(t, ds, host.UUID), 2)

	// other hosts are independent, but can't take an ID that's bound to another host, in any casing
	for _, taken := range []string{"device-a", "DEVICE-A"} {
		_, err = ds.SetHostGoogleDeviceID(ctx, other.UUID, googleTestCustomer, taken, []string{"devices/device-a/deviceUsers/9"})
		requireOtherHostConflict(t, err)
		requireID(other, "")
		require.False(t, googleDeviceRowExists(t, ds, other.UUID))
	}
	// nor a device with a name stored on another host, and nothing is written
	_, err = ds.SetHostGoogleDeviceID(ctx, other.UUID, googleTestCustomer, "device-x", []string{"devices/device-x/deviceUsers/1", names[0]})
	var nameConflict *fleet.GoogleConditionalAccessDeviceUserConflictError
	require.ErrorAs(t, err, &nameConflict)
	requireID(other, "")
	_, err = ds.SetHostGoogleDeviceID(ctx, other.UUID, googleTestCustomer, "device-b", []string{"devices/device-b/deviceUsers/1"})
	require.NoError(t, err)

	// deleting only removes the given value
	deleted, err := ds.DeleteHostGoogleDeviceID(ctx, host.UUID, "device-b")
	require.NoError(t, err)
	require.False(t, deleted)
	requireID(host, "device-a")

	// deleting clears the ID at once and queues the names for the unmark
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.UUID, "device-a")
	require.NoError(t, err)
	require.True(t, deleted)
	requireID(host, "")
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		names[0]: fleet.GoogleConditionalAccessDeviceUserUnmanaging,
		names[1]: fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, googleDeviceUserStates(t, ds, host.UUID))

	// a new ID can be set right away, while the old names wait for the unmark
	_, err = ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, "device-c", []string{"devices/device-c/deviceUsers/1"})
	require.NoError(t, err)
	requireID(host, "device-c")
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		names[0]:                         fleet.GoogleConditionalAccessDeviceUserUnmanaging,
		names[1]:                         fleet.GoogleConditionalAccessDeviceUserUnmanaging,
		"devices/device-c/deviceUsers/1": fleet.GoogleConditionalAccessDeviceUserPending,
	}, googleDeviceUserStates(t, ds, host.UUID))

	// deleting when no ID is set is a no-op
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.UUID, "device-c")
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.UUID, "device-c")
	require.NoError(t, err)
	require.False(t, deleted)

	requireID(other, "device-b")

	// an ID with no names left goes with its row when it is deleted
	migrated := test.NewHost(t, ds, "h3", "", "h3key", "h3uuid", time.Now())
	insertMigratedGoogleDeviceID(t, ds, migrated.UUID, "device-m")
	requireID(migrated, "device-m")
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, migrated.UUID, "device-m")
	require.NoError(t, err)
	require.True(t, deleted)
	require.False(t, googleDeviceRowExists(t, ds, migrated.UUID))
}

func testHostGoogleDeviceIDKeptOnAppleReenrollment(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "iphone", "", "iphonekey", "iphone-uuid", time.Now())
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET platform = 'ios', hardware_serial = 'iphone-serial' WHERE id = ?`, host.ID)
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)
	_, err = ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, "device-a", []string{"devices/device-a/deviceUsers/1"})
	require.NoError(t, err)

	// the device enrolls again with the same serial and UDID, so the host row is reused
	reenroll := func() {
		err := ds.MDMAppleUpsertHost(ctx, &fleet.Host{
			UUID:           "iphone-uuid",
			HardwareSerial: "iphone-serial",
			Platform:       "ios",
		}, fleet.PersonalEnrollmentTypeNone)
		require.NoError(t, err)
		hosts := listHostsCheckCount(t, ds, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{}, 1)
		require.Equal(t, host.ID, hosts[0].ID)
		id, err := ds.GetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer)
		require.NoError(t, err)
		require.Equal(t, "device-a", id)
	}

	reenroll()

	// only the delete endpoint removes the ID, even after a wipe
	appleStore, err := ds.NewMDMAppleMDMStorage()
	require.NoError(t, err)
	cmd := &mdm.Command{CommandUUID: uuid.NewString(), Raw: []byte("<?xml")}
	cmd.Command.RequestType = "EraseDevice"
	require.NoError(t, appleStore.EnqueueDeviceWipeCommand(ctx, host, cmd))
	reenroll()
}

func testHostGoogleDeviceIDKeptOnAndroidReenrollment(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	android, err := ds.NewAndroidHost(ctx, createAndroidHost("es-id-reenroll"), false)
	require.NoError(t, err)
	_, err = ds.SetHostGoogleDeviceID(ctx, android.Host.UUID, googleTestCustomer, "device-a", []string{"devices/device-a/deviceUsers/1"})
	require.NoError(t, err)

	requireID := func(hostUUID string, want string) {
		id, err := ds.GetHostGoogleDeviceID(ctx, hostUUID, googleTestCustomer)
		require.NoError(t, err)
		require.Equal(t, want, id)
	}

	require.NoError(t, ds.UpdateAndroidHost(ctx, android, false, false))
	requireID(android.Host.UUID, "device-a")

	// a factory-reset device enrolls again with the same enterprise-specific ID
	require.NoError(t, ds.UpdateAndroidHost(ctx, android, true, false))
	requireID(android.Host.UUID, "device-a")

	// a host enrolled by the Fleet Android agent is reused when it enrolls in MDM
	agentHost := test.NewHost(t, ds, "agent", "", "agentkey", "es-id-agent", time.Now())
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET platform = 'android' WHERE id = ?`, agentHost.ID)
	require.NoError(t, err)
	_, err = ds.SetHostGoogleDeviceID(ctx, agentHost.UUID, googleTestCustomer, "device-b", []string{"devices/device-b/deviceUsers/1"})
	require.NoError(t, err)
	reused, err := ds.NewAndroidHost(ctx, createAndroidHost("es-id-agent"), false)
	require.NoError(t, err)
	require.Equal(t, agentHost.ID, reused.Host.ID)
	requireID(reused.Host.UUID, "device-b")
}

func testHostGoogleDeviceIDConcurrentDifferentValues(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "h1", "", "h1key", "h1uuid", time.Now())

	values := []string{"device-a", "device-b"}
	errs := make([]error, len(values))
	var wg sync.WaitGroup
	for i, v := range values {
		wg.Go(func() {
			_, errs[i] = ds.SetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer, v, []string{"devices/" + v + "/deviceUsers/1"})
		})
	}
	wg.Wait()

	// exactly one bind wins and the other is refused with a conflict
	var winners []string
	for i, err := range errs {
		if err == nil {
			winners = append(winners, values[i])
			continue
		}
		var statusErr interface{ Status() int }
		require.ErrorAs(t, err, &statusErr)
		require.Equal(t, http.StatusConflict, statusErr.Status()) //nolint:nilaway // cannot be nil due to require.ErrorAs above
	}
	require.Len(t, winners, 1)
	id, err := ds.GetHostGoogleDeviceID(ctx, host.UUID, googleTestCustomer)
	require.NoError(t, err)
	require.Equal(t, winners[0], id)
}

func testHostGoogleDeviceIDConcurrentSameValueDifferentHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hosts := []*fleet.Host{
		test.NewHost(t, ds, "h1", "", "h1key", "h1uuid", time.Now()),
		test.NewHost(t, ds, "h2", "", "h2key", "h2uuid", time.Now()),
	}

	errs := make([]error, len(hosts))
	var wg sync.WaitGroup
	for i, h := range hosts {
		wg.Go(func() {
			_, errs[i] = ds.SetHostGoogleDeviceID(ctx, h.UUID, googleTestCustomer, "device-a", []string{fmt.Sprintf("devices/device-a/deviceUsers/%d", i)})
		})
	}
	wg.Wait()

	// exactly one host gets the ID and the other is refused
	var winners []uint
	for i, err := range errs {
		if err == nil {
			winners = append(winners, hosts[i].ID)
			continue
		}
		requireOtherHostConflict(t, err)
	}
	require.Len(t, winners, 1)
	for _, h := range hosts {
		id, err := ds.GetHostGoogleDeviceID(ctx, h.UUID, googleTestCustomer)
		require.NoError(t, err)
		if h.ID == winners[0] {
			require.Equal(t, "device-a", id)
		} else {
			require.Empty(t, id)
		}
	}
}

const googleTestCustomer = "C02cajzvg"

// insertMigratedGoogleDeviceID inserts a row as the migration from host_google_device_ids
// leaves it: a device ID with no customer and no names.
func insertMigratedGoogleDeviceID(t *testing.T, ds *Datastore, hostUUID, googleDeviceID string) {
	t.Helper()
	_, err := ds.writer(t.Context()).ExecContext(t.Context(),
		`INSERT INTO google_conditional_access_devices (host_uuid, customer_id, google_device_id) VALUES (?, '', ?)`, hostUUID, googleDeviceID)
	require.NoError(t, err)
}

func requireOtherHostConflict(t *testing.T, err error) {
	t.Helper()
	var statusErr interface{ Status() int }
	require.ErrorAs(t, err, &statusErr)
	require.Equal(t, http.StatusConflict, statusErr.Status()) //nolint:nilaway // cannot be nil due to require.ErrorAs above
	var invalidErr interface{ Invalid() []map[string]string }
	require.ErrorAs(t, err, &invalidErr)
	require.Equal(t, []map[string]string{{
		"name":   "google_device_id",
		"reason": "This Google device ID is already used by another host.",
	}}, invalidErr.Invalid()) //nolint:nilaway // cannot be nil due to require.ErrorAs above
}
