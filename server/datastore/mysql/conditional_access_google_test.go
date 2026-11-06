package mysql

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessGoogle(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Integration", testConditionalAccessGoogleIntegration},
		{"DeviceUsers", testConditionalAccessGoogleDeviceUsers},
		{"ConcurrentFirstSave", testConditionalAccessGoogleConcurrentFirstSave},
		{"RegistrationAttempts", testConditionalAccessGoogleRegistrationAttempts},
		{"RecordRegistration", testConditionalAccessGoogleRecordRegistration},
		{"SyncQueries", testConditionalAccessGoogleSyncQueries},
		{"HostHooks", testConditionalAccessGoogleHostHooks},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testConditionalAccessGoogleIntegration(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	_, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.True(t, fleet.IsNotFound(err))

	pending := &fleet.ConditionalAccessGoogleIntegration{
		CustomerID: "C02cajzvg",
		AdminEmail: "admin@example.com",
		State:      fleet.ConditionalAccessGoogleStateDelegationFailed, ErrorMessage: new("delegation failed"),
	}
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, pending))
	got, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Equal(t, pending, got)
	require.False(t, got.Registered())

	// Saving again updates the single row in place.
	connected := &fleet.ConditionalAccessGoogleIntegration{
		CustomerID:        "C03other",
		AdminEmail:        "admin2@example.com",
		ProxyServerSecret: "secret",
		OAuthClientID:     "client-id",
		State:             fleet.ConditionalAccessGoogleStateConnected,
	}
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, connected))
	// Saving the same values again is still a no-op insert-wise.
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, connected))
	got, err = ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Equal(t, connected, got)
	require.True(t, got.Registered())
	var ids []uint
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &ids, `SELECT id FROM google_conditional_access_integrations`))
	require.Equal(t, []uint{1}, ids)

	require.NoError(t, ds.DeleteGoogleConditionalAccessIntegration(ctx))
	_, err = ds.GetGoogleConditionalAccessIntegration(ctx)
	require.True(t, fleet.IsNotFound(err))
	// Deleting with nothing stored is not an error.
	require.NoError(t, ds.DeleteGoogleConditionalAccessIntegration(ctx))
}

