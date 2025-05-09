package _package

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/cmd/fleetctl/fleetctl"
	"github.com/fleetdm/fleet/v4/orbit/pkg/packaging"
	"github.com/fleetdm/fleet/v4/orbit/pkg/update"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/stretchr/testify/require"
)

func TestPackage(t *testing.T) {
	nettest.Run(t)

	updateOpt := update.DefaultOptions
	updateOpt.RootDirectory = t.TempDir()
	updatesData, err := packaging.InitializeUpdates(updateOpt)
	require.NoError(t, err)

	// --type is required
	fleetctl.RunAppCheckErr(t, []string{"package", "deb"}, "Required flag \"type\" not set")

	// if you provide -fleet-url & --enroll-secret are required together
	fleetctl.RunAppCheckErr(t, []string{"package", "--type=deb", "--fleet-url=https://localhost:8080"},
		"--enroll-secret and --fleet-url must be provided together")
	fleetctl.RunAppCheckErr(t, []string{"package", "--type=deb", "--enroll-secret=foobar"}, "--enroll-secret and --fleet-url must be provided together")

	// --insecure and --fleet-certificate are mutually exclusive
	fleetctl.RunAppCheckErr(t, []string{"package", "--type=deb", "--insecure", "--fleet-certificate=test123"},
		"--insecure and --fleet-certificate may not be provided together")

	// Test invalid PEM file provided in --fleet-certificate.
	certDir := t.TempDir()
	fleetCertificate := filepath.Join(certDir, "fleet.pem")
	err = os.WriteFile(fleetCertificate, []byte("undefined"), os.FileMode(0o644))
	require.NoError(t, err)
	fleetctl.RunAppCheckErr(t, []string{"package", "--type=deb", fmt.Sprintf("--fleet-certificate=%s", fleetCertificate)},
		fmt.Sprintf("failed to read fleet server certificate %q: invalid PEM file", fleetCertificate))

	if runtime.GOOS != "linux" {
		fleetctl.RunAppCheckErr(t, []string{"package", "--type=msi", "--native-tooling"}, "native tooling is only available in Linux")
	}

	t.Run("deb", func(t *testing.T) {
		fleetctl.RunAppForTest(t, []string{"package", "--type=deb", "--insecure", "--disable-open-folder"})
		info, err := os.Stat(fmt.Sprintf("fleet-osquery_%s_amd64.deb", updatesData.OrbitVersion))
		require.NoError(t, err)
		require.Greater(t, info.Size(), int64(0)) // TODO verify contents
	})

	t.Run("--use-sytem-configuration can't be used on installers that aren't pkg", func(t *testing.T) {
		for _, p := range []string{"deb", "msi", "rpm", ""} {
			fleetctl.RunAppCheckErr(
				t,
				[]string{"package", fmt.Sprintf("--type=%s", p), "--use-system-configuration"},
				"--use-system-configuration is only available for pkg installers",
			)
		}
	})

	// fleet-osquery.msi
	// runAppForTest(t, []string{"package", "--type=msi", "--insecure"}) TODO: this is currently failing on Github runners due to permission issues
	// info, err = os.Stat("orbit-osquery_0.0.3.msi")
	// require.NoError(t, err)
	// require.Greater(t, info.Size(), int64(0))

	// runAppForTest(t, []string{"package", "--type=pkg", "--insecure"}) TODO: had a hard time getting xar installed on Ubuntu
}
