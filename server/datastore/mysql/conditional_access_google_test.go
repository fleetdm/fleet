package mysql

import (
	"fmt"
	"sync"
	"testing"

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
		State:      fleet.ConditionalAccessGoogleStateKeyRequired, ErrorMessage: new("key required"),
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
