package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
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
				got := extractAppCfgMacOSCustomSettings(specs.AppConfig)
				require.Equal(t, c.want, got)
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
				got := extractAppCfgWindowsCustomSettings(specs.AppConfig)
				require.Equal(t, c.want, got)
			}
		})
	}
}

func TestExtractTeamSpecsMDMCustomSettings(t *testing.T) {
	cases := []struct {
		desc string
		yaml string
		want map[string][]fleet.MDMProfileSpec
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
`,
			map[string][]fleet.MDMProfileSpec{"Fleet": {}, "Fleet2": {}},
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
			map[string][]fleet.MDMProfileSpec{"Fleet": {
				{Path: "a", Labels: []string{"foo", "bar"}},
				{Path: "b"},
				{Path: "c"},
				{Path: "d", Labels: []string{"foo", "baz"}},
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
			map[string][]fleet.MDMProfileSpec{"Fleet": {{Path: "a"}, {Path: "b"}, {Path: "c"}, {Path: "d"}}},
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
			map[string][]fleet.MDMProfileSpec{},
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
			map[string][]fleet.MDMProfileSpec{},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			specs, err := spec.GroupFromBytes([]byte(c.yaml))
			require.NoError(t, err)
			if len(specs.Teams) > 0 {
				got := extractTmSpecsMDMCustomSettings(specs.Teams)
				require.Equal(t, c.want, got)
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

	tests := []struct {
		name        string
		baseDir     string
		setupFiles  [][2]string
		labels      []string
		environment map[string]string
		expandEnv   bool
		expectError bool
		want        []fleet.MDMProfileBatchPayload
	}{
		{
			name:    "invalid darwin xml",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>`},
			},
			expectError: true,
			want:        []fleet.MDMProfileBatchPayload{{Name: "foo"}},
		},
		{
			name:    "windows and darwin files",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
				{"bar.mobileconfig", string(darwinProfile)},
			},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile},
				{Name: "bar", Contents: darwinProfile},
			},
		},
		{
			name:    "windows and darwin files with labels",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
				{"bar.mobileconfig", string(darwinProfile)},
			},
			labels:      []string{"foo", "bar"},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile, Labels: []string{"foo", "bar"}},
				{Name: "bar", Contents: darwinProfile, Labels: []string{"foo", "bar"}},
			},
		},
		{
			name:    "darwin files with file name != PayloadDisplayName",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.xml", string(windowsProfile)},
				{"bar.mobileconfig", string(darwinProfile)},
			},
			expectError: false,
			want: []fleet.MDMProfileBatchPayload{
				{Name: "foo", Contents: windowsProfile},
				{Name: "bar", Contents: darwinProfile},
			},
		},
		{
			name:    "duplicate names across windows and darwin",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"baz.xml", string(windowsProfile)},
				{"bar.mobileconfig", string(mobileconfigForTest("baz", "I"))},
			},
			expectError: true,
		},
		{
			name:    "duplicate file names",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"baz.xml", string(windowsProfile)},
				{"baz.xml", string(windowsProfile)},
			},
			expectError: true,
		},
		{
			name:    "with environment variables",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"bar.mobileconfig", darwinProfileWithFooEnv},
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
			setupFiles: [][2]string{
				{"bar.mobileconfig", darwinProfileWithFooEnv},
				{"foo.xml", windowsProfileWithBarEnv},
			},
			environment: map[string]string{},
			expandEnv:   true,
			expectError: true,
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
			paths := []fleet.MDMProfileSpec{}
			for _, fileSpec := range tt.setupFiles {
				filePath := filepath.Join(tempDir, fileSpec[0])
				require.NoError(t, os.WriteFile(filePath, []byte(fileSpec[1]), 0o644))
				paths = append(paths, fleet.MDMProfileSpec{Path: filePath, Labels: tt.labels})
			}

			profileContents, err := getProfilesContents(tt.baseDir, paths, tt.expandEnv)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profileContents)
				require.Len(t, profileContents, len(tt.want))
				require.ElementsMatch(t, tt.want, profileContents)
			}
		})
	}
}
