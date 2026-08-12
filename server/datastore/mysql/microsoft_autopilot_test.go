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
		{"IngestCollapsesDuplicateSerials", testIngestCollapsesDuplicateSerials},
		{"IngestSpansChunkedTransactions", testIngestSpansChunkedTransactions},
		{"OrbitEnrollReusesPendingAutopilotHost", testOrbitEnrollReusesPendingAutopilotHost},
		{"HostResponsesCarryGroupTag", testHostResponsesCarryGroupTag},
		{"HostIDByAutopilotDeviceID", testHostIDByAutopilotDeviceID},
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
	require.NoError(t, ds.BatchUpsertHostAutopilotDevices(t.Context(), devices))
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

	seedGraphCredential(t, ds, testTenantA)
	_, err := ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)

	meta, err := ds.ListMicrosoftGraphCredentialMetadata(ctx)
	require.NoError(t, err)
	require.Len(t, meta, 1)
	assert.Equal(t, testTenantA, meta[0].TenantID)
	assert.Equal(t, "client-"+testTenantA, meta[0].ClientID)
	assert.Empty(t, meta[0].ClientSecret, "the metadata read must not carry the secret")
	// The fields the UI needs, including the banner flag, still come through.
	assert.True(t, meta[0].CredentialInvalid)
}

// The sync writes three pieces of state back onto the credential row: the invalid flag that drives the banner, and the
// last-synced timestamp and error that are displayed beside it.
func testGraphCredentialSyncState(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedGraphCredential(t, ds, testTenantA)

	// The first transition reports a change; a repeat does not, so a sync that keeps failing does not re-notify.
	wasSet, err := ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)
	assert.True(t, wasSet)

	wasSet, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)
	assert.False(t, wasSet)

	got := storedGraphCredential(t, ds, testTenantA)
	assert.True(t, got.CredentialInvalid)

	// Clearing works the same way.
	wasSet, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, false)
	require.NoError(t, err)
	assert.True(t, wasSet)

	got = storedGraphCredential(t, ds, testTenantA)
	assert.False(t, got.CredentialInvalid)

	// A failed pass records the message alongside the timestamp.
	syncErr := "AADSTS7000222: client secret expired"
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))
	got = storedGraphCredential(t, ds, testTenantA)
	require.NotNil(t, got.LastSyncedAt)
	require.NotNil(t, got.LastSyncError)
	assert.Equal(t, syncErr, *got.LastSyncError)

	// A subsequent success clears the error.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, nil))
	got = storedGraphCredential(t, ds, testTenantA)
	require.NotNil(t, got.LastSyncedAt)
	assert.Nil(t, got.LastSyncError)

	// Rotating the credential clears all of its sync state, which describes the credential being replaced.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))
	_, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)

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
// from the serial.
func autopilotDevice(serial, tag string) *fleet.HostAutopilotDevice {
	return &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-" + serial,
		EntraDeviceID:     "aad-" + serial,
		GroupTag:          tag,
		HardwareSerial:    serial,
		TenantID:          testTenantA,
	}
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
	}
	require.NoError(t, sqlx.SelectContext(ctx, ds.reader(ctx), &got, `
SELECT h.id, h.platform, h.hardware_serial, h.osquery_host_id,
       hm.enrolled, hm.installed_from_dep, hm.enrollment_status, hm.server_url
FROM hosts h JOIN host_mdm hm ON hm.host_id = h.id
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
	var hostCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &hostCount,
		`SELECT COUNT(*) FROM hosts WHERE hardware_serial = 'AP-SERIAL-1'`))
	assert.Equal(t, 1, hostCount, "re-ingesting must not create a second host")
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

// Several Autopilot records can share one hardware serial, and they can only ever become one host. The collapse must be
// deterministic: keying off Graph's page order would make the stored group tag flap between syncs.
func testIngestCollapsesDuplicateSerials(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	seedWindowsBuiltinLabels(t, ds)

	first := &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-aaa", EntraDeviceID: "aad-1", GroupTag: "Engineering",
		HardwareSerial: "DUP-SERIAL", TenantID: testTenantA,
	}
	second := &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-bbb", EntraDeviceID: "aad-2", GroupTag: "Marketing",
		HardwareSerial: "DUP-SERIAL", TenantID: testTenantA,
	}

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{first, second}))

	var hostCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &hostCount,
		`SELECT COUNT(*) FROM hosts WHERE hardware_serial = 'DUP-SERIAL'`))
	assert.Equal(t, 1, hostCount, "two Autopilot records for one serial are one host")

	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "ap-aaa", devices[0].AutopilotDeviceID, "lowest Autopilot device ID wins")
	assert.Equal(t, "Engineering", devices[0].GroupTag)

	// A record that has been Entra-joined beats one that never has, regardless of Autopilot device ID ordering: it is
	// the registration that actually deployed, so its group tag is the operative one.
	joined := &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-zzz", EntraDeviceID: "aad-joined", GroupTag: "Deployed",
		HardwareSerial: "DUP-SERIAL", TenantID: testTenantA,
	}
	neverJoined := &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-aaa", EntraDeviceID: "", GroupTag: "Stale",
		HardwareSerial: "DUP-SERIAL", TenantID: testTenantA,
	}
	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{neverJoined, joined}))
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "Deployed", devices[0].GroupTag, "the Entra-joined record wins even with a higher Autopilot ID")

	// Reversing the order Graph returned them in must not change the outcome.
	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, []*fleet.HostAutopilotDevice{second, first}))
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, "ap-aaa", devices[0].AutopilotDeviceID, "winner is stable across page ordering")
}

// Ingestion runs one transaction per chunk so a 100k-device tenant never becomes a single unbounded transaction. This
// shrinks the chunk size and asserts a device set spanning several chunks, with duplicates straddling a boundary,
// reconciles to exactly one host per distinct serial.
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
	// Duplicates of serials that land in different chunks once sorted.
	devices = append(devices, &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-zzz", GroupTag: "loser", HardwareSerial: "CHUNK-0", TenantID: testTenantA,
	}, &fleet.HostAutopilotDevice{
		AutopilotDeviceID: "ap-zzz", GroupTag: "loser", HardwareSerial: "CHUNK-9", TenantID: testTenantA,
	})

	require.NoError(t, ds.IngestWindowsAutopilotDevices(ctx, devices))

	var hostCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &hostCount,
		`SELECT COUNT(*) FROM hosts WHERE hardware_serial LIKE 'CHUNK-%'`))
	assert.Equal(t, distinct, hostCount, "one host per distinct serial across chunk boundaries")

	stored, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, stored, distinct)
	for _, d := range stored {
		assert.NotEqual(t, "loser", d.GroupTag, "the lower Autopilot device ID wins in every chunk")
		assert.NotZero(t, d.HostID)
	}
}

// A pending Autopilot host must be reused when the device actually enrolls, rather than a second host being created.
// The serial is the only identifier orbit and Autopilot share, so this is what the Windows serial branch exists for.
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

	var hostCount int
	require.NoError(t, sqlx.GetContext(ctx, ds.reader(ctx), &hostCount,
		`SELECT COUNT(*) FROM hosts WHERE hardware_serial = ?`, serial))
	assert.Equal(t, 1, hostCount)

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

// The group tag is what automation targets when placing a device, so both the list and the detail endpoint have to
// carry it, and neither may invent one for a host that is not Autopilot-registered.
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
