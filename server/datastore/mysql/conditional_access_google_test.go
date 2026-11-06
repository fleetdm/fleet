package mysql

import (
	"crypto/sha256"
	"fmt"
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
		{"ConcurrentRegistrationStarts", testConditionalAccessGoogleConcurrentRegistrationStarts},
		{"RecordRegistration", testConditionalAccessGoogleRecordRegistration},
		{"ConcurrentRegistrationRecords", testConditionalAccessGoogleConcurrentRegistrationRecords},
		{"DeviceID", testConditionalAccessGoogleDeviceID},
		{"SyncQueries", testConditionalAccessGoogleSyncQueries},
		{"UnresolvedAndroidHosts", testConditionalAccessGoogleUnresolvedAndroidHosts},
		{"DeleteHosts", testConditionalAccessGoogleDeleteHosts},
		{"UnmarkUnenrolledHosts", testConditionalAccessGoogleUnmarkUnenrolledHosts},
		{"AndroidUnenroll", testConditionalAccessGoogleAndroidUnenroll},
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
		State: fleet.ConditionalAccessGoogleStateRegistrationFailed, ErrorMessage: new("fleetdm.com unavailable"),
	}
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, pending))
	got, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Equal(t, pending, got)
	require.False(t, got.Registered())

	// Saving again updates the single row in place.
	connected := &fleet.ConditionalAccessGoogleIntegration{
		ProxyServerSecret: "secret",
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
				ProxyServerSecret: fmt.Sprintf("secret-%02d", i),
				State:             fleet.ConditionalAccessGoogleStatePending,
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
	require.Regexp(t, `^secret-\d{2}$`, got.ProxyServerSecret)
}

func testConditionalAccessGoogleDeviceUsers(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.Empty(t, googleDeviceUserNames(t, ds, "C02cajzvg"))

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
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO google_conditional_access_device_users (host_uuid, device_user_name) VALUES (?, ?)`, "uuid-1", "devices/d2/deviceUsers/1")
	require.True(t, IsDuplicate(err))
	insertDevice("C02cajzvg", "uuid-3", "devices/D2/deviceUsers/1")
	// Rows left by a previous customer are listed only for that customer.
	insertDevice("C03other", "uuid-4", "devices/d4/deviceUsers/1")
	insertDevice("C03other", "uuid-5", "devices/d5/deviceUsers/1")
	// A row migrated from host_google_device_ids has no customer and is never another customer's.
	insertMigratedGoogleDeviceID(t, ds, "uuid-migrated", "device-migrated")

	require.Equal(t, []string{"devices/D2/deviceUsers/1", "devices/d1/deviceUsers/1", "devices/d1/deviceUsers/2", "devices/d2/deviceUsers/1"},
		googleDeviceUserNames(t, ds, "C02cajzvg"))
	require.Equal(t, []string{"devices/d4/deviceUsers/1", "devices/d5/deviceUsers/1"}, googleDeviceUserNames(t, ds, "C03other"))

	// Deleting a device row takes its names with it.
	_, err = ds.writer(ctx).ExecContext(ctx, `DELETE FROM google_conditional_access_devices WHERE host_uuid = ?`, "uuid-3")
	require.NoError(t, err)
	require.Len(t, googleDeviceUserNames(t, ds, "C02cajzvg"), 3)

	// The rows of other customers are deleted in batches, keeping the integration.
	integration := &fleet.ConditionalAccessGoogleIntegration{ProxyServerSecret: "secret", State: fleet.ConditionalAccessGoogleStateConnected}
	require.NoError(t, ds.SaveGoogleConditionalAccessIntegration(ctx, integration))
	deleted, err := ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 1)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	require.Len(t, googleDeviceUserNames(t, ds, "C03other"), 1)
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 10)
	require.NoError(t, err)
	require.Equal(t, 1, deleted)
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "C02cajzvg", 10)
	require.NoError(t, err)
	require.Zero(t, deleted)
	require.Empty(t, googleDeviceUserNames(t, ds, "C03other"))
	require.Equal(t, 3, countDevices())
	got, err := ds.GetGoogleConditionalAccessIntegration(ctx)
	require.NoError(t, err)
	require.Equal(t, integration, got)

	// An empty customer ID deletes every row that has a customer.
	deleted, err = ds.DeleteGoogleConditionalAccessDevicesExcept(ctx, "", 10)
	require.NoError(t, err)
	require.Equal(t, 2, deleted)
	require.Equal(t, 1, countDevices())
	require.True(t, googleDeviceRowExists(t, ds, "uuid-migrated"))
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

func googleDeviceUserNames(t *testing.T, ds *Datastore, customerID string) []string {
	var names []string
	require.NoError(t, sqlx.SelectContext(t.Context(), ds.reader(t.Context()), &names, `
		SELECT du.device_user_name FROM google_conditional_access_device_users du
		JOIN google_conditional_access_devices d ON d.host_uuid = du.host_uuid
		WHERE d.customer_id = ? ORDER BY du.device_user_name`, customerID))
	return names
}

func googleDeviceID(t *testing.T, ds *Datastore, hostUUID string) string {
	t.Helper()
	var id *string
	require.NoError(t, sqlx.GetContext(t.Context(), ds.reader(t.Context()), &id,
		`SELECT google_device_id FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID))
	if id == nil {
		return ""
	}
	return *id
}

