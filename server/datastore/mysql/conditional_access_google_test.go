package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleCloudIdentityClientStates(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"UpsertHostGoogleCloudIdentityResolution", testUpsertHostGoogleCloudIdentityResolution},
		{"UpsertIsIdempotentOnSameTriple", testUpsertIsIdempotentOnSameTriple},
		{"SetHostGoogleCloudIdentityResolvedDeviceUser", testSetHostGoogleCloudIdentityResolvedDeviceUser},
		{"LoadHostGoogleCloudIdentityClientStatesMultiRow", testLoadHostGoogleCloudIdentityClientStatesMultiRow},
		{"LoadHostGoogleCloudIdentityClientStatesEmpty", testLoadHostGoogleCloudIdentityClientStatesEmpty},
		{"SetHostGoogleCloudIdentityClientState", testSetHostGoogleCloudIdentityClientState},
		{"DeleteHostGoogleCloudIdentityClientStates", testDeleteHostGoogleCloudIdentityClientStates},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

// newGCIHost returns a freshly created Host record we can FK ClientState rows
// against (even though the schema has no actual FK, we still need a host_id
// that exists for the application-side lookups to make sense).
func newGCIHost(t *testing.T, ds *Datastore, uuid string) *fleet.Host {
	t.Helper()
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("nk-" + uuid),
		OsqueryHostID:   ptr.String("oh-" + uuid),
		UUID:            uuid,
		Hostname:        "host-" + uuid + ".local",
		Platform:        "darwin",
		HardwareSerial:  "S-" + uuid,
	})
	require.NoError(t, err)
	return host
}

func testUpsertHostGoogleCloudIdentityResolution(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h1")

	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "user@example.com", "fleet"))

	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	r := rows[0]
	assert.Equal(t, host.ID, r.HostID)
	assert.Equal(t, "user@example.com", r.WorkspaceEmail)
	assert.Equal(t, "fleet", r.PartnerSuffix)
	assert.Nil(t, r.DeviceUserResource, "device_user_resource is nil until resolution")
	assert.Nil(t, r.LastCompliant)
	assert.Nil(t, r.LastManaged)
	assert.Nil(t, r.LastEtag)
}

func testUpsertIsIdempotentOnSameTriple(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h2")

	for range 3 {
		require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "user@example.com", "fleet"))
	}
	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "three upserts on the same (host,email,suffix) triple should produce one row")

	// Different suffix on the same host+email should produce a separate row.
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "user@example.com", "fleet-engineering"))
	rows, err = ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

func testSetHostGoogleCloudIdentityResolvedDeviceUser(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h3")
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "u@example.com", "fleet"))

	require.NoError(t, ds.SetHostGoogleCloudIdentityResolvedDeviceUser(ctx,
		host.ID, "u@example.com", "fleet", "devices/d-1/deviceUsers/du-1"))

	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.NotNil(t, rows[0].DeviceUserResource)
	assert.Equal(t, "devices/d-1/deviceUsers/du-1", *rows[0].DeviceUserResource)

	// Set on a (host,email,suffix) that doesn't exist must succeed silently
	// (UPDATE with WHERE clause matching no rows). Sync layer should rely on
	// Upsert to create the row first; this guards against panics on the
	// secondary path.
	require.NoError(t, ds.SetHostGoogleCloudIdentityResolvedDeviceUser(ctx,
		host.ID, "missing@example.com", "fleet", "devices/x/deviceUsers/y"))
}

func testLoadHostGoogleCloudIdentityClientStatesMultiRow(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h4")
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "a@example.com", "fleet"))
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "b@example.com", "fleet"))
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "b@example.com", "fleet-eng"))

	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	emails := make(map[string]struct{})
	for _, r := range rows {
		emails[r.WorkspaceEmail+"/"+r.PartnerSuffix] = struct{}{}
	}
	assert.Contains(t, emails, "a@example.com/fleet")
	assert.Contains(t, emails, "b@example.com/fleet")
	assert.Contains(t, emails, "b@example.com/fleet-eng")
}

func testLoadHostGoogleCloudIdentityClientStatesEmpty(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h5")

	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err, "empty result is not an error")
	assert.Empty(t, rows)
}

func testSetHostGoogleCloudIdentityClientState(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host := newGCIHost(t, ds, "h6")
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host.ID, "u@example.com", "fleet"))

	require.NoError(t, ds.SetHostGoogleCloudIdentityClientState(ctx,
		host.ID, "u@example.com", "fleet",
		true, // managed
		true, // compliant
		"all CA policies passing",
		"etag-1",
	))

	rows, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	r := rows[0]
	require.NotNil(t, r.LastManaged)
	require.NotNil(t, r.LastCompliant)
	require.NotNil(t, r.LastScoreReason)
	require.NotNil(t, r.LastEtag)
	require.NotNil(t, r.LastSyncedAt)
	assert.True(t, *r.LastManaged)
	assert.True(t, *r.LastCompliant)
	assert.Equal(t, "all CA policies passing", *r.LastScoreReason)
	assert.Equal(t, "etag-1", *r.LastEtag)
	assert.WithinDuration(t, time.Now(), *r.LastSyncedAt, 30*time.Second)

	// Subsequent set with different values must overwrite, not append.
	require.NoError(t, ds.SetHostGoogleCloudIdentityClientState(ctx,
		host.ID, "u@example.com", "fleet",
		false, false, "1 policy failed: Disk encryption", "etag-2",
	))
	rows, err = ds.LoadHostGoogleCloudIdentityClientStates(ctx, host.ID)
	require.NoError(t, err)
	require.Len(t, rows, 1, "SetHostGoogleCloudIdentityClientState updates, never inserts")
	r = rows[0]
	assert.False(t, *r.LastManaged)
	assert.False(t, *r.LastCompliant)
	assert.Equal(t, "1 policy failed: Disk encryption", *r.LastScoreReason)
	assert.Equal(t, "etag-2", *r.LastEtag)

	// Update against a non-existent triple is a no-op (zero rows affected,
	// no error). Mirrors the SetHostGoogleCloudIdentityResolvedDeviceUser test.
	require.NoError(t, ds.SetHostGoogleCloudIdentityClientState(ctx,
		host.ID, "ghost@example.com", "fleet",
		true, true, "shouldn't matter", "etag-x",
	))
}

func testDeleteHostGoogleCloudIdentityClientStates(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	host1 := newGCIHost(t, ds, "h7-1")
	host2 := newGCIHost(t, ds, "h7-2")
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host1.ID, "a@example.com", "fleet"))
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host1.ID, "b@example.com", "fleet"))
	require.NoError(t, ds.UpsertHostGoogleCloudIdentityResolution(ctx, host2.ID, "c@example.com", "fleet"))

	// Delete host1's rows; host2's row should be untouched.
	require.NoError(t, ds.DeleteHostGoogleCloudIdentityClientStates(ctx, host1.ID))

	rows1, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host1.ID)
	require.NoError(t, err)
	assert.Empty(t, rows1, "host1 rows must all be deleted")

	rows2, err := ds.LoadHostGoogleCloudIdentityClientStates(ctx, host2.ID)
	require.NoError(t, err)
	require.Len(t, rows2, 1, "host2's row must be preserved")
	assert.Equal(t, "c@example.com", rows2[0].WorkspaceEmail)

	// Delete on a host with no rows is a no-op (zero rows affected, no error).
	require.NoError(t, ds.DeleteHostGoogleCloudIdentityClientStates(ctx, host1.ID))
}
