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
- name: Subquery aliases
  platform: darwin
  interval: 3600
  query: SELECT t.uid FROM (SELECT uid FROM users) AS t JOIN (SELECT gid FROM groups) g ON g.gid = t.uid;
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
	out, _ := cmd.CombinedOutput() // exits 1 on the unknown table
	if strings.Contains(string(out), "no YAML parser") {
		t.Skip("validator needs PyYAML or Ruby to parse YAML")
	}

	// The unknown table proves the table check ran; it's reported once even though two fleets reference the file.
	var fails []string
	for line := range strings.Lines(string(out)) {
		if strings.HasPrefix(line, "FAIL") {
			fails = append(fails, strings.TrimSpace(line))
		}
	}
	require.Len(t, fails, 1, string(out))
	require.Contains(t, fails[0], "report 'Unknown table': table 'no_such_table' isn't in Fleet's osquery schema", string(out))
}
