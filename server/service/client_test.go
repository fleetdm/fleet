package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/stretchr/testify/require"
)

func TestExtractAppConfigMacOSCustomSettings(t *testing.T) {
	cases := []struct {
		desc string
		yaml string
		want []string
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
			[]string{},
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
        - "a"
        - "b"
`,
			[]string{"a", "b"},
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
        - "a"
        - ""
        - 4
        - "c"
`,
			[]string{"a", "c"},
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
		want []string
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
			[]string{},
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
        - "a"
        - "b"
`,
			[]string{"a", "b"},
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
        - "a"
        - ""
        - 4
        - "c"
`,
			[]string{"a", "c"},
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
		want map[string][]string
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
			map[string][]string{"Fleet": {}, "Fleet2": {}},
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
          - "a"
          - "b"
      windows_settings:
        custom_settings:
          - "c"
          - "d"
`,
			map[string][]string{"Fleet": {"a", "b", "c", "d"}},
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
			map[string][]string{},
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

func TestExtractFilenameFromPath(t *testing.T) {
	cases := []struct {
		in  string
		out string
	}{
		{"http://example.com", ""},
		{"http://example.com/", ""},
		{"http://example.com?foo=bar", ""},
		{"http://example.com/foo.pkg", "foo.pkg"},
		{"http://example.com/foo.exe", "foo.exe"},
		{"http://example.com/foo.pkg?bar=baz", "foo.pkg"},
		{"http://example.com/foo.bar.pkg", "foo.bar.pkg"},
		{"http://example.com/foo", "foo.pkg"},
		{"http://example.com/foo/bar/baz", "baz.pkg"},
		{"http://example.com/foo?bar=baz", "foo.pkg"},
	}

	for _, c := range cases {
		got := extractFilenameFromPath(c.in)
		require.Equalf(t, c.out, got, "for URL %s", c.in)
	}
}

func TestGetProfilesContents(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		baseDir      string
		setupFiles   [][2]string
		expectError  bool
		expectedKeys []string
	}{
		{
			name:    "invalid darwin xml",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.mobileconfig", `<?xml version="1.0" encoding="UTF-8"?>`},
			},
			expectError:  true,
			expectedKeys: []string{"foo"},
		},
		{
			name:    "windows and darwin files",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.xml", string(syncMLForTest("./some/path"))},
				{"bar.mobileconfig", string(mobileconfigForTest("bar", "I"))},
			},
			expectError:  false,
			expectedKeys: []string{"foo", "bar"},
		},
		{
			name:    "darwin files with file name != PayloadDisplayName",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"foo.xml", string(syncMLForTest("./some/path"))},
				{"bar.mobileconfig", string(mobileconfigForTest("fizz", "I"))},
			},
			expectError:  false,
			expectedKeys: []string{"foo", "fizz"},
		},
		{
			name:    "duplicate names across windows and darwin",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"baz.xml", string(syncMLForTest("./some/path"))},
				{"bar.mobileconfig", string(mobileconfigForTest("baz", "I"))},
			},
			expectError: true,
		},
		{
			name:    "duplicate file names",
			baseDir: tempDir,
			setupFiles: [][2]string{
				{"baz.xml", string(syncMLForTest("./some/path"))},
				{"baz.xml", string(syncMLForTest("./some/path"))},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := []string{}
			for _, fileSpec := range tt.setupFiles {
				filePath := filepath.Join(tempDir, fileSpec[0])
				require.NoError(t, os.WriteFile(filePath, []byte(fileSpec[1]), 0666))
				paths = append(paths, filePath)
			}

			profileContents, err := getProfilesContents(tt.baseDir, paths)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, profileContents)
				require.Len(t, profileContents, len(tt.expectedKeys))
				for _, key := range tt.expectedKeys {
					_, exists := profileContents[key]
					require.True(t, exists, fmt.Sprintf("Expected key %s not found", key))
				}
			}
		})
	}
}
