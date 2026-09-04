package mysql

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// upsertCred and deleteCred seed and remove a single credential through the transactional reconcile method, which is the
// only write path. They keep the tests focused on one credential at a time.
func upsertCred(ctx context.Context, ds *Datastore, cred *fleet.MicrosoftGraphCredential) error {
	return ds.ReplaceMicrosoftGraphCredentials(ctx, []*fleet.MicrosoftGraphCredential{cred}, nil)
}

func deleteCred(ctx context.Context, ds *Datastore, tenantID string) error {
	return ds.ReplaceMicrosoftGraphCredentials(ctx, nil, []string{tenantID})
}

func TestMicrosoftAutopilot(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GraphCredentialCRUD", testGraphCredentialCRUD},
		{"GraphCredentialSecretEncryptedAtRest", testGraphCredentialSecretEncryptedAtRest},
		{"GraphCredentialMetadataOmitsSecret", testGraphCredentialMetadataOmitsSecret},
		{"GraphCredentialSyncState", testGraphCredentialSyncState},
		{"HostAutopilotDeviceUpsertAndGet", testHostAutopilotDeviceUpsertAndGet},
		{"HostAutopilotDeviceListByTenant", testHostAutopilotDeviceListByTenant},
		{"HostAutopilotDeviceBatchSpansBatches", testHostAutopilotDeviceBatchSpansBatches},
		{"HostAutopilotDeviceUnchangedUpsertIsNotAWrite", testHostAutopilotDeviceUnchangedUpsertIsNotAWrite},
		{"HostAutopilotDeviceBatchSoftDelete", testHostAutopilotDeviceBatchSoftDelete},
		{"IngestWindowsAutopilotDevices", testIngestWindowsAutopilotDevices},
		{"RemoveWindowsAutopilotHosts", testRemoveWindowsAutopilotHosts},
		{"IngestResolvesByAutopilotDeviceID", testIngestResolvesByAutopilotDeviceID},
		{"IngestSpansChunkedTransactions", testIngestSpansChunkedTransactions},
		{"OrbitEnrollReusesPendingAutopilotHost", testOrbitEnrollReusesPendingAutopilotHost},
		{"HostResponsesCarryGroupTag", testHostResponsesCarryGroupTag},
		{"HostIDByAutopilotDeviceID", testHostIDByAutopilotDeviceID},
		{"PendingHostVisibilityAndRemovalSafety", testPendingHostVisibilityAndRemovalSafety},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

const (
	testTenantA = "5b1fc5b6-9502-4cf9-90cf-d0b656eaf7a4"
	testTenantB = "11111111-1111-1111-1111-111111111111"

	selectAutopilotUpdatedAt = `SELECT updated_at FROM host_autopilot_devices WHERE host_id = ?`
	selectAutopilotDeletedAt = `SELECT deleted_at FROM host_autopilot_devices WHERE host_id = ?`
)

// seedGraphCredential stores a credential for a tenant with values derived from the tenant
func seedGraphCredential(t *testing.T, ds *Datastore, tenantID string) {
	t.Helper()
	require.NoError(t, upsertCred(t.Context(), ds, &fleet.MicrosoftGraphCredential{
		TenantID: tenantID, ClientID: "client-" + tenantID, ClientSecret: "secret-" + tenantID,
	}))
}

// newAutopilotHosts creates n hosts with distinct identities, returning them in creation order (so their IDs ascend).
func newAutopilotHosts(t *testing.T, ds *Datastore, prefix string, n int) []*fleet.Host {
	t.Helper()
	hosts := make([]*fleet.Host, 0, n)
	for i := range n {
		suffix := prefix + "-" + strconv.Itoa(i)
		hosts = append(hosts, test.NewHost(t, ds, "host-"+suffix, "10.0.0."+strconv.Itoa(i), "key-"+suffix, "uuid-"+suffix, time.Now()))
	}
	return hosts
}

func upsertAutopilotDevices(t *testing.T, ds *Datastore, devices ...*fleet.HostAutopilotDevice) {
	t.Helper()
	require.NoError(t, batchUpsertHostAutopilotDevicesDB(t.Context(), ds.writer(t.Context()), devices))
}

// autopilotTimestamp reads a timestamp column straight from the row, bypassing the datastore's soft-delete filtering.
func autopilotTimestamp(t *testing.T, ds *Datastore, stmt string, hostID uint) time.Time {
	t.Helper()
	var ts time.Time
	require.NoError(t, sqlx.GetContext(t.Context(), ds.reader(t.Context()), &ts, stmt, hostID))
	return ts
}

// storedGraphCredential reads one credential back through the decrypting list, which is the only credential read path
// production uses. There is no single-tenant datastore read because nothing needs one. It fails the test when the
// tenant is absent, so callers can use the result directly.
// graphCredentialBanner reads the app-config flag that drives the unhealthy-credential banner.
func graphCredentialBanner(t *testing.T, ds *Datastore) bool {
	t.Helper()
	appCfg, err := ds.AppConfig(t.Context())
	require.NoError(t, err)
	return appCfg.MDM.MicrosoftGraphCredentialInvalid
}

func storedGraphCredential(t *testing.T, ds *Datastore, tenantID string) *fleet.MicrosoftGraphCredential {
	t.Helper()
	cred := findGraphCredential(t, ds, tenantID)
	require.NotNil(t, cred, "no stored credential for tenant %s", tenantID)
	return cred
}

// findGraphCredential is the same read without the assertion, for the cases that expect nothing.
func findGraphCredential(t *testing.T, ds *Datastore, tenantID string) *fleet.MicrosoftGraphCredential {
	t.Helper()
	creds, err := ds.ListMicrosoftGraphCredentials(t.Context())
	require.NoError(t, err)
	for _, cred := range creds {
		if cred.TenantID == tenantID {
			return cred
		}
	}
	return nil
}

// seedAutopilotDevices shrinks the batch size, creates n hosts, and stores one device per host, returning both so a
// test can assert against either. Both batch tests need exactly this, differing only in count and batch size.
func seedAutopilotDevices(t *testing.T, ds *Datastore, prefix string, n, batchSize int) ([]*fleet.Host, []*fleet.HostAutopilotDevice) {
	t.Helper()
	original := hostAutopilotDeviceBatchSize
	hostAutopilotDeviceBatchSize = batchSize
	t.Cleanup(func() { hostAutopilotDeviceBatchSize = original })

	hosts := newAutopilotHosts(t, ds, prefix, n)
	devices := make([]*fleet.HostAutopilotDevice, 0, n)
	for i, host := range hosts {
		devices = append(devices, &fleet.HostAutopilotDevice{
			HostID: host.ID, TenantID: testTenantA,
			HardwareSerial: "serial-" + strconv.Itoa(i), GroupTag: "tag-" + strconv.Itoa(i),
		})
	}
	upsertAutopilotDevices(t, ds, devices...)
	return hosts, devices
}

func requireAutopilotDeviceNotFound(t *testing.T, ds *Datastore, hostID uint) {
	t.Helper()
	_, err := ds.GetHostAutopilotDevice(t.Context(), hostID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err), "expected a not-found error, got %v", err)
}

