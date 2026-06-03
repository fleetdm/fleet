package preview

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/fleetctltest"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/stretchr/testify/require"
)

func TestPreviewFailsOnInvalidLicenseKey(t *testing.T) {
	_, err := fleetctltest.RunAppNoChecks([]string{"preview", "--license-key", "0xDEADBEEF"})
	require.ErrorContains(t, err, "--license-key")
}

func TestIntegrationsPreview(t *testing.T) {
	nettest.Run(t)

	t.Setenv("FLEET_SERVER_ADDRESS", "https://localhost:8412")
	fleetctl.TestOverridePreviewDirectory = t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config")
	t.Log("config path: ", configPath)

	t.Cleanup(func() {
		require.Empty(t, fleetctltest.RunAppForTest(t, []string{"preview", "--config", configPath, "stop"}))
	})

	fleetTag := os.Getenv("FLEET_PREVIEW_TAG")
	if fleetTag == "" {
		fleetTag = "main"
	}

	require.NoError(t, nettest.RunWithNetRetry(t, func() error {
		_, err := fleetctltest.RunAppNoChecks([]string{
			"preview",
			"--config", configPath,
			"--preview-config-path", filepath.Join(gitRootPath(t), "tools", "osquery", "in-a-box"),
			"--tag", fleetTag,
			"--disable-open-browser",
		})
		return err
	}))

	// run some sanity checks on the preview environment

	// app configuration must disable analytics
	appConf := fleetctltest.RunAppForTest(t, []string{"get", "config", "--include-server-config", "--config", configPath, "--yaml"})
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
