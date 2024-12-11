package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/stretchr/testify/require"
)

func TestPreviewFailsOnInvalidLicenseKey(t *testing.T) {
	_, err := runAppNoChecks([]string{"preview", "--license-key", "0xDEADBEEF"})
	require.ErrorContains(t, err, "--license-key")
}

func TestIntegrationsPreview(t *testing.T) {
	nettest.Run(t)

	t.Setenv("FLEET_SERVER_ADDRESS", "https://localhost:8412")
	testOverridePreviewDirectory = t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config")
	t.Log("config path: ", configPath)

	t.Cleanup(func() {
		require.Equal(t, "", runAppForTest(t, []string{"preview", "--config", configPath, "stop"}))
	})

	var output *bytes.Buffer
	require.NoError(t, nettest.RunWithNetRetry(t, func() error {
		var err error
		output, err = runAppNoChecks([]string{
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
	queries := runAppForTest(t, []string{"get", "queries", "--config", configPath, "--json"})
	n := strings.Count(queries, `"kind":"query"`)
	require.Greater(t, n, 10)

	// app configuration must disable analytics
	appConf := runAppForTest(t, []string{"get", "config", "--include-server-config", "--config", configPath, "--yaml"})
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

func TestDockerCompose(t *testing.T) {
	t.Parallel()
	t.Run("returns the right command according to the version", func(t *testing.T) {
		v1 := dockerCompose{dockerComposeV1}
		cmd1 := v1.Command("up")
		require.Equal(t, []string{"docker-compose", "up"}, cmd1.Args)

		v2 := dockerCompose{dockerComposeV2}
		cmd2 := v2.Command("up")
		require.Equal(t, []string{"docker", "compose", "up"}, cmd2.Args)
	})

	t.Run("strings according to the version", func(t *testing.T) {
		v1 := dockerCompose{dockerComposeV1}
		require.Equal(t, v1.String(), "`docker-compose`")

		v2 := dockerCompose{dockerComposeV2}
		require.Equal(t, v2.String(), "`docker compose`")
	})
}
