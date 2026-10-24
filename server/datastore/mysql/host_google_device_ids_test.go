package mysql

import (
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

	id, err := ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, id)

	changed, err := ds.SetHostGoogleDeviceID(ctx, host.ID, "device-a")
	require.NoError(t, err)
	require.True(t, changed)
	id, err = ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "device-a", id)

	// same value again is idempotent
	changed, err = ds.SetHostGoogleDeviceID(ctx, host.ID, "device-a")
	require.NoError(t, err)
	require.False(t, changed)

	// a different value is refused and the stored one is kept
	_, err = ds.SetHostGoogleDeviceID(ctx, host.ID, "device-b")
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
	id, err = ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "device-a", id)

	// other hosts are independent, but can't take an ID that's bound to another host, in any casing
	for _, taken := range []string{"device-a", "DEVICE-A"} {
		_, err = ds.SetHostGoogleDeviceID(ctx, other.ID, taken)
		requireOtherHostConflict(t, err)
		id, err = ds.GetHostGoogleDeviceID(ctx, other.ID)
		require.NoError(t, err)
		require.Empty(t, id)
	}
	_, err = ds.SetHostGoogleDeviceID(ctx, other.ID, "device-b")
	require.NoError(t, err)

	// deleting only removes the given value, then allows a new one
	deleted, err := ds.DeleteHostGoogleDeviceID(ctx, host.ID, "device-b")
	require.NoError(t, err)
	require.False(t, deleted)
	id, err = ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "device-a", id)
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.ID, "device-a")
	require.NoError(t, err)
	require.True(t, deleted)
	id, err = ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Empty(t, id)
	_, err = ds.SetHostGoogleDeviceID(ctx, host.ID, "device-c")
	require.NoError(t, err)
	id, err = ds.GetHostGoogleDeviceID(ctx, host.ID)
	require.NoError(t, err)
	require.Equal(t, "device-c", id)

	// deleting when no ID is set is a no-op
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.ID, "device-c")
	require.NoError(t, err)
	require.True(t, deleted)
	deleted, err = ds.DeleteHostGoogleDeviceID(ctx, host.ID, "device-c")
	require.NoError(t, err)
	require.False(t, deleted)

	id, err = ds.GetHostGoogleDeviceID(ctx, other.ID)
	require.NoError(t, err)
	require.Equal(t, "device-b", id)
}

func testHostGoogleDeviceIDKeptOnAppleReenrollment(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "iphone", "", "iphonekey", "iphone-uuid", time.Now())
	_, err := ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET platform = 'ios', hardware_serial = 'iphone-serial' WHERE id = ?`, host.ID)
	require.NoError(t, err)
	nanoEnroll(t, ds, host, false)
	_, err = ds.SetHostGoogleDeviceID(ctx, host.ID, "device-a")
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
		id, err := ds.GetHostGoogleDeviceID(ctx, host.ID)
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
	_, err = ds.SetHostGoogleDeviceID(ctx, android.Host.ID, "device-a")
	require.NoError(t, err)

	requireID := func(hostID uint, want string) {
		id, err := ds.GetHostGoogleDeviceID(ctx, hostID)
		require.NoError(t, err)
		require.Equal(t, want, id)
	}

	require.NoError(t, ds.UpdateAndroidHost(ctx, android, false, false))
	requireID(android.Host.ID, "device-a")

	// a factory-reset device enrolls again with the same enterprise-specific ID
	require.NoError(t, ds.UpdateAndroidHost(ctx, android, true, false))
	requireID(android.Host.ID, "device-a")

	// a host enrolled by the Fleet Android agent is reused when it enrolls in MDM
	agentHost := test.NewHost(t, ds, "agent", "", "agentkey", "es-id-agent", time.Now())
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET platform = 'android' WHERE id = ?`, agentHost.ID)
	require.NoError(t, err)
	_, err = ds.SetHostGoogleDeviceID(ctx, agentHost.ID, "device-b")
	require.NoError(t, err)
	reused, err := ds.NewAndroidHost(ctx, createAndroidHost("es-id-agent"), false)
	require.NoError(t, err)
	require.Equal(t, agentHost.ID, reused.Host.ID)
	requireID(agentHost.ID, "device-b")
}

func testHostGoogleDeviceIDConcurrentDifferentValues(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "h1", "", "h1key", "h1uuid", time.Now())

	values := []string{"device-a", "device-b"}
	errs := make([]error, len(values))
	var wg sync.WaitGroup
	for i, v := range values {
		wg.Go(func() {
			_, errs[i] = ds.SetHostGoogleDeviceID(ctx, host.ID, v)
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
	id, err := ds.GetHostGoogleDeviceID(ctx, host.ID)
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
			_, errs[i] = ds.SetHostGoogleDeviceID(ctx, h.ID, "device-a")
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
		id, err := ds.GetHostGoogleDeviceID(ctx, h.ID)
		require.NoError(t, err)
		if h.ID == winners[0] {
			require.Equal(t, "device-a", id)
		} else {
			require.Empty(t, id)
		}
	}
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
