package gdmf

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestSyncMacOSCurrencyPolicies(t *testing.T) {
	ds := new(mock.Store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	orig := getAssetMetadataFn
	t.Cleanup(func() { getAssetMetadataFn = orig })
	getAssetMetadataFn = func(ctx context.Context) (*AssetMetadata, error) {
		return &AssetMetadata{
			AssetSets: AssetSets{
				MacOS: []Asset{
					{ProductVersion: "26.4.1", PostingDate: "2026-07-20", Build: "a", SupportedDevices: []string{"J1"}},
					{ProductVersion: "26.4.0", PostingDate: "2026-07-01", Build: "b", SupportedDevices: []string{"J1"}},
					{ProductVersion: "15.7.5", PostingDate: "2026-07-20", Build: "c", SupportedDevices: []string{"J2"}},
					{ProductVersion: "15.7.4", PostingDate: "2026-06-15", Build: "d", SupportedDevices: []string{"J2"}},
				},
			},
		}, nil
	}

	var replaced []fleet.AppleSoftwareUpdateAsset
	ds.ReplaceAppleSoftwareUpdateAssetsFunc = func(ctx context.Context, class fleet.AppleSoftwareUpdateAssetClass, assets []fleet.AppleSoftwareUpdateAsset) error {
		require.Equal(t, fleet.AppleSoftwareUpdateAssetClassMacOS, class)
		replaced = assets
		return nil
	}

	updated := map[string]string{}
	ds.UpdateFleetManagedPolicyQueriesFunc = func(ctx context.Context, key string, query string) ([]uint, error) {
		updated[key] = query
		return []uint{1}, nil
	}

	err := SyncMacOSCurrencyPolicies(context.Background(), ds, slog.Default(), now)
	require.NoError(t, err)
	require.True(t, ds.ReplaceAppleSoftwareUpdateAssetsFuncInvoked)
	require.Len(t, replaced, 4)

	require.Equal(t,
		"SELECT 1 FROM os_version WHERE (major = 26 AND version_compare(version, '26.4.1') >= 0) OR (major = 15 AND version_compare(version, '15.7.5') >= 0);",
		updated[FleetManagedKeyMacOSUpToDate],
	)
	require.Equal(t,
		"SELECT 1 FROM os_version WHERE (major = 26 AND version_compare(version, '26.4.0') >= 0) OR (major = 15 AND version_compare(version, '15.7.4') >= 0);",
		updated[FleetManagedKeyMacOSAcceptable],
	)
	require.Len(t, updated, 2)
}

func TestSyncMacOSCurrencyPoliciesPreservesCacheOnEmptyFeed(t *testing.T) {
	ds := new(mock.Store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	orig := getAssetMetadataFn
	t.Cleanup(func() { getAssetMetadataFn = orig })
	getAssetMetadataFn = func(ctx context.Context) (*AssetMetadata, error) {
		return &AssetMetadata{}, nil
	}

	ds.ReplaceAppleSoftwareUpdateAssetsFunc = func(ctx context.Context, class fleet.AppleSoftwareUpdateAssetClass, assets []fleet.AppleSoftwareUpdateAsset) error {
		t.Fatal("must not replace cache on empty feed")
		return nil
	}
	ds.UpdateFleetManagedPolicyQueriesFunc = func(ctx context.Context, key string, query string) ([]uint, error) {
		t.Fatal("must not update policies on empty feed")
		return nil, nil
	}

	err := SyncMacOSCurrencyPolicies(context.Background(), ds, slog.Default(), now)
	require.NoError(t, err)
	require.False(t, ds.ReplaceAppleSoftwareUpdateAssetsFuncInvoked)
}

func TestSyncMacOSCurrencyPoliciesPreservesCacheOnUnusableCatalog(t *testing.T) {
	ds := new(mock.Store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	cached := []fleet.AppleSoftwareUpdateAsset{
		{Class: fleet.AppleSoftwareUpdateAssetClassMacOS, ProductVersion: "26.4.1", Build: "cached"},
	}

	orig := getAssetMetadataFn
	t.Cleanup(func() { getAssetMetadataFn = orig })
	getAssetMetadataFn = func(ctx context.Context) (*AssetMetadata, error) {
		return &AssetMetadata{
			AssetSets: AssetSets{MacOS: []Asset{
				{ProductVersion: "", Build: "blank"},
				{ProductVersion: "not-a-version", Build: "invalid"},
				{ProductVersion: "26..5", Build: "malformed-numeric-dot"},
				{ProductVersion: "26.4.1' OR '1'='1", Build: "unsafe"},
			}},
		}, nil
	}

	ds.ReplaceAppleSoftwareUpdateAssetsFunc = func(ctx context.Context, class fleet.AppleSoftwareUpdateAssetClass, assets []fleet.AppleSoftwareUpdateAsset) error {
		cached = assets
		return nil
	}
	ds.UpdateFleetManagedPolicyQueriesFunc = func(ctx context.Context, key string, query string) ([]uint, error) {
		t.Fatal("must not update policies from an unusable catalog")
		return nil, nil
	}

	err := SyncMacOSCurrencyPolicies(context.Background(), ds, slog.Default(), now)
	require.ErrorContains(t, err, "GDMF response contains no usable macOS assets")
	require.Equal(t, []fleet.AppleSoftwareUpdateAsset{
		{Class: fleet.AppleSoftwareUpdateAssetClassMacOS, ProductVersion: "26.4.1", Build: "cached"},
	}, cached)
	require.False(t, ds.ReplaceAppleSoftwareUpdateAssetsFuncInvoked)
	require.False(t, ds.UpdateFleetManagedPolicyQueriesFuncInvoked)
}

func TestSyncMacOSCurrencyPoliciesFiltersMixedCatalog(t *testing.T) {
	ds := new(mock.Store)
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	orig := getAssetMetadataFn
	t.Cleanup(func() { getAssetMetadataFn = orig })
	getAssetMetadataFn = func(ctx context.Context) (*AssetMetadata, error) {
		return &AssetMetadata{
			AssetSets: AssetSets{MacOS: []Asset{
				{ProductVersion: "", Build: "blank"},
				{ProductVersion: "26..5", PostingDate: "2026-07-25", Build: "malformed-numeric-dot"},
				{ProductVersion: "26.4.1", PostingDate: "2026-07-20", Build: "a", SupportedDevices: []string{"J1"}},
				{ProductVersion: "26.4.0", PostingDate: "2026-07-01", Build: "b", SupportedDevices: []string{"J1"}},
				{ProductVersion: "15.7.5' OR '1'='1", PostingDate: "2026-07-20", Build: "unsafe"},
				{ProductVersion: "15.7.5", PostingDate: "2026-07-20", Build: "c", SupportedDevices: []string{"J2"}},
				{ProductVersion: "15.7.4", PostingDate: "2026-06-15", Build: "d", SupportedDevices: []string{"J2"}},
			}},
		}, nil
	}

	var replaced []fleet.AppleSoftwareUpdateAsset
	ds.ReplaceAppleSoftwareUpdateAssetsFunc = func(ctx context.Context, class fleet.AppleSoftwareUpdateAssetClass, assets []fleet.AppleSoftwareUpdateAsset) error {
		require.Equal(t, fleet.AppleSoftwareUpdateAssetClassMacOS, class)
		replaced = assets
		return nil
	}

	updated := map[string]string{}
	ds.UpdateFleetManagedPolicyQueriesFunc = func(ctx context.Context, key string, query string) ([]uint, error) {
		updated[key] = query
		return nil, nil
	}

	err := SyncMacOSCurrencyPolicies(context.Background(), ds, slog.Default(), now)
	require.NoError(t, err)
	require.Len(t, replaced, 4)
	require.Equal(t, []string{"26.4.1", "26.4.0", "15.7.5", "15.7.4"}, []string{
		replaced[0].ProductVersion,
		replaced[1].ProductVersion,
		replaced[2].ProductVersion,
		replaced[3].ProductVersion,
	})
	require.Equal(t,
		"SELECT 1 FROM os_version WHERE (major = 26 AND version_compare(version, '26.4.1') >= 0) OR (major = 15 AND version_compare(version, '15.7.5') >= 0);",
		updated[FleetManagedKeyMacOSUpToDate],
	)
	require.Equal(t,
		"SELECT 1 FROM os_version WHERE (major = 26 AND version_compare(version, '26.4.0') >= 0) OR (major = 15 AND version_compare(version, '15.7.4') >= 0);",
		updated[FleetManagedKeyMacOSAcceptable],
	)
	require.Len(t, updated, 2)
}