func testGraphCredentialCRUD(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Nothing configured yet.
	creds, err := ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Empty(t, creds)

	require.NoError(t, upsertCred(ctx, ds, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "secret-a",
	}))

	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, testTenantA, creds[0].TenantID)
	assert.Equal(t, "client-a", creds[0].ClientID)
	assert.Equal(t, "secret-a", creds[0].ClientSecret)
	// A fresh credential starts with clean sync state, which is what testGraphCredentialSyncState then moves off.
	assert.False(t, creds[0].CredentialInvalid)
	assert.Nil(t, creds[0].LastSyncedAt)
	assert.Nil(t, creds[0].LastSyncError)

	// Upserting the same tenant updates in place rather than creating a second row: tenant_id is the credential's
	// identity because the Autopilot registry is per-tenant.
	require.NoError(t, upsertCred(ctx, ds, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a-rotated", ClientSecret: "secret-a-rotated",
	}))

	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, "client-a-rotated", creds[0].ClientID)
	assert.Equal(t, "secret-a-rotated", creds[0].ClientSecret)

	// A second, distinct tenant coexists.
	seedGraphCredential(t, ds, testTenantB)
	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 2)

	got := storedGraphCredential(t, ds, testTenantB)
	assert.Equal(t, "client-"+testTenantB, got.ClientID)
	assert.Equal(t, "secret-"+testTenantB, got.ClientSecret)

	assert.Nil(t, findGraphCredential(t, ds, "no-such-tenant"), "an unconfigured tenant is simply absent")

	// Deleting one tenant leaves the other alone.
	require.NoError(t, deleteCred(ctx, ds, testTenantA))
	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, testTenantB, creds[0].TenantID)

	// Deleting an absent tenant is a no-op, not an error, so a reconciling caller need not check first.
	require.NoError(t, deleteCred(ctx, ds, "no-such-tenant"))
}

func testGraphCredentialSecretEncryptedAtRest(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, upsertCred(ctx, ds, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "plaintext-secret",
	}))

	var stored []byte
	err := ds.writer(ctx).GetContext(ctx, &stored,
		`SELECT client_secret FROM mdm_microsoft_graph_credentials WHERE tenant_id = ?`, testTenantA)
	require.NoError(t, err)
	require.NotEmpty(t, stored)
	assert.NotContains(t, string(stored), "plaintext-secret", "The stored blob must not contain the plaintext")

	// And it decrypts back to the original on read.
	got := storedGraphCredential(t, ds, testTenantA)
	assert.Equal(t, "plaintext-secret", got.ClientSecret)
}

