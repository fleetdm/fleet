package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractAppConfigMacOSCustomSettings(t *testing.T) {
	cases := []struct {
		desc string
		yaml string
		want []fleet.MDMProfileSpec
	}{
		{
			"no settings",
			`
apiVersion: v1
kind: config
spec:
`,
			nil,
		},
		{
			"no custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
`,
			nil,
		},
		{
			"empty custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
      custom_settings:
`,
			[]fleet.MDMProfileSpec{},
		},
		{
			"custom settings specified",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
      custom_settings:
        - path: "a"
          labels:
            - "foo"
            - bar
        - path: "b"
`,
			[]fleet.MDMProfileSpec{{Path: "a", Labels: []string{"foo", "bar"}}, {Path: "b"}},
		},
		{
			"empty and invalid custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
      custom_settings:
        - path: "a"
          labels:
        - path: ""
          labels:
            - "foo"
        - path: 4
          labels:
            - "foo"
            - "bar"
        - path: "c"
          labels:
            - baz
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "c", Labels: []string{"baz"}}},
		},
		{
			"old custom settings specified",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
      custom_settings:
        - "a"
        - "b"
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "b"}},
		},
		{
			"old empty and invalid custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    macos_settings:
      custom_settings:
        - "a"
        - ""
        - 4
        - "c"
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "c"}},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			specs, err := spec.GroupFromBytes([]byte(c.yaml))
			require.NoError(t, err)
			if specs.AppConfig != nil {
				// Legacy fleetctl apply
				got := extractAppCfgMacOSCustomSettings(specs.AppConfig)
				assert.Equal(t, c.want, got)

				// GitOps
				mdm, ok := specs.AppConfig.(map[string]interface{})["mdm"].(map[string]interface{})
				require.True(t, ok)
				mdm["macos_settings"] = fleet.MacOSSettings{CustomSettings: c.want}
				got = extractAppCfgMacOSCustomSettings(specs.AppConfig)
				assert.Equal(t, c.want, got)
			}
		})
	}
}

func TestExtractMacOSAssetSpecs(t *testing.T) {
	// GitOps decodes macos_settings into a fleet.MacOSSettings struct. Whenever
	// the struct is present, assets must be reconciled (a nil/empty asset set
	// clears existing assets), so the extractor returns a non-nil slice.
	t.Run("gitops struct, no assets => non-nil empty (reconcile to empty)", func(t *testing.T) {
		got := extractMacOSAssetSpecs(fleet.MacOSSettings{CustomSettings: []fleet.MDMProfileSpec{{Path: "a"}}})
		require.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("gitops struct with assets", func(t *testing.T) {
		got := extractMacOSAssetSpecs(fleet.MacOSSettings{Assets: []fleet.MDMProfileSpec{{Path: "x"}}})
		assert.Equal(t, []fleet.MDMProfileSpec{{Path: "x"}}, got)
	})
	t.Run("gitops struct pointer with assets", func(t *testing.T) {
		got := extractMacOSAssetSpecs(&fleet.MacOSSettings{Assets: []fleet.MDMProfileSpec{{Path: "x"}}})
		assert.Equal(t, []fleet.MDMProfileSpec{{Path: "x"}}, got)
	})
	// Legacy fleetctl apply passes a raw map; there the assets key drives
	// behavior: absent means "leave untouched" (nil), present means reconcile.
	t.Run("legacy map, no assets key => nil (leave untouched)", func(t *testing.T) {
		got := extractMacOSAssetSpecs(map[string]any{"custom_settings": []any{}})
		assert.Nil(t, got)
	})
	t.Run("legacy map, empty assets => non-nil empty", func(t *testing.T) {
		got := extractMacOSAssetSpecs(map[string]any{"assets": []any{}})
		require.NotNil(t, got)
		assert.Empty(t, got)
	})
	t.Run("legacy map with assets", func(t *testing.T) {
		got := extractMacOSAssetSpecs(map[string]any{"assets": []any{map[string]any{"path": "x"}}})
		assert.Equal(t, []fleet.MDMProfileSpec{{Path: "x"}}, got)
	})
	t.Run("unrelated type => nil", func(t *testing.T) {
		assert.Nil(t, extractMacOSAssetSpecs("nope"))
	})
}

func TestExtractTmSpecsMDMAssets(t *testing.T) {
	cases := []struct {
		desc string
		spec string
		// want is the expected value keyed by team name; nil means the team must
		// be absent from the result map (assets left untouched).
		want map[string][]fleet.MDMProfileSpec
	}{
		{
			"macos_settings absent => untouched",
			`{"name":"T1","mdm":{}}`,
			nil,
		},
		{
			"macos_settings present, no assets => reconcile to empty (clear)",
			`{"name":"T1","mdm":{"macos_settings":{"custom_settings":[]}}}`,
			map[string][]fleet.MDMProfileSpec{"T1": {}},
		},
		{
			"macos_settings present, empty assets => reconcile to empty (clear)",
			`{"name":"T1","mdm":{"macos_settings":{"assets":[]}}}`,
			map[string][]fleet.MDMProfileSpec{"T1": {}},
		},
		{
			"macos_settings present with assets",
			`{"name":"T1","mdm":{"macos_settings":{"assets":[{"path":"x"}]}}}`,
			map[string][]fleet.MDMProfileSpec{"T1": {{Path: "x"}}},
		},
		{
			"empty team name => skipped",
			`{"name":"","mdm":{"macos_settings":{"assets":[]}}}`,
			nil,
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			got := extractTmSpecsMDMAssets([]json.RawMessage{json.RawMessage(c.spec)})
			assert.Equal(t, c.want, got)
		})
	}
}

func TestExtractAppConfigWindowsCustomSettings(t *testing.T) {
	cases := []struct {
		desc string
		yaml string
		want []fleet.MDMProfileSpec
	}{
		{
			"no settings",
			`
apiVersion: v1
kind: config
spec:
`,
			nil,
		},
		{
			"no custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
`,
			nil,
		},
		{
			"empty custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
      custom_settings:
`,
			[]fleet.MDMProfileSpec{},
		},
		{
			"custom settings specified",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
      custom_settings:
        - path: "a"
          labels:
            - "foo"
            - bar
        - path: "b"
`,
			[]fleet.MDMProfileSpec{{Path: "a", Labels: []string{"foo", "bar"}}, {Path: "b"}},
		},
		{
			"empty and invalid custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
      custom_settings:
        - path: "a"
          labels:
        - path: ""
          labels:
            - "foo"
        - path: 4
          labels:
            - "foo"
            - "bar"
        - path: "c"
          labels:
            - baz
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "c", Labels: []string{"baz"}}},
		},
		{
			"old custom settings specified",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
      custom_settings:
        - "a"
        - "b"
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "b"}},
		},
		{
			"old empty and invalid custom settings",
			`
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: "Fleet"
  mdm:
    windows_settings:
      custom_settings:
        - "a"
        - ""
        - 4
        - "c"
`,
			[]fleet.MDMProfileSpec{{Path: "a"}, {Path: "c"}},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			specs, err := spec.GroupFromBytes([]byte(c.yaml))
			require.NoError(t, err)
			if specs.AppConfig != nil {
				// Legacy fleetctl apply
				got := extractAppCfgWindowsCustomSettings(specs.AppConfig)
				assert.Equal(t, c.want, got)

				// GitOps
				mdm, ok := specs.AppConfig.(map[string]interface{})["mdm"].(map[string]interface{})
				require.True(t, ok)
				windowsSettings := fleet.WindowsSettings{}
				windowsSettings.CustomSettings = optjson.SetSlice(c.want)
				mdm["windows_settings"] = windowsSettings
				got = extractAppCfgWindowsCustomSettings(specs.AppConfig)
				assert.Equal(t, c.want, got)
			}
		})
	}
}