func googleDeviceRowExists(t *testing.T, ds *Datastore, hostUUID string) bool {
	var n int
	require.NoError(t, sqlx.GetContext(t.Context(), ds.reader(t.Context()), &n,
		`SELECT COUNT(*) FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID))
	return n > 0
}

// googleRegistrationAttempts returns the host's attempt counter and when its window started.
func googleRegistrationAttempts(t *testing.T, ds *Datastore, hostUUID string) (uint, *time.Time) {
	t.Helper()
	var row struct {
		Attempts uint       `db:"registration_attempts"`
		FirstAt  *time.Time `db:"first_registration_attempt_at"`
	}
	require.NoError(t, sqlx.GetContext(t.Context(), ds.reader(t.Context()), &row, `
		SELECT registration_attempts, first_registration_attempt_at
		FROM google_conditional_access_devices WHERE host_uuid = ?`, hostUUID))
	return row.Attempts, row.FirstAt
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
	attempts, firstAt := googleRegistrationAttempts(t, ds, "uuid-1")
	require.EqualValues(t, fleet.GoogleConditionalAccessMaxRegistrationAttempts, attempts)
	require.NotNil(t, firstAt)
	dev, err := ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
	require.Empty(t, dev.DeviceUsers)
	require.False(t, dev.Registered())

	// The next one is refused and leaves the last token in place.
	ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-blocked"), expires)
	require.NoError(t, err)
	require.False(t, ok)
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-blocked"))
	require.True(t, fleet.IsNotFound(err))

	// Once the first attempt is older than the window, the count starts over.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_devices
		SET first_registration_attempt_at = NOW(6) - INTERVAL 25 HOUR WHERE host_uuid = ?`, "uuid-1")
	require.NoError(t, err)
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-new"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	attempts, firstAt = googleRegistrationAttempts(t, ds, "uuid-1")
	require.EqualValues(t, 1, attempts)
	require.NotNil(t, firstAt)
	require.WithinDuration(t, time.Now(), *firstAt, time.Minute)

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
	attempts, firstAt = googleRegistrationAttempts(t, ds, "uuid-1")
	require.Zero(t, attempts)
	require.Nil(t, firstAt)

	// An expired token is refused.
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t-expired"), time.Now().Add(-time.Second))
	require.NoError(t, err)
	require.True(t, ok)
	_, err = ds.ConsumeGoogleConditionalAccessRegistrationToken(ctx, customer, googleTokenHash("t-expired"))
	require.True(t, fleet.IsNotFound(err))

	// Another customer takes the row over: its names go and the counter starts over.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, []string{"devices/d2/deviceUsers/1"}))
	for i := range 2 {
		ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, googleTokenHash(fmt.Sprint("t-uuid-2-", i)), expires)
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", "C03other", googleTokenHash("t-other"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	_, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", customer)
	require.True(t, fleet.IsNotFound(err))
	dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-2", "C03other")
	require.NoError(t, err)
	require.Empty(t, dev.DeviceUsers)
	attempts, _ = googleRegistrationAttempts(t, ds, "uuid-2")
	require.EqualValues(t, 1, attempts)
}