// The config API reads metadata rather than the full credential, so a missing or rotated server private key cannot fail.
func testGraphCredentialMetadataOmitsSecret(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	empty, err := ds.ListMicrosoftGraphCredentialMetadata(ctx)
	require.NoError(t, err)
	require.NotNil(t, empty, "an empty read must return an empty slice, not nil")
	require.Empty(t, empty)

	seedGraphCredential(t, ds, testTenantA)
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true))

	meta, err := ds.ListMicrosoftGraphCredentialMetadata(ctx)
	require.NoError(t, err)
	require.Len(t, meta, 1)
	assert.Equal(t, testTenantA, meta[0].TenantID)
	assert.Equal(t, "client-"+testTenantA, meta[0].ClientID)
	// The fields the UI needs, including the banner flag, still come through.
	assert.True(t, meta[0].CredentialInvalid)
}

// The sync writes three pieces of state back onto the credential row: the invalid flag that drives the banner, and the
// last-synced timestamp and error that are displayed beside it.
func testGraphCredentialSyncState(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedGraphCredential(t, ds, testTenantA)

	// Setting the flag is idempotent, so a sync that keeps failing keeps calling this without rewriting the row.
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true))
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true))

	got := storedGraphCredential(t, ds, testTenantA)
	assert.True(t, got.CredentialInvalid)

	// The app-config banner is a stored aggregate over the per-tenant flags, so it only reflects the flag above once
	// it is recomputed. The service and the sync cron both call this after a change; nothing else keeps it in step.
	require.NoError(t, ds.UpdateMicrosoftGraphCredentialInvalidAggregate(ctx))
	assert.True(t, graphCredentialBanner(t, ds), "one unhealthy credential raises the banner")

	// Clearing works the same way.
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, false))

	got = storedGraphCredential(t, ds, testTenantA)
	assert.False(t, got.CredentialInvalid)

	require.NoError(t, ds.UpdateMicrosoftGraphCredentialInvalidAggregate(ctx))
	assert.False(t, graphCredentialBanner(t, ds), "the banner clears once no credential is unhealthy")

	// Recomputing when nothing changed is a no-op rather than a rewrite, which is what lets the sync call it freely.
	require.NoError(t, ds.UpdateMicrosoftGraphCredentialInvalidAggregate(ctx))
	assert.False(t, graphCredentialBanner(t, ds))

	// A credential deleted between listing and flagging is not an error: there is nothing left to flag, and the sync
	// must not fail the whole pass over it.
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, "8f1e0b1c-0000-0000-0000-000000000000", true))

	// A failed pass records the message without advancing last_synced_at, which reports the last SUCCESSFUL sync.
	syncErr := "AADSTS7000222: client secret expired"
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))
	got = storedGraphCredential(t, ds, testTenantA)
	assert.Nil(t, got.LastSyncedAt, "a failed pass must not claim the tenant synced")
	require.NotNil(t, got.LastSyncError)
	assert.Equal(t, syncErr, *got.LastSyncError)

	// A success stamps the time and clears the error.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, nil))
	got = storedGraphCredential(t, ds, testTenantA)
	require.NotNil(t, got.LastSyncedAt)
	assert.Nil(t, got.LastSyncError)
	syncedAt := *got.LastSyncedAt

	// A failure after a success keeps the earlier success time, so the UI can show how stale the data is.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))
	got = storedGraphCredential(t, ds, testTenantA)
	require.NotNil(t, got.LastSyncedAt)
	assert.WithinDuration(t, syncedAt, *got.LastSyncedAt, 0, "a later failure must not move the last successful sync")

	// Rotating the credential clears all of its sync state, which describes the credential being replaced.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))
	require.NoError(t, ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true))

	require.NoError(t, upsertCred(ctx, ds, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "rotated-secret",
	}))

	got = storedGraphCredential(t, ds, testTenantA)
	assert.False(t, got.CredentialInvalid, "rotating a verified credential must clear the banner flag")
	assert.Nil(t, got.LastSyncError, "rotating a credential must clear the error recorded against the old secret")
	assert.Nil(t, got.LastSyncedAt, "the previous sync time describes the replaced credential, not the new one")
}

func testHostAutopilotDeviceUpsertAndGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := newAutopilotHosts(t, ds, "get", 1)[0]

	requireAutopilotDeviceNotFound(t, ds, host.ID)

	dev := &fleet.HostAutopilotDevice{
		HostID:            host.ID,
		AutopilotDeviceID: "747c1c60-fecb-4533-a5de-81c3068091d8",
		EntraDeviceID:     "261b8f91-f3fb-4f3d-bc31-de657b7f002b",
		GroupTag:          "Engineering",
		HardwareSerial:    "VICTOR1776257483",
		TenantID:          testTenantA,
	}
	upsertAutopilotDevices(t, ds, dev)

	got, err := ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, *dev, *got, "every field must round-trip")

	// Upserting the same host updates in place. Group tags are mutable in Intune, so this is the normal sync path.
	dev.GroupTag = "Sales"
	upsertAutopilotDevices(t, ds, dev)
	got, err = ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sales", got.GroupTag)

	// A soft-deleted record reads as not found: the device left Autopilot and only the host survives.
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, []uint{host.ID}))
	requireAutopilotDeviceNotFound(t, ds, host.ID)

	// Re-registering the device revives the record rather than leaving it hidden behind the tombstone.
	upsertAutopilotDevices(t, ds, dev)
	got, err = ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sales", got.GroupTag)
}

