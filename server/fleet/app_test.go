package fleet

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
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

func TestAppleOSUpdatesLatestValidate(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		cases := []struct {
			name string
			m    AppleOSUpdateSettings
		}{
			{
				"latest with deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					DeadlineDays:   optjson.SetInt(14),
				},
			},
			{
				"latest with deadline_days of 1",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					DeadlineDays:   optjson.SetInt(1),
				},
			},
			{
				"latest with explicitly empty deadline",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					Deadline:       optjson.SetString(""),
					DeadlineDays:   optjson.SetInt(14),
				},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				require.NoError(t, tc.m.Validate())
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		cases := []struct {
			name string
			m    AppleOSUpdateSettings
		}{
			{
				"latest without deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
				},
			},
			{
				"latest with null deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					DeadlineDays:   optjson.Int{Set: true, Valid: false},
				},
			},
			{
				"latest with deadline",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					Deadline:       optjson.SetString("2026-09-01"),
					DeadlineDays:   optjson.SetInt(14),
				},
			},
			{
				"latest with zero deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					DeadlineDays:   optjson.SetInt(0),
				},
			},
			{
				"latest with negative deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("latest"),
					DeadlineDays:   optjson.SetInt(-1),
				},
			},
			{
				"specific version with deadline_days",
				AppleOSUpdateSettings{
					MinimumVersion: optjson.SetString("15.1"),
					Deadline:       optjson.SetString("2026-09-01"),
					DeadlineDays:   optjson.SetInt(14),
				},
			},
			{
				"deadline_days with no version",
				AppleOSUpdateSettings{
					DeadlineDays: optjson.SetInt(14),
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
	// nullDeadlineDays is what `"deadline_days": null` unmarshals to: the key was
	// present but carried no value.
	nullDeadlineDays := optjson.Int{Set: true, Valid: false}

	cases := []struct {
		name         string
		version      string
		deadline     string
		deadlineDays optjson.Int
		out          bool
	}{
		{"empty", "", "", optjson.Int{}, false},
		{"version only", "12.3", "", optjson.Int{}, false},
		{"deadline only", "", "12-03-2022", optjson.Int{}, false},
		{"version and deadline", "12.3", "12-03-2022", optjson.Int{}, true},

		// "latest" mode: DeadlineDays stands in for Deadline.
		{"latest with deadline_days", AppleOSUpdateLatestVersion, "", optjson.SetInt(14), true},
		{"latest without deadline_days", AppleOSUpdateLatestVersion, "", optjson.Int{}, false},
		{"latest with null deadline_days", AppleOSUpdateLatestVersion, "", nullDeadlineDays, false},
		{"latest with zero deadline_days", AppleOSUpdateLatestVersion, "", optjson.SetInt(0), false},
		{"cleared", "", "", nullDeadlineDays, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := AppleOSUpdateSettings{
				MinimumVersion: optjson.SetString(tc.version),
				Deadline:       optjson.SetString(tc.deadline),
				DeadlineDays:   tc.deadlineDays,
			}
			require.Equal(t, tc.out, m.Configured())
		})
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
		msg          string
		in           json.RawMessage
		wantFeatures Features
		// wantDiskEncryption is the expected flat mdm.enable_disk_encryption.
		wantDiskEncryption      bool
		wantMacOSDiskEncryption bool
	}{
		{"both empty", json.RawMessage(`{}`), Features{}, false, false},
		{"only one feature set", json.RawMessage(`{"host_settings": {"enable_host_users": true}}`), Features{EnableHostUsers: true}, false, false},
		{
			// macos_settings.enable_disk_encryption is no longer an alias of
			// the flat toggle: it enables macOS enforcement only, so the
			// virtual flat toggle reads false.
			"a feature and macOS disk encryption set",
			json.RawMessage(`{"host_settings": {"enable_host_users": true}, "mdm": {"macos_settings": {"enable_disk_encryption": true}}}`),
			Features{EnableHostUsers: true},
			false,
			true,
		},
		{
			"features legacy and new setting set",
			json.RawMessage(`{"host_settings": {"enable_host_users": true}, "features": {"enable_host_users": false}}`),
			Features{EnableHostUsers: true},
			false,
			false,
		},
		{
			"flat disk encryption false and macOS setting true",
			json.RawMessage(`{"mdm": {"enable_disk_encryption": false, "macos_settings": {"enable_disk_encryption": true}}}`),
			Features{},
			false,
			true,
		},
		{
			// a legacy-only write fans out to every per-platform setting on
			// serialization
			"flat disk encryption only",
			json.RawMessage(`{"mdm": {"enable_disk_encryption": true}}`),
			Features{},
			true,
			true,
		},
	}

	for _, c := range cases {
		t.Run(c.msg, func(t *testing.T) {
			ac := AppConfig{}
			err := json.Unmarshal(c.in, &ac)
			require.NoError(t, err)
			require.Nil(t, ac.DeprecatedHostSettings)
			require.Equal(t, c.wantFeatures, ac.Features)

			acJSON, err := json.Marshal(ac)
			require.NoError(t, err)
			var resultMap map[string]interface{}
			err = json.Unmarshal(acJSON, &resultMap)
			require.NoError(t, err)

			// host_settings is not present
			_, exists := resultMap["host_settings"]
			require.False(t, exists)

			// the per-platform settings are always explicit booleans
			mdm, ok := resultMap["mdm"].(map[string]any)
			require.True(t, ok)
			macosSettings, ok := mdm["macos_settings"].(map[string]any)
			require.True(t, ok)
			require.EqualValues(t, c.wantMacOSDiskEncryption, macosSettings["enable_disk_encryption"])
			require.EqualValues(t, c.wantDiskEncryption, macosSettings["enable_escrow_disk_encryption_key"])
			windowsSettings, ok := mdm["windows_settings"].(map[string]any)
			require.True(t, ok)
			require.EqualValues(t, c.wantDiskEncryption, windowsSettings["enable_disk_encryption"])
			linuxSettings, ok := mdm["linux_settings"].(map[string]any)
			require.True(t, ok)
			require.EqualValues(t, c.wantDiskEncryption, linuxSettings["enable_escrow_disk_encryption_key"])

			diskEncryption, exists := mdm["enable_disk_encryption"]
			require.True(t, exists)
			require.EqualValues(t, c.wantDiskEncryption, diskEncryption)
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

	t.Run("copy HistoricalData is independent", func(t *testing.T) {
		f := &Features{
			HistoricalData: HistoricalDataSettings{Uptime: true, Vulnerabilities: false},
		}
		clone := f.Copy()
		require.NotNil(t, clone)
		require.Equal(t, f.HistoricalData, clone.HistoricalData)

		// Mutating the original should not affect the clone.
		f.HistoricalData.Uptime = false
		f.HistoricalData.Vulnerabilities = true
		require.True(t, clone.HistoricalData.Uptime)
		require.False(t, clone.HistoricalData.Vulnerabilities)
	})
}

func TestFeaturesApplyDefaults(t *testing.T) {
	t.Run("ApplyDefaults sets historical_data sub-fields true", func(t *testing.T) {
		var f Features
		f.ApplyDefaults()
		require.True(t, f.HistoricalData.Uptime)
		require.True(t, f.HistoricalData.Vulnerabilities)
		require.True(t, f.EnableHostUsers)
	})

	t.Run("ApplyDefaultsForNewInstalls sets historical_data sub-fields true", func(t *testing.T) {
		var f Features
		f.ApplyDefaultsForNewInstalls()
		require.True(t, f.HistoricalData.Uptime)
		require.True(t, f.HistoricalData.Vulnerabilities)
		require.True(t, f.EnableHostUsers)
		require.True(t, f.EnableSoftwareInventory)
	})
}

func TestHistoricalDataSettingsEnabled(t *testing.T) {
	t.Run("uptime dataset returns Uptime field", func(t *testing.T) {
		h := HistoricalDataSettings{Uptime: true, Vulnerabilities: false}
		v, err := h.Enabled("uptime")
		require.NoError(t, err)
		require.True(t, v)

		h.Uptime = false
		v, err = h.Enabled("uptime")
		require.NoError(t, err)
		require.False(t, v)
	})

	t.Run("cve dataset returns Vulnerabilities field", func(t *testing.T) {
		h := HistoricalDataSettings{Uptime: false, Vulnerabilities: true}
		v, err := h.Enabled("cve")
		require.NoError(t, err)
		require.True(t, v)

		h.Vulnerabilities = false
		v, err = h.Enabled("cve")
		require.NoError(t, err)
		require.False(t, v)
	})

	t.Run("unknown dataset returns error", func(t *testing.T) {
		h := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		v, err := h.Enabled("policy_compliance")
		require.Error(t, err)
		require.False(t, v)
		require.Contains(t, err.Error(), "policy_compliance")
	})

	t.Run("empty dataset name returns error", func(t *testing.T) {
		h := HistoricalDataSettings{Uptime: true, Vulnerabilities: true}
		_, err := h.Enabled("")
		require.Error(t, err)
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

func TestAppConfig_ConditionalAccessIdPSSOURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		serverURL string
		envVar    string
		want      string
		wantErr   bool
	}{
		{
			name:      "transforms hostname with okta prefix",
			serverURL: "https://fleet.example.com",
			want:      "https://okta.fleet.example.com",
			wantErr:   false,
		},
		{
			name:      "preserves port in URL",
			serverURL: "https://fleet.example.com:8080",
			want:      "https://okta.fleet.example.com:8080",
			wantErr:   false,
		},
		{
			name:      "handles http scheme",
			serverURL: "http://fleet.localhost",
			want:      "http://okta.fleet.localhost",
			wantErr:   false,
		},
		{
			name:      "handles subdomain",
			serverURL: "https://my.fleet.example.com",
			want:      "https://okta.my.fleet.example.com",
			wantErr:   false,
		},
		{
			name:      "dev override takes precedence",
			serverURL: "https://fleet.example.com",
			envVar:    "https://dev.okta.example.com",
			want:      "https://dev.okta.example.com",
			wantErr:   false,
		},
		{
			name:      "empty server URL returns error",
			serverURL: "",
			want:      "",
			wantErr:   true,
		},
		{
			name:      "invalid URL returns error",
			serverURL: "://invalid-url",
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock getenv function
			getenv := func(key string) string {
				if key == "FLEET_DEV_OKTA_SSO_SERVER_URL" {
					return tt.envVar
				}
				return ""
			}

			appConfig := &AppConfig{
				ServerSettings: ServerSettings{
					ServerURL: tt.serverURL,
				},
			}

			got, err := appConfig.ConditionalAccessIdPSSOURL(getenv)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGoogleCalendarApiKeyMarshalUnmarshal(t *testing.T) {
	t.Run("marshal masked", func(t *testing.T) {
		key := GoogleCalendarApiKey{
			Values: map[string]string{
				"client_email": "test@example.com",
				"private_key":  "secret-key",
			},
		}
		key.SetMasked()

		data, err := json.Marshal(key)
		require.NoError(t, err)
		require.Equal(t, fmt.Sprintf(`"%s"`, MaskedPassword), string(data))
	})

	t.Run("marshal unmasked", func(t *testing.T) {
		key := GoogleCalendarApiKey{
			Values: map[string]string{
				"client_email": "test@example.com",
				"private_key":  "secret-key",
			},
		}

		data, err := json.Marshal(key)
		require.NoError(t, err)
		// Unmarshal to verify it's a valid JSON object
		var parsed map[string]string
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		require.Equal(t, "test@example.com", parsed["client_email"])
		require.Equal(t, "secret-key", parsed["private_key"])
	})

	t.Run("unmarshal masked string", func(t *testing.T) {
		data := fmt.Appendf(nil, `"%s"`, MaskedPassword)
		var key GoogleCalendarApiKey
		err := json.Unmarshal(data, &key)
		require.NoError(t, err)
		require.True(t, key.IsMasked())
		require.True(t, key.IsEmpty())
	})

	t.Run("unmarshal json object", func(t *testing.T) {
		data := []byte(`{"client_email": "test@example.com", "private_key": "secret-key"}`)
		var key GoogleCalendarApiKey
		err := json.Unmarshal(data, &key)
		require.NoError(t, err)
		require.False(t, key.IsMasked())
		require.False(t, key.IsEmpty())
		require.Equal(t, "test@example.com", key.Values["client_email"])
		require.Equal(t, "secret-key", key.Values["private_key"])
	})

	t.Run("unmarshal invalid string", func(t *testing.T) {
		data := []byte(`"some-invalid-string"`)
		var key GoogleCalendarApiKey
		err := json.Unmarshal(data, &key)
		require.Error(t, err)
	})

	t.Run("unmarshal null", func(t *testing.T) {
		data := []byte(`null`)
		var key GoogleCalendarApiKey
		err := json.Unmarshal(data, &key)
		require.NoError(t, err)
		require.False(t, key.IsMasked())
		require.True(t, key.IsEmpty())
	})

	t.Run("full integration roundtrip", func(t *testing.T) {
		// Test the full struct with the API key
		intg := GoogleCalendarIntegration{
			Domain: "example.com",
			ApiKey: GoogleCalendarApiKey{
				Values: map[string]string{
					"client_email": "test@example.com",
					"private_key":  "secret-key",
				},
			},
		}

		// Marshal with unmasked key
		data, err := json.Marshal(intg)
		require.NoError(t, err)

		// Unmarshal and verify
		var parsed GoogleCalendarIntegration
		err = json.Unmarshal(data, &parsed)
		require.NoError(t, err)
		require.Equal(t, "example.com", parsed.Domain)
		require.Equal(t, "test@example.com", parsed.ApiKey.Values["client_email"])
		require.False(t, parsed.ApiKey.IsMasked())

		// Now mask and marshal again
		intg.ApiKey.SetMasked()
		data, err = json.Marshal(intg)
		require.NoError(t, err)
		require.Contains(t, string(data), `"api_key_json":"********"`)
	})
}

func TestOrgInfoNormalizeLogoFields(t *testing.T) {
	cases := []struct {
		name    string
		in      OrgInfo
		want    OrgInfo
		wantErr bool
	}{
		{
			name: "all empty",
			in:   OrgInfo{},
			want: OrgInfo{},
		},
		{
			name: "deprecated dark only -> mirrored to new",
			in:   OrgInfo{OrgLogoURL: "https://example.com/d.png"},
			want: OrgInfo{
				OrgLogoURL:         "https://example.com/d.png",
				OrgLogoURLDarkMode: "https://example.com/d.png",
			},
		},
		{
			name: "new dark only -> mirrored to deprecated",
			in:   OrgInfo{OrgLogoURLDarkMode: "https://example.com/d.png"},
			want: OrgInfo{
				OrgLogoURL:         "https://example.com/d.png",
				OrgLogoURLDarkMode: "https://example.com/d.png",
			},
		},
		{
			name: "deprecated light only -> mirrored to new",
			in:   OrgInfo{OrgLogoURLLightBackground: "https://example.com/l.png"},
			want: OrgInfo{
				OrgLogoURLLightBackground: "https://example.com/l.png",
				OrgLogoURLLightMode:       "https://example.com/l.png",
			},
		},
		{
			name: "new light only -> mirrored to deprecated",
			in:   OrgInfo{OrgLogoURLLightMode: "https://example.com/l.png"},
			want: OrgInfo{
				OrgLogoURLLightBackground: "https://example.com/l.png",
				OrgLogoURLLightMode:       "https://example.com/l.png",
			},
		},
		{
			name: "both modes via deprecated",
			in: OrgInfo{
				OrgLogoURL:                "https://example.com/d.png",
				OrgLogoURLLightBackground: "https://example.com/l.png",
			},
			want: OrgInfo{
				OrgLogoURL:                "https://example.com/d.png",
				OrgLogoURLLightBackground: "https://example.com/l.png",
				OrgLogoURLDarkMode:        "https://example.com/d.png",
				OrgLogoURLLightMode:       "https://example.com/l.png",
			},
		},
		{
			name: "matching dark old + new -> kept",
			in: OrgInfo{
				OrgLogoURL:         "https://example.com/d.png",
				OrgLogoURLDarkMode: "https://example.com/d.png",
			},
			want: OrgInfo{
				OrgLogoURL:         "https://example.com/d.png",
				OrgLogoURLDarkMode: "https://example.com/d.png",
			},
		},
		{
			name: "matching light old + new -> kept",
			in: OrgInfo{
				OrgLogoURLLightBackground: "https://example.com/l.png",
				OrgLogoURLLightMode:       "https://example.com/l.png",
			},
			want: OrgInfo{
				OrgLogoURLLightBackground: "https://example.com/l.png",
				OrgLogoURLLightMode:       "https://example.com/l.png",
			},
		},
		{
			name: "conflicting dark old + new -> error",
			in: OrgInfo{
				OrgLogoURL:         "https://example.com/d1.png",
				OrgLogoURLDarkMode: "https://example.com/d2.png",
			},
			wantErr: true,
		},
		{
			name: "conflicting light old + new -> error",
			in: OrgInfo{
				OrgLogoURLLightBackground: "https://example.com/l1.png",
				OrgLogoURLLightMode:       "https://example.com/l2.png",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			err := got.NormalizeLogoFields()
			if tc.wantErr {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestOrgInfoAbsolutizeLogoURLs(t *testing.T) {
	const fleetHostedDark = "/api/latest/fleet/logo?mode=dark"
	const fleetHostedLight = "/api/latest/fleet/logo?mode=light"
	const externalDark = "https://example.com/dark.png"
	const externalLight = "https://example.com/light.png"

	cases := []struct {
		name      string
		serverURL string
		in        OrgInfo
		want      OrgInfo
	}{
		{
			name:      "empty serverURL is no-op",
			serverURL: "",
			in:        OrgInfo{OrgLogoURL: fleetHostedDark},
			want:      OrgInfo{OrgLogoURL: fleetHostedDark},
		},
		{
			name:      "all empty fields stay empty",
			serverURL: "https://fleet.example.com",
			in:        OrgInfo{},
			want:      OrgInfo{},
		},
		{
			name:      "fleet-hosted relative URL gets absolutized",
			serverURL: "https://fleet.example.com",
			in: OrgInfo{
				OrgLogoURL:                fleetHostedDark,
				OrgLogoURLDarkMode:        fleetHostedDark,
				OrgLogoURLLightBackground: fleetHostedLight,
				OrgLogoURLLightMode:       fleetHostedLight,
			},
			want: OrgInfo{
				OrgLogoURL:                "https://fleet.example.com" + fleetHostedDark,
				OrgLogoURLDarkMode:        "https://fleet.example.com" + fleetHostedDark,
				OrgLogoURLLightBackground: "https://fleet.example.com" + fleetHostedLight,
				OrgLogoURLLightMode:       "https://fleet.example.com" + fleetHostedLight,
			},
		},
		{
			name:      "external URLs are left unchanged",
			serverURL: "https://fleet.example.com",
			in: OrgInfo{
				OrgLogoURL:          externalDark,
				OrgLogoURLDarkMode:  externalDark,
				OrgLogoURLLightMode: externalLight,
			},
			want: OrgInfo{
				OrgLogoURL:          externalDark,
				OrgLogoURLDarkMode:  externalDark,
				OrgLogoURLLightMode: externalLight,
			},
		},
		{
			name:      "trailing slash on serverURL is stripped",
			serverURL: "https://fleet.example.com/",
			in:        OrgInfo{OrgLogoURL: fleetHostedDark},
			want:      OrgInfo{OrgLogoURL: "https://fleet.example.com" + fleetHostedDark},
		},
		{
			name:      "already-absolute fleet URL is left alone (idempotent)",
			serverURL: "https://fleet.example.com",
			in: OrgInfo{
				OrgLogoURL: "https://fleet.example.com" + fleetHostedDark,
			},
			want: OrgInfo{
				OrgLogoURL: "https://fleet.example.com" + fleetHostedDark,
			},
		},
		{
			name:      "mixed external and fleet-hosted",
			serverURL: "https://fleet.example.com",
			in: OrgInfo{
				OrgLogoURL:          fleetHostedDark, // fleet-hosted
				OrgLogoURLLightMode: externalLight,   // external
			},
			want: OrgInfo{
				OrgLogoURL:          "https://fleet.example.com" + fleetHostedDark,
				OrgLogoURLLightMode: externalLight,
			},
		},
		{
			name:      "subdomain serverURL",
			serverURL: "https://eu.acme.fleet.example.com",
			in:        OrgInfo{OrgLogoURL: fleetHostedDark},
			want: OrgInfo{
				OrgLogoURL: "https://eu.acme.fleet.example.com" + fleetHostedDark,
			},
		},
		{
			name:      "serverURL with explicit port",
			serverURL: "https://fleet.example.com:8443",
			in:        OrgInfo{OrgLogoURL: fleetHostedDark},
			want: OrgInfo{
				OrgLogoURL: "https://fleet.example.com:8443" + fleetHostedDark,
			},
		},
		{
			name:      "serverURL with URL prefix path",
			serverURL: "https://example.com/fleet",
			in:        OrgInfo{OrgLogoURL: fleetHostedDark},
			want: OrgInfo{
				OrgLogoURL: "https://example.com/fleet" + fleetHostedDark,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.in
			got.AbsolutizeLogoURLs(tc.serverURL)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestMacOSSetupValidate(t *testing.T) {
	t.Run("validate", func(t *testing.T) {
		cases := []struct {
			name    string
			m       MacOSSetup
			wantErr *string
		}{
			{"empty", MacOSSetup{}, nil},
			{"manual_agent_install without bootstrap package fails", MacOSSetup{ManualAgentInstall: optjson.SetBool(true)}, new("Couldn't enable macos_manual_agent_install. To use this option, first specify a bootstrap package.")},
			{"manual_agent_install with bootstrap package succeeds", MacOSSetup{ManualAgentInstall: optjson.SetBool(true), BootstrapPackage: optjson.SetString("https://example.com/bootstrap.pkg")}, nil},
			{"end_user_local_account_type fails with non accepted value", MacOSSetup{EndUserLocalAccountType: optjson.SetString("invalid")}, new(`end_user_local_account_type only "admin", "standard", and "none" are supported`)},
			{"end_user_local_account_type with accepted value succeeds", MacOSSetup{EndUserLocalAccountType: optjson.SetString("admin")}, nil},
			{"end_user_local_account_type none fails with enable_end_user_local_account false", MacOSSetup{EndUserLocalAccountType: optjson.SetString("none"), EnableManagedLocalAccount: optjson.SetBool(false)}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "none" for the end_user_local_account_type`)},
			{"end_user_local_account_type standard fails with enable_end_user_local_account false", MacOSSetup{EndUserLocalAccountType: optjson.SetString("standard"), EnableManagedLocalAccount: optjson.SetBool(false)}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "standard" for the end_user_local_account_type`)},
			{"end_user_local_account_type none succeeds with enable_end_user_local_account true", MacOSSetup{EndUserLocalAccountType: optjson.SetString("none"), EnableManagedLocalAccount: optjson.SetBool(true)}, nil},
			{"end_user_local_account_type standard succeeds with enable_end_user_local_account true", MacOSSetup{EndUserLocalAccountType: optjson.SetString("standard"), EnableManagedLocalAccount: optjson.SetBool(true)}, nil},
			{"end_user_local_account_type admin succeeds with enable_end_user_local_account false", MacOSSetup{EndUserLocalAccountType: optjson.SetString("admin"), EnableManagedLocalAccount: optjson.SetBool(false)}, nil},
			{"end_user_local_account_type admin succeeds with enable_end_user_local_account true", MacOSSetup{EndUserLocalAccountType: optjson.SetString("admin"), EnableManagedLocalAccount: optjson.SetBool(true)}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.wantErr != nil {
					require.ErrorContains(t, tc.m.Validate(), *tc.wantErr)
					return
				}

				require.NoError(t, tc.m.Validate())
			})
		}
	})

	t.Run("validate against", func(t *testing.T) {
		cases := []struct {
			name    string
			m       MacOSSetup
			against MacOSSetup
			wantErr *string
		}{
			{"empty against empty", MacOSSetup{}, MacOSSetup{}, nil},
			{"manual_agent_install enabled against empty fails", MacOSSetup{ManualAgentInstall: optjson.SetBool(true)}, MacOSSetup{}, new("Couldn't enable macos_manual_agent_install. To use this option, first specify a bootstrap package.")},
			{"manual_agent_install enabled with bootstrap package succeeds", MacOSSetup{ManualAgentInstall: optjson.SetBool(true), BootstrapPackage: optjson.SetString("https://example.com/bootstrap.pkg")}, MacOSSetup{}, nil},
			{"end_user_local_account_type none against empty fails", MacOSSetup{EndUserLocalAccountType: optjson.SetString("none")}, MacOSSetup{}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "none" for the end_user_local_account_type`)},
			{"end_user_local_account_type standard against empty fails", MacOSSetup{EndUserLocalAccountType: optjson.SetString("standard")}, MacOSSetup{}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "standard" for the end_user_local_account_type`)},
			{"enable_end_user_local_account false against none user type fails", MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false)}, MacOSSetup{EndUserLocalAccountType: optjson.SetString("none")}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "none" for the end_user_local_account_type`)},
			{"enable_end_user_local_account false against standard user type fails", MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false)}, MacOSSetup{EndUserLocalAccountType: optjson.SetString("standard")}, new(`enable_create_local_admin_account enable_create_local_admin_account is required to be enabled when using "standard" for the end_user_local_account_type`)},
			{"enable_end_user_local_account false against admin user type succeeds", MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false)}, MacOSSetup{EndUserLocalAccountType: optjson.SetString("admin")}, nil},
			{"enable_end_user_local_account false against none user type succeeds when both are set", MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false), EndUserLocalAccountType: optjson.SetString("admin")}, MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(true), EndUserLocalAccountType: optjson.SetString("none")}, nil},
			{"enable_end_user_local_account false against standard user type succeeds when both are set", MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(false), EndUserLocalAccountType: optjson.SetString("admin")}, MacOSSetup{EnableManagedLocalAccount: optjson.SetBool(true), EndUserLocalAccountType: optjson.SetString("standard")}, nil},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if tc.wantErr != nil {
					require.ErrorContains(t, tc.m.ValidateAgainst(tc.against), *tc.wantErr)
					return
				}

				require.NoError(t, tc.m.ValidateAgainst(tc.against))
			})
		}
	})
}

// TestEnableManagedLocalAccountMarshalDefaults verifies every marshal/save path defaults the
// Windows managed local account toggle to false and preserves a set value.
func TestEnableManagedLocalAccountMarshalDefaults(t *testing.T) {
	windowsToggle := func(b []byte) any {
		var out map[string]any
		require.NoError(t, json.Unmarshal(b, &out))
		return out["mdm"].(map[string]any)["windows_settings"].(map[string]any)["enable_managed_local_account"]
	}
	marshaled := func(v any) any {
		b, err := json.Marshal(v)
		require.NoError(t, err)
		return windowsToggle(b)
	}

	var ac AppConfig
	require.Equal(t, false, marshaled(ac))
	ac.MDM.WindowsSettings.EnableManagedLocalAccount = optjson.SetBool(true)
	require.Equal(t, true, marshaled(ac))

	team := Team{ID: 1, Name: "t1"}
	require.Equal(t, false, marshaled(team))

	// the DB save path (TeamConfig.Value) applies the same default
	v, err := team.Config.Value()
	require.NoError(t, err)
	require.Equal(t, false, windowsToggle(v.([]byte)))
}

func TestMDMAppleABMAssignmentInfoRenameTeam(t *testing.T) {
	base := MDMAppleABMAssignmentInfo{
		OrganizationName: "Acme Inc",
		MacOSTeam:        "Workstations",
		IOSTeam:          "Phones",
		IpadOSTeam:       "Workstations",
		BYODTeam:         "",
	}

	t.Run("renames every platform pointing at the old name", func(t *testing.T) {
		info := base
		require.True(t, info.RenameTeam("Workstations", "Laptops"))
		require.Equal(t, "Laptops", info.MacOSTeam)
		require.Equal(t, "Laptops", info.IpadOSTeam)
		require.Equal(t, "Phones", info.IOSTeam, "unrelated fleets must not move")
		require.Empty(t, info.BYODTeam, `"no fleet" must stay unset`)
	})

	t.Run("no match reports false and changes nothing", func(t *testing.T) {
		info := base
		require.False(t, info.RenameTeam("Servers", "Laptops"))
		require.Equal(t, base, info)
	})

	t.Run("empty old name must not claim the unassigned platforms", func(t *testing.T) {
		info := base
		require.False(t, info.RenameTeam("", "Laptops"))
		require.Equal(t, base, info)
	})
}

func TestMDMAppleVolumePurchasingProgramInfoRenameTeam(t *testing.T) {
	base := MDMAppleVolumePurchasingProgramInfo{
		Location: "Acme HQ",
		Teams:    []string{"Workstations", "Servers", "Workstations"},
	}
	clone := func() MDMAppleVolumePurchasingProgramInfo {
		return MDMAppleVolumePurchasingProgramInfo{Location: base.Location, Teams: slices.Clone(base.Teams)}
	}

	t.Run("renames every occurrence and leaves the rest", func(t *testing.T) {
		info := clone()
		require.True(t, info.RenameTeam("Workstations", "Laptops"))
		require.Equal(t, []string{"Laptops", "Servers", "Laptops"}, info.Teams)
	})

	t.Run("no match reports false and changes nothing", func(t *testing.T) {
		info := clone()
		require.False(t, info.RenameTeam("Tablets", "Laptops"))
		require.Equal(t, base.Teams, info.Teams)
	})

	t.Run("empty old name must not claim entries", func(t *testing.T) {
		info := clone()
		info.Teams = append(info.Teams, "")
		require.False(t, info.RenameTeam("", "Laptops"))
		require.Equal(t, append(slices.Clone(base.Teams), ""), info.Teams)
	})

	t.Run("all-teams is a display name, not a fleet", func(t *testing.T) {
		info := MDMAppleVolumePurchasingProgramInfo{Location: "Acme HQ", Teams: []string{DisplayNameAllTeams}}
		require.False(t, info.RenameTeam("Workstations", "Laptops"))
		require.Equal(t, []string{DisplayNameAllTeams}, info.Teams)
	})
}

func TestBitLockerPINAliasSync(t *testing.T) {
	t.Run("marshal mirrors the deprecated key and its canonical home", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			mdm  MDM
			want bool
		}{
			{"deprecated only", MDM{RequireBitLockerPIN: optjson.SetBool(true)}, true},
			{"canonical only", MDM{WindowsSettings: WindowsSettings{RequireBitLockerPIN: optjson.SetBool(true)}}, true},
			{"canonical wins", MDM{
				RequireBitLockerPIN: optjson.SetBool(false),
				WindowsSettings:     WindowsSettings{RequireBitLockerPIN: optjson.SetBool(true)},
			}, true},
			{"neither defaults to false", MDM{}, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				b, err := json.Marshal(&AppConfig{MDM: tc.mdm})
				require.NoError(t, err)
				var doc map[string]any
				require.NoError(t, json.Unmarshal(b, &doc))
				mdm := doc["mdm"].(map[string]any)
				require.Equal(t, tc.want, mdm["windows_require_bitlocker_pin"])
				require.Equal(t, tc.want, mdm["windows_settings"].(map[string]any)["require_bitlocker_pin"])
			})
		}
	})

	t.Run("unmarshal fills the canonical home from a legacy document", func(t *testing.T) {
		var ac AppConfig
		require.NoError(t, json.Unmarshal([]byte(`{"mdm": {"windows_require_bitlocker_pin": true}}`), &ac))
		require.True(t, ac.MDM.WindowsSettings.RequireBitLockerPIN.Value)
		require.True(t, ac.MDM.BitLockerPINRequired())

		// an explicitly present canonical value is never overridden
		ac = AppConfig{}
		require.NoError(t, json.Unmarshal([]byte(`{"mdm": {
			"windows_require_bitlocker_pin": true,
			"windows_settings": {"require_bitlocker_pin": false}
		}}`), &ac))
		require.False(t, ac.MDM.BitLockerPINRequired())
	})
}

func TestBitLockerPINRequirementError(t *testing.T) {
	for _, tc := range []struct {
		name              string
		oldWindowsEnabled bool
		cfg               DiskEncryptionConfig
		wantField         string
		wantMsg           string
	}{
		{
			name: "PIN with encryption on is valid",
			cfg:  DiskEncryptionConfig{WindowsEnabled: true, BitLockerPINRequired: true},
		},
		{
			name: "no PIN with encryption off is valid",
			cfg:  DiskEncryptionConfig{},
		},
		{
			name:      "enabling the PIN without encryption blames the PIN field",
			cfg:       DiskEncryptionConfig{BitLockerPINRequired: true},
			wantField: "mdm.windows_settings.require_bitlocker_pin",
			wantMsg:   CantEnablePINRequiredIfDiskEncryptionEnabled,
		},
		{
			name:              "disabling encryption while the PIN is required blames the encryption field",
			oldWindowsEnabled: true,
			cfg:               DiskEncryptionConfig{BitLockerPINRequired: true},
			wantField:         "mdm.windows_settings.enable_disk_encryption",
			wantMsg:           CantDisableDiskEncryptionIfPINRequiredErrMsg,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			field, msg := BitLockerPINRequirementError(tc.oldWindowsEnabled, tc.cfg)
			require.Equal(t, tc.wantField, field)
			require.Equal(t, tc.wantMsg, msg)
		})
	}
}

func TestResolveBitLockerPINAlias(t *testing.T) {
	for _, tc := range []struct {
		name       string
		deprecated optjson.Bool
		canonical  optjson.Bool
		want       optjson.Bool
		wantErr    bool
	}{
		{name: "neither provided leaves the stored value alone"},
		{
			name:       "deprecated only",
			deprecated: optjson.SetBool(true),
			want:       optjson.SetBool(true),
		},
		{
			name:      "canonical only",
			canonical: optjson.SetBool(true),
			want:      optjson.SetBool(true),
		},
		{
			name:       "both agreeing is accepted, as a round-tripped document sends",
			deprecated: optjson.SetBool(true),
			canonical:  optjson.SetBool(true),
			want:       optjson.SetBool(true),
		},
		{
			name:       "explicit null on the canonical key falls back to the deprecated one",
			deprecated: optjson.SetBool(true),
			canonical:  optjson.Bool{Set: true},
			want:       optjson.SetBool(true),
		},
		{
			name:       "disagreeing values are rejected",
			deprecated: optjson.SetBool(false),
			canonical:  optjson.SetBool(true),
			wantErr:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveBitLockerPINAlias(tc.deprecated, tc.canonical)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want.Valid, got.Valid)
			if tc.want.Valid {
				require.Equal(t, tc.want.Value, got.Value)
			}
		})
	}
}

func TestFleetDesktopBrowserUrl(t *testing.T) {
	cases := []struct {
		name                   string
		serverURL              string
		alternativeBrowserHost string
		want                   string
	}{
		{
			name:      "no alternative browser host",
			serverURL: "https://fleet.example.com",
			want:      "https://fleet.example.com/device/abc123",
		},
		{
			name:                   "alternative browser host replaces only the host",
			serverURL:              "https://fleet.example.com",
			alternativeBrowserHost: "proxy.example.com",
			want:                   "https://proxy.example.com/device/abc123",
		},
		{
			name:                   "alternative browser host keeps its port",
			serverURL:              "https://fleet.example.com",
			alternativeBrowserHost: "proxy.example.com:8443",
			want:                   "https://proxy.example.com:8443/device/abc123",
		},
		{
			name:                   "alternative browser host given with a scheme is tolerated",
			serverURL:              "https://fleet.example.com",
			alternativeBrowserHost: "https://proxy.example.com",
			want:                   "https://proxy.example.com/device/abc123",
		},
		{
			name:      "url_prefix on the server URL is preserved",
			serverURL: "https://fleet.example.com/prefix",
			want:      "https://fleet.example.com/prefix/device/abc123",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var appConfig AppConfig
			appConfig.ServerSettings.ServerURL = c.serverURL
			appConfig.FleetDesktop.AlternativeBrowserHost = c.alternativeBrowserHost

			base, err := appConfig.FleetDesktopBrowserUrl()
			require.NoError(t, err)
			require.Equal(t, c.want, base.JoinPath("/device/abc123").String())
		})
	}
}
