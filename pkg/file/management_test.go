package file

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	update = flag.Bool("update", false, "update the golden files of this test")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

// Note: to update the goldens, run the tests with `-update`:
//
// go test ./pkg/file/... -update
func TestGetInstallAndRemoveScript(t *testing.T) {
	scriptsByType := map[string][2]string{
		"msi": {
			"./scripts/install_msi.ps1",
			"./scripts/remove_msi.ps1",
		},
		"pkg": {
			"./scripts/install_pkg.sh",
			"./scripts/remove_pkg.sh",
		},
		"deb": {
			"./scripts/install_deb.sh",
			"./scripts/remove_deb.sh",
		},
		"exe": {
			"./scripts/install_exe.ps1",
			"./scripts/remove_exe.ps1",
		},
	}

	for itype, scripts := range scriptsByType {
		gotScript := GetInstallScript(itype)
		assertGoldenMatches(t, scripts[0], gotScript, *update)

		gotScript = GetRemoveScript(itype)
		assertGoldenMatches(t, scripts[1], gotScript, *update)
	}
}

func assertGoldenMatches(t *testing.T, goldenFile string, actual string, update bool) {
	t.Helper()
	goldenPath := filepath.Join("testdata", goldenFile+".golden")

	f, err := os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE, 0644)
	require.NoError(t, err)
	defer f.Close()

	if update {
		_, err := f.WriteString(actual)
		require.NoError(t, err)
		return
	}

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, string(content), actual)
}