func testHostAutopilotDeviceListByTenant(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hosts := newAutopilotHosts(t, ds, "list", 3)

	upsertAutopilotDevices(t, ds,
		&fleet.HostAutopilotDevice{HostID: hosts[0].ID, TenantID: testTenantA, HardwareSerial: "serial-a", GroupTag: "Engineering"},
		&fleet.HostAutopilotDevice{HostID: hosts[1].ID, TenantID: testTenantA, HardwareSerial: "serial-b"},
		&fleet.HostAutopilotDevice{HostID: hosts[2].ID, TenantID: testTenantB, HardwareSerial: "serial-c"},
	)

	// The listing is scoped to one tenant and ordered by host ID, which the hosts were created in.
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 2)
	assert.Equal(t, []uint{hosts[0].ID, hosts[1].ID}, []uint{devices[0].HostID, devices[1].HostID})

	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantB)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, hosts[2].ID, devices[0].HostID)

	// Soft-deleted records drop out of the listing.
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, []uint{hosts[1].ID}))
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, hosts[0].ID, devices[0].HostID)
}

// A tenant's device list is written in batches, so the batch boundary must not drop or duplicate devices. The batch
// size is shrunk here rather than inserting 1000+ real hosts.
func testHostAutopilotDeviceBatchSpansBatches(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// 7 devices at a batch size of 3 is deliberately not a multiple, so the final partial batch is exercised.
	const deviceCount = 7
	_, devices := seedAutopilotDevices(t, ds, "batch", deviceCount, 3)

	assertStoredTags := func(msg string) {
		t.Helper()
		stored, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
		require.NoError(t, err)
		require.Len(t, stored, deviceCount, msg)
		byHostID := make(map[uint]string, len(stored))
		for _, d := range stored {
			byHostID[d.HostID] = d.GroupTag
		}
		for _, want := range devices {
			assert.Equal(t, want.GroupTag, byHostID[want.HostID], "%s: host %d", msg, want.HostID)
		}
	}
	assertStoredTags("insert across batches")

	// A second pass updates every row, so the boundary is exercised on the update path too, not just the insert path.
	for _, dev := range devices {
		dev.GroupTag += "-updated"
	}
	upsertAutopilotDevices(t, ds, devices...)
	assertStoredTags("update across batches")

	// An empty slice is the steady state of a sync where nothing changed, and must not build a syntactically invalid statement.
	upsertAutopilotDevices(t, ds)
}

// Re-upserting unchanged devices must not dirty the rows.
func testHostAutopilotDeviceUnchangedUpsertIsNotAWrite(t *testing.T, ds *Datastore) {
	host := newAutopilotHosts(t, ds, "noop", 1)[0]

	dev := &fleet.HostAutopilotDevice{
		HostID: host.ID, TenantID: testTenantA, HardwareSerial: "serial-noop", GroupTag: "Engineering",
	}
	upsertAutopilotDevices(t, ds, dev)
	afterInsert := autopilotTimestamp(t, ds, selectAutopilotUpdatedAt, host.ID)

	// updated_at has microsecond resolution, so a real write would land on a different value.
	time.Sleep(10 * time.Millisecond)
	upsertAutopilotDevices(t, ds, dev)
	assert.Equal(t, afterInsert, autopilotTimestamp(t, ds, selectAutopilotUpdatedAt, host.ID),
		"re-upserting identical values must not rewrite the row")

	dev.GroupTag = "Sales"
	upsertAutopilotDevices(t, ds, dev)
	assert.True(t, autopilotTimestamp(t, ds, selectAutopilotUpdatedAt, host.ID).After(afterInsert),
		"a changed group tag must rewrite the row")
}

func testHostAutopilotDeviceBatchSoftDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	const deviceCount = 5
	hosts, devices := seedAutopilotDevices(t, ds, "del", deviceCount, 2)

	// Tombstone three of five, spanning the batch boundary.
	toDelete := []uint{hosts[0].ID, hosts[2].ID, hosts[4].ID}
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, toDelete))

	live, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, live, 2)
	assert.Equal(t, []uint{hosts[1].ID, hosts[3].ID}, []uint{live[0].HostID, live[1].HostID})

	// The host rows survive: deregistering from Autopilot does not delete an enrolled host.
	for _, h := range hosts {
		_, err := ds.Host(ctx, h.ID)
		require.NoError(t, err, "host %d must outlive its Autopilot record", h.ID)
	}

	// A repeat pass must not move deleted_at forward, or the record would misreport when the device left Autopilot.
	firstDeletion := autopilotTimestamp(t, ds, selectAutopilotDeletedAt, hosts[0].ID)
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, toDelete))
	assert.Equal(t, firstDeletion, autopilotTimestamp(t, ds, selectAutopilotDeletedAt, hosts[0].ID),
		"re-deleting must leave the original deletion timestamp")

	// Re-registering revives the record, which is how a device that returns to Autopilot comes back.
	upsertAutopilotDevices(t, ds, devices[0])
	live, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, live, 3)

	// An empty slice is the steady state of a sync where nothing was deregistered.
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, nil))
}

