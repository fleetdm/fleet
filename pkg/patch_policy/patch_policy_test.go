package patch_policy_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/stretchr/testify/require"
)

func TestGenerateQueryForManifest(t *testing.T) {
	tests := []struct {
		name string
		want string
		p    patch_policy.PolicyData
	}{
		{
			name: "fills in query",
			p: patch_policy.PolicyData{
				Query:   "SELECT $FMA_VERSION;",
				Version: "1.0",
			},
			want: "SELECT 1.0;",
		},
		{
			name: "darwin",
			p: patch_policy.PolicyData{
				Platform:         "darwin",
				Version:          "1.0",
				BundleIdentifier: "com.foo",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo' AND version_compare(bundle_short_version, '1.0') < 0);",
		},
		{
			name: "windows with publisher",
			p: patch_policy.PolicyData{
				Platform:      "windows",
				Version:       "1.0",
				SoftwareTitle: "Foo x64",
				Publisher:     "Bar, Inc.",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo x64' AND publisher = 'Bar, Inc.' AND version_compare(version, '1.0') < 0);",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := patch_policy.GenerateQueryForManifest(tt.p)
			require.NoError(t, err)
			require.Equal(t, tt.want, query)
		})
	}
}
