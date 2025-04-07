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
		{"GetAllCAConfigAssetsByType", testGetAllCAConfigAssetsByType},
		{"SaveCAConfigAssets", testSaveCAConfigAssets},
		{"DeleteCAConfigAssets", testDeleteCAConfigAssets},
		{"GetCAConfigAsset", testGetCAConfigAsset},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds, "ca_config_assets")
			c.fn(t, ds)
		})
	}
}

func testGetAllCAConfigAssetsByType(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty table - should return not found error for both types
	_, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	_, err = ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Insert some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
		{Name: "asset3", Type: fleet.CAConfigDigiCert, Value: []byte("value3")},
	}
	err = ds.SaveCAConfigAssets(ctx, testAssets)
	require.NoError(t, err)

	// Test retrieving assets by type
	digiCertAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
	require.NoError(t, err)
	assert.Len(t, digiCertAssets, 2)

	// Verify only DigiCert assets were retrieved
	_, ok := digiCertAssets["asset1"]
	assert.True(t, ok, "Asset 'asset1' should be in DigiCert assets")
	_, ok = digiCertAssets["asset3"]
	assert.True(t, ok, "Asset 'asset3' should be in DigiCert assets")
	_, ok = digiCertAssets["asset2"]
	assert.False(t, ok, "Asset 'asset2' should not be in DigiCert assets")

	// Test retrieving assets by another type
	scepProxyAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	require.NoError(t, err)
	assert.Len(t, scepProxyAssets, 1)

	// Verify only CustomSCEPProxy assets were retrieved
	_, ok = scepProxyAssets["asset2"]
	assert.True(t, ok, "Asset 'asset2' should be in CustomSCEPProxy assets")
	_, ok = scepProxyAssets["asset1"]
	assert.False(t, ok, "Asset 'asset1' should not be in CustomSCEPProxy assets")
	_, ok = scepProxyAssets["asset3"]
	assert.False(t, ok, "Asset 'asset3' should not be in CustomSCEPProxy assets")

	// Test with non-existent type - should return not found error
	nonExistentAssets, err := ds.GetAllCAConfigAssetsByType(ctx, "non-existent-type")
	assert.Error(t, err)
	assert.Empty(t, nonExistentAssets)
	assert.True(t, fleet.IsNotFound(err))
}

// Helper function to add test assets and verify they were added correctly
func addAndVerifyTestAssets(t *testing.T, ds *Datastore, ctx context.Context, assets []fleet.CAConfigAsset) {
	err := ds.SaveCAConfigAssets(ctx, assets)
	require.NoError(t, err)

	// Group assets by type
	digiCertAssets := make([]fleet.CAConfigAsset, 0)
	scepProxyAssets := make([]fleet.CAConfigAsset, 0)

	for _, asset := range assets {
		switch asset.Type {
		case fleet.CAConfigDigiCert:
			digiCertAssets = append(digiCertAssets, asset)
		case fleet.CAConfigCustomSCEPProxy:
			scepProxyAssets = append(scepProxyAssets, asset)
		default:
			t.Fatalf("Unsupported asset type: %s", asset.Type)
		}
	}

	// Verify DigiCert assets if any
	if len(digiCertAssets) > 0 {
		retrievedAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
		require.NoError(t, err)

		for _, asset := range digiCertAssets {
			retrievedAsset, ok := retrievedAssets[asset.Name]
			assert.True(t, ok, "DigiCert asset %s not found in retrieved assets", asset.Name)
			assert.Equal(t, asset.Type, retrievedAsset.Type)
			assert.Equal(t, asset.Value, retrievedAsset.Value)
		}
	}

	// Verify SCEP Proxy assets if any
	if len(scepProxyAssets) > 0 {
		retrievedAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
		require.NoError(t, err)

		for _, asset := range scepProxyAssets {
			retrievedAsset, ok := retrievedAssets[asset.Name]
			assert.True(t, ok, "SCEP Proxy asset %s not found in retrieved assets", asset.Name)
			assert.Equal(t, asset.Type, retrievedAsset.Type)
			assert.Equal(t, asset.Value, retrievedAsset.Value)
		}
	}
}

