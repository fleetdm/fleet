package azure

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestOrgLogo(t *testing.T) {
	ctx := context.Background()
	store := setupTestOrgLogoStore(t, "org-logos-unit-test")

	getAndCheck := func(mode fleet.OrgLogoMode, expected []byte) {
		rc, sz, err := store.Get(ctx, mode)
		require.NoError(t, err)
		require.EqualValues(t, len(expected), sz)
		defer rc.Close()

		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, expected, got)

		exists, err := store.Exists(ctx, mode)
		require.NoError(t, err)
		require.True(t, exists)
	}

	// get a non-existing logo
	blob, length, err := store.Get(ctx, fleet.OrgLogoModeLight)
	require.Error(t, err)
	require.True(t, fleet.IsNotFound(err))
	require.Nil(t, blob)
	require.Zero(t, length)

	exists, err := store.Exists(ctx, fleet.OrgLogoModeLight)
	require.NoError(t, err)
	require.False(t, exists)

	// store the light logo
	light := []byte("light-logo-content")
	err = store.Put(ctx, fleet.OrgLogoModeLight, bytes.NewReader(light))
	require.NoError(t, err)
	getAndCheck(fleet.OrgLogoModeLight, light)

	// the dark logo is still missing
	exists, err = store.Exists(ctx, fleet.OrgLogoModeDark)
	require.NoError(t, err)
	require.False(t, exists)

	// store the dark logo
	dark := []byte("dark-logo-content")
	err = store.Put(ctx, fleet.OrgLogoModeDark, bytes.NewReader(dark))
	require.NoError(t, err)
	getAndCheck(fleet.OrgLogoModeDark, dark)

	// replace the light logo
	light2 := []byte("light-logo-content-v2")
	err = store.Put(ctx, fleet.OrgLogoModeLight, bytes.NewReader(light2))
	require.NoError(t, err)
	getAndCheck(fleet.OrgLogoModeLight, light2)

	// delete the light logo, the dark one is unaffected
	err = store.Delete(ctx, fleet.OrgLogoModeLight)
	require.NoError(t, err)

	exists, err = store.Exists(ctx, fleet.OrgLogoModeLight)
	require.NoError(t, err)
	require.False(t, exists)

	getAndCheck(fleet.OrgLogoModeDark, dark)

	// deleting again is a no-op
	err = store.Delete(ctx, fleet.OrgLogoModeLight)
	require.NoError(t, err)
}
