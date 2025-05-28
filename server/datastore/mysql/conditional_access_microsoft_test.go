package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccess(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Setup", testConditionalAccessSetup},
		{"Hosts", testConditionalAccessHosts},
	}

	for _, c := range cases {
		t.Helper()
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)

			c.fn(t, ds)
		})
	}
}

func testConditionalAccessSetup(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	_, err := ds.ConditionalAccessMicrosoftGet(ctx)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	err = ds.ConditionalAccessMicrosoftCreateIntegration(ctx, "foobar", "insecure")
	require.NoError(t, err)
	ca, err := ds.ConditionalAccessMicrosoftGet(ctx)
	require.NoError(t, err)
	require.NotNil(t, ca)
	require.False(t, ca.SetupDone)
	require.Equal(t, "insecure", ca.ProxyServerSecret)
	require.Equal(t, "foobar", ca.TenantID)

	// ConditionalAccessMicrosoftCreateIntegration replaces the existing one.
	err = ds.ConditionalAccessMicrosoftCreateIntegration(ctx, "foobar2", "insecure2")
	require.NoError(t, err)
	ca, err = ds.ConditionalAccessMicrosoftGet(ctx)
	require.NoError(t, err)
	require.NotNil(t, ca)
	require.False(t, ca.SetupDone)
	require.Equal(t, "insecure2", ca.ProxyServerSecret)
	require.Equal(t, "foobar2", ca.TenantID)

	err = ds.ConditionalAccessMicrosoftMarkSetupDone(ctx)
	require.NoError(t, err)

	ca, err = ds.ConditionalAccessMicrosoftGet(ctx)
	require.NoError(t, err)
	require.NotNil(t, ca)
	require.True(t, ca.SetupDone)
	require.Equal(t, "insecure2", ca.ProxyServerSecret)
	require.Equal(t, "foobar2", ca.TenantID)

	err = ds.ConditionalAccessMicrosoftDelete(ctx)
	require.NoError(t, err)

	_, err = ds.ConditionalAccessMicrosoftGet(ctx)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Create a new one after deleting the existing.
	err = ds.ConditionalAccessMicrosoftCreateIntegration(ctx, "foobar3", "insecure3")
	require.NoError(t, err)
	ca, err = ds.ConditionalAccessMicrosoftGet(ctx)
	require.NoError(t, err)
	require.NotNil(t, ca)
	require.False(t, ca.SetupDone)
	require.Equal(t, "insecure3", ca.ProxyServerSecret)
	require.Equal(t, "foobar3", ca.TenantID)
}

func testConditionalAccessHosts(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	err := ds.ConditionalAccessMicrosoftCreateIntegration(ctx, "foobar", "insecure")
	require.NoError(t, err)

	// Test with non-existent host.
	_, err = ds.LoadHostConditionalAccessStatus(ctx, 999_999)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	noTeamHost := newTestHostWithPlatform(t, ds, "host1", "darwin", nil)

	// Test with an existent host but no status yet.
	_, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	// Nothing happens if the host doesn't have an entry yet.
	err = ds.SetHostConditionalAccessStatus(ctx, noTeamHost.ID, false, false)
	require.NoError(t, err)
	_, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))

	err = ds.CreateHostConditionalAccessStatus(ctx, noTeamHost.ID, "entraDeviceID", "foobar@example.onmicrosoft.com")
	require.NoError(t, err)

	s, err := ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.NoError(t, err)
	require.Equal(t, noTeamHost.ID, s.HostID)
	require.Equal(t, "entraDeviceID", s.DeviceID)
	require.Equal(t, "foobar@example.onmicrosoft.com", s.UserPrincipalName)
	require.Equal(t, "host1", s.DisplayName)
	require.Equal(t, "15.4.1", s.OSVersion)
	require.NotZero(t, s.CreatedAt)
	require.NotZero(t, s.UpdatedAt)
	// When the status entry is created, these are not set yet.
	// These values are updated during detail query ingestion.
	require.Nil(t, s.Managed)
	require.Nil(t, s.Compliant)

	// Execute with same values should do nothing.
	err = ds.CreateHostConditionalAccessStatus(ctx, noTeamHost.ID, "entraDeviceID", "foobar@example.onmicrosoft.com")
	require.NoError(t, err)
	s, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.NoError(t, err)
	require.Equal(t, noTeamHost.ID, s.HostID)
	require.Equal(t, "entraDeviceID", s.DeviceID)
	require.Equal(t, "foobar@example.onmicrosoft.com", s.UserPrincipalName)
	require.Equal(t, "host1", s.DisplayName)
	require.Equal(t, "15.4.1", s.OSVersion)
	require.NotZero(t, s.CreatedAt)
	require.NotZero(t, s.UpdatedAt)
	// These values are updated during detail query ingestion.
	require.Nil(t, s.Managed)
	require.Nil(t, s.Compliant)

	err = ds.SetHostConditionalAccessStatus(ctx, noTeamHost.ID, true, false)
	require.NoError(t, err)

	s, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.NoError(t, err)
	require.Equal(t, noTeamHost.ID, s.HostID)
	require.Equal(t, "entraDeviceID", s.DeviceID)
	require.Equal(t, "foobar@example.onmicrosoft.com", s.UserPrincipalName)
	require.Equal(t, "host1", s.DisplayName)
	require.Equal(t, "15.4.1", s.OSVersion)
	require.NotZero(t, s.CreatedAt)
	require.NotZero(t, s.UpdatedAt)
	require.NotNil(t, s.Managed)
	require.True(t, *s.Managed)
	require.NotNil(t, s.Compliant)
	require.False(t, *s.Compliant)

	err = ds.SetHostConditionalAccessStatus(ctx, noTeamHost.ID, false, true)
	require.NoError(t, err)

	s, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.NoError(t, err)
	require.NotNil(t, s.Managed)
	require.False(t, *s.Managed)
	require.NotNil(t, s.Compliant)
	require.True(t, *s.Compliant)

	// Simulate a device changing its device ID and user principal name
	// (e.g. log out from Entra on device and log in again).
	// We should update its data and clear its statuses.
	err = ds.CreateHostConditionalAccessStatus(ctx, noTeamHost.ID, "entraDeviceID2", "foobar2@example.onmicrosoft.com")
	require.NoError(t, err)
	s, err = ds.LoadHostConditionalAccessStatus(ctx, noTeamHost.ID)
	require.NoError(t, err)
	require.Equal(t, noTeamHost.ID, s.HostID)
	require.Equal(t, "entraDeviceID2", s.DeviceID)
	require.Equal(t, "foobar2@example.onmicrosoft.com", s.UserPrincipalName)
	require.Equal(t, "host1", s.DisplayName)
	require.Equal(t, "15.4.1", s.OSVersion)
	require.NotZero(t, s.CreatedAt)
	require.NotZero(t, s.UpdatedAt)
	require.Nil(t, s.Managed)
	require.Nil(t, s.Compliant)
}
