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
	got := patch_policy.GenerateOpenQuery("darwin", "org.mozilla.firefox", "", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON substr(p.path, 1, LENGTH(a.path) + 1) = concat(a.path, '/') WHERE a.bundle_identifier = 'org.mozilla.firefox');", got)

	// Apostrophes in the bundle identifier are escaped so they can't break the literal.
	got = patch_policy.GenerateOpenQuery("darwin", "com.oreilly.o'reilly", "", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON substr(p.path, 1, LENGTH(a.path) + 1) = concat(a.path, '/') WHERE a.bundle_identifier = 'com.oreilly.o''reilly');", got)

	// Windows matches a process named "<title>.exe".
	got = patch_policy.GenerateOpenQuery("windows", "", "Slack", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'slack.exe');", got)

	// An apostrophe in the derived executable is escaped.
	got = patch_policy.GenerateOpenQuery("windows", "", "O'Reilly", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'o''reilly.exe');", got)

	// A per-app override (keyed by software title) supplies the process-name predicate, in any of
	// its forms: LIKE, exact, or IN.
	got = patch_policy.GenerateOpenQuery("windows", "", "OneDrive", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) LIKE 'onedrive%');", got)

	got = patch_policy.GenerateOpenQuery("windows", "", "Google Chrome", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'chrome.exe');", got)

	got = patch_policy.GenerateOpenQuery("windows", "", "Microsoft Teams", nil)
	require.Equal(t, "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) IN ('teams.exe','ms-teams.exe'));", got)

	// A multi-word title without an override yields no query: the derived
	// "<title>.exe" ("xnsoft xnconvert.exe") would never match a real process,
	// silently defeating the app-open gate.
	require.Empty(t, patch_policy.GenerateOpenQuery("windows", "", "XnSoft XnConvert", nil))
	require.Empty(t, patch_policy.GenerateOpenQuery("windows", "", "Microsoft Visual C++ 2015-2022 Redistributable (x64)", nil))

	// Unknown platform yields no query.
	require.Empty(t, patch_policy.GenerateOpenQuery("linux", "com.example.foo", "", nil))
}

func TestGenerateOpenQueryWithProcessNames(t *testing.T) {
	t.Parallel()

	windows := func(title string, processNames []string) string {
		return patch_policy.GenerateOpenQuery("windows", "", title, processNames)
	}

	// A single verified process name.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'acrobat.exe');",
		windows("Adobe Acrobat Pro", []string{"Acrobat.exe"}))

	// Several collapse into an IN(...), preserving author order.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) IN ('7zfm.exe','7zg.exe'));",
		windows("7-zip", []string{"7zFM.exe", "7zG.exe"}))

	// A trailing "*" becomes a prefix match.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) LIKE 'onedrive%');",
		windows("OneDrive", []string{"OneDrive*"}))

	// Exact and prefix entries mix into a parenthesized OR.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE (LOWER(name) IN ('code.exe','code - insiders.exe') OR LOWER(name) LIKE 'codehelper%'));",
		windows("Microsoft Visual Studio Code", []string{"Code.exe", "Code - Insiders.exe", "CodeHelper*"}))

	// Apostrophes are escaped rather than breaking out of the literal.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'o''reilly.exe');",
		windows("O'Reilly", []string{"O'Reilly.exe"}))

	// Blank and duplicate entries are dropped.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'slack.exe');",
		windows("Slack", []string{"Slack.exe", "  ", "slack.exe"}))

	// process_names beats the curated override map...
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'new-chrome.exe');",
		windows("Google Chrome", []string{"new-chrome.exe"}))

	// ...but an all-blank list falls back to it rather than emitting an empty predicate.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'chrome.exe');",
		windows("Google Chrome", []string{"   "}))

	// process_names is what gives a multi-word app an open query at all: without it, a title
	// with a space and no override yields nothing rather than a guess that can never match.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM processes WHERE LOWER(name) = 'xnconvert.exe');",
		windows("XnSoft XnConvert", []string{"XnConvert.exe"}))
	require.Empty(t, windows("XnSoft XnConvert", nil))
	require.Empty(t, windows("XnSoft XnConvert", []string{"  "}))

	// Process names are windows-only; darwin ignores them.
	require.Equal(t,
		"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON substr(p.path, 1, LENGTH(a.path) + 1) = concat(a.path, '/') WHERE a.bundle_identifier = 'org.mozilla.firefox');",
		patch_policy.GenerateOpenQuery("darwin", "org.mozilla.firefox", "", []string{"firefox.exe"}))
}

func TestValidateProcessNames(t *testing.T) {
	t.Parallel()

	require.NoError(t, patch_policy.ValidateProcessNames(nil))
	require.NoError(t, patch_policy.ValidateProcessNames([]string{"7zFM.exe", "7zG.exe"}))
	require.NoError(t, patch_policy.ValidateProcessNames([]string{"1password*"}))

	for name, processNames := range map[string][]string{
		"empty entry":       {"chrome.exe", ""},
		"bare wildcard":     {"*"},
		"windows path":      {`C:\Program Files\7-Zip\7zFM.exe`},
		"unix path":         {"bin/chrome.exe"},
		"literal percent":   {"onedrive%"},
		"missing extension": {"chrome"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, patch_policy.ValidateProcessNames(processNames))
		})
	}
}