func testConditionalAccessGoogleConcurrentFirstSave(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	const n = 10
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			errs[i] = ds.SaveGoogleConditionalAccessIntegration(ctx, &fleet.ConditionalAccessGoogleIntegration{
				CustomerID: fmt.Sprintf("C%02d", i),
				AdminEmail: "admin@example.com",
				State:      fleet.ConditionalAccessGoogleStatePending,
			})
		})
	}
	close(start)
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}

	var count int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM google_conditional_access_integrations`))
	require.Equal(t, 1, count)
	got, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Regexp(t, `^C\d{2}$`, got.CustomerID)
}

func testConditionalAccessGoogleDeviceUsers(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	names, err := ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C02cajzvg")
	require.NoError(t, err)
	require.Empty(t, names)

	insertDevice := func(customerID, hostUUID string, names ...string) {
		_, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO google_conditional_access_devices (host_uuid, customer_id) VALUES (?, ?)`, hostUUID, customerID)
		require.NoError(t, err)
		for _, name := range names {
			_, err := ds.writer(ctx).ExecContext(ctx,
				`INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name) VALUES (?, ?)`, hostUUID, name)
			require.NoError(t, err)
		}
	}
	countDevices := func() int {
		var n int
		require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &n, `SELECT COUNT(*) FROM google_conditional_access_devices`))
		return n
	}
	insertDevice("C02cajzvg", "uuid-1", "devices/d1/deviceUsers/1", "devices/d1/deviceUsers/2")
	insertDevice("C02cajzvg", "uuid-2", "devices/d2/deviceUsers/1")
	// Names are unique across hosts, case-sensitively.
	_, err = ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name) VALUES (?, ?)`, "uuid-1", "devices/d2/deviceUsers/1")
	require.True(t, IsDuplicate(err))
	insertDevice("C02cajzvg", "uuid-3", "devices/D2/deviceUsers/1")
	// Rows left by a previous customer are listed only for that customer.
	insertDevice("C03other", "uuid-4", "devices/d4/deviceUsers/1")
	insertDevice("C03other", "uuid-5", "devices/d5/deviceUsers/1")

	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C02cajzvg")
	require.NoError(t, err)
	require.Equal(t, []string{"devices/D2/deviceUsers/1", "devices/d1/deviceUsers/1", "devices/d1/deviceUsers/2", "devices/d2/deviceUsers/1"}, names)
	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C03other")
	require.NoError(t, err)
	require.Equal(t, []string{"devices/d4/deviceUsers/1", "devices/d5/deviceUsers/1"}, names)

	// Deleting a device row takes its names with it.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM google_conditional_access_devices WHERE host_uuid = ?`, "uuid-3")
	require.NoError(t, err)
	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C02cajzvg")
	require.NoError(t, err)
	require.Len(t, names, 3)

	// The rows of other customers are deleted in batches, keeping the integration.
	integration := &fleet.ConditionalAccessGoogleIntegration{CustomerID: "C02cajzvg", AdminEmail: "a@example.com"}
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, integration))
	deleted, err := ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C03other")
	require.NoError(t, err)
	require.Len(t, names, 1)
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 10)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 10)
	require.NoError(t, err)
	require.Zero(t, deleted)
	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C03other")
	require.NoError(t, err)
	require.Empty(t, names)
	require.Equal(t, 2, countDevices())
	got, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Equal(t, integration, got)

	// An empty customer ID deletes every row.
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "", 10)
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
	require.Zero(t, countDevices())

	// Deleting the integration removes everything.
	insertDevice("C02cajzvg", "uuid-1", "devices/d1/deviceUsers/1")
	require.NoError(t, ds.DeleteGoogleConditionalAccessIntegration(ctx))
	names, err = ds.ListGoogleConditionalAccessDeviceUserNames(ctx, "C02cajzvg")
	require.NoError(t, err)
	require.Empty(t, names)
	require.Zero(t, countDevices())
}

func googleTokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

func googleDeviceUserStates(t *testing.T, ds *Datastore, hostUUID string) map[string]fleet.GoogleConditionalAccessDeviceUserState {
	var rows []fleet.GoogleConditionalAccessDeviceUser
	require.NoError(t, sqlx.SelectContext(t.Context(), ds.reader(t.Context()), &rows,
		`SELECT host_uuid, device_user_name, state, error_message, updated_at FROM google_conditional_access_device_users WHERE host_uuid = ?`, hostUUID))
	states := make(map[string]fleet.GoogleConditionalAccessDeviceUserState, len(rows))
	for _, r := range rows {
		states[r.Name] = r.State
	}
	return states
}

