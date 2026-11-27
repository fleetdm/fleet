package patch_policy_test

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/patch_policy"
	"github.com/stretchr/testify/require"
)

func TestParseWindowsProgramsRule(t *testing.T) {
	type cond = patch_policy.ProgramsCondition
	name := func(op patch_policy.ProgramsOp, values ...string) cond {
		return cond{Column: patch_policy.ProgramsName, Op: op, Values: values}
	}
	publisher := func(op patch_policy.ProgramsOp, values ...string) cond {
		return cond{Column: patch_policy.ProgramsPublisher, Op: op, Values: values}
	}
	version := func(op patch_policy.ProgramsOp, values ...string) cond {
		return cond{Column: patch_policy.ProgramsVersion, Op: op, Values: values}
	}
	one := func(c cond) []cond { return []cond{c} }

	cases := []struct {
		name  string
		query string
		want  [][]cond
	}{
		{
			name:  "exact name",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Mozilla Firefox (x64 en-US)' AND publisher = 'Mozilla' AND version_compare(version, '160.0.1') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpEqual, "Mozilla Firefox (x64 en-US)")),
				one(publisher(patch_policy.OpEqual, "Mozilla")),
			},
		},
		{
			name:  "trimmed name",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE TRIM(name) = 'Foo' AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
			want: [][]cond{
				one(cond{Column: patch_policy.ProgramsName, Op: patch_policy.OpEqual, Values: []string{"Foo"}, Trim: true}),
				one(publisher(patch_policy.OpEqual, "Bar")),
			},
		},
		{
			name:  "name IN",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name IN ('eM Client (Updater)', 'eM Client') AND publisher = 'eM Client s.r.o.' AND version_compare(version, '11.0.1054') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpEqual, "eM Client (Updater)", "eM Client")),
				one(publisher(patch_policy.OpEqual, "eM Client s.r.o.")),
			},
		},
		{
			name:  "two name clauses",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Microsoft .NET Runtime - 8.%' AND name LIKE '%(x64)' AND publisher = 'Microsoft Corporation' AND version_compare(version, '8.0.34') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpLike, "Microsoft .NET Runtime - 8.%")),
				one(name(patch_policy.OpLike, "%(x64)")),
				one(publisher(patch_policy.OpEqual, "Microsoft Corporation")),
			},
		},
		{
			name:  "name NOT LIKE",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Chatbox %' AND name NOT LIKE 'Chatbox CE %' AND publisher = 'Mediocre Company' AND version_compare(version, '1.24.7') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpLike, "Chatbox %")),
				one(name(patch_policy.OpNotLike, "Chatbox CE %")),
				one(publisher(patch_policy.OpEqual, "Mediocre Company")),
			},
		},
		{
			name:  "publisher LIKE and IN",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Autopsy' AND publisher IN ('Sleuth Kit Labs', 'The Sleuth Kit') AND publisher LIKE '%Sleuth%' AND version_compare(version, '4.23.1') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpEqual, "Autopsy")),
				one(publisher(patch_policy.OpEqual, "Sleuth Kit Labs", "The Sleuth Kit")),
				one(publisher(patch_policy.OpLike, "%Sleuth%")),
			},
		},
		{
			name:  "version LIKE and NOT LIKE",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'QGIS %' AND publisher = 'QGIS.org' AND version LIKE '3.%' AND version NOT LIKE '3.44.%' AND version_compare(version, '4.2.3') < 0);",
			want: [][]cond{
				one(name(patch_policy.OpLike, "QGIS %")),
				one(publisher(patch_policy.OpEqual, "QGIS.org")),
				one(version(patch_policy.OpLike, "3.%")),
				one(version(patch_policy.OpNotLike, "3.44.%")),
			},
		},
		{
			name:  "OR group of names inside the generator's parens",
			query: "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE ((name LIKE 'Burp Suite 2%' OR name LIKE 'Burp Suite Professional %' OR name LIKE 'Burp Suite Community Edition %') AND publisher = 'PortSwigger Web Security') AND version_compare(version, '2026.11.2') < 0);",
			want: [][]cond{
				{
					name(patch_policy.OpLike, "Burp Suite 2%"),
					name(patch_policy.OpLike, "Burp Suite Professional %"),
					name(patch_policy.OpLike, "Burp Suite Community Edition %"),
				},
				one(publisher(patch_policy.OpEqual, "PortSwigger Web Security")),
			},
		},
		{
			name:  "ARM-scoped",
			query: "SELECT 1 WHERE EXISTS (SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox%ESR (AArch64 %' AND publisher = 'Mozilla' AND version_compare(version, '153.8.0') < 0)) OR NOT EXISTS (SELECT 1 FROM system_info WHERE cpu_type LIKE 'ARM%');",
			want: [][]cond{
				one(name(patch_policy.OpLike, "Mozilla Firefox%ESR (AArch64 %")),
				one(publisher(patch_policy.OpEqual, "Mozilla")),
			},
		},
		{
			name:  "quotes unescaped, keywords case-insensitive, no trailing semicolon",
			query: "select 1 where not exists (select 1 from programs where NAME like 'Bob''s App %' and Publisher = 'Bob''s' and version_compare(version, '1.0') < 0)",
			want: [][]cond{
				one(name(patch_policy.OpLike, "Bob's App %")),
				one(publisher(patch_policy.OpEqual, "Bob's")),
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rule, ok := patch_policy.ParseWindowsProgramsRule(c.query)
			require.True(t, ok)
			require.Equal(t, c.want, rule.Clauses)
		})
	}
}

