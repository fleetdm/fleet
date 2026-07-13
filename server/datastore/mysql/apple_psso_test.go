package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplePSSO(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SetOrUpdateAndGet", testPSSOSetOrUpdateAndGet},
		{"ReRegistrationKeepsOldKeys", testPSSOReRegistrationKeepsOldKeys},
		{"RejectsKIDOwnedByAnotherHost", testPSSORejectsKIDOwnedByAnotherHost},
		{"DeleteDevice", testPSSODeleteDevice},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testPSSOSetOrUpdateAndGet(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const hostUUID = "ABCDEFGH-0000-0000-0000-111111111111"

	keys := []fleet.PSSOKey{
		{KID: "kid-sign-1", KeyType: fleet.PSSOKeyTypeSigning, PEM: "sign-pem-1"},
		{KID: "kid-enc-1", KeyType: fleet.PSSOKeyTypeEncryption, PEM: "enc-pem-1"},
	}
	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID, keys))

	device, err := ds.GetPSSODevice(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, hostUUID, device.HostUUID)
	assert.False(t, device.CreatedAt.IsZero())
	assert.False(t, device.UpdatedAt.IsZero())

	_, err = ds.GetPSSODevice(ctx, "unregistered-uuid")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	signKey, err := ds.GetPSSOKey(ctx, "kid-sign-1")
	require.NoError(t, err)
	assert.Equal(t, hostUUID, signKey.HostUUID)
	assert.Equal(t, fleet.PSSOKeyTypeSigning, signKey.KeyType)
	assert.Equal(t, "sign-pem-1", signKey.PEM)

	encKey, err := ds.GetPSSOKey(ctx, "kid-enc-1")
	require.NoError(t, err)
	assert.Equal(t, fleet.PSSOKeyTypeEncryption, encKey.KeyType)
	assert.Equal(t, "enc-pem-1", encKey.PEM)

	_, err = ds.GetPSSOKey(ctx, "no-such-kid")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	listed, err := ds.ListPSSOKeys(ctx, hostUUID)
	require.NoError(t, err)
	assert.Len(t, listed, 2)

	listed, err = ds.ListPSSOKeys(ctx, "unregistered-uuid")
	require.NoError(t, err)
	assert.Empty(t, listed)

	// Upserting the same kid updates the row in place.
	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID, []fleet.PSSOKey{
		{KID: "kid-sign-1", KeyType: fleet.PSSOKeyTypeSigning, PEM: "sign-pem-1-rotated"},
	}))
	signKey, err = ds.GetPSSOKey(ctx, "kid-sign-1")
	require.NoError(t, err)
	assert.Equal(t, "sign-pem-1-rotated", signKey.PEM)

	listed, err = ds.ListPSSOKeys(ctx, hostUUID)
	require.NoError(t, err)
	assert.Len(t, listed, 2)
}

func testPSSOReRegistrationKeepsOldKeys(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const hostUUID = "ABCDEFGH-0000-0000-0000-222222222222"

	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID, []fleet.PSSOKey{
		{KID: "kid-sign-old", KeyType: fleet.PSSOKeyTypeSigning, PEM: "sign-pem-old"},
		{KID: "kid-enc-old", KeyType: fleet.PSSOKeyTypeEncryption, PEM: "enc-pem-old"},
	}))

	// Re-register with fresh keys: old keys must remain resolvable.
	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID, []fleet.PSSOKey{
		{KID: "kid-sign-new", KeyType: fleet.PSSOKeyTypeSigning, PEM: "sign-pem-new"},
		{KID: "kid-enc-new", KeyType: fleet.PSSOKeyTypeEncryption, PEM: "enc-pem-new"},
	}))

	for _, kid := range []string{"kid-sign-old", "kid-enc-old", "kid-sign-new", "kid-enc-new"} {
		key, err := ds.GetPSSOKey(ctx, kid)
		require.NoError(t, err, "kid %s", kid)
		assert.Equal(t, hostUUID, key.HostUUID)
	}

	listed, err := ds.ListPSSOKeys(ctx, hostUUID)
	require.NoError(t, err)
	assert.Len(t, listed, 4)
}

func testPSSORejectsKIDOwnedByAnotherHost(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const (
		hostUUID1 = "ABCDEFGH-0000-0000-0000-555555555555"
		hostUUID2 = "ABCDEFGH-0000-0000-0000-666666666666"
	)

	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID1, []fleet.PSSOKey{
		{KID: "kid-shared", KeyType: fleet.PSSOKeyTypeSigning, PEM: "host1-key"},
	}))

	// A second host must not be able to claim (and overwrite) a kid host1 owns.
	err := ds.SetOrUpdatePSSODevice(ctx, hostUUID2, []fleet.PSSOKey{
		{KID: "kid-shared", KeyType: fleet.PSSOKeyTypeSigning, PEM: "host2-key"},
	})
	require.Error(t, err)
	var conflict *fleet.ConflictError
	require.ErrorAs(t, err, &conflict)

	// host1's key row is untouched.
	key, err := ds.GetPSSOKey(ctx, "kid-shared")
	require.NoError(t, err)
	assert.Equal(t, hostUUID1, key.HostUUID)
	assert.Equal(t, "host1-key", key.PEM)

	// The whole registration rolled back: host2 got no device row.
	_, err = ds.GetPSSODevice(ctx, hostUUID2)
	assert.True(t, fleet.IsNotFound(err))
}

func testPSSODeleteDevice(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	const (
		hostUUID1 = "ABCDEFGH-0000-0000-0000-333333333333"
		hostUUID2 = "ABCDEFGH-0000-0000-0000-444444444444"
	)

	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID1, []fleet.PSSOKey{
		{KID: "kid-sign-h1", KeyType: fleet.PSSOKeyTypeSigning, PEM: "p"},
		{KID: "kid-enc-h1", KeyType: fleet.PSSOKeyTypeEncryption, PEM: "p"},
	}))
	require.NoError(t, ds.SetOrUpdatePSSODevice(ctx, hostUUID2, []fleet.PSSOKey{
		{KID: "kid-sign-h2", KeyType: fleet.PSSOKeyTypeSigning, PEM: "p"},
	}))

	require.NoError(t, ds.DeletePSSODevice(ctx, hostUUID1))

	_, err := ds.GetPSSODevice(ctx, hostUUID1)
	assert.True(t, fleet.IsNotFound(err))

	// Keys cascade with the device row.
	_, err = ds.GetPSSOKey(ctx, "kid-sign-h1")
	assert.True(t, fleet.IsNotFound(err))
	listed, err := ds.ListPSSOKeys(ctx, hostUUID1)
	require.NoError(t, err)
	assert.Empty(t, listed)

	// Other hosts are untouched.
	_, err = ds.GetPSSODevice(ctx, hostUUID2)
	require.NoError(t, err)
	_, err = ds.GetPSSOKey(ctx, "kid-sign-h2")
	require.NoError(t, err)

	// Deleting an unregistered host is a no-op.
	require.NoError(t, ds.DeletePSSODevice(ctx, "never-registered"))
}