// autopilotDevice builds an Autopilot record for ingestion. HostID is left unset on purpose: the datastore resolves it
// from the Autopilot device ID, falling back to the serial for a host Fleet already knows about.
func autopilotDevice(serial, tag string) *fleet.HostAutopilotDevice {
	return &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-" + serial,
		EntraDeviceID:     "aad-" + serial,
		GroupTag:          tag,
		HardwareSerial:    serial,
		HardwareModel:     "Virtual Machine",
		HardwareVendor:    "Microsoft Corporation",
		TenantID:          testTenantA,
	}
}

// autopilotDeviceWithID builds a record whose Autopilot device ID is not derived from the serial, which is what the
// duplicate-serial cases need.
func autopilotDeviceWithID(deviceID, serial, tag string) *fleet.HostAutopilotDevice {
	dev := autopilotDevice(serial, tag)
	dev.AutopilotDeviceID = deviceID
	dev.EntraDeviceID = "aad-" + deviceID
	return dev
}

// hostIDsBySerial returns every host carrying a serial, lowest ID first. Read straight from the column because no
// Datastore method returns more than one host for a serial, and more than one is exactly what these tests assert on.
func hostIDsBySerial(t *testing.T, ds *Datastore, serial string) []uint {
	t.Helper()
	var ids []uint
	require.NoError(t, sqlx.SelectContext(t.Context(), ds.reader(t.Context()), &ids,
		`SELECT id FROM hosts WHERE hardware_serial = ? ORDER BY id`, serial))
	return ids
}

// autopilotDevicesByDeviceID keys a tenant's stored records by Autopilot device ID, which is how the ingest resolves
// them and therefore how the assertions read.
func autopilotDevicesByDeviceID(t *testing.T, ds *Datastore, tenantID string) map[string]fleet.HostAutopilotDevice {
	t.Helper()
	devices, err := ds.ListHostAutopilotDevices(t.Context(), tenantID)
	require.NoError(t, err)
	byID := make(map[string]fleet.HostAutopilotDevice, len(devices))
	for _, d := range devices {
		byID[d.AutopilotDeviceID] = *d
	}
	require.Len(t, byID, len(devices), "two records shared an Autopilot device ID")
	return byID
}

// seedWindowsBuiltinLabels recreates the two builtin labels a pending Windows host joins. TruncateTables clears the
// labels table between subtests, and the shared createBuiltinLabels helper only seeds the Apple ones.
func seedWindowsBuiltinLabels(t *testing.T, ds *Datastore) {
	t.Helper()
	_, err := ds.writer(t.Context()).ExecContext(t.Context(), `
INSERT INTO labels (name, description, query, platform, label_type) VALUES (?, '', '', '', ?), (?, '', '', '', ?)
ON DUPLICATE KEY UPDATE name = name`,
		fleet.BuiltinLabelNameAllHosts, fleet.LabelTypeBuiltIn,
		fleet.BuiltinLabelNameWindows, fleet.LabelTypeBuiltIn)
	require.NoError(t, err)
}

