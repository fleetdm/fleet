package maintainedapps

import (
	"encoding/json"
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

func TestScriptGeneration(t *testing.T) {
	appsJSON, err := os.ReadFile("apps.json")
	require.NoError(t, err)

	var apps []maintainedApp
	err = json.Unmarshal(appsJSON, &apps)
	require.NoError(t, err)

	for _, app := range apps {
		caskJSON, err := os.ReadFile(filepath.Join("testdata", app.Identifier+".json"))
		require.NoError(t, err)

		var cask brewCask
		err = json.Unmarshal(caskJSON, &cask)
		require.NoError(t, err)

		cask.PreUninstallScripts = app.PreUninstallScripts
		cask.PostUninstallScripts = app.PostUninstallScripts

		t.Run(app.Identifier, func(t *testing.T) {
			installScript, err := installScriptForApp(app, &cask)
			require.NoError(t, err)
			assertGoldenMatches(t, app.Identifier+"_install", installScript, *update)
			assertGoldenMatches(t, app.Identifier+"_uninstall", uninstallScriptForApp(&cask), *update)
		})
	}

}

func assertGoldenMatches(t *testing.T, goldenFile string, actual string, update bool) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "scripts", goldenFile+".golden.sh")

	var f *os.File
	var err error
	if update {
		f, err = os.OpenFile(goldenPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	} else {
		f, err = os.OpenFile(goldenPath, os.O_RDONLY, 0644)
	}
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