func testConditionalAccessGoogleConcurrentRegistrationStarts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"
	expires := time.Now().Add(10 * time.Minute)

	// Half the goroutines race on the same new host, the rest each create their own.
	const n = 20
	start := make(chan struct{})
	errs := make([]error, n)
	started := make([]bool, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			hostUUID := fmt.Sprintf("uuid-%02d", i)
			if i%2 == 0 {
				hostUUID = "uuid-shared"
			}
			started[i], errs[i] = ds.StartGoogleConditionalAccessRegistration(ctx, hostUUID, customer, googleTokenHash(fmt.Sprint("t", i)), expires)
		})
	}
	close(start)
	wg.Wait()

	var sharedStarted int
	for i := range n {
		require.NoError(t, errs[i])
		if i%2 == 0 {
			if started[i] {
				sharedStarted++
			}
		} else {
			require.True(t, started[i])
		}
	}
	require.Equal(t, fleet.GoogleConditionalAccessMaxRegistrationAttempts, sharedStarted)
	attempts, _ := googleRegistrationAttempts(t, ds, "uuid-shared")
	require.EqualValues(t, fleet.GoogleConditionalAccessMaxRegistrationAttempts, attempts)
}

func testConditionalAccessGoogleRecordRegistration(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, googleTokenHash("t1"), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, ok)

	// Every name is stored as pending, and the token and the counter are cleared.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, []string{"devices/d1/deviceUsers/a", "devices/d1/deviceUsers/b"}))
	attempts, firstAt := googleRegistrationAttempts(t, ds, "uuid-1")
	require.Zero(t, attempts)
	require.Nil(t, firstAt)
	dev, err := ds.GetGoogleConditionalAccessDevice(ctx, "uuid-1", customer)
	require.NoError(t, err)
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
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, googleTokenHash("t2"), time.Now().Add(time.Hour))
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
	attempts, _ = googleRegistrationAttempts(t, ds, "uuid-2")
	require.EqualValues(t, 1, attempts)
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

	// A repeated failure with the same message still moves updated_at.
	for range 2 {
		_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE google_conditional_access_device_users
			SET updated_at = NOW(6) - INTERVAL 1 HOUR WHERE device_user_name = ?`, "devices/d3/deviceUsers/keep")
		require.NoError(t, err)
		ok, err = ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/d3/deviceUsers/keep",
			fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserPending, new("HTTP 500"))
		require.NoError(t, err)
		require.True(t, ok)
		dev, err = ds.GetGoogleConditionalAccessDevice(ctx, "uuid-3", customer)
		require.NoError(t, err)
		for _, du := range dev.DeviceUsers {
			if du.Name == "devices/d3/deviceUsers/keep" {
				require.WithinDuration(t, time.Now(), du.UpdatedAt, time.Minute)
			}
		}
	}

	// Names can't be recorded without a customer.
	require.Error(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", "", []string{"devices/d3/deviceUsers/keep"}))
	require.Len(t, googleDeviceUserStates(t, ds, "uuid-3"), 4)

	// An empty list is refused.
	require.Error(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", customer, nil))
}

func testConditionalAccessGoogleConcurrentRegistrationRecords(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"
	const shared = "devices/shared/deviceUsers/a"

	// Half the goroutines register a name of their own, the rest race for the same name.
	const n = 20
	start := make(chan struct{})
	errs := make([]error, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			<-start
			name := fmt.Sprintf("devices/d%02d/deviceUsers/a", i)
			if i%2 == 0 {
				name = shared
			}
			errs[i] = ds.RecordGoogleConditionalAccessRegistration(ctx, fmt.Sprintf("uuid-%02d", i), customer, []string{name})
		})
	}
	close(start)
	wg.Wait()

	var winner string
	for i := range n {
		hostUUID := fmt.Sprintf("uuid-%02d", i)
		if i%2 != 0 {
			require.NoError(t, errs[i])
			require.Len(t, googleDeviceUserStates(t, ds, hostUUID), 1)
			continue
		}
		if errs[i] == nil {
			require.Empty(t, winner)
			winner = hostUUID
			continue
		}
		var conflict *fleet.GoogleConditionalAccessDeviceUserConflictError
		require.ErrorAs(t, errs[i], &conflict)
		require.False(t, googleDeviceRowExists(t, ds, hostUUID))
	}
	require.NotEmpty(t, winner)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		shared: fleet.GoogleConditionalAccessDeviceUserPending,
	}, googleDeviceUserStates(t, ds, winner))
}

func testConditionalAccessGoogleSyncQueries(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	live := newTestHostWithPlatform(t, ds, "live", "ios", nil)
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, live.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
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
	queued, err := ds.QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts(ctx, customer, 10)
	require.NoError(t, err)
	require.Equal(t, 1, queued)
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
		if h == live {
			// A row of the customer whose names are all gone.
			_, err := ds.writer(ctx).ExecContext(ctx,
				`UPDATE google_conditional_access_devices SET google_device_id = 'gdid-3' WHERE host_uuid = ?`, h.UUID)
			require.NoError(t, err)
		} else {
			insertMigratedGoogleDeviceID(t, ds, h.UUID, fmt.Sprint("gdid-", i))
		}
		automatic := h != personal
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, true, "https://example.com", automatic, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	}
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, registered.UUID, customer, []string{"devices/gdid-2/deviceUsers/a"}))
	got, err := ds.ListGoogleConditionalAccessBookmarkHosts(ctx, customer, 0, 10)
	require.NoError(t, err)
	want := []fleet.GoogleConditionalAccessBookmarkHost{
		{HostID: live.ID, HostUUID: live.UUID, GoogleDeviceID: "gdid-3"},
		{HostID: bm1.ID, HostUUID: bm1.UUID, GoogleDeviceID: "gdid-0"},
		{HostID: bm2.ID, HostUUID: bm2.UUID, GoogleDeviceID: "gdid-1"},
	}
	require.Equal(t, want, got)
	got, err = ds.ListGoogleConditionalAccessBookmarkHosts(ctx, customer, want[0].HostID, 1)
	require.NoError(t, err)
	require.Equal(t, want[1:2], got)
	// Another customer only sees the rows that have no customer yet.
	got, err = ds.ListGoogleConditionalAccessBookmarkHosts(ctx, "C03other", 0, 10)
	require.NoError(t, err)
	require.Equal(t, want[1:], got)
}

func testConditionalAccessGoogleDeleteHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	register := func(h *fleet.Host) string {
		name := "devices/" + h.UUID + "/deviceUsers/a"
		require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, h.UUID, customer, []string{name}))
		return name
	}

	registered := newTestHostWithPlatform(t, ds, "registered", "ios", nil)
	register(registered)
	unregistered := newTestHostWithPlatform(t, ds, "unregistered", "ipados", nil)
	ok, err := ds.StartGoogleConditionalAccessRegistration(ctx, unregistered.UUID, customer, googleTokenHash("t1"), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.True(t, ok)
	stale := newTestHostWithPlatform(t, ds, "stale", "android", nil)
	ok, err = ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, register(stale),
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserStale, nil)
	require.NoError(t, err)
	require.True(t, ok)
	migrated := newTestHostWithPlatform(t, ds, "migrated", "ios", nil)
	insertMigratedGoogleDeviceID(t, ds, migrated.UUID, "device-migrated")
	// A host that never enrolled has no UUID: deleting it leaves a row keyed by none alone.
	noUUID := newTestHostWithPlatform(t, ds, "no-uuid", "ios", nil)
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE hosts SET uuid = '' WHERE id = ?`, noUUID.ID)
	require.NoError(t, err)
	insertMigratedGoogleDeviceID(t, ds, "", "device-no-uuid")
	require.Equal(t, registered.UUID, googleDeviceID(t, ds, registered.UUID))
	require.NoError(t, ds.DeleteHosts(ctx, []uint{registered.ID, unregistered.ID, stale.ID, migrated.ID, noUUID.ID}))
	require.Equal(t, "device-no-uuid", googleDeviceID(t, ds, ""))

	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/" + registered.UUID + "/deviceUsers/a": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, googleDeviceUserStates(t, ds, registered.UUID))
	require.False(t, googleDeviceRowExists(t, ds, unregistered.UUID))
	require.False(t, googleDeviceRowExists(t, ds, stale.UUID))
	// The device ID is released at once, also when Fleet never stored names for it.
	require.Empty(t, googleDeviceID(t, ds, registered.UUID))
	require.False(t, googleDeviceRowExists(t, ds, migrated.UUID))
}