func TestExtractTeamSpecsMDMCustomSettings(t *testing.T) {
	cases := []struct {
		desc string
		yaml string
		want map[string]profileSpecsByPlatform
	}{
		{
			"no settings",
			`
apiVersion: v1
kind: team
spec:
  team:
`,
			nil,
		},
		{
			"no custom settings",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: Fleet
    mdm:
      macos_settings:
      windows_settings:
---
apiVersion: v1
kind: team
spec:
  team:
    name: Fleet2
    mdm:
      macos_settings:
      windows_settings:
`,
			nil,
		},
		{
			"empty custom settings",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet"
    mdm:
      macos_settings:
        custom_settings:
      windows_settings:
        custom_settings:
      android_settings:
        custom_settings:
---
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet2"
    mdm:
      macos_settings:
        custom_settings:
      windows_settings:
        custom_settings:
      android_settings:
        custom_settings:
`,
			map[string]profileSpecsByPlatform{"Fleet": {windows: []fleet.MDMProfileSpec{}, macos: []fleet.MDMProfileSpec{}, android: []fleet.MDMProfileSpec{}}, "Fleet2": {windows: []fleet.MDMProfileSpec{}, macos: []fleet.MDMProfileSpec{}, android: []fleet.MDMProfileSpec{}}},
		},
		{
			"custom settings specified",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet"
    mdm:
      android_settings:
        custom_settings:
          - path: "e"
            labels:
              - "foo"
          - path: "f"
      macos_settings:
        custom_settings:
          - path: "a"
            labels:
              - "foo"
              - bar
          - path: "b"
      windows_settings:
        custom_settings:
           - path: "c"
           - path: "d"
             labels:
               - "foo"
               - baz
`,
			map[string]profileSpecsByPlatform{"Fleet": {
				macos: []fleet.MDMProfileSpec{
					{Path: "a", Labels: []string{"foo", "bar"}},
					{Path: "b"},
				},
				windows: []fleet.MDMProfileSpec{
					{Path: "c"},
					{Path: "d", Labels: []string{"foo", "baz"}},
				},
				android: []fleet.MDMProfileSpec{
					{Path: "e", Labels: []string{"foo"}},
					{Path: "f"},
				},
			}},
		},
		{
			"old custom settings specified",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet"
    mdm:
      macos_settings:
        custom_settings:
          - "a"
          - "b"
      windows_settings:
        custom_settings:
          - "c"
          - "d"
`,
			map[string]profileSpecsByPlatform{"Fleet": {
				macos: []fleet.MDMProfileSpec{{Path: "a"}, {Path: "b"}},
				windows: []fleet.MDMProfileSpec{
					{Path: "c"},
					{Path: "d"},
				},
			}},
		},
		{
			"invalid custom settings",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet"
    mdm:
      android_settings:
        custom_settings:
          - path: "e"
            labels:
              - "y"
          - path: ""
      macos_settings:
        custom_settings:
          - path: "a"
            labels:
              - "y"
          - path: ""
          - path: 42
            labels:
              - "x"
          - path: "c"
      windows_settings:
        custom_settings:
          - path: "x"
          - path: ""
            labels:
              - "x"
          - path: 24
          - path: "y"
`,
			map[string]profileSpecsByPlatform{},
		},
		{
			"old invalid custom settings",
			`
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet"
    mdm:
      macos_settings:
        custom_settings:
          - "a"
          - ""
          - 42
          - "c"
      windows_settings:
        custom_settings:
          - "x"
          - ""
          - 24
          - "y"
`,
			map[string]profileSpecsByPlatform{},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			specs, err := spec.GroupFromBytes([]byte(c.yaml))
			require.NoError(t, err)
			if len(specs.Teams) > 0 {
				gotSpecs := extractTmSpecsMDMCustomSettings(specs.Teams)
				for k, wantProfs := range c.want {
					gotProfs, ok := gotSpecs[k]
					require.True(t, ok)
					require.Equal(t, wantProfs.macos, gotProfs.macos)
					require.Equal(t, wantProfs.windows, gotProfs.windows)
					require.Equal(t, wantProfs.android, gotProfs.android)
				}
			}
		})
	}
}

func TestGetProfilesContents(t *testing.T) {
	tempDir := t.TempDir()
	darwinProfile := mobileconfigForTest("bar", "I")
	darwinProfileWithFooEnv := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>bar</string>
	<key>PayloadIdentifier</key>
	<string>123</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>123</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>someConfig</key>
	<integer>$FOO</integer>
</dict>
</plist>`
	windowsProfile := syncMLForTest("./some/path")
	windowsProfileWithBarEnv := `<Add>
  <Item>
    <Target>
      <LocURI>./some/path</LocURI>
    </Target>
  </Item>
</Add>
<Replace>
  <Item>
    <Target>
      <LocURI>${BAR}/some/path</LocURI>
    </Target>
  </Item>
</Replace>`
	androidProfile := []byte(`{
		"name": "My Profile",
		"modifyAccountsDisabled": true,
		"maximumTimeToLock": "1234567",
		"something": {"else": true},
		"anotherThing": null,
		"numeric": 12345,
		"decimal": 1.23,
		"aList": ["1", "2"]
	}`)

	tests := []struct {
		name              string
		baseDir           string
		macSetupFiles     [][2]string
		winSetupFiles     [][2]string
		androidSetupFiles [][2]string
		labels            []string
		environment       map[string]string
		expandEnv         bool
		expectError       bool
		want              []fleet.MDMProfileBatchPayload
		wantErr           string
	}{
		{
			name:    "invalid darwin xml",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"foo.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>`},
			},
			expectError: true,
			want:        []fleet.MDMProfileBatchPayload{{Name: "foo"}},
		},
		{
			name:    "windows, darwin and android files",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", string(darwinProfile)},
			},
			winSetupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
			},
			androidSetupFiles: [][2]string{
				{"android.json", string(androidProfile)},
			},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile},
				{Name: "bar", Contents: darwinProfile},
				{Name: "android", Contents: androidProfile},
			},
		},
		{
			name:    "windows, darwin and android files with labels",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", string(darwinProfile)},
			},
			winSetupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
			},
			androidSetupFiles: [][2]string{
				{"android.json", string(androidProfile)},
			},
			labels:      []string{"foo", "bar"},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile, Labels: []string{"foo", "bar"}},
				{Name: "bar", Contents: darwinProfile, Labels: []string{"foo", "bar"}},
				{Name: "android", Contents: androidProfile, Labels: []string{"foo", "bar"}},
			},
		},
		{
			name:    "darwin files with file name != PayloadDisplayName",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", string(darwinProfile)},
			},
			winSetupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
			},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile},
				{Name: "bar", Contents: darwinProfile},
			},
		},
		{
			name:    "duplicate names across windows, darwin and android",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", string(mobileconfigForTest("baz", "I"))},
			},
			winSetupFiles: [][2]string{
				{"baz.xml", string(windowsProfile)},
			},
			androidSetupFiles: [][2]string{
				{"baz.json", string(androidProfile)},
			},
			expectError: true,
		},
		{
			name:    "duplicate windows file names",
			baseDir: tempDir,
			winSetupFiles: [][2]string{
				{"baz.xml", string(windowsProfile)},
				{"baz.xml", string(windowsProfile)},
			},
			expectError: true,
		},
		{
			name:    "duplicate android file names",
			baseDir: tempDir,
			androidSetupFiles: [][2]string{
				{"baz.json", string(androidProfile)},
				{"baz.json", string(androidProfile)},
			},
			expectError: true,
		},
		{
			name:    "with environment variables",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", darwinProfileWithFooEnv},
			},
			winSetupFiles: [][2]string{
				{"foo.xml", windowsProfileWithBarEnv},
			},
			environment: map[string]string{"FOO": "42", "BAR": "24"},
			expandEnv:   true,
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{
					Name: "bar",
					Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array/>
	<key>PayloadDisplayName</key>
	<string>bar</string>
	<key>PayloadIdentifier</key>
	<string>123</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>123</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>someConfig</key>
	<integer>42</integer>
</dict>
</plist>`),
				},
				{
					Name: "foo",
					Contents: []byte(`<Add>
  <Item>
    <Target>
      <LocURI>./some/path</LocURI>
    </Target>
  </Item>
</Add>
<Replace>
  <Item>
    <Target>
      <LocURI>24/some/path</LocURI>
    </Target>
  </Item>
</Replace>`),
				},
			},
		},
		{
			name:    "with environment variables but not set",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.mobileconfig", darwinProfileWithFooEnv},
			},
			winSetupFiles: [][2]string{
				{"foo.xml", windowsProfileWithBarEnv},
			},
			environment: map[string]string{},
			expandEnv:   true,
			expectError: true,
		},
		{
			name:    "with unprocessable json",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.json", string(windowsProfile)},
			},
			expectError: true,
			wantErr:     "Couldn't edit macos_settings.custom_settings (bar.json): Declaration profiles should include valid JSON",
		},
		{
			name:    "with unprocessable xml",
			baseDir: tempDir,
			winSetupFiles: [][2]string{
				{"bar.xml", string(darwinProfile)},
			},
			expectError: true,
			wantErr:     "Couldn't edit windows_settings.custom_settings (bar.xml): Windows configuration profiles can only have <Replace> or <Add> top level elements",
		},
		{
			name:    "with unsupported extension",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"bar.cfg", string(darwinProfile)},
			},
			expectError: true,
			wantErr:     "Couldn't edit macos_settings.custom_settings (bar.cfg): macOS configuration profiles must be .mobileconfig or .json files",
		},
		{
			name:    "with FLEET_SECRET in data tag",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"cert.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>com.apple.security.root</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>PayloadIdentifier</key>
			<string>com.example.cert</string>
			<key>PayloadUUID</key>
			<string>11111111-2222-3333-4444-555555555555</string>
			<key>PayloadDisplayName</key>
			<string>Test Certificate</string>
			<key>PayloadContent</key>
			<data>$FLEET_SECRET_CERT_DATA</data>
		</dict>
	</array>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadUUID</key>
	<string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadDisplayName</key>
	<string>Certificate Profile</string>
</dict>
</plist>`},
			},
			environment: map[string]string{
				"FLEET_SECRET_CERT_DATA": "VGVzdENlcnREYXRhQmFzZTY0", // "TestCertDataBase64" in base64
			},
			expandEnv:   true,
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{
					Name: "Certificate Profile",
					Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>com.apple.security.root</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>PayloadIdentifier</key>
			<string>com.example.cert</string>
			<key>PayloadUUID</key>
			<string>11111111-2222-3333-4444-555555555555</string>
			<key>PayloadDisplayName</key>
			<string>Test Certificate</string>
			<key>PayloadContent</key>
			<data>$FLEET_SECRET_CERT_DATA</data>
		</dict>
	</array>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadUUID</key>
	<string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadDisplayName</key>
	<string>Certificate Profile</string>
</dict>
</plist>`),
				},
			},
		},
		{
			name:    "with FLEET_SECRET in PayloadDisplayName - should reject",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"secret_name.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadUUID</key>
	<string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadDisplayName</key>
	<string>Profile $FLEET_SECRET_NAME</string>
</dict>
</plist>`},
			},
			environment: map[string]string{
				"FLEET_SECRET_NAME": "SecretProfileName",
			},
			expandEnv:   true,
			expectError: true,
			wantErr:     "PayloadDisplayName cannot contain FLEET_SECRET variables",
		},
		{
			name:    "with FLEET_VAR in profile - should not expand",
			baseDir: tempDir,
			macSetupFiles: [][2]string{
				{"fleet_var.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadUUID</key>
	<string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadDisplayName</key>
	<string>Profile with FLEET_VAR</string>
	<key>SomeValue</key>
	<string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>
</dict>
</plist>`},
			},
			expandEnv:   true,
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{
					Name: "Profile with FLEET_VAR",
					Contents: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadUUID</key>
	<string>aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee</string>
	<key>PayloadDisplayName</key>
	<string>Profile with FLEET_VAR</string>
	<key>SomeValue</key>
	<string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>
</dict>
</plist>`),
				},
			},
		},
		{
			name:    "android files with env var should expand",
			baseDir: tempDir,
			androidSetupFiles: [][2]string{
				{"env_secrets.json", `{"name": "env secrets", "testKey": "$FOO"}`},
			},
			environment: map[string]string{
				"FOO": "testValue",
			},
			expandEnv: true,
			want: []fleet.MDMProfileBatchPayload{
				{
					Name:     "env_secrets",
					Contents: []byte(`{"name": "env secrets", "testKey": "testValue"}`),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expandEnv {
				if len(tt.environment) > 0 {
					for k, v := range tt.environment {
						os.Setenv(k, v)
					}
					t.Cleanup(func() {
						for k := range tt.environment {
							os.Unsetenv(k)
						}
					})
				}
			}
			macPaths := []fleet.MDMProfileSpec{}
			for _, fileSpec := range tt.macSetupFiles {
				filePath := filepath.Join(tempDir, fileSpec[0])
				require.NoError(t, os.WriteFile(filePath, []byte(fileSpec[1]), 0o644))
				macPaths = append(macPaths, fleet.MDMProfileSpec{Path: filePath, Labels: tt.labels})
			}

			winPaths := []fleet.MDMProfileSpec{}
			for _, fileSpec := range tt.winSetupFiles {
				filePath := filepath.Join(tempDir, fileSpec[0])
				require.NoError(t, os.WriteFile(filePath, []byte(fileSpec[1]), 0o644))
				winPaths = append(winPaths, fleet.MDMProfileSpec{Path: filePath, Labels: tt.labels})
			}

			androidPaths := []fleet.MDMProfileSpec{}
			for _, fileSpec := range tt.androidSetupFiles {
				filePath := filepath.Join(tempDir, fileSpec[0])
				require.NoError(t, os.WriteFile(filePath, []byte(fileSpec[1]), 0o644))
				androidPaths = append(androidPaths, fleet.MDMProfileSpec{Path: filePath, Labels: tt.labels})
			}

			profileContents, err := getProfilesContents(tt.baseDir, macPaths, winPaths, androidPaths, tt.expandEnv)

			if tt.expectError {
				require.Error(t, err)
				if tt.wantErr != "" {
					require.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, profileContents)
				require.Len(t, profileContents, len(tt.want))
				require.ElementsMatch(t, tt.want, profileContents)
			}
		})
	}
}

