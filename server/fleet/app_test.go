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

func TestAppConfigDeprecatedFields(t *testing.T) {
	cases := []struct {
		msg                string
		in                 json.RawMessage
		wantFeatures       Features
		wantDiskEncryption bool
	}{
		{"both empty", json.RawMessage(`{}`), Features{}, false},
		{"only one feature set", json.RawMessage(`{"host_settings": {"enable_host_users": true}}`), Features{EnableHostUsers: true}, false},
		{
			"a feature and disk encryption set",
			json.RawMessage(`{"host_settings": {"enable_host_users": true}, "mdm": {"macos_settings": {"enable_disk_encryption": true}}}`),
			Features{EnableHostUsers: true},
			true,
		},
		{
			"features legacy and new setting set",
			json.RawMessage(`{"host_settings": {"enable_host_users": true}, "features": {"enable_host_users": false}}`),
			Features{EnableHostUsers: true},
			false,
		},
		{
			"disk encryption legacy and new setting set",
			json.RawMessage(`{"mdm": {"enable_disk_encryption": false, "macos_settings": {"enable_disk_encryption": true}}}`),
			Features{},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			ac := AppConfig{}
			err := json.Unmarshal(c.in, &ac)
			require.NoError(t, err)
			require.Nil(t, ac.DeprecatedHostSettings)
			require.Nil(t, ac.MDM.MacOSSettings.DeprecatedEnableDiskEncryption)
			require.Equal(t, c.wantFeatures, ac.Features)
			require.Equal(t, c.wantDiskEncryption, ac.MDM.EnableDiskEncryption.Value)

			// marshalling the fields again doesn't contain deprecated fields
			acJSON, err := json.Marshal(ac)
			require.NoError(t, err)
			var resultMap map[string]interface{}
			err = json.Unmarshal(acJSON, &resultMap)
			require.NoError(t, err)

			// host_settings is not present
			_, exists := resultMap["host_settings"]
			require.False(t, exists)

			// mdm.macos_settings.enable_disk_encryption is not present
			mdm, ok := resultMap["mdm"].(map[string]interface{})
			require.True(t, ok)
			macosSettings, ok := mdm["macos_settings"].(map[string]interface{})
			require.True(t, ok)
			_, exists = macosSettings["enable_disk_encryption"]
			require.False(t, exists)

			diskEncryption, exists := mdm["enable_disk_encryption"]
			require.True(t, exists)
			require.EqualValues(t, c.wantDiskEncryption, diskEncryption)

		})
	}

}

func TestAtLeastOnePlatformEnabledAndConfigured(t *testing.T) {
	tests := []struct {
		name                        string
		macOSEnabledAndConfigured   bool
		windowsEnabledAndConfigured bool
		isMDMFeatureFlagEnabled     bool
		expectedResult              bool
	}{
		{
			name:                        "None enabled, feature flag disabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: false,
			isMDMFeatureFlagEnabled:     false,
			expectedResult:              false,
		},
		{
			name:                        "MacOS enabled, feature flag disabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: false,
			isMDMFeatureFlagEnabled:     false,
			expectedResult:              true,
		},
		{
			name:                        "Windows enabled, feature flag disabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: true,
			isMDMFeatureFlagEnabled:     false,
			expectedResult:              false,
		},
		{
			name:                        "Both enabled, feature flag disabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: true,
			isMDMFeatureFlagEnabled:     false,
			expectedResult:              true,
		},
		{
			name:                        "None enabled, feature flag enabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: false,
			isMDMFeatureFlagEnabled:     true,
			expectedResult:              false,
		},
		{
			name:                        "MacOS enabled, feature flag enabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: false,
			isMDMFeatureFlagEnabled:     true,
			expectedResult:              true,
		},
		{
			name:                        "Windows enabled, feature flag enabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: true,
			isMDMFeatureFlagEnabled:     true,
			expectedResult:              true,
		},
		{
			name:                        "Both enabled, feature flag enabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: true,
			isMDMFeatureFlagEnabled:     true,
			expectedResult:              true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.isMDMFeatureFlagEnabled {
				t.Setenv("FLEET_DEV_MDM_ENABLED", "1")
			} else {
				t.Setenv("FLEET_DEV_MDM_ENABLED", "0")
			}

			mdm := MDM{
				EnabledAndConfigured:        test.macOSEnabledAndConfigured,
				WindowsEnabledAndConfigured: test.windowsEnabledAndConfigured,
			}
			result := mdm.AtLeastOnePlatformEnabledAndConfigured()
			require.Equal(t, test.expectedResult, result)
		})
	}
}

func TestFeaturesCopy(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var f *Features
		require.Nil(t, f.Copy())
	})

	t.Run("copy value fields", func(t *testing.T) {
		f := &Features{
			EnableHostUsers:         true,
			EnableSoftwareInventory: false,
		}
		clone := f.Copy()
		require.NotNil(t, clone)
		require.Equal(t, f.EnableHostUsers, clone.EnableHostUsers)
		require.Equal(t, f.EnableSoftwareInventory, clone.EnableSoftwareInventory)
		require.Nil(t, clone.AdditionalQueries)
		require.Nil(t, clone.DetailQueryOverrides)
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
		// map values are pointers, check that they have been cloned
		require.NotSame(t, f.DetailQueryOverrides["foo"], clone.DetailQueryOverrides["foo"])
		// the map content itself is equal
		require.Equal(t, f.DetailQueryOverrides, clone.DetailQueryOverrides)
	})
}
