package apple_mdm

import (
	"strings"
	"testing"

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
				"<integer>"+itoa(c.wantMgmt)+"</integer>",
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n == 1 {
		return "1"
	}
	// Builder only emits 0 or 1 today.
	t := []byte{}
	for n > 0 {
		t = append([]byte{byte('0' + n%10)}, t...)
		n /= 10
	}
	return string(t)
}