func TestGetProfilesContentsActivation(t *testing.T) {
	tempDir := t.TempDir()

	activation := []byte(`{"Type":"com.apple.activation.simple","Identifier":"com.example.act","Payload":{"StandardConfigurations":["com.example.decl"]}}`)
	activationPath := filepath.Join(tempDir, "activation.json")
	require.NoError(t, os.WriteFile(activationPath, activation, 0o644))

	declPath := filepath.Join(tempDir, "decl.json")
	require.NoError(t, os.WriteFile(declPath,
		[]byte(`{"Type":"com.apple.configuration.passcode.settings","Identifier":"com.example.decl","Payload":{}}`), 0o644))

	mobileconfigPath := filepath.Join(tempDir, "profile.mobileconfig")
	require.NoError(t, os.WriteFile(mobileconfigPath, mobileconfigForTest("bar", "I"), 0o644))

	t.Run("declaration carries the activation contents", func(t *testing.T) {
		got, err := getProfilesContents(tempDir,
			[]fleet.MDMProfileSpec{{Path: declPath, Activation: activationPath}}, nil, nil, false)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, activation, got[0].Activation)
	})

	t.Run("no activation leaves the payload field empty", func(t *testing.T) {
		got, err := getProfilesContents(tempDir,
			[]fleet.MDMProfileSpec{{Path: declPath}}, nil, nil, false)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Empty(t, got[0].Activation)
	})

	t.Run("mobileconfig with an activation is rejected", func(t *testing.T) {
		_, err := getProfilesContents(tempDir,
			[]fleet.MDMProfileSpec{{Path: mobileconfigPath, Activation: activationPath}}, nil, nil, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "activation is only supported for declaration (DDM) profiles")
	})

	t.Run("unreadable activation file reports the path", func(t *testing.T) {
		_, err := getProfilesContents(tempDir,
			[]fleet.MDMProfileSpec{{Path: declPath, Activation: filepath.Join(tempDir, "missing.json")}}, nil, nil, false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "reading activation")
	})
}

func TestGitOpsErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	client, err := NewClient("https://foo.bar", true, "", "")
	require.NoError(t, err)

	tests := []struct {
		name    string
		rawJSON string
		wantErr string
	}{
		{
			name:    "invalid integrations value",
			rawJSON: `{ "integrations": false }`,
			wantErr: "org_settings.integrations",
		},
		{
			name:    "invalid integrations.ndes_scep_proxy key",
			rawJSON: `{ "integrations": { "ndes_scep_proxy": [] } }`,
			wantErr: "org_settings.integrations.ndes_scep_proxy is not supported",
		},
		{
			name:    "invalid certificate_authorities.ndes_scep_proxy value",
			rawJSON: `{ "integrations": null, "certificate_authorities": { "ndes_scep_proxy": [] } }`,
			wantErr: "org_settings.certificate_authorities.ndes_scep_proxy config is not a map",
		},
		// TODO(hca): add more tests for other certificate authority types
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &spec.GitOps{}
			config.OrgSettings = make(map[string]interface{})
			// Signal that we don't want to send any labels.
			// This avoids this test attempting to make a request to the GetLabels endpoint.
			config.Labels = make([]*fleet.LabelSpec, 0)
			err = json.Unmarshal([]byte(tt.rawJSON), &config.OrgSettings)
			require.NoError(t, err)
			config.OrgSettings["secrets"] = []*fleet.EnrollSecret{}
			settings := fleet.IconGitOpsSettings{ConcurrentUpdates: 1, ConcurrentUploads: 1}
			_, err = client.DoGitOps(ctx, config, "/filename", nil, false, nil, nil, nil, nil, nil, &settings)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestResolvePolicySoftwareTitleID(t *testing.T) {
	// uintPtr keeps the test cases readable — no ptr package import needed here.
	uintPtr := func(v uint) *uint { return &v }

	byURL := map[string]uint{
		"https://example.com/pkg.pkg":      100,
		"https://example.com/in-house.pkg": 400, // in-house scenario: title map has it, installer map doesn't
	}
	byAppStoreID := map[string]uint{
		"com.example.app": 200,
	}
	byHash := map[string]uint{
		"abc123hash":     100, // same title as the URL entry
		"different-hash": 999, // different title — used to test URL-over-hash precedence
		"in-house-hash":  401, // in-house scenario: title map has it, installer map doesn't
	}
	bySlug := map[string]uint{
		"some-fma-slug": 300,
	}
	installerIDsByURL := map[string]uint{
		"https://example.com/pkg.pkg": 500,
	}
	installerIDsByHash := map[string]uint{
		"abc123hash":     500,
		"different-hash": 501,
	}

	tests := []struct {
		name            string
		policy          *spec.GitOpsPolicySpec
		wantTitleID     uint
		wantInstallerID *uint
		wantResolved    bool
	}{
		{
			name: "URL lookup succeeds",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftwareURL: "https://example.com/pkg.pkg",
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "abc123hash",
					},
				},
			},
			wantTitleID:     100,
			wantInstallerID: uintPtr(500),
			wantResolved:    true,
		},
		{
			name: "URL takes precedence over hash when both match different titles",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftwareURL: "https://example.com/pkg.pkg",
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "different-hash",
					},
				},
			},
			wantTitleID:     100, // URL's title (100), not hash's title (999)
			wantInstallerID: uintPtr(500),
			wantResolved:    true,
		},
		{
			name: "URL lookup fails, hash fallback succeeds",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftwareURL: "https://example.com/DIFFERENT-url.pkg",
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "abc123hash",
					},
				},
			},
			wantTitleID:     100,
			wantInstallerID: uintPtr(500),
			wantResolved:    true,
		},
		{
			name: "URL matches title map but not installer map (in-house app path)",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftwareURL: "https://example.com/in-house.pkg",
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other:   &spec.PolicyInstallSoftware{},
				},
			},
			wantTitleID:     400,
			wantInstallerID: nil,
			wantResolved:    true,
		},
		{
			name: "hash matches title map but not installer map (in-house app path)",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "in-house-hash",
					},
				},
			},
			wantTitleID:     401,
			wantInstallerID: nil,
			wantResolved:    true,
		},
		{
			name: "App Store ID lookup succeeds",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						AppStoreID: "com.example.app",
					},
				},
			},
			wantTitleID:     200,
			wantInstallerID: nil,
			wantResolved:    true,
		},
		{
			name: "FMA slug lookup succeeds",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						FleetMaintainedAppSlug: "some-fma-slug",
					},
				},
			},
			wantTitleID:     300,
			wantInstallerID: nil,
			wantResolved:    true,
		},
		{
			name: "all lookups fail",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftwareURL: "https://example.com/nonexistent.pkg",
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "nonexistent-hash",
					},
				},
			},
			wantTitleID:     0,
			wantInstallerID: nil,
			wantResolved:    false,
		},
		{
			name: "hash-only policy (no URL)",
			policy: &spec.GitOpsPolicySpec{
				InstallSoftware: optjson.BoolOr[*spec.PolicyInstallSoftware]{
					IsOther: true,
					Other: &spec.PolicyInstallSoftware{
						HashSHA256: "abc123hash",
					},
				},
			},
			wantTitleID:     100,
			wantInstallerID: uintPtr(500),
			wantResolved:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			titleID, installerID, resolved := resolvePolicySoftwareTitleID(tt.policy, byURL, byAppStoreID, byHash, bySlug, installerIDsByURL, installerIDsByHash)
			require.Equal(t, tt.wantResolved, resolved)
			require.Equal(t, tt.wantTitleID, titleID)
			if tt.wantInstallerID == nil {
				require.Nil(t, installerID)
			} else {
				require.NotNil(t, installerID)
				require.Equal(t, *tt.wantInstallerID, *installerID)
			}
		})
	}
}

