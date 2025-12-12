package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
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
			_, _, err = client.DoGitOps(ctx, config, "/filename", nil, false, nil, nil, nil, nil, nil, &settings)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}
