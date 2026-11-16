package softwaredb

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToMapsExtensionFor(t *testing.T) {
	db := &DB{
		Darwin: []DarwinSoftware{
			{Name: "a", Source: "vscode_extensions"},
			{Name: "b", Source: "vscode_extensions", ExtensionFor: ptrString("")},
			{Name: "c", Source: "vscode_extensions", ExtensionFor: ptrString("cursor")},
			{Name: "d", Source: "chrome_extensions", ExtensionFor: ptrString("chrome")},
			{Name: "e", Source: "apps"},
		},
		Windows: []WindowsSoftware{
			{Name: "a", Source: "vscode_extensions"},
			{Name: "b", Source: "vscode_extensions", ExtensionFor: ptrString("")},
			{Name: "c", Source: "vscode_extensions", ExtensionFor: ptrString("cursor")},
			{Name: "d", Source: "chrome_extensions", ExtensionFor: ptrString("chrome")},
			{Name: "e", Source: "programs"},
		},
		Ubuntu: []UbuntuSoftware{
			{Name: "a", Source: "vscode_extensions"},
			{Name: "b", Source: "vscode_extensions", ExtensionFor: ptrString("")},
			{Name: "c", Source: "vscode_extensions", ExtensionFor: ptrString("cursor")},
			{Name: "d", Source: "chrome_extensions", ExtensionFor: ptrString("chrome")},
			{Name: "e", Source: "deb_packages"},
		},
	}
	indices := []uint32{0, 1, 2, 3, 4}

	for name, rows := range map[string][]map[string]string{
		"darwin":  db.DarwinToMaps(indices),
		"windows": db.WindowsToMaps(indices),
		"ubuntu":  db.UbuntuToMaps(indices),
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, "vscode", rows[0]["extension_for"])
			require.Equal(t, "vscode", rows[1]["extension_for"])
			require.Equal(t, "cursor", rows[2]["extension_for"])
			require.Equal(t, "chrome", rows[3]["extension_for"])
			require.NotContains(t, rows[4], "extension_for")
		})
	}
}

func TestLoadedEditorPluginsReportExtensionFor(t *testing.T) {
	const dbPath = "../software-library/software.db"
	if _, err := os.Stat(dbPath); err != nil {
		t.Skipf("%s not present: %v", dbPath, err)
	}
	db, err := LoadFromDatabase(dbPath)
	require.NoError(t, err)

	all := func(n int) []uint32 {
		indices := make([]uint32, n)
		for i := range indices {
			indices[i] = uint32(i) //nolint:gosec // test-only, bounded by slice length
		}
		return indices
	}
	for platform, rows := range map[string][]map[string]string{
		"darwin":  db.DarwinToMaps(all(len(db.Darwin))),
		"windows": db.WindowsToMaps(all(len(db.Windows))),
		"ubuntu":  db.UbuntuToMaps(all(len(db.Ubuntu))),
	} {
		t.Run(platform, func(t *testing.T) {
			counts := map[string]int{}
			for _, r := range rows {
				if r["source"] == "vscode_extensions" || r["source"] == "jetbrains_plugins" {
					require.NotEmpty(t, r["extension_for"], r["name"])
					counts[r["source"]]++
				}
			}
			require.Positive(t, counts["vscode_extensions"])
			require.Positive(t, counts["jetbrains_plugins"])
		})
	}
}
