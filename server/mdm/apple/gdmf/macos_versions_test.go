package gdmf

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRequiredMacOSVersions(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)

	assets := []Asset{
		// macOS 26 track
		{ProductVersion: "26.4.1", PostingDate: "2026-07-20", Build: "a"},
		{ProductVersion: "26.4.0", PostingDate: "2026-07-01", Build: "b"},
		{ProductVersion: "26.3.0", PostingDate: "2026-06-01", Build: "c"},
		// macOS 15 track
		{ProductVersion: "15.7.5", PostingDate: "2026-07-20", Build: "d"},
		{ProductVersion: "15.7.4", PostingDate: "2026-06-15", Build: "e"},
		// older major should be ignored (only top 2 majors)
		{ProductVersion: "14.7.6", PostingDate: "2026-07-20", Build: "f"},
	}

	t.Run("grace 0 requires latest per major", func(t *testing.T) {
		floors := RequiredMacOSVersions(assets, 0, now)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.1"},
			{Major: 15, Version: "15.7.5"},
		}, floors)
		require.Equal(t,
			"SELECT 1 FROM os_version WHERE (major = 26 AND version_compare(version, '26.4.1') >= 0) OR (major = 15 AND version_compare(version, '15.7.5') >= 0);",
			PolicyQuery(floors),
		)
	})

	t.Run("grace 30 within window allows previous", func(t *testing.T) {
		// 26.4.1 and 15.7.5 posted 2026-07-20 → 11 days old < 30
		floors := RequiredMacOSVersions(assets, 30, now)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.0"},
			{Major: 15, Version: "15.7.4"},
		}, floors)
	})

	t.Run("grace 30 on day 29 still allows previous", func(t *testing.T) {
		day29 := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC) // 29 days after 2026-07-20
		floors := RequiredMacOSVersions(assets, 30, day29)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.0"},
			{Major: 15, Version: "15.7.4"},
		}, floors)
	})

	t.Run("grace 30 on day 30 requires latest", func(t *testing.T) {
		day30 := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC) // 30 days after 2026-07-20
		floors := RequiredMacOSVersions(assets, 30, day30)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.1"},
			{Major: 15, Version: "15.7.5"},
		}, floors)
	})

	t.Run("missing previous falls back to latest", func(t *testing.T) {
		onlyLatest := []Asset{
			{ProductVersion: "26.4.1", PostingDate: "2026-07-29", Build: "a"},
			{ProductVersion: "15.7.5", PostingDate: "2026-07-29", Build: "b"},
		}
		floors := RequiredMacOSVersions(onlyLatest, 30, now)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.1"},
			{Major: 15, Version: "15.7.5"},
		}, floors)
	})

	t.Run("missing posting date requires latest under grace", func(t *testing.T) {
		noDate := []Asset{
			{ProductVersion: "26.4.1", Build: "a"},
			{ProductVersion: "26.4.0", PostingDate: "2026-07-01", Build: "b"},
			{ProductVersion: "15.7.5", Build: "c"},
			{ProductVersion: "15.7.4", PostingDate: "2026-06-15", Build: "d"},
		}
		floors := RequiredMacOSVersions(noDate, 30, now)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.1"},
			{Major: 15, Version: "15.7.5"},
		}, floors)
	})

	t.Run("empty assets", func(t *testing.T) {
		require.Nil(t, RequiredMacOSVersions(nil, 0, now))
		require.Empty(t, PolicyQuery(nil))
	})

	t.Run("dedupes builds preferring earliest posting date", func(t *testing.T) {
		dupes := []Asset{
			{ProductVersion: "26.4.1", PostingDate: "2026-07-22", Build: "late"},
			{ProductVersion: "26.4.1", PostingDate: "2026-07-20", Build: "early"},
			{ProductVersion: "26.4.0", PostingDate: "2026-07-01", Build: "b"},
			{ProductVersion: "15.7.5", PostingDate: "2026-07-20", Build: "d"},
			{ProductVersion: "15.7.4", PostingDate: "2026-06-15", Build: "e"},
		}
		// Within grace based on earliest posting date 2026-07-20
		floors := RequiredMacOSVersions(dupes, 30, now)
		require.Equal(t, []VersionFloor{
			{Major: 26, Version: "26.4.0"},
			{Major: 15, Version: "15.7.4"},
		}, floors)
	})
}

func TestMacOSAssetsForCurrencyPolicies(t *testing.T) {
	meta := &AssetMetadata{
		PublicAssetSets: AssetSets{MacOS: []Asset{{ProductVersion: "26.0"}}},
		AssetSets:       AssetSets{MacOS: []Asset{{ProductVersion: "26.0"}, {ProductVersion: "25.0"}}},
	}
	require.Equal(t, meta.AssetSets.MacOS, MacOSAssetsForCurrencyPolicies(meta))

	meta.AssetSets.MacOS = nil
	require.Equal(t, meta.PublicAssetSets.MacOS, MacOSAssetsForCurrencyPolicies(meta))
	require.Nil(t, MacOSAssetsForCurrencyPolicies(nil))
}

func TestMacOSCurrencyPolicies(t *testing.T) {
	policies := MacOSCurrencyPolicies()
	require.Len(t, policies, 2)
	require.Equal(t, FleetManagedKeyMacOSUpToDate, policies[0].Key)
	require.Equal(t, GraceDaysUpToDate, policies[0].GraceDays)
	require.Equal(t, FleetManagedKeyMacOSAcceptable, policies[1].Key)
	require.Equal(t, GraceDaysAcceptable, policies[1].GraceDays)
}

func TestPolicyQueryRejectsUnsafeOrMalformedVersions(t *testing.T) {
	require.Empty(t, PolicyQuery([]VersionFloor{{Major: 15, Version: "15.7.5' OR '1'='1"}}))
	require.Empty(t, PolicyQuery([]VersionFloor{{Major: 26, Version: "26..5"}}))
	require.Equal(t,
		"SELECT 1 FROM os_version WHERE (major = 15 AND version_compare(version, '15.7.5') >= 0);",
		PolicyQuery([]VersionFloor{
			{Major: 15, Version: "15.7.5' OR '1'='1"},
			{Major: 15, Version: "15.7.5"},
		}),
	)
}
