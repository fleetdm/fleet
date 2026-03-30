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
			name: "darwin from exists query",
			p: patch_policy.PolicyData{
				Platform:    "darwin",
				Version:     "1.0",
				ExistsQuery: "SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo';",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo' AND version_compare(bundle_short_version, '1.0') < 0);",
		},
		{
			name: "windows from exists query",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "1.0",
				ExistsQuery: "SELECT 1 FROM programs WHERE name = 'Foo x64' AND publisher = 'Bar, Inc.';",
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