func googleDeviceRowExists(t *testing.T, ds *Datastore, hostUUID string) bool {
	var n int
	require.NoError(t, sqlx.GetContext(t.Context(), ds.reader(t.Context()), &n,
		`SELECT COUNT(*) FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID))
	return n > 0
}

func testConditionalAccessGoogleRegistrationAttempts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"
	expires := time.Now().Add(10 * time.Minute)

	_, err := ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.True(t, fleet.IsNotFound(err))

	// Each attempt is counted, up to the cap.
	for i := range fleet.GoogleConditionalAccessMaxRegistrationAttempts {
		ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash(fmt.Sprint("t", i)), expires)
		require.NoError(t, err)
		require.True(t, ok)
	}
	dev, err := ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.EqualValues(t, fleet.GoogleConditionalAccessMaxRegistrationAttempts, dev.RegistrationAttempts)
	require.NotNil(t, dev.FirstRegistrationAttemptAt)
	require.Empty(t, dev.DeviceUsers)
	require.False(t, dev.Registered())

	// The next one is refused and leaves the last token in place.
	ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-blocked"), expires)
	require.NoError(t, err)
	require.False(t, ok)
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-blocked"))
	require.True(t, fleet.IsNotFound(err))

	// The row isn't visible to another customer.
	_, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", "C03other")
	require.True(t, fleet.IsNotFound(err))

	// Once the first attempt is older than the window, the count starts over.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_devices
		SET first_registration_attempt_at = NOW(6) - INTERVAL 25 HOUR WHERE host_uuid = ?`, "uuid-1")
	require.NoError(t, err)
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-new"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.EqualValues(t, 1, dev.RegistrationAttempts)
	require.WithinDuration(t, time.Now(), *dev.FirstRegistrationAttemptAt, time.Minute)

	// Only the latest token is valid. It is consumed once, for its customer only, and
	// consuming it resets the counter.
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t2"))
	require.True(t, fleet.IsNotFound(err))
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, "C03other", googleTokenHash("t-new"))
	require.True(t, fleet.IsNotFound(err))
	hostUUID, err := ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-new"))
	require.NoError(t, err)
	require.Equal(t, "uuid-1", hostUUID)
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-new"))
	require.True(t, fleet.IsNotFound(err))
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.Zero(t, dev.RegistrationAttempts)
	require.Nil(t, dev.FirstRegistrationAttemptAt)

	// An expired token is refused.
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-expired"), time.Now().Add(-time.Second))
	require.NoError(t, err)
	require.True(t, ok)
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-expired"))
	require.True(t, fleet.IsNotFound(err))

	// Android attempts carry no token, and the reset clears the counter.
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, nil, time.Time{})
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, nil, time.Time{})
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, ds.ResetGoogleConditionalAccessRegistrationAttempts(ctx, "uuid-2", customer))
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", customer)
	require.NoError(t, err)
	require.Zero(t, dev.RegistrationAttempts)
	// Resetting a row that doesn't exist is a no-op.
	require.NoError(t, ds.ResetGoogleConditionalAccessRegistrationAttempts(ctx, "uuid-none", customer))

	// Another customer takes the row over: its names go and the counter starts over.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, []string{"devices/d2/deviceUsers/1"}))
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", "C03other", googleTokenHash("t-other"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	_, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", customer)
	require.True(t, fleet.IsNotFound(err))
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", "C03other")
	require.NoError(t, err)
	require.EqualValues(t, 1, dev.RegistrationAttempts)
	require.Empty(t, dev.DeviceUsers)
}

