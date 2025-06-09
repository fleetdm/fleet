package preview

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/stretchr/testify/require"
)

func TestPreviewFailsOnInvalidLicenseKey(t *testing.T) {
	_, err := fleetctl.RunAppNoChecks([]string{"preview", "--license-key", "0xDEADBEEF"})
	require.ErrorContains(t, err, "--license-key")
}

func TestIntegrationsPreview(t *testing.T) {
	nettest.Run(t)

	t.Setenv("FLEET_SERVER_ADDRESS", "https://localhost:8412")
	fleetctl.TestOverridePreviewDirectory = t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config")
	t.Log("config path: ", configPath)

	t.Cleanup(func() {
		require.Equal(t, "", fleetctl.RunAppForTest(t, []string{"preview", "--config", configPath, "stop"}))
	})

	var output *bytes.Buffer
	require.NoError(t, nettest.RunWithNetRetry(t, func() error {
		var err error
		output, err = fleetctl.RunAppNoChecks([]string{
			"preview",
			"--config", configPath,
			"--preview-config-path", filepath.Join(gitRootPath(t), "tools", "osquery", "in-a-box"),
			"--tag", "main",
			"--disable-open-browser",
		})
		return err
	}))

	queriesRe := regexp.MustCompile(`applied ([0-9]+) queries`)
	policiesRe := regexp.MustCompile(`applied ([0-9]+) policies`)
	require.True(t, queriesRe.MatchString(output.String()))
	require.True(t, policiesRe.MatchString(output.String()))

	// run some sanity checks on the preview environment

	// standard queries must have been loaded
	queries := fleetctl.RunAppForTest(t, []string{"get", "queries", "--config", configPath, "--json"})
	n := strings.Count(queries, `"kind":"query"`)
	require.Greater(t, n, 10)

	// app configuration must disable analytics
	appConf := fleetctl.RunAppForTest(t, []string{"get", "config", "--include-server-config", "--config", configPath, "--yaml"})
	ok := strings.Contains(appConf, `enable_analytics: false`)
	require.True(t, ok, appConf)

	// software inventory must be enabled
	ok = strings.Contains(appConf, `enable_software_inventory: true`)
	require.True(t, ok, appConf)

	// current instance checks must be on
	ok = strings.Contains(appConf, `current_instance_checks: "yes"`)
	require.True(t, ok, appConf)

	// a vulnerability database path must be set
	ok = strings.Contains(appConf, `databases_path: /vulndb`)
	require.True(t, ok, appConf)
}

func gitRootPath(t *testing.T) string {
	path, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	require.NoError(t, err)
	return strings.TrimSpace(string(path))
}