func testIngestWindowsAutopilotDevices(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	// A device found in Autopilot is placed into the default fleet as it is created
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "Workstations"})
	require.NoError(t, err)
	require.NoError(t, ds.SetWindowsEnrollmentDefaultFleet(ctx, &team.ID))

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("AP-SERIAL-1", "Engineering"),
		autopilotDevice("AP-SERIAL-2", ""),
	}))

	// A pending Autopilot host must look like a pending host everywhere the UI and the host filters look.
	var got []struct {
		ID               uint    `db:"id"`
		Platform         string  `db:"platform"`
		HardwareSerial   string  `db:"hardware_serial"`
		OsqueryHostID    *string `db:"osquery_host_id"`
		Enrolled         bool    `db:"enrolled"`
		InstalledFromDep bool    `db:"installed_from_dep"`
		EnrollmentStatus *string `db:"enrollment_status"`
		ServerURL        string  `db:"server_url"`
		TeamID           *uint   `db:"team_id"`
		HardwareModel    string  `db:"hardware_model"`
		HardwareVendor   string  `db:"hardware_vendor"`
		DisplayName      *string `db:"display_name"`
	}
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &got, `
SELECT h.id, h.platform, h.hardware_serial, h.osquery_host_id, h.team_id,
       h.hardware_model, h.hardware_vendor, hdn.display_name,
       hm.enrolled, hm.installed_from_dep, hm.enrollment_status, hm.server_url
FROM hosts h
JOIN host_mdm hm ON hm.host_id = h.id
LEFT JOIN host_display_names hdn ON hdn.host_id = h.id
WHERE h.hardware_serial LIKE 'AP-SERIAL-%' ORDER BY h.hardware_serial`))
	require.Len(t, got, 2)
	for _, h := range got {
		assert.Equal(t, "windows", h.Platform)
		assert.Nil(t, h.OsqueryHostID, "a pending host has not run osquery yet")
		assert.False(t, h.Enrolled)
		assert.True(t, h.InstalledFromDep, "drives the generated Pending enrollment status")
		require.NotNil(t, h.EnrollmentStatus)
		assert.Equal(t, "Pending", *h.EnrollmentStatus)
		assert.Contains(t, h.ServerURL, "/api/mdm/microsoft/discovery",
			"pending hosts must share the Windows MDM solution row, not the Apple one")
		require.NotNil(t, h.TeamID, "a pending host lands in the default fleet, not No team")
		assert.Equal(t, team.ID, *h.TeamID)

		// Autopilot is the only source of hardware identity until osquery runs.
		assert.Equal(t, "Virtual Machine", h.HardwareModel)
		assert.Equal(t, "Microsoft Corporation", h.HardwareVendor)
		require.NotNil(t, h.DisplayName)
		assert.Equal(t, "Virtual Machine ("+h.HardwareSerial+")", *h.DisplayName,
			"a pending host must be identifiable in the host list before it boots")
	}

	// The Autopilot metadata is stored against the created hosts.
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 2)
	byS := map[string]fleet.HostAutopilotDevice{}
	for _, d := range devices {
		byS[d.HardwareSerial] = *d
		assert.NotZero(t, d.HostID, "host id resolved from the serial")
	}
	require.Contains(t, byS, "AP-SERIAL-1")
	first := byS["AP-SERIAL-1"]
	assert.Equal(t, "Engineering", first.GroupTag)

	// Builtin label membership so the host shows up before osquery runs.
	var labelCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &labelCount, `
SELECT COUNT(*) FROM label_membership lm
JOIN labels l ON l.id = lm.label_id
WHERE lm.host_id = ? AND l.name IN (?, ?)`, first.HostID, fleet.BuiltinLabelNameAllHosts, fleet.BuiltinLabelNameWindows))
	assert.Equal(t, 2, labelCount)

	// Re-ingesting is idempotent and updates a mutable group tag in place.
	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("AP-SERIAL-1", "Marketing"),
	}))
	assert.Len(t, hostIDsBySerial(t, ds, "AP-SERIAL-1"), 1, "re-ingesting must not create a second host")
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	for _, d := range devices {
		if d.HardwareSerial == "AP-SERIAL-1" {
			assert.Equal(t, "Marketing", d.GroupTag)
		}
	}
}

