package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCAConfigAssets(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"GetAllCAConfigAssets", testGetAllCAConfigAssets},
		{"SaveCAConfigAssets", testSaveCAConfigAssets},
		{"DeleteCAConfigAssets", testDeleteCAConfigAssets},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds, "ca_config_assets")
			c.fn(t, ds)
		})
	}
}

func testGetAllCAConfigAssets(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty table - should return not found error
	assets, err := ds.GetAllCAConfigAssets(ctx)
	assert.Error(t, err)
	assert.Empty(t, assets)
	assert.True(t, fleet.IsNotFound(err))

	// Insert some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
	}
	err = ds.SaveCAConfigAssets(ctx, testAssets)
	require.NoError(t, err)

	// Test retrieving the assets
	retrievedAssets, err := ds.GetAllCAConfigAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, retrievedAssets, 2)

	// Verify the assets were correctly stored and retrieved
	for _, asset := range testAssets {
		retrievedAsset, ok := retrievedAssets[asset.Name]
		assert.True(t, ok, "Asset %s not found in retrieved assets", asset.Name)
		assert.Equal(t, asset.Type, retrievedAsset.Type)
		assert.Equal(t, asset.Value, retrievedAsset.Value)
	}
}

func testSaveCAConfigAssets(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty assets slice - should be a no-op
	err := ds.SaveCAConfigAssets(ctx, []fleet.CAConfigAsset{})
	assert.NoError(t, err)

	// Insert some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
	}
	err = ds.SaveCAConfigAssets(ctx, testAssets)
	require.NoError(t, err)

	// Verify the assets were correctly stored
	retrievedAssets, err := ds.GetAllCAConfigAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, retrievedAssets, 2)

	// Update an existing asset and add a new one
	updatedAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value1-updated")},
		{Name: "asset3", Type: fleet.CAConfigDigiCert, Value: []byte("value3")},
	}
	err = ds.SaveCAConfigAssets(ctx, updatedAssets)
	require.NoError(t, err)

	// Verify the updates were correctly applied
	retrievedAssets, err = ds.GetAllCAConfigAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, retrievedAssets, 3)

	// Check the updated asset
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, retrievedAssets["asset1"].Type)
	assert.Equal(t, []byte("value1-updated"), retrievedAssets["asset1"].Value)

	// Check the new asset
	assert.Equal(t, fleet.CAConfigDigiCert, retrievedAssets["asset3"].Type)
	assert.Equal(t, []byte("value3"), retrievedAssets["asset3"].Value)

	// Check the unchanged asset
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, retrievedAssets["asset2"].Type)
	assert.Equal(t, []byte("value2"), retrievedAssets["asset2"].Value)
}

func testDeleteCAConfigAssets(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty slice - should be a no-op
	err := ds.DeleteCAConfigAssets(ctx, []string{})
	assert.NoError(t, err)

	// Insert some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
		{Name: "asset3", Type: fleet.CAConfigDigiCert, Value: []byte("value3")},
	}
	err = ds.SaveCAConfigAssets(ctx, testAssets)
	require.NoError(t, err)

	// Verify assets were inserted
	retrievedAssets, err := ds.GetAllCAConfigAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, retrievedAssets, 3)

	// Delete one asset
	err = ds.DeleteCAConfigAssets(ctx, []string{"asset1"})
	require.NoError(t, err)

	// Verify the asset was deleted
	retrievedAssets, err = ds.GetAllCAConfigAssets(ctx)
	require.NoError(t, err)
	assert.Len(t, retrievedAssets, 2)
	_, ok := retrievedAssets["asset1"]
	assert.False(t, ok, "Asset 'asset1' should have been deleted")
	_, ok = retrievedAssets["asset2"]
	assert.True(t, ok, "Asset 'asset2' should still exist")
	_, ok = retrievedAssets["asset3"]
	assert.True(t, ok, "Asset 'asset3' should still exist")

	// Delete multiple assets
	err = ds.DeleteCAConfigAssets(ctx, []string{"asset2", "asset3"})
	require.NoError(t, err)

	// Verify all assets were deleted
	_, err = ds.GetAllCAConfigAssets(ctx)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "Expected NotFound error, got: %v", err)

	// Delete non-existent asset - should not error
	err = ds.DeleteCAConfigAssets(ctx, []string{"non-existent-asset"})
	assert.NoError(t, err)
}
