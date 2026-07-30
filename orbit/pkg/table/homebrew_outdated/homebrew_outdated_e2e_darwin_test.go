//go:build darwin

package homebrew_outdated

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGenerateE2E exercises the full pipeline against the real Homebrew
// installation on the machine running the test. It runs `brew outdated` and
// `brew info` directly (as the test user, without the console-user setuid that
// Generate performs), then feeds the real output through the parse/build helpers
// and asserts invariants. It is skipped when Homebrew is not installed.
func TestGenerateE2E(t *testing.T) {
	brewPath := findBrew("")
	if brewPath == "" {
		t.Skip("Homebrew is not installed; skipping e2e test")
	}

	outdatedOut, err := exec.Command(brewPath, "outdated", "--json=v2").Output()
	require.NoError(t, err, "brew outdated should succeed")

	pkgs, err := parseOutdated(outdatedOut)
	require.NoError(t, err, "real brew outdated output should parse")

	if len(pkgs) == 0 {
		t.Log("no outdated Homebrew packages on this machine; parse of empty result verified")
		return
	}

	// Enrich casks using real `brew info` output for the outdated packages.
	names := make([]string, 0, len(pkgs))
	for _, p := range pkgs {
		names = append(names, p.name)
	}
	infoOut, err := exec.Command(brewPath, append([]string{"info", "--json=v2"}, names...)...).Output()
	require.NoError(t, err, "brew info should succeed")
	casks, err := parseCaskInfo(infoOut)
	require.NoError(t, err, "real brew info output should parse")

	prefix := brewPath[:strings.LastIndex(brewPath, "/bin/brew")]
	rows := buildRows(pkgs, casks, prefix)
	require.Len(t, rows, len(pkgs))

	cols := Columns()
	for _, row := range rows {
		// Every declared column must be present in every row.
		for _, col := range cols {
			_, ok := row[col.Name]
			require.Truef(t, ok, "row missing column %q: %v", col.Name, row)
		}

		require.NotEmpty(t, row["name"], "name should be populated")
		require.Contains(t, []string{typeFormula, typeCask}, row["type"])
		require.NotEmpty(t, row["installed_version"])
		require.NotEmpty(t, row["current_version"])
		require.True(t, strings.HasPrefix(row["install_path"], prefix), "install_path should be under the brew prefix")

		if row["type"] == typeFormula {
			// Formula rows carry no cask-specific enrichment.
			require.Empty(t, row["app_name"])
			require.Empty(t, row["auto_updates"])
		} else {
			// auto_updates is a boolean flag for casks.
			require.Contains(t, []string{"0", "1"}, row["auto_updates"])
		}
	}
}