func testRemoveWindowsAutopilotHosts(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("RM-PENDING", "tag"),
		autopilotDevice("RM-ENROLLED", "tag"),
	}))
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 2)
	byS := map[string]uint{}
	for _, d := range devices {
		byS[d.HardwareSerial] = d.HostID
	}

	// Simulate the second device having enrolled since the sync created it.
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE host_mdm SET enrolled = 1 WHERE host_id = ?`, byS["RM-ENROLLED"])
	require.NoError(t, err)

	require.NoError(t, ds.RemoveWindowsAutopilotHosts(ctx, []uint{byS["RM-PENDING"], byS["RM-ENROLLED"]}))

	// The still-pending host is gone; the enrolled host survives.
	var remaining []string
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &remaining,
		`SELECT hardware_serial FROM hosts WHERE hardware_serial IN ('RM-PENDING','RM-ENROLLED')`))
	assert.Equal(t, []string{"RM-ENROLLED"}, remaining)

	// Both Autopilot rows are tombstoned, so neither device is reported as live.
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	assert.Empty(t, devices)
}

func testIngestResolvesByAutopilotDeviceID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	t.Run("records sharing a serial become separate hosts, stable across syncs", func(t *testing.T) {
		const serial = "DUP-SERIAL"
		first := autopilotDeviceWithID("ap-aaa", serial, "Engineering")
		second := autopilotDeviceWithID("ap-bbb", serial, "Marketing")
		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{first, second}))

		require.Len(t, hostIDsBySerial(t, ds, serial), 2, "two records for one serial are two devices, so two hosts")
		stored := autopilotDevicesByDeviceID(t, ds, testTenantA)
		require.Len(t, stored, 2)
		assert.Equal(t, "Engineering", stored["ap-aaa"].GroupTag)
		assert.Equal(t, "Marketing", stored["ap-bbb"].GroupTag)
		assert.NotEqual(t, stored["ap-aaa"].HostID, stored["ap-bbb"].HostID)

		// Re-ingest reversed and retagged.
		second.GroupTag = "Sales"
		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{second, first}))

		resynced := autopilotDevicesByDeviceID(t, ds, testTenantA)
		require.Len(t, resynced, 2, "re-ingesting the same registry must not create more hosts")
		assert.Equal(t, stored["ap-aaa"].HostID, resynced["ap-aaa"].HostID, "hosts are reused regardless of page order")
		assert.Equal(t, stored["ap-bbb"].HostID, resynced["ap-bbb"].HostID)
		assert.Equal(t, "Sales", resynced["ap-bbb"].GroupTag)
	})

	t.Run("an existing host is adopted rather than duplicated", func(t *testing.T) {
		const serial = "ADOPT-SERIAL"
		// A fully enrolled Windows host, the shape Fleet already knows about before a device is registered in Autopilot.
		existing := test.NewHost(t, ds, "DESKTOP-ADOPT", "10.10.10.20", "nodekey-adopt", "uuid-adopt", time.Now(),
			test.WithPlatform("windows"), test.WithHardwareSerial(serial))

		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx,
			[]*fleet.HostAutopilotDevice{autopilotDeviceWithID("ap-adopt", serial, "Engineering")}))
		assert.Equal(t, []uint{existing.ID}, hostIDsBySerial(t, ds, serial), "the enrolled host is adopted, not duplicated")

		// A second, genuinely different registration on that serial cannot claim the same host again.
		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx,
			[]*fleet.HostAutopilotDevice{autopilotDeviceWithID("ap-adopt-2", serial, "")}))

		stored := autopilotDevicesByDeviceID(t, ds, testTenantA)
		assert.NotEqual(t, stored["ap-adopt"].HostID, stored["ap-adopt-2"].HostID,
			"a claimed host is never handed to a second registration")
		assert.ElementsMatch(t, hostIDsBySerial(t, ds, serial),
			[]uint{stored["ap-adopt"].HostID, stored["ap-adopt-2"].HostID})
	})

	t.Run("adopting and creating for one serial in a single batch", func(t *testing.T) {
		const serial = "MIXED-SERIAL"
		existing := test.NewHost(t, ds, "DESKTOP-MIXED", "10.10.10.21", "nodekey-mixed", "uuid-mixed", time.Now(),
			test.WithPlatform("windows"), test.WithHardwareSerial(serial))

		require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
			autopilotDeviceWithID("ap-mixed-1", serial, ""),
			autopilotDeviceWithID("ap-mixed-2", serial, ""),
		}))

		hostIDs := hostIDsBySerial(t, ds, serial)
		require.Len(t, hostIDs, 2, "one host per Autopilot device, with no orphan left behind")
		assert.Equal(t, existing.ID, hostIDs[0], "the pre-existing host is adopted rather than duplicated")

		stored := autopilotDevicesByDeviceID(t, ds, testTenantA)
		assert.ElementsMatch(t, hostIDs, []uint{stored["ap-mixed-1"].HostID, stored["ap-mixed-2"].HostID},
			"every host carries a record, so neither device was dropped onto the other's host")
	})
}

func testIngestSpansChunkedTransactions(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	original := hostAutopilotDeviceBatchSize
	hostAutopilotDeviceBatchSize = 3
	t.Cleanup(func() { hostAutopilotDeviceBatchSize = original })

	const distinct = 10
	devices := make([]*fleet.HostAutopilotDevice, 0, distinct+2)
	for i := range distinct {
		serial := "CHUNK-" + strconv.Itoa(i)
		devices = append(devices, &fleet.HostAutopilotDevice{
			AutopilotDeviceID: "ap-" + serial, EntraDeviceID: "aad-" + serial,
			GroupTag: "tag-" + strconv.Itoa(i), HardwareSerial: serial, TenantID: testTenantA,
		})
	}
	// Second registrations on serials that land in different chunks once sorted, so serial collisions are resolved
	// both within a chunk and across a transaction boundary.
	devices = append(devices,
		autopilotDeviceWithID("ap-dup-0", "CHUNK-0", "second"),
		autopilotDeviceWithID("ap-dup-9", "CHUNK-9", "second"))

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, devices))

	stored := autopilotDevicesByDeviceID(t, ds, testTenantA)
	require.Len(t, stored, len(devices), "one record per Autopilot device across chunk boundaries")
	uniqueHostIDs := make(map[uint]struct{}, len(stored))
	for _, d := range stored {
		assert.NotZero(t, d.HostID)
		uniqueHostIDs[d.HostID] = struct{}{}
	}
	assert.Len(t, uniqueHostIDs, len(devices), "two Autopilot records must never share a host")
	assert.Len(t, hostIDsBySerial(t, ds, "CHUNK-0"), 2, "a serial carrying two registrations gets two hosts")
}

// A pending Autopilot host must be reused when the device actually enrolls, rather than a second host being created.
func testOrbitEnrollReusesPendingAutopilotHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	const serial = "ENROLL-SERIAL-1"
	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice(serial, "Engineering"),
	}))
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	pendingHostID := devices[0].HostID

	host, err := ds.EnrollOrbit(ctx,
		fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
			HardwareUUID: "uuid-" + serial, HardwareSerial: serial, Hostname: "DESKTOP-AUTOPILOT", Platform: "windows",
		}),
		fleet.WithEnrollOrbitNodeKey("orbit-key-"+serial),
	)
	require.NoError(t, err)
	assert.Equal(t, pendingHostID, host.ID, "the pending Autopilot host is reused, not duplicated")

	assert.Len(t, hostIDsBySerial(t, ds, serial), 1)

	// The Autopilot metadata is keyed by host_id, so the group tag survives enrollment.
	device, err := ds.GetHostAutopilotDevice(ctx, pendingHostID)
	require.NoError(t, err)
	assert.Equal(t, "Engineering", device.GroupTag)

	// A Windows host with no Autopilot record must not match by serial, which is the legacy behaviour.
	const legacySerial = "LEGACY-SERIAL-1"
	legacy := test.NewHost(t, ds, "legacy-host", "10.0.0.99", "legacy-key", "legacy-uuid", time.Now())
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE hosts SET hardware_serial = ?, platform = 'windows' WHERE id = ?`, legacySerial, legacy.ID)
	require.NoError(t, err)

	other, err := ds.EnrollOrbit(ctx,
		fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
			HardwareUUID: "uuid-different", HardwareSerial: legacySerial, Hostname: "DESKTOP-LEGACY", Platform: "windows",
		}),
		fleet.WithEnrollOrbitNodeKey("orbit-key-legacy"),
	)
	require.NoError(t, err)
	assert.NotEqual(t, legacy.ID, other.ID,
		"a Windows host with no pending Autopilot row still matches on its identifier, not its serial")
}