func testConditionalAccessGoogleUnmarkUnenrolledHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	register := func(customerID, hostUUID string) {
		require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, hostUUID, customerID,
			[]string{"devices/" + hostUUID + "/deviceUsers/a"}))
	}
	newHost := func(name, platform string, enrolled, companyOwned bool) string {
		h := newTestHostWithPlatform(t, ds, name, platform, nil)
		require.NoError(t, ds.SetOrUpdateMDMData(ctx, h.ID, false, enrolled, "https://example.com", companyOwned,
			fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
		return h.UUID
	}

	// An enrolled host keeps its names, whoever owns it.
	kept := []string{newHost("enrolled", "ios", true, true), newHost("personal", "android", true, false)}
	// A registration that committed after its host unenrolled or was deleted.
	late := []string{
		newHost("unenrolled", "ios", false, false),
		newTestHostWithPlatform(t, ds, "no-mdm", "ipados", nil).UUID,
		"gone-uuid",
	}
	for _, hostUUID := range append(kept, late...) {
		register(customer, hostUUID)
	}
	// A previous customer's rows are left to their own cleanup.
	register("C03other", "other-uuid")

	// The hosts are worked through in batches.
	for _, want := range []int{2, 1, 0} {
		queued, err := ds.QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts(ctx, customer, 2)
		require.NoError(t, err)
		require.Equal(t, want, queued)
	}

	state := func(hostUUID string) fleet.GoogleConditionalAccessDeviceUserState {
		return googleDeviceUserStates(t, ds, hostUUID)["devices/"+hostUUID+"/deviceUsers/a"]
	}
	for _, hostUUID := range late {
		require.Equal(t, fleet.GoogleConditionalAccessDeviceUserUnmanaging, state(hostUUID), hostUUID)
		require.Empty(t, googleDeviceID(t, ds, hostUUID), hostUUID)
	}
	for _, hostUUID := range append(kept, "other-uuid") {
		require.Equal(t, fleet.GoogleConditionalAccessDeviceUserPending, state(hostUUID), hostUUID)
		require.Equal(t, hostUUID, googleDeviceID(t, ds, hostUUID), hostUUID)
	}
}

func testConditionalAccessGoogleDeviceID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = googleTestCustomer
	pending := fleet.GoogleConditionalAccessDeviceUserPending

	// A sign-in sets the ID to the device of the first name and stores the names of
	// every device.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer,
		[]string{"devices/d1/deviceUsers/a", "devices/d2/deviceUsers/a"}))
	require.Equal(t, "d1", googleDeviceID(t, ds, "uuid-1"))
	require.Len(t, googleDeviceUserStates(t, ds, "uuid-1"), 2)
	// The stored ID is kept while one of the names is under it.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer,
		[]string{"devices/d2/deviceUsers/a", "devices/d1/deviceUsers/a"}))
	require.Equal(t, "d1", googleDeviceID(t, ds, "uuid-1"))

	// When the stored device's last name is deleted the ID moves to a device that still
	// has one, and it is cleared with the last name.
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d1/deviceUsers/a", pending, true))
	require.Equal(t, "d2", googleDeviceID(t, ds, "uuid-1"))
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d2/deviceUsers/a", pending, false))
	require.Empty(t, googleDeviceID(t, ds, "uuid-1"))
	require.True(t, googleDeviceRowExists(t, ds, "uuid-1"))

	// A sign-in replaces an ID none of its names are under.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, []string{"devices/d3/deviceUsers/a"}))
	ok, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/d3/deviceUsers/a", pending, fleet.GoogleConditionalAccessDeviceUserStale, nil)
	require.NoError(t, err)
	require.True(t, ok)
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-1", customer, []string{"devices/d4/deviceUsers/a"}))
	require.Equal(t, "d4", googleDeviceID(t, ds, "uuid-1"))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{"devices/d4/deviceUsers/a": pending}, googleDeviceUserStates(t, ds, "uuid-1"))
	// The row goes with its last name once the ID is cleared.
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d4/deviceUsers/a", pending, true))
	require.False(t, googleDeviceRowExists(t, ds, "uuid-1"))

	// A device bound to another host is refused, also when that host has no names yet.
	// The error names the DeviceUser the device came from.
	insertMigratedGoogleDeviceID(t, ds, "uuid-migrated", "d5")
	err = ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-2", customer, []string{"not-a-device-user", "devices/d5/deviceUsers/a"})
	var conflict *fleet.GoogleConditionalAccessDeviceUserConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict)
	require.Equal(t, "devices/d5/deviceUsers/a", conflict.Name)
	require.False(t, googleDeviceRowExists(t, ds, "uuid-2"))

	// The ID moves to a device a previous customer's row still holds, which doesn't count.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-4", customer,
		[]string{"devices/d7/deviceUsers/a", "devices/d8/deviceUsers/a"}))
	_, err = ds.writer(ctx).ExecContext(ctx, `INSERT INTO google_conditional_access_devices
		(host_uuid, customer_id, google_device_id) VALUES ('uuid-old', 'C03old', 'd8')`)
	require.NoError(t, err)
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d7/deviceUsers/a", pending, true))
	require.Equal(t, "d8", googleDeviceID(t, ds, "uuid-4"))
	require.Empty(t, googleDeviceID(t, ds, "uuid-old"))
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/d8/deviceUsers/a", pending, true))

	// A migrated row keeps its ID when the host's first status call claims it for the
	// customer, and a row left by another customer doesn't.
	expires := time.Now().Add(time.Hour)
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-migrated", customer, googleTokenHash("t1"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "d5", googleDeviceID(t, ds, "uuid-migrated"))
	id, err := ds.GetHostGoogleDeviceID(ctx, "uuid-migrated", "C03other")
	require.NoError(t, err)
	require.Empty(t, id)
	ok, err = ds.StartGoogleConditionalAccessRegistration(ctx, "uuid-migrated", "C03other", googleTokenHash("t2"), expires)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, googleDeviceID(t, ds, "uuid-migrated"))

	// Queueing the unmark releases the ID.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, "uuid-3", customer, []string{"devices/d6/deviceUsers/a"}))
	queued, err := ds.QueueGoogleConditionalAccessUnmarkOfUnenrolledHosts(ctx, customer, 10)
	require.NoError(t, err)
	require.Equal(t, 1, queued)
	require.Empty(t, googleDeviceID(t, ds, "uuid-3"))
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		"devices/d6/deviceUsers/a": fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, googleDeviceUserStates(t, ds, "uuid-3"))

	require.Equal(t, "d1", fleet.GoogleDeviceIDFromDeviceUserName("devices/d1/deviceUsers/a"))
	for _, malformed := range []string{"", "devices/", "devices/d1", "d1/deviceUsers/a", "devices//deviceUsers/a"} {
		require.Empty(t, fleet.GoogleDeviceIDFromDeviceUserName(malformed), malformed)
	}
}

