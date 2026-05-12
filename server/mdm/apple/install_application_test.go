package apple_mdm

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func TestBuildInstallApplicationCommand_VPP(t *testing.T) {
	const commandUUID = "abc-123"
	cases := []struct {
		name     string
		params   InstallApplicationParams
		wantMgmt int
		wantHas  []string
		wantNot  []string
	}{
		{
			name: "iOS VPP, no configuration → ManagementFlags=1, iTunesStoreID, no Configuration key",
			params: InstallApplicationParams{
				CommandUUID:   commandUUID,
				HostPlatform:  "ios",
				ITunesStoreID: "12345",
			},
			wantMgmt: 1,
			wantHas:  []string{"<key>iTunesStoreID</key>", "<integer>12345</integer>", commandUUID},
			wantNot:  []string{"<key>Configuration</key>", "<key>ManifestURL</key>"},
		},
		{
			name: "iPadOS VPP with configuration → Configuration dict injected",
			params: InstallApplicationParams{
				CommandUUID:   commandUUID,
				HostPlatform:  "ipados",
				ITunesStoreID: "67890",
				Configuration: []byte("<dict><key>K</key><string>v</string></dict>"),
			},
			wantMgmt: 1,
			wantHas: []string{
				"<key>Configuration</key>",
				"<dict><key>K</key><string>v</string></dict>",
				"<key>iTunesStoreID</key>",
			},
		},
		{
			name: "macOS VPP with configuration → Configuration silently dropped",
			params: InstallApplicationParams{
				CommandUUID:   commandUUID,
				HostPlatform:  "darwin",
				ITunesStoreID: "11111",
				Configuration: []byte("<dict><key>K</key><string>should-not-leak</string></dict>"),
			},
			wantMgmt: 0,
			wantHas:  []string{"<key>iTunesStoreID</key>"},
			wantNot:  []string{"<key>Configuration</key>", "should-not-leak"},
		},
		{
			name: "iOS in-house with configuration → ManifestURL + Configuration injected",
			params: InstallApplicationParams{
				CommandUUID:   commandUUID,
				HostPlatform:  "ios",
				ManifestURL:   "https://fleet.example.com/manifest",
				Configuration: []byte("<dict><key>K</key><string>v</string></dict>"),
			},
			wantMgmt: 1,
			wantHas: []string{
				"<key>ManifestURL</key>",
				"<string>https://fleet.example.com/manifest</string>",
				"<key>Configuration</key>",
				"<dict><key>K</key><string>v</string></dict>",
			},
			wantNot: []string{"<key>iTunesStoreID</key>"},
		},
		{
			name: "iOS VPP with empty configuration bytes → key omitted (clear semantics)",
			params: InstallApplicationParams{
				CommandUUID:   commandUUID,
				HostPlatform:  "ios",
				ITunesStoreID: "55555",
				Configuration: []byte{},
			},
			wantMgmt: 1,
			wantNot:  []string{"<key>Configuration</key>"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := string(BuildInstallApplicationCommand(c.params))

			// Roundtrip through the plist parser to confirm we always emit a
			// well-formed XML plist regardless of inputs. This is the same
			// shape Apple MDM accepts; if this fails, devices will reject.
			var anything any
			format, err := plist.Unmarshal([]byte(out), &anything)
			require.NoError(t, err, "output must be a valid plist")
			require.Equal(t, plist.XMLFormat, format, "output must be XML plist (not binary/OpenStep)")

			for _, s := range c.wantHas {
				require.Contains(t, out, s, "expected to contain %q", s)
			}
			for _, s := range c.wantNot {
				require.NotContains(t, out, s, "did not expect %q in output", s)
			}
			require.Contains(t, out,
				"<integer>"+strconv.Itoa(c.wantMgmt)+"</integer>",
				"ManagementFlags should be %d", c.wantMgmt)
			require.Contains(t, out, "<string>"+commandUUID+"</string>")
		})
	}
}

