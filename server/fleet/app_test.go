package fleet

import (
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/ptr"
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

func TestMacOSUpdatesEnabledForHost(t *testing.T) {
	hostWithRequirements := &Host{
		OsqueryHostID: ptr.String("notempty"),
		MDMInfo: &HostMDM{
			IsServer: false,
			Enrolled: true,
			Name:     WellKnownMDMFleet,
		},
	}
	cases := []struct {
		version  string
		deadline string
		host     *Host
		out      bool
	}{
		{"", "", &Host{}, false},
		{"", "", hostWithRequirements, false},
		{"12.3", "", hostWithRequirements, false},
		{"", "12-03-2022", hostWithRequirements, false},
		{"12.3", "12-03-2022", &Host{}, false},
		{"12.3", "12-03-2022", hostWithRequirements, true},
	}

	for _, tc := range cases {
		m := MacOSUpdates{
			MinimumVersion: optjson.SetString(tc.version),
			Deadline:       optjson.SetString(tc.deadline),
		}
		require.Equal(t, tc.out, m.EnabledForHost(tc.host))
	}
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

func TestFeaturesCopy(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var f *Features
		require.Nil(t, f.Copy())
	})

	t.Run("shallow copy", func(t *testing.T) {
		f := &Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: false,
		}
		clone := f.Copy()
		require.NotNil(t, clone)
		require.Equal(t, f.EnableHostUsers, clone.EnableHostUsers)
		require.Equal(t, f.EnableSoftwareInventory, clone.EnableSoftwareInventory)
	})

	t.Run("copy AdditionalQueries", func(t *testing.T) {
		rawMessage := json.RawMessage(`{"test": "data"}`)
		f := &Features{
			AdditionalQueries: &rawMessage,
		}
		clone := f.Copy()
		require.NotNil(t, clone.AdditionalQueries)
		require.NotSame(t, f.AdditionalQueries, clone.AdditionalQueries)
		require.Equal(t, *f.AdditionalQueries, *clone.AdditionalQueries)
	})

	t.Run("copy DetailQueryOverrides", func(t *testing.T) {
		f := &Features{
			DetailQueryOverrides: map[string]*string{
				"foo": ptr.String("bar"),
				"baz": nil,
			},
		}
		clone := f.Copy()
		require.NotNil(t, clone.DetailQueryOverrides)
		require.NotSame(t, f.DetailQueryOverrides, clone.DetailQueryOverrides)
		require.Equal(t, f.DetailQueryOverrides["testKey"], clone.DetailQueryOverrides["testKey"])
	})
}
