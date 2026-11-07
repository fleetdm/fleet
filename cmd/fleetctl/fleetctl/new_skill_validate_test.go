package fleetctl

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Runs the validator that `fleetctl new` scaffolds on trimmed copies of
// dogfood reports it used to flag, both referenced from two fleets.
func TestNewFleetGitopsSkillValidatorQueries(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not installed")
	}

	dir := t.TempDir()
	// The validator reads Fleet's osquery schema from schema/ in any parent of the repo it checks.
	schema, err := os.ReadFile(filepath.Join("..", "..", "..", "schema", "osquery_fleet_schema.json"))
	require.NoError(t, err)
	root := filepath.Join(dir, "gitops")
	files := map[string]string{
		filepath.Join(dir, "schema", "osquery_fleet_schema.json"): string(schema),
		filepath.Join(root, "default.yml"):                        "org_settings:\n  org_info:\n    org_name: Test\n",
		filepath.Join(root, "fleets", "a.yml"):                    "name: A\nreports:\n  - path: ../lib/reports.yml\n",
		filepath.Join(root, "fleets", "b.yml"):                    "name: B\nreports:\n  - path: ../lib/reports.yml\n",
		filepath.Join(root, "lib", "reports.yml"): `- name: SQLite table-valued functions
  platform: darwin
  interval: 3600
  query: |
    SELECT
      (SELECT j.key FROM pmset s, json_each(s.json_result) j WHERE s.getting = 'stats') AS sleep_count,
      (SELECT COUNT(*) FROM pmset s JOIN json_tree(s.json_result) t) AS nodes
    FROM pmset c WHERE c.getting = 'custom';
- name: CTEs with column lists and VALUES bodies
  platform: windows
  interval: 3600
  query: |
    WITH
    scheme AS (SELECT data AS guid FROM registry WHERE path = 'a'),
    vals AS (
      SELECT lower(path) AS p, data FROM registry WHERE path LIKE 'b%'
      UNION ALL
      SELECT lower(path), data FROM registry WHERE path LIKE 'c%'
    ),
    setting(name, guid) AS (VALUES ('sleep', 'd'), ('display', 'e')),
    source(name, idx) AS (VALUES ('ac', 'acsettingindex'), ('dc', 'dcsettingindex'))
    SELECT st.name || '_' || so.name AS k, (SELECT data FROM vals WHERE p = sc.guid || st.guid || so.idx) AS v
    FROM setting st, source so, scheme sc;
- name: Materialized CTEs
  platform: windows
  interval: 3600
  query: WITH m AS MATERIALIZED (SELECT data FROM registry WHERE path = 'a'), n AS NOT MATERIALIZED (SELECT data FROM registry WHERE path = 'b') SELECT * FROM m, n;
- name: Function argument after an unaliased source
  platform: darwin
  interval: 3600
  query: SELECT (SELECT COUNT(*) FROM pmset s, json_each(s.json_result)) AS nodes FROM users
- name: Subquery aliases
  platform: darwin
  interval: 3600
  query: SELECT t.uid FROM (SELECT uid FROM users) AS t JOIN (SELECT gid FROM groups) g ON g.gid = t.uid;
- name: Joins without aliases
  platform: darwin
  interval: 3600
  query: SELECT * FROM users JOIN no_such_table, groups JOIN no_such_table_2;
- name: Named window
  platform: darwin
  interval: 3600
  query: SELECT users.no_such_column, COUNT(*) OVER users FROM users WINDOW users AS (PARTITION BY uid);
- name: Unknown table after a function
  platform: darwin
  interval: 3600
  query: SELECT * FROM json_each(json_extract('[]', '$')) AS j, no_such_table AS n;
- name: Unknown table in a function argument
  platform: darwin
  interval: 3600
  query: SELECT * FROM json_each((SELECT value FROM no_such_table)) AS j;
- name: Unknown table
  platform: darwin
  interval: 3600
  query: SELECT * FROM no_such_table;
`,
	}
	for path, content := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}

	validator := filepath.Join("templates", "new", ".agents", "skills", "fleet-gitops", "scripts", "validate.py")
	cmd := exec.Command(python, validator, root, "--offline")
	cmd.Env = append(os.Environ(), "XDG_CACHE_HOME="+t.TempDir())
	out, err := cmd.CombinedOutput()
	if strings.Contains(string(out), "no YAML parser") {
		t.Skip("validator needs PyYAML or Ruby to parse YAML")
	}
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, string(out))
	require.Equal(t, 1, exitErr.ExitCode(), string(out))

	// Each FAIL is reported once even though two fleets reference the file. The validator sorts findings by message.
	var fails []string
	for line := range strings.Lines(string(out)) {
		if strings.HasPrefix(line, "FAIL") {
			fails = append(fails, strings.TrimSpace(line))
		}
	}
	want := []string{
		"report 'Joins without aliases': table 'no_such_table' isn't in Fleet's osquery schema",
		"report 'Joins without aliases': table 'no_such_table_2' isn't in Fleet's osquery schema",
		"report 'Named window': column 'no_such_column' isn't in table 'users'",
		"report 'Unknown table after a function': table 'no_such_table' isn't in Fleet's osquery schema",
		"report 'Unknown table in a function argument': table 'no_such_table' isn't in Fleet's osquery schema",
		"report 'Unknown table': table 'no_such_table' isn't in Fleet's osquery schema",
	}
	require.Len(t, fails, len(want), string(out))
	for i, f := range fails {
		require.Contains(t, f, want[i], string(out))
	}
}