func TestBuildInstallApplicationCommand_FullPlistDocumentNormalized(t *testing.T) {
	fullDoc := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ServerURL</key>
	<string>https://example.com</string>
</dict>
</plist>`)

	out := string(BuildInstallApplicationCommand(InstallApplicationParams{
		CommandUUID:   "uuid",
		HostPlatform:  "ios",
		ITunesStoreID: "1",
		Configuration: fullDoc,
	}))

	var parsed map[string]any
	_, err := plist.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err, "output with full-doc config must be a valid plist")
	require.Equal(t, 1, strings.Count(out, "<?xml"), "only one XML declaration allowed")
	require.Equal(t, 1, strings.Count(out, "<plist"), "only one <plist> element allowed")

	cmd := parsed["Command"].(map[string]any)
	cfgDict, ok := cmd["Configuration"].(map[string]any)
	require.True(t, ok, "Configuration value should be a dict")
	require.Equal(t, "https://example.com", cfgDict["ServerURL"])
}

func TestBuildInstallApplicationCommand_ConfigurationOuterDictPreserved(t *testing.T) {
	// The validator stores the bytes including the outer <dict>...</dict>.
	// Builder must inline them as-is so the resulting plist nests correctly:
	//   <key>Configuration</key><dict>...</dict>
	cfg := []byte(`<dict>
	<key>ServerURL</key>
	<string>https://example.com</string>
</dict>`)

	out := string(BuildInstallApplicationCommand(InstallApplicationParams{
		CommandUUID:   "uuid",
		HostPlatform:  "ios",
		ITunesStoreID: "1",
		Configuration: cfg,
	}))

	// The Configuration value should be the stored dict, not double-wrapped.
	configIdx := strings.Index(out, "<key>Configuration</key>")
	require.NotEqual(t, -1, configIdx, "Configuration key present")
	tail := out[configIdx:]
	require.Contains(t, tail, string(cfg), "stored bytes inlined verbatim")

	// Re-parse and verify Configuration is a nested dict, not a string.
	var parsed map[string]any
	_, err := plist.Unmarshal([]byte(out), &parsed)
	require.NoError(t, err)
	cmd, ok := parsed["Command"].(map[string]any)
	require.True(t, ok, "Command should be a dict")
	cfgVal, ok := cmd["Configuration"]
	require.True(t, ok, "Configuration key should be present")
	cfgDict, ok := cfgVal.(map[string]any)
	require.True(t, ok, "Configuration value should be a dict, got %T", cfgVal)
	require.Equal(t, "https://example.com", cfgDict["ServerURL"])
}

func TestSubstituteFleetVarsInAppConfig(t *testing.T) {
	ctx := context.Background()
	host := AppConfigSubstitutionHost{
		UUID:           "host-uuid-1",
		HardwareSerial: "ABC123",
		Platform:       "ios",
	}

	// emptyDS is a non-nil placeholder for paths that don't actually consult
	// the datastore (host-only variables). The nilaway linter treats nil
	// arguments as nilable through the call chain even when the called branch
	// can never dereference them, so always pass a real mock.
	emptyDS := new(mock.Store)

	t.Run("no config returns nil error", func(t *testing.T) {
		got, err := SubstituteFleetVarsInAppConfig(ctx, emptyDS, nil, host)
		require.NoError(t, err)
		require.Nil(t, got)
	})

	t.Run("config without variables returns unchanged", func(t *testing.T) {
		cfg := []byte(`<dict><key>K</key><string>plain</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Equal(t, cfg, got)
	})

	t.Run("HOST_UUID substituted", func(t *testing.T) {
		cfg := []byte(`<dict><key>K</key><string>$FLEET_VAR_HOST_UUID</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "host-uuid-1")
		require.NotContains(t, string(got), "$FLEET_VAR_HOST_UUID")
	})

	t.Run("HOST_HARDWARE_SERIAL substituted", func(t *testing.T) {
		cfg := []byte(`<dict><key>K</key><string>${FLEET_VAR_HOST_HARDWARE_SERIAL}</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, emptyDS, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "ABC123")
	})

	t.Run("HOST_PLATFORM darwin maps to macos", func(t *testing.T) {
		darwinHost := host
		darwinHost.Platform = "darwin"
		cfg := []byte(`<dict><key>K</key><string>$FLEET_VAR_HOST_PLATFORM</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, emptyDS, cfg, darwinHost)
		require.NoError(t, err)
		require.Contains(t, string(got), "macos")
	})

	t.Run("HOST_END_USER_EMAIL_IDP resolves via datastore", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			require.Equal(t, "host-uuid-1", hostUUID)
			return []string{"user@example.com"}, nil
		}
		cfg := []byte(`<dict><key>K</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		require.Contains(t, string(got), "user@example.com")
	})

	t.Run("HOST_END_USER_EMAIL_IDP missing returns ErrUnresolvable", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return nil, nil
		}
		cfg := []byte(`<dict><key>K</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, ds, cfg, host)
		require.ErrorIs(t, err, ErrUnresolvableAppConfigVar)
		require.Nil(t, got)
	})

	t.Run("XML special chars in substituted value are escaped", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return []string{"a&b@example.com"}, nil
		}
		cfg := []byte(`<dict><key>K</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		// Round-trip the resulting XML through the plist parser; if escaping
		// is broken the parser will fail or decode incorrectly.
		var parsed map[string]any
		_, perr := plist.Unmarshal(got, &parsed)
		require.NoError(t, perr)
		require.Equal(t, "a&b@example.com", parsed["K"])
	})

	t.Run("multiple variables substituted independently", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetHostEmailsFunc = func(ctx context.Context, hostUUID string, source string) ([]string, error) {
			return []string{"user@example.com"}, nil
		}
		cfg := []byte(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string><key>S</key><string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string><key>E</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		got, err := SubstituteFleetVarsInAppConfig(ctx, ds, cfg, host)
		require.NoError(t, err)
		s := string(got)
		require.Contains(t, s, "host-uuid-1")
		require.Contains(t, s, "ABC123")
		require.Contains(t, s, "user@example.com")
	})
}

// Ensure the result of substitution slots into BuildInstallApplicationCommand
// without breaking the plist envelope.
func TestSubstituteThenBuildRoundTrip(t *testing.T) {
	ctx := context.Background()
	cfg := []byte(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`)
	host := AppConfigSubstitutionHost{UUID: "uuid-x", Platform: "ios"}

	substituted, err := SubstituteFleetVarsInAppConfig(ctx, new(mock.Store), cfg, host)
	require.NoError(t, err)

	cmd := BuildInstallApplicationCommand(InstallApplicationParams{
		CommandUUID:   "cmd",
		HostPlatform:  "ios",
		ITunesStoreID: "1",
		Configuration: substituted,
	})

	var parsed map[string]any
	format, err := plist.Unmarshal(cmd, &parsed)
	require.NoError(t, err)
	require.Equal(t, plist.XMLFormat, format)
	command, _ := parsed["Command"].(map[string]any)
	configDict, ok := command["Configuration"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "uuid-x", configDict["UUID"])
}
