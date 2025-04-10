package fleet

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestMacOSUpdatesValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := []struct {
			name string
			m    AppleOSUpdateSettings
		}{
			{"empty", AppleOSUpdateSettings{}},
			{
				"with full version",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"without patch version",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("10.15"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"only major version",
				AppleOSUpdateSettings{
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
			m    AppleOSUpdateSettings
		}{
			{
				"version but no deadline",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString(""),
				},
			},
			{
				"deadline with timestamp",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("10.15.0"),
					Deadline:       optjson.SetString("2020-01-01T00:00:00Z"),
				},
			},
			{
				"incomplete date",
				AppleOSUpdateSettings{
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
			m    AppleOSUpdateSettings
		}{
			{
				"deadline but no version",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString(""),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"version with build info",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("10.15.0 (19A583)"),
					Deadline:       optjson.SetString("2020-01-01"),
				},
			},
			{
				"version with patch info",
				AppleOSUpdateSettings{
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

func TestWindowsUpdatesValidate(t *testing.T) {
	cases := []struct {
		name    string
		w       WindowsUpdates
		wantErr string
	}{
		{"empty", WindowsUpdates{}, ""},
		{"explicitly unset", WindowsUpdates{DeadlineDays: optjson.Int{Set: false}, GracePeriodDays: optjson.Int{Set: false}}, ""},
		{"explicitly null", WindowsUpdates{DeadlineDays: optjson.Int{Set: true, Valid: false}, GracePeriodDays: optjson.Int{Set: true, Valid: false}}, ""},
		{"explicitly set to 0", WindowsUpdates{DeadlineDays: optjson.SetInt(0), GracePeriodDays: optjson.SetInt(0)}, ""},
		{"set to valid values", WindowsUpdates{DeadlineDays: optjson.SetInt(20), GracePeriodDays: optjson.SetInt(4)}, ""},
		{"deadline null grace set", WindowsUpdates{DeadlineDays: optjson.Int{Set: true, Valid: false}, GracePeriodDays: optjson.SetInt(2)}, "deadline_days is required when grace_period_days is provided"},
		{"grace null deadline set", WindowsUpdates{DeadlineDays: optjson.SetInt(10), GracePeriodDays: optjson.Int{Set: true, Valid: false}}, "grace_period_days is required when deadline_days is provided"},
		{"negative deadline", WindowsUpdates{DeadlineDays: optjson.SetInt(-1), GracePeriodDays: optjson.SetInt(2)}, "deadline_days must be an integer between 0 and 30"},
		{"negative grace", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(-2)}, "grace_period_days must be an integer between 0 and 7"},
		{"deadline out of range", WindowsUpdates{DeadlineDays: optjson.SetInt(1000), GracePeriodDays: optjson.SetInt(2)}, "deadline_days must be an integer between 0 and 30"},
		{"grace out of range", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(1000)}, "grace_period_days must be an integer between 0 and 7"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.w.Validate()
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestWindowsUpdatesEqual(t *testing.T) {
	cases := []struct {
		name   string
		w1, w2 WindowsUpdates
		want   bool
	}{
		{"both empty", WindowsUpdates{}, WindowsUpdates{}, true},
		{"both all set", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(2)}, WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(2)}, true},
		{"both all null", WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}}, WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}}, true},
		{"both all set to 0", WindowsUpdates{DeadlineDays: optjson.SetInt(0), GracePeriodDays: optjson.SetInt(0)}, WindowsUpdates{DeadlineDays: optjson.SetInt(0), GracePeriodDays: optjson.SetInt(0)}, true},
		{"different all set", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(2)}, WindowsUpdates{DeadlineDays: optjson.SetInt(3), GracePeriodDays: optjson.SetInt(4)}, false},
		{"different set deadline", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(2)}, WindowsUpdates{DeadlineDays: optjson.SetInt(3), GracePeriodDays: optjson.SetInt(2)}, false},
		{"different set grace", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(2)}, WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(3)}, false},
		{"different null deadline", WindowsUpdates{DeadlineDays: optjson.SetInt(0), GracePeriodDays: optjson.SetInt(2)}, WindowsUpdates{DeadlineDays: optjson.Int{Set: true, Valid: false}, GracePeriodDays: optjson.SetInt(2)}, false},
		{"different null grace", WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.SetInt(0)}, WindowsUpdates{DeadlineDays: optjson.SetInt(1), GracePeriodDays: optjson.Int{Set: true, Valid: false}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.w1.Equal(tc.w2)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMacOSUpdatesConfigured(t *testing.T) {
	cases := []struct {
		version  string
		deadline string
		out      bool
	}{
		{"", "", false},
		{"", "", false},
		{"12.3", "", false},
		{"", "12-03-2022", false},
		{"12.3", "12-03-2022", true},
	}

	for _, tc := range cases {
		m := AppleOSUpdateSettings{
			MinimumVersion: optjson.SetString(tc.version),
			Deadline:       optjson.SetString(tc.deadline),
		}
		require.Equal(t, tc.out, m.Configured())
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
		expectedResult              bool
	}{
		{
			name:                        "None enabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: false,
			expectedResult:              false,
		},
		{
			name:                        "MacOS enabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: false,
			expectedResult:              true,
		},
		{
			name:                        "Both enabled",
			macOSEnabledAndConfigured:   true,
			windowsEnabledAndConfigured: true,
			expectedResult:              true,
		},
		{
			name:                        "Windows enabled",
			macOSEnabledAndConfigured:   false,
			windowsEnabledAndConfigured: true,
			expectedResult:              true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
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
		require.NotEqual(t,
			reflect.ValueOf(f.DetailQueryOverrides).Pointer(),
			reflect.ValueOf(clone.DetailQueryOverrides).Pointer(),
		)
		// map values are pointers, check that they have been cloned
		require.NotSame(t, f.DetailQueryOverrides["foo"], clone.DetailQueryOverrides["foo"])
		// the map content itself is equal
		require.Equal(t, f.DetailQueryOverrides, clone.DetailQueryOverrides)
	})
}

func TestMDMUrl(t *testing.T) {
	cases := []struct {
		name      string
		mdmURL    string
		serverURL string
		want      string
	}{
		{
			name:      "mdm url set",
			mdmURL:    "https://mdm.example.com",
			serverURL: "https://fleet.example.com",
			want:      "https://mdm.example.com",
		},
		{
			name:      "mdm url not set",
			mdmURL:    "",
			serverURL: "https://mdm.example.com",
			want:      "https://mdm.example.com",
		},
		{
			name:      "mdm url and server url not set",
			mdmURL:    "",
			serverURL: "",
			want:      "",
		},
		{
			name:      "server url not set",
			mdmURL:    "https://mdm.example.com",
			serverURL: "",
			want:      "https://mdm.example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appConfig := AppConfig{
				MDM:            MDM{AppleServerURL: tc.mdmURL},
				ServerSettings: ServerSettings{ServerURL: tc.serverURL},
			}
			require.Equal(t, tc.want, appConfig.MDMUrl())
		})
	}
}