func TestEnsureHistoricalDataDefaults(t *testing.T) {
	cases := []struct {
		name      string
		features  map[string]any
		wantErr   string
		wantValue map[string]any
	}{
		{
			name:      "missing key gets defaults",
			features:  map[string]any{},
			wantValue: map[string]any{"uptime": true, "vulnerabilities": true},
		},
		{
			name:      "nil value gets defaults",
			features:  map[string]any{"historical_data": nil},
			wantValue: map[string]any{"uptime": true, "vulnerabilities": true},
		},
		{
			name:      "partial map fills missing sub-keys",
			features:  map[string]any{"historical_data": map[string]any{"uptime": false}},
			wantValue: map[string]any{"uptime": false, "vulnerabilities": true},
		},
		{
			name:      "explicit values are preserved",
			features:  map[string]any{"historical_data": map[string]any{"uptime": false, "vulnerabilities": false}},
			wantValue: map[string]any{"uptime": false, "vulnerabilities": false},
		},
		{
			name:     "scalar value is rejected",
			features: map[string]any{"historical_data": true},
			wantErr:  "features.historical_data must be a map, got bool",
		},
		{
			name:     "list value is rejected",
			features: map[string]any{"historical_data": []any{"uptime"}},
			wantErr:  "features.historical_data must be a map, got []interface {}",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			err := ensureHistoricalDataDefaults(tt.features)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantValue, tt.features["historical_data"])
		})
	}
}

