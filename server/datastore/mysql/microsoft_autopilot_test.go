package mysql

import (
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMicrosoftAutopilot(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GraphCredentialUpsertListGet", testGraphCredentialUpsertListGet},
		{"GraphCredentialMetadataOmitsSecret", testGraphCredentialMetadataOmitsSecret},
		{"GraphCredentialSecretEncryptedAtRest", testGraphCredentialSecretEncryptedAtRest},
		{"GraphCredentialDelete", testGraphCredentialDelete},
		{"GraphCredentialInvalidFlag", testGraphCredentialInvalidFlag},
		{"GraphCredentialSyncResult", testGraphCredentialSyncResult},
		{"HostAutopilotDeviceUpsertAndGet", testHostAutopilotDeviceUpsertAndGet},
		{"HostAutopilotDeviceListByTenant", testHostAutopilotDeviceListByTenant},
		{"HostAutopilotDeviceMaxGroupTag", testHostAutopilotDeviceMaxGroupTag},
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
)

func testGraphCredentialUpsertListGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Nothing configured yet.
	creds, err := ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Empty(t, creds)

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "secret-a",
	}))

	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, testTenantA, creds[0].TenantID)
	assert.Equal(t, "client-a", creds[0].ClientID)
	assert.Equal(t, "secret-a", creds[0].ClientSecret)
	assert.False(t, creds[0].CredentialInvalid)
	assert.Nil(t, creds[0].LastSyncedAt)
	assert.Nil(t, creds[0].LastSyncError)

	// Upserting the same tenant updates in place rather than creating a second row: tenant_id is the credential's
	// identity because the Autopilot registry is per-tenant.
	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a-rotated", ClientSecret: "secret-a-rotated",
	}))

	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, "client-a-rotated", creds[0].ClientID)
	assert.Equal(t, "secret-a-rotated", creds[0].ClientSecret)

	// A second, distinct tenant coexists.
	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantB, ClientID: "client-b", ClientSecret: "secret-b",
	}))
	creds, err = ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Len(t, creds, 2)

	got, err := ds.GetMicrosoftGraphCredential(ctx, testTenantB)
	require.NoError(t, err)
	assert.Equal(t, "client-b", got.ClientID)
	assert.Equal(t, "secret-b", got.ClientSecret)

	_, err = ds.GetMicrosoftGraphCredential(ctx, "no-such-tenant")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testGraphCredentialSecretEncryptedAtRest(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "plaintext-secret",
	}))

	// The stored blob must not contain the plaintext: a leaked DB dump should not leak the credential.
	var stored []byte
	err := ds.writer(ctx).GetContext(ctx, &stored,
		`SELECT client_secret FROM mdm_microsoft_graph_credentials WHERE tenant_id = ?`, testTenantA)
	require.NoError(t, err)
	require.NotEmpty(t, stored)
	assert.NotContains(t, string(stored), "plaintext-secret")

	// And it decrypts back to the original on read.
	got, err := ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	assert.Equal(t, "plaintext-secret", got.ClientSecret)
}

func testGraphCredentialDelete(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "secret-a",
	}))
	require.NoError(t, ds.DeleteMicrosoftGraphCredential(ctx, testTenantA))

	creds, err := ds.ListMicrosoftGraphCredentials(ctx)
	require.NoError(t, err)
	require.Empty(t, creds)

	// Deleting an absent tenant is a no-op, not an error, so a reconciling caller need not check first.
	require.NoError(t, ds.DeleteMicrosoftGraphCredential(ctx, "no-such-tenant"))
}

func testGraphCredentialInvalidFlag(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "secret-a",
	}))

	// First transition reports a change; a repeat does not, so a sync that keeps failing does not re-notify.
	wasSet, err := ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)
	assert.True(t, wasSet)

	wasSet, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)
	assert.False(t, wasSet)

	got, err := ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	assert.True(t, got.CredentialInvalid)

	// Clearing works the same way.
	wasSet, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, false)
	require.NoError(t, err)
	assert.True(t, wasSet)

	got, err = ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	assert.False(t, got.CredentialInvalid)

	// A rotated credential keeps whatever sync state it had: a config write replaces the secret, not the record of the
	// last sync.
	_, err = ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)
	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "rotated",
	}))
	got, err = ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	assert.True(t, got.CredentialInvalid)
}

func testGraphCredentialSyncResult(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "secret-a",
	}))

	syncErr := "AADSTS7000222: client secret expired"
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, &syncErr))

	got, err := ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	require.NotNil(t, got.LastSyncedAt)
	require.NotNil(t, got.LastSyncError)
	assert.Equal(t, syncErr, *got.LastSyncError)

	// A subsequent success clears the error.
	require.NoError(t, ds.RecordMicrosoftGraphSyncResult(ctx, testTenantA, nil))
	got, err = ds.GetMicrosoftGraphCredential(ctx, testTenantA)
	require.NoError(t, err)
	require.NotNil(t, got.LastSyncedAt)
	assert.Nil(t, got.LastSyncError)
}

func testHostAutopilotDeviceUpsertAndGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "autopilot-1", "1.1.1.1", "ap-key-1", "ap-uuid-1", time.Now())

	// Unknown host reads as not found.
	_, err := ds.GetHostAutopilotDevice(ctx, host.ID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	dev := &fleet.HostAutopilotDevice{
		HostID:            host.ID,
		AutopilotDeviceID: "747c1c60-fecb-4533-a5de-81c3068091d8",
		AzureADDeviceID:   "261b8f91-f3fb-4f3d-bc31-de657b7f002b",
		GroupTag:          "Engineering",
		HardwareSerial:    "VICTOR1776257483",
		TenantID:          testTenantA,
	}
	require.NoError(t, ds.UpsertHostAutopilotDevice(ctx, dev))

	got, err := ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "Engineering", got.GroupTag)
	assert.Equal(t, "747c1c60-fecb-4533-a5de-81c3068091d8", got.AutopilotDeviceID)

	// Upserting the same host updates in place. Group tags are mutable in Intune, so this is the normal sync path.
	dev.GroupTag = "Sales"
	require.NoError(t, ds.UpsertHostAutopilotDevice(ctx, dev))
	got, err = ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sales", got.GroupTag)

	// A soft-deleted record reads as not found: the device left Autopilot and only the host survives.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_autopilot_devices SET deleted_at = NOW() WHERE host_id = ?`, host.ID)
	require.NoError(t, err)
	_, err = ds.GetHostAutopilotDevice(ctx, host.ID)
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Re-registering the device revives the record rather than leaving it hidden behind the tombstone.
	require.NoError(t, ds.UpsertHostAutopilotDevice(ctx, dev))
	got, err = ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Equal(t, "Sales", got.GroupTag)
}

func testHostAutopilotDeviceListByTenant(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostA := test.NewHost(t, ds, "autopilot-a", "1.1.1.1", "ap-key-a", "ap-uuid-a", time.Now())
	hostB := test.NewHost(t, ds, "autopilot-b", "1.1.1.2", "ap-key-b", "ap-uuid-b", time.Now())
	hostC := test.NewHost(t, ds, "autopilot-c", "1.1.1.3", "ap-key-c", "ap-uuid-c", time.Now())

	for _, d := range []*fleet.HostAutopilotDevice{
		{HostID: hostA.ID, TenantID: testTenantA, HardwareSerial: "serial-a", GroupTag: "Engineering"},
		{HostID: hostB.ID, TenantID: testTenantA, HardwareSerial: "serial-b"},
		{HostID: hostC.ID, TenantID: testTenantB, HardwareSerial: "serial-c"},
	} {
		require.NoError(t, ds.UpsertHostAutopilotDevice(ctx, d))
	}

	devices, err := ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 2)

	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantB)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, hostC.ID, devices[0].HostID)

	// Soft-deleted records drop out of the listing.
	_, err = ds.writer(ctx).ExecContext(ctx, `UPDATE host_autopilot_devices SET deleted_at = NOW() WHERE host_id = ?`, hostB.ID)
	require.NoError(t, err)

	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, hostA.ID, devices[0].HostID)

	// Deleting the host cascades the Autopilot record away.
	require.NoError(t, ds.DeleteHost(ctx, hostA.ID))
	devices, err = ds.ListHostAutopilotDevices(ctx, testTenantA)
	require.NoError(t, err)
	require.Empty(t, devices)
}

func testHostAutopilotDeviceMaxGroupTag(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	host := test.NewHost(t, ds, "autopilot-maxtag", "1.1.1.9", "ap-key-max", "ap-uuid-max", time.Now())

	// 2048 characters is Intune's documented maximum. The column is sized for it exactly so real tags are never
	// truncated on the way in.
	maxTag := strings.Repeat("z", 2048)
	require.NoError(t, ds.UpsertHostAutopilotDevice(ctx, &fleet.HostAutopilotDevice{
		HostID: host.ID, TenantID: testTenantA, HardwareSerial: "serial-max", GroupTag: maxTag,
	}))

	got, err := ds.GetHostAutopilotDevice(ctx, host.ID)
	require.NoError(t, err)
	assert.Len(t, got.GroupTag, 2048)
	assert.Equal(t, maxTag, got.GroupTag)
}

// The config API reads metadata rather than the full credential, so a missing or rotated server private key cannot
// fail GET /config: nothing is decrypted on that path, and the secret is masked in the response anyway.
func testGraphCredentialMetadataOmitsSecret(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	require.NoError(t, ds.UpsertMicrosoftGraphCredential(ctx, &fleet.MicrosoftGraphCredential{
		TenantID: testTenantA, ClientID: "client-a", ClientSecret: "plaintext-secret",
	}))
	_, err := ds.SetMicrosoftGraphCredentialInvalid(ctx, testTenantA, true)
	require.NoError(t, err)

	meta, err := ds.ListMicrosoftGraphCredentialMetadata(ctx)
	require.NoError(t, err)
	require.Len(t, meta, 1)
	assert.Equal(t, testTenantA, meta[0].TenantID)
	assert.Equal(t, "client-a", meta[0].ClientID)
	assert.Empty(t, meta[0].ClientSecret, "the metadata read must not carry the secret")
	// The fields the UI needs, including the banner flag, still come through.
	assert.True(t, meta[0].CredentialInvalid)
}