func TestParseWindowsProgramsRuleUnsupported(t *testing.T) {
	cases := map[string]string{
		"empty":                     "",
		"file table":                "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM file WHERE path = 'C:\\foo.exe' AND version_compare(file_version, '1.0') < 0);",
		"unknown column among many": "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'ChatGPT' AND publisher = 'OpenAI' AND install_location LIKE '%\\WindowsApps\\%' AND version_compare(version, '1.0') < 0);",
		"GLOB":                      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name GLOB 'Citrix Workspace [0-9]*' AND publisher = 'Citrix Systems, Inc.' AND version_compare(version, '1.0') < 0);",
		"nested subquery":           "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' AND publisher = 'Bar' AND EXISTS (SELECT 1 FROM registry WHERE name = 'x') AND version_compare(version, '1.0') < 0);",
		"not equal":                 "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name != 'Foo' AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"NOT IN":                    "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name NOT IN ('Foo') AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"empty IN":                  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name IN () AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"less or equal":             "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' AND version <= '2.0' AND version_compare(version, '1.0') < 0);",
		"LIKE ESCAPE":               "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name LIKE 'Foo!_%' ESCAPE '!' AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"COLLATE":                   "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' COLLATE NOCASE AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"TRIM with characters":      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE TRIM(name, 'x') = 'Foo' AND publisher = 'Bar' AND version_compare(version, '1.0') < 0);",
		"OR of non-name terms":      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE (name = 'Foo' OR publisher = 'Bar') AND version_compare(version, '1.0') < 0);",
		"OR of conjunctions":        "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE ((name = 'Foo' AND publisher = 'Bar') OR (name = 'Baz' AND publisher = 'Qux')) AND version_compare(version, '1.0') < 0);",
		"top-level OR":              "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' OR name = 'Bar' AND version_compare(version, '1.0') < 0);",
		"no version_compare":        "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' AND publisher = 'Bar');",
		"version_compare not last":  "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE version_compare(version, '1.0') < 0 AND name = 'Foo');",
		"only version_compare":      "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE version_compare(version, '1.0') < 0);",
		"trailing tokens":           "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' AND version_compare(version, '1.0') < 0); SELECT 1;",
		"unterminated string":       "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo AND version_compare(version, '1.0') < 0);",
		"wrong ARM scope":           "SELECT 1 WHERE EXISTS (SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM programs WHERE name = 'Foo' AND version_compare(version, '1.0') < 0)) OR NOT EXISTS (SELECT 1 FROM system_info WHERE cpu_type LIKE 'x86%');",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			_, ok := patch_policy.ParseWindowsProgramsRule(query)
			require.False(t, ok)
		})
	}
}

// The parser inverts the generator, so a change to the wrapper must break this test
// rather than silently turning every app back to prefix-only matching.
func TestParseWindowsProgramsRuleRoundTrip(t *testing.T) {
	existsQueries := []string{
		"SELECT 1 FROM programs WHERE name LIKE 'Python 3.13.% (64-bit)' AND publisher = 'Python Software Foundation';",
		"SELECT 1 FROM programs WHERE (name LIKE 'RealVNC Viewer %' OR name LIKE 'RealVNC Connect Viewer %') AND publisher = 'RealVNC';",
	}
	for _, exists := range existsQueries {
		patched, err := patch_policy.GenerateQueryForManifest(patch_policy.PolicyData{
			Platform: "windows", Version: "1.2.3", ExistsQuery: exists,
		})
		require.NoError(t, err)

		plain, ok := patch_policy.ParseWindowsProgramsRule(patched)
		require.True(t, ok, patched)
		require.Len(t, plain.Clauses, 2)

		scoped := patch_policy.ScopeToWindowsARM(patched)
		arm, ok := patch_policy.ParseWindowsProgramsRule(scoped)
		require.True(t, ok, scoped)
		require.Equal(t, plain, arm)
	}
}
