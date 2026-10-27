package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestConditionalAccessGoogle(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	_, err := ds.ConditionalAccessGoogleGet(ctx)
	require.True(t, fleet.IsNotFound(err))
	// Deleting and setting an error with no integration are no-ops.
	require.NoError(t, ds.ConditionalAccessGoogleDelete(ctx))
	require.NoError(t, ds.ConditionalAccessGoogleSetSetupError(ctx, new("boom")))

	require.NoError(t, ds.ConditionalAccessGoogleCreateIntegration(ctx, "C01234abc", "secret1", "client1"))
	intg, err := ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, &fleet.ConditionalAccessGoogleIntegration{
		CustomerID:        "C01234abc",
		ProxyServerSecret: "secret1",
		OAuthClientID:     "client1",
		SetupDone:         true,
	}, intg)

	require.NoError(t, ds.ConditionalAccessGoogleSetSetupError(ctx, new("token revoked")))
	intg, err = ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, new("token revoked"), intg.SetupError)
	require.NoError(t, ds.ConditionalAccessGoogleSetSetupError(ctx, nil))
	intg, err = ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Nil(t, intg.SetupError)

	// Creating replaces the existing integration, including one with the same customer ID.
	require.NoError(t, ds.ConditionalAccessGoogleCreateIntegration(ctx, "C05678def", "secret2", "client2"))
	require.NoError(t, ds.ConditionalAccessGoogleCreateIntegration(ctx, "C05678def", "secret3", "client3"))
	intg, err = ds.ConditionalAccessGoogleGet(ctx)
	require.NoError(t, err)
	require.Equal(t, "C05678def", intg.CustomerID)
	require.Equal(t, "secret3", intg.ProxyServerSecret)
	require.Equal(t, "client3", intg.OAuthClientID)

	require.NoError(t, ds.ConditionalAccessGoogleDelete(ctx))
	_, err = ds.ConditionalAccessGoogleGet(ctx)
	require.True(t, fleet.IsNotFound(err))
}
