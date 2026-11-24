package homebrew

import (
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/stretchr/testify/require"
)

func TestExcludeDiskImages(t *testing.T) {
	apps := " AND " + patch_policy.MacOSNotOnDiskImage("apps")
	a := " AND " + patch_policy.MacOSNotOnDiskImage("a")

	cases := []struct {
		name    string
		query   string
		want    string
		wantErr string
	}{
		{
			name:  "exists",
			query: "SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome';",
			want:  "SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome'" + apps + ";",
		},
		{
			name:  "patched",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome' AND version_compare(bundle_short_version, '158.0') < 0);",
			want:  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.google.Chrome' AND version_compare(bundle_short_version, '158.0') < 0" + apps + ");",
		},
		{
			name:  "open, qualified by its alias",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON (p.path = concat(a.path, '/Contents/MacOS/', a.bundle_executable) OR p.path LIKE '%/x') WHERE a.bundle_identifier = 'com.x' AND a.bundle_executable != '');",
			want:  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON (p.path = concat(a.path, '/Contents/MacOS/', a.bundle_executable) OR p.path LIKE '%/x') WHERE a.bundle_identifier = 'com.x' AND a.bundle_executable != ''" + a + ");",
		},
		{
			name:  "AS alias",
			query: "SELECT 1 FROM apps AS a WHERE a.bundle_identifier = 'com.x';",
			want:  "SELECT 1 FROM apps AS a WHERE a.bundle_identifier = 'com.x'" + a + ";",
		},
		{
			name:  "a top-level OR is parenthesized",
			query: "SELECT 1 FROM apps WHERE bundle_identifier = 'com.openai.chat' OR (bundle_identifier = 'com.openai.codex' AND bundle_name = 'ChatGPT');",
			want:  "SELECT 1 FROM apps WHERE (bundle_identifier = 'com.openai.chat' OR (bundle_identifier = 'com.openai.codex' AND bundle_name = 'ChatGPT'))" + apps + ";",
		},
		{
			name:  "an OR inside parentheses is left alone",
			query: "SELECT 1 FROM apps WHERE (bundle_identifier = 'x' OR bundle_identifier = 'y') AND path != '/z';",
			want:  "SELECT 1 FROM apps WHERE (bundle_identifier = 'x' OR bundle_identifier = 'y') AND path != '/z'" + apps + ";",
		},
		{
			name:  "every read of apps",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x') OR EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x' AND version_compare(bundle_short_version, '1') >= 0);",
			want:  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x'" + apps + ") OR EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x' AND version_compare(bundle_short_version, '1') >= 0" + apps + ");",
		},
		{
			name:  "other tables are left alone",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM package_receipts WHERE package_id = 'p') AND (EXISTS (SELECT 1 FROM package_receipts WHERE package_id = 'p') OR NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x'));",
			want:  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM package_receipts WHERE package_id = 'p') AND (EXISTS (SELECT 1 FROM package_receipts WHERE package_id = 'p') OR NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'x'" + apps + "));",
		},
		{
			name:  "a query without apps is unchanged",
			query: "SELECT 1 FROM signature WHERE path = '/Applications/WezTerm.app' AND cdhash IN ('a', 'b');",
			want:  "SELECT 1 FROM signature WHERE path = '/Applications/WezTerm.app' AND cdhash IN ('a', 'b');",
		},
		{
			name:  "string literals aren't parsed",
			query: "SELECT 1 FROM apps WHERE bundle_name = 'o''reilly FROM apps ) WHERE OR' AND path NOT LIKE '%\\_x' ESCAPE '\\';",
			want:  "SELECT 1 FROM apps WHERE bundle_name = 'o''reilly FROM apps ) WHERE OR' AND path NOT LIKE '%\\_x' ESCAPE '\\'" + apps + ";",
		},
		{
			name:  "no query",
			query: "",
			want:  "",
		},
		{
			name:    "no WHERE clause",
			query:   "SELECT 1 FROM apps;",
			wantErr: "without a WHERE clause",
		},
		{
			name:    "a clause after WHERE",
			query:   "SELECT 1 FROM apps WHERE bundle_identifier = 'x' LIMIT 1;",
			wantErr: "with LIMIT after its WHERE clause",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := excludeDiskImages(c.query)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, c.want, got)
		})
	}
}

var fromAppsPattern = regexp.MustCompile(`(?i)\bFROM\s+apps\b`)

// withoutDiskImageCondition checks that every read of the apps table in query leaves out disk images, and returns
// the query without that condition.
func withoutDiskImageCondition(t *testing.T, query string) string {
	t.Helper()
	reads := len(fromAppsPattern.FindAllStringIndex(query, -1))
	conditions := 0
	for _, qualifier := range []string{"apps", "a"} {
		condition := " AND " + patch_policy.MacOSNotOnDiskImage(qualifier)
		conditions += strings.Count(query, condition)
		query = strings.ReplaceAll(query, condition, "")
	}
	require.Equal(t, reads, conditions, "every read of apps should leave out disk images: %s", query)
	return query
}
