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
		{
			name: "windows from exists query with LIKE percent wildcard",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "12.5.6",
				ExistsQuery: "SELECT 1 FROM programs WHERE name LIKE 'Postman x64 %' AND publisher = 'Postman';",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Postman x64 %' AND publisher = 'Postman' AND version_compare(version, '12.5.6') < 0);",
		},
		{
			name: "windows from exists query with multiple LIKE percent wildcards",
			p: patch_policy.PolicyData{
				Platform:    "windows",
				Version:     "139.0.0",
				ExistsQuery: "SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox % ESR %' AND publisher = 'Mozilla';",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox % ESR %' AND publisher = 'Mozilla' AND version_compare(version, '139.0.0') < 0);",
		},
		{
			name: "codex-cli portable install OR precedence and file_version",
			p: patch_policy.PolicyData{
				Platform: "windows",
				Version:  "0.130.0",
				ExistsQuery: "SELECT 1 FROM file WHERE path = 'C:\\Program Files\\Codex CLI\\codex.exe' " +
					"OR path LIKE '%\\AppData\\Local\\Programs\\Codex CLI\\codex.exe';",
			},
			want: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM file WHERE (path = 'C:\\Program Files\\Codex CLI\\codex.exe' " +
				"OR path LIKE '%\\AppData\\Local\\Programs\\Codex CLI\\codex.exe') AND version_compare(file_version, '0.130.0') < 0);",
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

func TestGenerateOpenQuery(t *testing.T) {
	// macOS resolves the app's install path from its bundle identifier and matches a process
	// running from inside it.
	got := patch_policy.GenerateOpenQuery("darwin", "org.mozilla.firefox", "")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path LIKE concat(a.path, '/%') WHERE a.bundle_identifier = 'org.mozilla.firefox');", got)

	// Apostrophes in the bundle identifier are escaped so they can't break the literal.
	got = patch_policy.GenerateOpenQuery("darwin", "com.oreilly.o'reilly", "")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path LIKE concat(a.path, '/%') WHERE a.bundle_identifier = 'com.oreilly.o''reilly');", got)

	// Windows matches a process named "<title>.exe".
	got = patch_policy.GenerateOpenQuery("windows", "", "Slack")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'slack.exe');", got)

	// An apostrophe in the derived executable is escaped.
	got = patch_policy.GenerateOpenQuery("windows", "", "O'Reilly")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'o''reilly.exe');", got)

	// A per-app override (keyed by software title) supplies the process-name predicate, in any of
	// its forms: LIKE, exact, or IN.
	got = patch_policy.GenerateOpenQuery("windows", "", "OneDrive")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) LIKE 'onedrive%');", got)

	got = patch_policy.GenerateOpenQuery("windows", "", "Google Chrome")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'chrome.exe');", got)

	got = patch_policy.GenerateOpenQuery("windows", "", "Microsoft Teams")
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) IN ('teams.exe','ms-teams.exe'));", got)

	// Unknown platform yields no query.
	require.Empty(t, patch_policy.GenerateOpenQuery("linux", "com.example.foo", ""))
}
