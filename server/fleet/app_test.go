package fleet

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/require"
)

func TestMacOSUpdatesValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := []struct {
			name string
			m    MacOSUpdates
		}{
			{"empty", MacOSUpdates{}},
			{
				"with full version",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"without patch version",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"only major version",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, tc.m.Validate())
			})
		}
	})

	t.Run("invalid deadline", func(t *testing.T) {
		cases := []struct {
			name string
			m    MacOSUpdates
		}{
			{
				"version but no deadline",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString(""),
				},
			},
			{
				"deadline with timestamp",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString("2020-01-01T00:00:00Z"),
				},
			},
			{
				"incomplete date",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString("2020-01"),
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				require.Error(t, tc.m.Validate())
			})
		}
	})

	t.Run("invalid version", func(t *testing.T) {
		cases := []struct {
			name string
			m    MacOSUpdates
		}{
			{
				"deadline but no version",
				MacOSUpdates{
					MinimumVersion: optjson.SetString(""),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"version with build info",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0 (19A583)"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"version with patch info",
				MacOSUpdates{
					MinimumVersion: optjson.SetString("10.15.0-patch1"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				require.Error(t, tc.m.Validate())
			})
		}
	})
}

func TestSSOSettingsIsEmpty(t *testing.T) {
	require.True(t, (SSOProviderSettings{}).IsEmpty())
	require.False(t, (SSOProviderSettings{EntityID: "fleet"}).IsEmpty())
}

func TestMacOSMigrationModeIsValid(t *testing.T) {
	require.True(t, (MacOSMigrationMode("forced")).IsValid())
	require.True(t, (MacOSMigrationMode("voluntary")).IsValid())
	require.False(t, (MacOSMigrationMode("")).IsValid())
	require.False(t, (MacOSMigrationMode("foo")).IsValid())
}
