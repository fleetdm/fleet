package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedLocalAccount(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"SaveAndGetPassword", testManagedLocalAccountSaveAndGetPassword},
		{"GetStatus", testManagedLocalAccountGetStatus},
		{"SetStatus", testManagedLocalAccountSetStatus},
		{"GetByCommandUUID", testManagedLocalAccountGetByCommandUUID},
		{"UpsertOverwrites", testManagedLocalAccountUpsertOverwrites},
		{"NotFound", testManagedLocalAccountNotFound},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
		})
	}
}

func testManagedLocalAccountSaveAndGetPassword(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-1"
	password := "TEST-PASS-WORD1"
	cmdUUID := "cmd-uuid-1"

	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, password, cmdUUID)
	require.NoError(t, err)

	got, err := ds.GetHostManagedLocalAccountPassword(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, "_fleetadmin", got.Username)
	assert.Equal(t, password, got.Password)
	assert.False(t, got.UpdatedAt.IsZero())
}

func testManagedLocalAccountGetStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-status"
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pass", "cmd-status")
	require.NoError(t, err)

	// Initially status is NULL in DB → should return "pending".
	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, "pending", *status.Status)
	assert.False(t, status.PasswordAvailable)

	// After setting to verified, password should be available.
	err = ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryVerified)
	require.NoError(t, err)

	status, err = ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryVerified), *status.Status)
	assert.True(t, status.PasswordAvailable)
}

func testManagedLocalAccountSetStatus(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-set-status"
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pass", "cmd-set-status")
	require.NoError(t, err)

	// Set to failed.
	err = ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryFailed)
	require.NoError(t, err)

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, string(fleet.MDMDeliveryFailed), *status.Status)
	assert.False(t, status.PasswordAvailable)
}

func testManagedLocalAccountGetByCommandUUID(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-cmd"
	cmdUUID := "cmd-uuid-lookup"
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "pass", cmdUUID)
	require.NoError(t, err)

	got, err := ds.GetManagedLocalAccountByCommandUUID(ctx, cmdUUID)
	require.NoError(t, err)
	assert.Equal(t, hostUUID, got)
}

func testManagedLocalAccountUpsertOverwrites(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	hostUUID := "host-uuid-upsert"

	// First save.
	err := ds.SaveHostManagedLocalAccount(ctx, hostUUID, "old-pass", "cmd-old")
	require.NoError(t, err)
	err = ds.SetHostManagedLocalAccountStatus(ctx, hostUUID, fleet.MDMDeliveryVerified)
	require.NoError(t, err)

	// Upsert with new password and command UUID should reset status to NULL (pending).
	err = ds.SaveHostManagedLocalAccount(ctx, hostUUID, "new-pass", "cmd-new")
	require.NoError(t, err)

	got, err := ds.GetHostManagedLocalAccountPassword(ctx, hostUUID)
	require.NoError(t, err)
	assert.Equal(t, "new-pass", got.Password)

	status, err := ds.GetHostManagedLocalAccountStatus(ctx, hostUUID)
	require.NoError(t, err)
	require.NotNil(t, status.Status)
	assert.Equal(t, "pending", *status.Status)

	// Command UUID should be the new one.
	foundHost, err := ds.GetManagedLocalAccountByCommandUUID(ctx, "cmd-new")
	require.NoError(t, err)
	assert.Equal(t, hostUUID, foundHost)

	// Old command UUID should no longer match.
	_, err = ds.GetManagedLocalAccountByCommandUUID(ctx, "cmd-old")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}

func testManagedLocalAccountNotFound(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	_, err := ds.GetHostManagedLocalAccountPassword(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetHostManagedLocalAccountStatus(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetManagedLocalAccountByCommandUUID(ctx, "nonexistent")
	require.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))
}