func testConditionalAccessGoogleRecordRegistration(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t1"), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, ok)

	// Every name is stored as pending, and the token and the counter are cleared.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, []string{"devices/d1/deviceUsers/a", "devices/d1/deviceUsers/b"}))
	dev, err := ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.Zero(t, dev.RegistrationAttempts)
	require.Nil(t, dev.FirstRegistrationAttemptAt)
	require.Len(t, dev.DeviceUsers, 2)
	require.True(t, dev.Registered())
	require.False(t, dev.Managed())
	for _, du := range dev.DeviceUsers {
		require.Equal(t, fleet.GoogleConditionalAccessDeviceUserPending, du.State)
	}
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t1"))
	require.True(t, fleet.IsNotFound(err))

	ok, err = ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/d1/deviceUsers/a",
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged, nil)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/d1/deviceUsers/b",
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged, nil)
	require.NoError(t, err)
	require.True(t, ok)
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.True(t, dev.Managed())

	// A name stored on another host is refused and nothing is written.
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, nil, time.Time{})
	require.NoError(t, err)
	require.True(t, ok)
	err = ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, []string{"devices/d2/deviceUsers/a", "devices/d1/deviceUsers/b"})
	var conflict *fleet.GoogleConditionalAccessDeviceUserConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict)
	require.Equal(t, "devices/d1/deviceUsers/b", conflict.Name)
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", customer)
	require.NoError(t, err)
	require.Empty(t, dev.DeviceUsers)
	require.EqualValues(t, 1, dev.RegistrationAttempts)
	require.Len(t, googleDeviceUserStates(t, ds, "uuid-1"), 2)

	// A name left by a previous customer doesn't conflict: it is dropped.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_devices SET customer_id = 'C03old' WHERE host_uuid = ?`, "uuid-1")
	require.NoError(t, err)
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, []string{"devices/d1/deviceUsers/b"}))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/d1/deviceUsers/a": fleet.GoogleConditionalAccessDeviceUserManaged,
	}, googleDeviceUserStates(t, ds, "uuid-1"))
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_devices SET customer_id = ? WHERE host_uuid = ?`, customer, "uuid-1")
	require.NoError(t, err)

	// Registering again: a re-registered name queued for unmark goes back to pending,
	// a name no longer returned is queued for unmark, and a stale one is deleted.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", customer,
		[]string{"devices/d3/deviceUsers/keep", "devices/d3/deviceUsers/drop", "devices/d3/deviceUsers/stale", "devices/d3/deviceUsers/back"}))
	for name, to := range map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/d3/deviceUsers/keep":  fleet.GoogleConditionalAccessDeviceUserManaged,
		"devices/d3/deviceUsers/drop":  fleet.GoogleConditionalAccessDeviceUserManaged,
		"devices/d3/deviceUsers/stale": fleet.GoogleConditionalAccessDeviceUserStale,
		"devices/d3/deviceUsers/back":  fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	} {
		ok, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, name, fleet.GoogleConditionalAccessDeviceUserPending, to, new("err"))
		require.NoError(t, err)
		require.True(t, ok)
	}
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", customer,
		[]string{"devices/d3/deviceUsers/keep", "devices/d3/deviceUsers/back", "devices/d3/deviceUsers/new"}))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/d3/deviceUsers/keep": fleet.GoogleConditionalAccessDeviceUserPending,
		"devices/d3/deviceUsers/back": fleet.GoogleConditionalAccessDeviceUserPending,
		"devices/d3/deviceUsers/new":  fleet.GoogleConditionalAccessDeviceUserPending,
		"devices/d3/deviceUsers/drop": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, googleDeviceUserStates(t, ds, "uuid-3"))
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-3", customer)
	require.NoError(t, err)
	for _, du := range dev.DeviceUsers {
		if du.State == fleet.GoogleConditionalAccessDeviceUserPending {
			require.Nil(t, du.ErrorMessage, du.Name)
		}
	}

	// A state update is skipped when the name moved on.
	ok, err = ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/d3/deviceUsers/drop",
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserManaged, nil)
	require.NoError(t, err)
	require.False(t, ok)

	// An empty list is refused.
	require.Error(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", customer, nil))
}

func testConditionalAccessGoogleSyncQueries(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	live := newTestHostWithPlatform(t, ds, "live", "ios", nil)
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, live.UUID, customer, []string{"devices/d1/deviceUsers/a", "devices/d1/deviceUsers/b"}))
	// A row whose host was deleted.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "gone-uuid", customer, []string{"devices/d2/deviceUsers/a"}))
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "other-uuid", "C03other", []string{"devices/d3/deviceUsers/a"}))

	list := func(f fleet.GoogleConditionalAccessDeviceUserFilter) []string {
		dus, err := ds.ListGoogleConditionalAccessDeviceUsers(ctx, f)
		require.NoError(t, err)
		var names []string
		for _, du := range dus {
			names = append(names, du.Name)
		}
		return names
	}
	pending := []fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserPending}
	require.Equal(t, []string{"devices/d1/deviceUsers/a", "devices/d1/deviceUsers/b", "devices/d2/deviceUsers/a"},
		list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: customer, States: pending, Limit: 10}))
	require.Equal(t, []string{"devices/d1/deviceUsers/a", "devices/d1/deviceUsers/b"},
		list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: customer, States: pending, HostExists: true, Limit: 10}))
	require.Equal(t, []string{"devices/d1/deviceUsers/b"},
		list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: customer, States: pending, AfterName: "devices/d1/deviceUsers/a", Limit: 1}))
	require.Equal(t, []string{"devices/d3/deviceUsers/a"},
		list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: "C03other", States: pending, Limit: 10}))
	require.Empty(t, list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: customer,
		States: []fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserUnmanaging}, Limit: 10}))

	// Queueing the unmark of the deleted host's row.
	require.NoError(t, ds.QueueGoogleConditionalAccessUnmanage(ctx, []string{"gone-uuid"}))
	unmanaging := []fleet.GoogleConditionalAccessDeviceUserState{fleet.GoogleConditionalAccessDeviceUserUnmanaging}
	require.Equal(t, []string{"devices/d2/deviceUsers/a"},
		list(fleet.GoogleConditionalAccessDeviceUserFilter{CustomerID: customer, States: unmanaging, Limit: 10}))

	// Deleting a name only when it is in the expected state.
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d2/deviceUsers/a", fleet.GoogleConditionalAccessDeviceUserPending, true))
	require.Len(t, googleDeviceUserStates(t, ds, "gone-uuid"), 1)
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d2/deviceUsers/a", fleet.GoogleConditionalAccessDeviceUserUnmanaging, true))
	require.False(t, googleDeviceRowExists(t, ds, "gone-uuid"))
	// Without deleteEmptyDevice the row stays.
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d1/deviceUsers/a", fleet.GoogleConditionalAccessDeviceUserPending, false))
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d1/deviceUsers/b", fleet.GoogleConditionalAccessDeviceUserPending, false))
	require.True(t, googleDeviceRowExists(t, ds, live.UUID))

	// Bookmark-era hosts: a Google device ID and no names for the customer.
	bm1 := newTestHostWithPlatform(t, ds, "bm1", "ios", nil)
	bm2 := newTestHostWithPlatform(t, ds, "bm2", "android", nil)
	registered := newTestHostWithPlatform(t, ds, "registered", "ios", nil)
	personal := newTestHostWithPlatform(t, ds, "personal", "ios", nil)
	for i, h := range []*fleet.Host{bm1, bm2, registered, live, personal} {
		_, err := ds.SetHostGoogleDeviceID(ctx, h.ID, fmt.Sprint("gdid-", i))
		require.NoError(t, err)
		automatic := h != personal
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", automatic, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, registered.UUID, customer, []string{"devices/gdid-2/deviceUsers/a"}))
	got, err := ds.ListGoogleConditionalAccessBookmarkHosts(ctx, customer, "", 10)
	require.NoError(t, err)
	want := []fleet.GoogleConditionalAccessBookmarkHost{
		{HostUUID: bm1.UUID, GoogleDeviceID: "gdid-0"},
		{HostUUID: bm2.UUID, GoogleDeviceID: "gdid-1"},
		{HostUUID: live.UUID, GoogleDeviceID: "gdid-3"},
	}
	slices.SortFunc(want, func(a, b fleet.GoogleConditionalAccessBookmarkHost) int {
		return strings.Compare(a.HostUUID, b.HostUUID)
	})
	require.Equal(t, want, got)
	got, err = ds.ListGoogleConditionalAccessBookmarkHosts(ctx, customer, want[0].HostUUID, 1)
	require.NoError(t, err)
	require.Equal(t, want[1:2], got)
	// Names of another customer don't count.
	got, err = ds.ListGoogleConditionalAccessBookmarkHosts(ctx, "C03other", "", 10)
	require.NoError(t, err)
	require.Len(t, got, 4)
}

func testConditionalAccessGoogleHostHooks(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	register := func(h *fleet.Host) string {
		name := "devices/" + h.UUID + "/deviceUsers/a"
		require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, h.UUID, customer, []string{name}))
		return name
	}
	requireQueued := func(h *fleet.Host) {
		t.Helper()
		require.True(t, googleDeviceRowExists(t, ds, h.UUID))
		require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
			"devices/" + h.UUID + "/deviceUsers/a": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
		}, googleDeviceUserStates(t, ds, h.UUID))
	}
	unregistered := func(h *fleet.Host) {
		ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, h.UUID, customer, nil, time.Time{})
		require.NoError(t, err)
		require.True(t, ok)
	}

	// Single delete: a registered host's row stays, queued; an unregistered host's goes.
	h1 := newTestHostWithPlatform(t, ds, "h1", "ios", nil)
	h2 := newTestHostWithPlatform(t, ds, "h2", "ios", nil)
	register(h1)
	unregistered(h2)
	require.NoError(t, ds.DeleteHost(ctx, h1.ID))
	require.NoError(t, ds.DeleteHost(ctx, h2.ID))
	requireQueued(h1)
	require.False(t, googleDeviceRowExists(t, ds, h2.UUID))

	// Bulk delete, with a stale name that is dropped.
	h3 := newTestHostWithPlatform(t, ds, "h3", "android", nil)
	h4 := newTestHostWithPlatform(t, ds, "h4", "ipados", nil)
	register(h3)
	stale := register(h4)
	ok, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, stale, fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserStale, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, ds.DeleteHosts(ctx, []uint{h3.ID, h4.ID}))
	requireQueued(h3)
	require.False(t, googleDeviceRowExists(t, ds, h4.UUID))

	// Expired-host cleanup.
	ac, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	ac.HostExpirySettings.HostExpiryEnabled = true
	ac.HostExpirySettings.HostExpiryWindow = 10
	require.NoError(t, ds.SaveAppConfig(ctx, ac))
	h5 := newTestHostWithPlatform(t, ds, "h5", "ios", nil)
	register(h5)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_seen_times SET seen_time = NOW() - INTERVAL 20 DAY WHERE host_id = ?`, h5.ID)
	require.NoError(t, err)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET created_at = NOW() - INTERVAL 20 DAY WHERE id = ?`, h5.ID)
	require.NoError(t, err)
	deleted, err := ds.CleanupExpiredHostsBatch(ctx, 100)
	require.NoError(t, err)
	require.NotEmpty(t, deleted)
	_, err = ds.HostLite(ctx, h5.ID)
	require.True(t, fleet.IsNotFound(err))
	requireQueued(h5)
	ac.HostExpirySettings.HostExpiryEnabled = false
	require.NoError(t, ds.SaveAppConfig(ctx, ac))

	// Apple MDM turn-off.
	h6 := newTestHostWithPlatform(t, ds, "h6", "ios", nil)
	register(h6)
	_, _, err = ds.MDMTurnOff(ctx, h6.UUID)
	require.NoError(t, err)
	requireQueued(h6)

	// Android unenrollment, single and bulk.
	h7 := newTestHostWithPlatform(t, ds, "h7", "android", nil)
	h8 := newTestHostWithPlatform(t, ds, "h8", "android", nil)
	h9 := newTestHostWithPlatform(t, ds, "h9", "ios", nil)
	for _, h := range []*fleet.Host{h7, h8} {
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", false, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}
	register(h7)
	unchanged, err := ds.SetAndroidHostUnenrolled(ctx, h7.ID)
	require.NoError(t, err)
	require.True(t, unchanged)
	requireQueued(h7)
	register(h8)
	register(h9)
	require.NoError(t, ds.BulkSetAndroidHostsUnenrolled(ctx))
	requireQueued(h8)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/" + h9.UUID + "/deviceUsers/a": fleet.GoogleConditionalAccessDeviceUserPending,
	}, googleDeviceUserStates(t, ds, h9.UUID))
}
