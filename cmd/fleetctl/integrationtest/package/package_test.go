package _package

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/fleetctltest"
	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl/testing_utils"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	nettest.Run(t)

	updateOpt := update.DefaultOptions
	updateOpt.RootDirectory = t.TempDir()
	_, err := packaging.InitializeUpdates(updateOpt)
	require.NoError(t, err)

	// --type is required
	fleetctltest.RunAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -fleet-url & --enroll-secret are required together
	fleetctltest.RunAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"},
		"--enroll-secret and --fleet-url must be provided together")
	fleetctltest.RunAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")

	// --insecure and --fleet-certificate are mutually exclusive
	fleetctltest.RunAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"},
		"--insecure and --fleet-certificate may not be provided together")

	// Test invalid PEM file provided in --fleet-certificate.
	certDir := t.TempDir()
	fleetCertificate := filepath.Join(certDir, "fleet.pem")
	err = os.WriteFile(fleetCertificate, []byte("undefined"), os.FileMode(0o644))
	require.NoError(t, err)
	fleetctltest.RunAppCheckErr(t, []string{"package", "--type=deb", fmt.Sprintf("--fleet-certificate=%s", fleetCertificate)},
		fmt.Sprintf("failed to read fleet server certificate %q: invalid PEM file", fleetCertificate))

	if runtime.GOOS != "linux" {
		fleetctltest.RunAppCheckErr(t, []string{"package", "--type=msi", "--native-tooling"}, "native tooling is only available in Linux")
	}

	t.Run("deb", func(t *testing.T) {
		t.Cleanup(func() { os.Remove("fleet-linux-amd64.deb") })
		fleetctltest.RunAppForTest(t, []string{"package", "--type=deb", "--insecure", "--disable-open-folder"})
		info, err := os.Stat("fleet-linux-amd64.deb")
		require.NoError(t, err)
		require.Positive(t, info.Size()) // TODO verify contents
	})

	t.Run("deb with the fleet name", func(t *testing.T) {
		server, ds := testing_utils.RunServerWithMockedDS(t, &service.TestServerOpts{
			License: &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)},
		})
		ds.ListTeamsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
			return []*fleet.Team{{ID: 1, Name: "Workstations (Canary)", Secrets: []*fleet.EnrollSecret{{Secret: "canary-secret"}}}}, nil
		}
		configPath := filepath.Join(t.TempDir(), "config")
		config := fmt.Sprintf("contexts:\n  default:\n    address: %s\n    token: admin-token\n    tls-skip-verify: true\n", server.URL)
		require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

		t.Cleanup(func() { os.Remove("fleet-linux-arm64-workstations-canary.deb") })
		fleetctltest.RunAppForTest(t, []string{
			"package", "--type=deb", "--arch=arm64", "--insecure", "--disable-open-folder",
			"--fleet-url=" + server.URL, "--enroll-secret=canary-secret", "--config=" + configPath,
		})
		info, err := os.Stat("fleet-linux-arm64-workstations-canary.deb")
		require.NoError(t, err)
		require.Positive(t, info.Size())
	})

	t.Run("--use-sytem-configuration can't be used on installers that aren't pkg", func(t *testing.T) {
		for _, p := range []string{"deb", "msi", "rpm", ""} {
			fleetctltest.RunAppCheckErr(
				t,
				[]string{"package", fmt.Sprintf("--type=%s", p), "--use-system-configuration"},
				"--use-system-configuration is only available for pkg installers",
			)
		}
	})

	t.Run("--cpu-quota is only available for Linux packages", func(t *testing.T) {
		for _, p := range []string{"pkg", "msi"} {
			fleetctltest.RunAppCheckErr(
				t,
				[]string{"package", fmt.Sprintf("--type=%s", p), "--cpu-quota=1"},
				"--cpu-quota is only supported for deb/rpm/pkg.tar.zst packages",
			)
		}
		fleetctltest.RunAppCheckErr(t, []string{"package", "--type=deb", "--cpu-quota=0"}, "--cpu-quota must be greater than 0")
	})

	// fleet-windows-amd64.msi
	// runAppForTest(t, []string{"package", "--type=msi", "--insecure"}) TODO: this is currently failing on Github runners due to permission issues
	// info, err = os.Stat("orbit-osquery_0.0.3.msi")
	// require.NoError(t, err)
	// require.Greater(t, info.Size(), int64(0))

	// runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu
}