func TestApplySoftwareInstallersProgress(t *testing.T) {
	pkg := func(name string, status fleet.SoftwarePackageDownloadStatus) fleet.SoftwarePackageDownloadProgress {
		return fleet.SoftwarePackageDownloadProgress{Name: name, Status: status}
	}
	poll := func(status string, progress ...fleet.SoftwarePackageDownloadProgress) batchSetSoftwareInstallersResultResponse {
		return batchSetSoftwareInstallersResultResponse{Status: status, DownloadProgress: progress}
	}
	// Fakes the batch endpoints, handing out one scripted response per poll.
	newClient := func(t *testing.T, polls []batchSetSoftwareInstallersResultResponse) *Client {
		var mu sync.Mutex
		var polled int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == "POST" {
				_ = json.NewEncoder(w).Encode(batchSetSoftwareInstallersResponse{RequestUUID: "test-uuid"})
				return
			}
			mu.Lock()
			defer mu.Unlock()
			_ = json.NewEncoder(w).Encode(polls[min(polled, len(polls)-1)])
			polled++
		}))
		t.Cleanup(srv.Close)
		client, err := NewClient(srv.URL, true, "", "")
		require.NoError(t, err)
		client.SetToken("test-token")
		return client
	}

	processing, completed := fleet.BatchSetSoftwareInstallersStatusProcessing, fleet.BatchSetSoftwareInstallersStatusCompleted
	downloading, downloaded := fleet.SoftwarePackageDownloadStarted, fleet.SoftwarePackageDownloadFinished

	testCases := []struct {
		name      string
		polls     []batchSetSoftwareInstallersResultResponse
		wantLines []string
	}{
		{
			// A package keeps its entry for the rest of the batch, so a line that already
			// printed must not print again on the next poll.
			name: "prints each line once however often it polls",
			polls: []batchSetSoftwareInstallersResultResponse{
				poll(processing, pkg("zoom.pkg", downloading)),
				poll(processing, pkg("zoom.pkg", downloaded), pkg("slack.pkg", downloading)),
				poll(completed, pkg("zoom.pkg", downloaded), pkg("slack.pkg", downloaded)),
			},
			wantLines: []string{
				"[+] downloading software package - zoom.pkg ...",
				"[+] downloaded software package - zoom.pkg",
				"[+] downloading software package - slack.pkg ...",
				"[+] downloaded software package - slack.pkg",
			},
		},
		{
			// Fleet already has the bytes, so there is no download to report.
			name:  "a package already in storage reports the skip and no download",
			polls: []batchSetSoftwareInstallersResultResponse{poll(completed, pkg("zoom.pkg", fleet.SoftwarePackageDownloadSkipped))},
			wantLines: []string{
				"[+] skipped downloading the software package (already in storage) - zoom.pkg",
			},
		},
		{
			// The same maintained app for two platforms carries one name.
			name: "two packages sharing a name each report",
			polls: []batchSetSoftwareInstallersResultResponse{
				poll(processing, pkg("OneDrive", downloaded), pkg("OneDrive", downloading)),
				poll(completed, pkg("OneDrive", downloaded), pkg("OneDrive", downloaded)),
			},
			wantLines: []string{
				"[+] downloading software package - OneDrive ...",
				"[+] downloaded software package - OneDrive",
				"[+] downloading software package - OneDrive ...",
				"[+] downloaded software package - OneDrive",
			},
		},
		{
			name:  "packages the batch never downloads stay silent",
			polls: []batchSetSoftwareInstallersResultResponse{poll(completed, fleet.SoftwarePackageDownloadProgress{}, pkg("zoom.pkg", downloaded), fleet.SoftwarePackageDownloadProgress{})},
			wantLines: []string{
				"[+] downloading software package - zoom.pkg ...",
				"[+] downloaded software package - zoom.pkg",
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var lines []string
			logFn := func(format string, args ...any) {
				lines = append(lines, strings.TrimSuffix(fmt.Sprintf(format, args...), "\n"))
			}

			client := newClient(t, tt.polls)
			_, _, _, err := client.applySoftwareInstallers(nil, url.Values{}, false, logFn)
			require.NoError(t, err)
			require.Equal(t, tt.wantLines, lines)
		})
	}
}