func testConditionalAccessGoogleUnresolvedAndroidHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	newAndroid := func(name string, companyOwned bool) fleet.GoogleConditionalAccessAndroidHost {
		ah := createAndroidHost(name)
		ah.Device.DeviceID = "amapi-" + name
		created, err := ds.NewAndroidHost(ctx, ah, companyOwned)
		require.NoError(t, err)
		return fleet.GoogleConditionalAccessAndroidHost{HostID: created.Host.ID, HostUUID: created.Host.UUID, AndroidDeviceID: "amapi-" + name}
	}
	list := func(customerID string, maxAttempts, limit int) []fleet.GoogleConditionalAccessAndroidHost {
		got, err := ds.ListGoogleConditionalAccessUnresolvedAndroidHosts(ctx, customerID, maxAttempts, limit)
		require.NoError(t, err)
		return got
	}

	fresh := newAndroid("fresh", true)
	retried := newAndroid("retried", true)
	registered := newAndroid("registered", true)
	stale := newAndroid("stale", true)
	bookmark := newAndroid("bookmark", true)
	newAndroid("personal", false)
	unenrolled := newAndroid("unenrolled", true)
	_, err := ds.SetAndroidHostUnenrolled(ctx, unenrolled.HostID)
	require.NoError(t, err)
	// An iOS host is never listed.
	ios := newTestHostWithPlatform(t, ds, "ios", "ios", nil)
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, ios.ID, false, true, "https://example.com", true, fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))

	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, registered.HostUUID, customer, []string{"devices/reg/deviceUsers/a"}))
	// A host whose only name went stale is resolved again.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, stale.HostUUID, customer, []string{"devices/stale/deviceUsers/a"}))
	ok, err := ds.UpdateGoogleConditionalAccessDeviceUserState(ctx, "devices/stale/deviceUsers/a",
		fleet.GoogleConditionalAccessDeviceUserPending, fleet.GoogleConditionalAccessDeviceUserStale, nil)
	require.NoError(t, err)
	require.True(t, ok)
	// A bookmark-era row has no customer yet, and no attempts.
	insertMigratedGoogleDeviceID(t, ds, bookmark.HostUUID, "gdid-bookmark")

	want := []fleet.GoogleConditionalAccessAndroidHost{fresh, retried, stale, bookmark}
	require.Equal(t, want, list(customer, 0, 10))
	require.Equal(t, want[:2], list(customer, 0, 2))

	// Counting an attempt creates the host's row and moves the host back in the order.
	require.False(t, googleDeviceRowExists(t, ds, retried.HostUUID))
	for range 2 {
		require.NoError(t, ds.CountGoogleConditionalAccessResolutionAttempt(ctx, retried.HostUUID, customer))
	}
	require.NoError(t, ds.CountGoogleConditionalAccessResolutionAttempt(ctx, fresh.HostUUID, customer))
	require.Equal(t, []fleet.GoogleConditionalAccessAndroidHost{stale, bookmark, fresh, retried}, list(customer, 0, 10))
	require.Equal(t, []fleet.GoogleConditionalAccessAndroidHost{stale, bookmark, fresh}, list(customer, 2, 10))

	// An attempt on a bookmark-era row keeps its device ID.
	require.NoError(t, ds.CountGoogleConditionalAccessResolutionAttempt(ctx, bookmark.HostUUID, customer))
	id, err := ds.GetHostGoogleDeviceID(ctx, bookmark.HostUUID, customer)
	require.NoError(t, err)
	require.Equal(t, "gdid-bookmark", id)

	// Another customer's attempts and names don't count.
	require.Equal(t, []fleet.GoogleConditionalAccessAndroidHost{fresh, retried, registered, stale, bookmark}, list("C03other", 1, 10))

	// Recording a registration resets the count.
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, retried.HostUUID, customer, []string{"devices/retried/deviceUsers/a"}))
	require.NoError(t, ds.DeleteGoogleConditionalAccessDeviceUser(ctx, "devices/retried/deviceUsers/a", fleet.GoogleConditionalAccessDeviceUserPending, false))
	require.Equal(t, []fleet.GoogleConditionalAccessAndroidHost{retried, stale, fresh, bookmark}, list(customer, 0, 10))
}

func testConditionalAccessGoogleAndroidUnenroll(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const customer = "C02cajzvg"

	host := newTestHostWithPlatform(t, ds, "android-unenroll", "android", nil)
	require.NoError(t, ds.SetOrUpdateMDMData(ctx, host.ID, false, true, "https://fleet.example.com", true,
		fleet.WellKnownMDMFleet, "", fleet.PersonalEnrollmentTypeNone))
	name := "devices/" + host.UUID + "/deviceUsers/a"
	require.NoError(t, ds.RecordGoogleConditionalAccessRegistration(ctx, host.UUID, customer, []string{name}))

	unenrolled, err := ds.SetAndroidHostUnenrolled(ctx, host.ID)
	require.NoError(t, err)
	require.True(t, unenrolled)
	require.Equal(t, map[string]fleet.GoogleConditionalAccessDeviceUserState{
		name: fleet.GoogleConditionalAccessDeviceUserUnmanaging,
	}, googleDeviceUserStates(t, ds, host.UUID))
}
