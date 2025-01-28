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
			"install":   "./scripts/install_exe.ps1",
			"remove":    "./scripts/remove_exe.ps1",
			"uninstall": "./scripts/uninstall_exe.ps1",
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

func assertGoldenMatches(t *testing.T, goldenFile string, actual string, update bool) {
	t.Helper()
	goldenPath := filepath.Join("testdata", goldenFile+".golden")

	f, err := os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE, 0o644)
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
