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
			want: "SELECT 1 WHERE NOT EXISTS ((SELECT 1 FROM apps WHERE bundle_identifier = 'com.foo') AND version_compare(bundle_short_version, '1.0') < 0);",
		},
		{
			name: "windows from exists query",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "1.0",
				ExistsQuery: "SELECT 1 FROM programs WHERE name = 'Foo x64' AND publisher = 'Bar, Inc.';",
			},
			want: "SELECT 1 WHERE NOT EXISTS ((SELECT 1 FROM programs WHERE name = 'Foo x64' AND publisher = 'Bar, Inc.') AND version_compare(version, '1.0') < 0);",
		},
		{
			name: "windows from exists query with LIKE percent wildcard",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "12.5.6",
				ExistsQuery: "SELECT 1 FROM programs WHERE name LIKE 'Postman x64 %' AND publisher = 'Postman';",
			},
			want: "SELECT 1 WHERE NOT EXISTS ((SELECT 1 FROM programs WHERE name LIKE 'Postman x64 %' AND publisher = 'Postman') AND version_compare(version, '12.5.6') < 0);",
		},
		{
			name: "windows from exists query with multiple LIKE percent wildcards",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "139.0.0",
				ExistsQuery: "SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox % ESR %' AND publisher = 'Mozilla';",
			},
			want: "SELECT 1 WHERE NOT EXISTS ((SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox % ESR %' AND publisher = 'Mozilla') AND version_compare(version, '139.0.0') < 0);",
		},
		{
			name: "windows from exists query containing OR (precedence fix)",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "0.130.0",
				ExistsQuery: "SELECT 1 FROM file WHERE path = 'C:\\a' OR path LIKE '%\\b';",
			},
			want: "SELECT 1 WHERE NOT EXISTS ((SELECT 1 FROM file WHERE path = 'C:\\a' OR path LIKE '%\\b') AND version_compare(version, '0.130.0') < 0);",
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