func testHostResponsesCarryGroupTag(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	// Intune caps the group tag at 2048 characters; a tag at the limit must round-trip intact.
	maxTag := strings.Repeat("a", 2048)
	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("TAG-SERIAL-1", maxTag),
	}))
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	autopilotHostID := devices[0].HostID

	plain := test.NewHost(t, ds, "plain-host", "10.0.0.50", "plain-key", "plain-uuid", time.Now())

	// Detail endpoint.
	got, err := ds.Host(ctx, autopilotHostID)
	require.NoError(t, err)
	require.NotNil(t, got.GroupTag)
	assert.Equal(t, maxTag, *got.GroupTag, "a 2048-character tag round-trips intact")

	gotPlain, err := ds.Host(ctx, plain.ID)
	require.NoError(t, err)
	assert.Nil(t, gotPlain.GroupTag, "a host with no Autopilot record has no group tag")

	// List endpoint.
	hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin}, fleet.HostListOptions{})
	require.NoError(t, err)
	tagByID := map[uint]*string{}
	for _, h := range hosts {
		tagByID[h.ID] = h.GroupTag
	}
	require.Contains(t, tagByID, autopilotHostID)
	listed := tagByID[autopilotHostID]
	require.NotNil(t, listed)
	assert.Equal(t, maxTag, *listed)
	require.Contains(t, tagByID, plain.ID)
	assert.Nil(t, tagByID[plain.ID])

	// A device that leaves Autopilot stops reporting a tag, without the host disappearing.
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, []uint{autopilotHostID}))
	got, err = ds.Host(ctx, autopilotHostID)
	require.NoError(t, err)
	assert.Nil(t, got.GroupTag, "a tombstoned Autopilot record reports no group tag")
}

// The ZTDID a device supplies at Windows MDM enrollment is the same GUID Graph returns as
// windowsAutopilotDeviceIdentity.id, so it resolves the pending host exactly, without the serial.
func testHostIDByAutopilotDeviceID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("ZTD-SERIAL-1", "Engineering"),
	}))
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)

	got, err := ds.HostIDByAutopilotDeviceID(ctx, devices[0].AutopilotDeviceID)
	require.NoError(t, err)
	assert.Equal(t, devices[0].HostID, got)

	_, err = ds.HostIDByAutopilotDeviceID(ctx, "no-such-ztdid")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "an unknown ZTDID is not found, not an error the caller has to special-case")

	// A device that has left Autopilot no longer resolves, so a stale enrollment cannot relink to a tombstoned host.
	require.NoError(t, ds.BatchSoftDeleteHostAutopilotDevices(ctx, []uint{devices[0].HostID}))
	_, err = ds.HostIDByAutopilotDeviceID(ctx, devices[0].AutopilotDeviceID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testPendingHostVisibilityAndRemovalSafety(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{
		autopilotDevice("VIS-SERIAL-1", "Engineering"),
	}))
	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	hostID := devices[0].HostID

	hosts, err := ds.ListHosts(ctx, fleet.TeamFilter{User: test.UserAdmin},
		fleet.HostListOptions{ListOptions: fleet.ListOptions{OrderKey: "display_name"}})
	require.NoError(t, err)
	var found bool
	for _, h := range hosts {
		if h.ID == hostID {
			found = true
		}
	}
	assert.True(t, found, "a pending Autopilot host must appear in the default host list ordering")

	// A device that runs fleetd but never MDM-enrolls keeps enrolled=0 and installed_from_dep=1, so removal must not
	// treat pending state alone as licence to delete a live host.
	_, err = ds.writer(ctx).ExecContext(ctx,
		`UPDATE hosts SET osquery_host_id = 'osq-vis-1', orbit_node_key = 'orbit-vis-1' WHERE id = ?`, hostID)
	require.NoError(t, err)

	require.NoError(t, ds.RemoveWindowsAutopilotHosts(ctx, []uint{hostID}))

	var stillThere int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &stillThere,
		`SELECT COUNT(*) FROM hosts WHERE id = ?`, hostID))
	assert.Equal(t, 1, stillThere, "a host that has checked in survives leaving the Autopilot registry")

	requireAutopilotDeviceNotFound(t, ds, hostID)
}