func testSaveCAConfigAssets(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty assets slice - should be a no-op
	err := ds.SaveCAConfigAssets(ctx, []fleet.CAConfigAsset{})
	assert.NoError(t, err)

	// Insert and verify some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
	}
	addAndVerifyTestAssets(t, ds, ctx, testAssets)

	// Update an existing asset and add a new one
	updatedAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value1-updated")},
		{Name: "asset3", Type: fleet.CAConfigDigiCert, Value: []byte("value3")},
	}
	err = ds.SaveCAConfigAssets(ctx, updatedAssets)
	require.NoError(t, err)

	// Verify the updates were correctly applied - check DigiCert assets
	digiCertAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
	require.NoError(t, err)
	assert.Len(t, digiCertAssets, 1)

	// Check the new DigiCert asset
	assert.Equal(t, fleet.CAConfigDigiCert, digiCertAssets["asset3"].Type)
	assert.Equal(t, []byte("value3"), digiCertAssets["asset3"].Value)

	// Verify the updates were correctly applied - check SCEP Proxy assets
	scepProxyAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	require.NoError(t, err)
	assert.Len(t, scepProxyAssets, 2)

	// Check the updated asset
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, scepProxyAssets["asset1"].Type)
	assert.Equal(t, []byte("value1-updated"), scepProxyAssets["asset1"].Value)

	// Check the unchanged asset
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, scepProxyAssets["asset2"].Type)
	assert.Equal(t, []byte("value2"), scepProxyAssets["asset2"].Value)
}

func testDeleteCAConfigAssets(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with empty slice - should be a no-op
	err := ds.DeleteCAConfigAssets(ctx, []string{})
	assert.NoError(t, err)

	// Insert and verify some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
		{Name: "asset3", Type: fleet.CAConfigDigiCert, Value: []byte("value3")},
	}
	addAndVerifyTestAssets(t, ds, ctx, testAssets)

	// Delete one asset
	err = ds.DeleteCAConfigAssets(ctx, []string{"asset1"})
	require.NoError(t, err)

	// Verify the asset was deleted by checking both types
	// Check DigiCert assets
	digiCertAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
	require.NoError(t, err)
	assert.Len(t, digiCertAssets, 1)
	_, ok := digiCertAssets["asset1"]
	assert.False(t, ok, "DigiCert asset 'asset1' should have been deleted")
	_, ok = digiCertAssets["asset3"]
	assert.True(t, ok, "DigiCert asset 'asset3' should still exist")

	// Check SCEP Proxy assets
	scepProxyAssets, err := ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	require.NoError(t, err)
	assert.Len(t, scepProxyAssets, 1)
	_, ok = scepProxyAssets["asset2"]
	assert.True(t, ok, "SCEP Proxy asset 'asset2' should still exist")

	// Delete multiple assets
	err = ds.DeleteCAConfigAssets(ctx, []string{"asset2", "asset3"})
	require.NoError(t, err)

	// Verify all assets were deleted - both types should return not found
	_, err = ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigDigiCert)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "Expected NotFound error for DigiCert assets, got: %v", err)

	_, err = ds.GetAllCAConfigAssetsByType(ctx, fleet.CAConfigCustomSCEPProxy)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err), "Expected NotFound error for SCEP Proxy assets, got: %v", err)

	// Delete non-existent asset - should not error
	err = ds.DeleteCAConfigAssets(ctx, []string{"non-existent-asset"})
	assert.NoError(t, err)
}

func testGetCAConfigAsset(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	// Test with non-existent asset - should return not found error
	asset, err := ds.GetCAConfigAsset(ctx, "non-existent-asset", fleet.CAConfigDigiCert)
	assert.Error(t, err)
	assert.Nil(t, asset)
	assert.True(t, fleet.IsNotFound(err))

	// Insert and verify some test assets
	testAssets := []fleet.CAConfigAsset{
		{Name: "asset1", Type: fleet.CAConfigDigiCert, Value: []byte("value1")},
		{Name: "asset2", Type: fleet.CAConfigCustomSCEPProxy, Value: []byte("value2")},
	}
	addAndVerifyTestAssets(t, ds, ctx, testAssets)

	// Test retrieving an existing asset by name and type
	asset, err = ds.GetCAConfigAsset(ctx, "asset1", fleet.CAConfigDigiCert)
	require.NoError(t, err)
	require.NotNil(t, asset)
	assert.Equal(t, "asset1", asset.Name)
	assert.Equal(t, fleet.CAConfigDigiCert, asset.Type)
	assert.Equal(t, []byte("value1"), asset.Value)

	// Test retrieving another existing asset
	asset, err = ds.GetCAConfigAsset(ctx, "asset2", fleet.CAConfigCustomSCEPProxy)
	require.NoError(t, err)
	require.NotNil(t, asset)
	assert.Equal(t, "asset2", asset.Name)
	assert.Equal(t, fleet.CAConfigCustomSCEPProxy, asset.Type)
	assert.Equal(t, []byte("value2"), asset.Value)

	// Test retrieving an asset with a matching name but different type
	asset, err = ds.GetCAConfigAsset(ctx, "asset1", fleet.CAConfigCustomSCEPProxy)
	assert.Error(t, err)
	assert.Nil(t, asset)
	assert.True(t, fleet.IsNotFound(err))
}
