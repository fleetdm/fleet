package service

import (
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

func TestExtractTeamSpecsMacOSCustomSettings(t *testing.T) {
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
---
apiVersion: v1
kind: team
spec:
  team:
    name: Fleet2
    mdm:
      macos_settings:
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
---
apiVersion: v1
kind: team
spec:
  team:
    name: "Fleet2"
    mdm:
      macos_settings:
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
`,
			map[string][]string{"Fleet": {"a", "b"}},
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
`,
			map[string][]string{},
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			specs, err := spec.GroupFromBytes([]byte(c.yaml))
			require.NoError(t, err)
			if len(specs.Teams) > 0 {
				got := extractTmSpecsMacOSCustomSettings(specs.Teams)
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
