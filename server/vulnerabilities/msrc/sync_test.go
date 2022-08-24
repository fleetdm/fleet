package msrc

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	msrc_io "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/io"
	"github.com/stretchr/testify/require"
)

func TestSync(t *testing.T) {
	t.Run("#bulletinsDelta", func(t *testing.T) {
		t.Run("win OS provided", func(t *testing.T) {
			os := []fleet.OperatingSystem{
				{
					Name:          "CentOS",
					Version:       "8.0.0",
					Platform:      "rhel",
					KernelVersion: "5.10.76-linuxkit",
				},
				{
					Name:          "Ubuntu",
					Version:       "20.4.0 LTS",
					Platform:      "ubuntu",
					KernelVersion: "5.10.76-linuxkit",
				},
				{
					Name:          "Ubuntu",
					Version:       "20.5.0 LTS",
					Platform:      "ubuntu",
					KernelVersion: "5.10.76-linuxkit",
				},
				{
					Name:          "Microsoft Windows 11 Enterprise",
					Version:       "21H2",
					Arch:          "64-bit",
					KernelVersion: "10.0.22000.795",
				},
				{
					Name:          "Microsoft Windows 10 Pro",
					Version:       "10.0.19044",
					Arch:          "64-bit",
					KernelVersion: "10.0.19044",
				},
			}

			remote := []msrc_io.SecurityBulletinName{
				"Windows_10-2022_10_10.json",
				"Windows_11-2022_10_10.json",
				"Windows_Server_2016-2022_10_10.json",
				"Windows_8.1-2022_10_10.json",
			}

			t.Run("no local bulletins", func(t *testing.T) {
				var local []msrc_io.SecurityBulletinName

				t.Run("should return the matching remote bulletins", func(t *testing.T) {
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []msrc_io.SecurityBulletinName{
						"Windows_10-2022_10_10.json",
						"Windows_11-2022_10_10.json",
					})
					require.Empty(t, toDelete)
				})
			})

			t.Run("missing some local bulletin", func(t *testing.T) {
				local := []msrc_io.SecurityBulletinName{
					"Windows_10-2022_10_10.json",
				}

				t.Run("should return what bulletin needs to be downloaded", func(t *testing.T) {
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []msrc_io.SecurityBulletinName{
						"Windows_11-2022_10_10.json",
					})
					require.Empty(t, toDelete)
				})
			})

			t.Run("out of date local bulletin", func(t *testing.T) {
				local := []msrc_io.SecurityBulletinName{
					"Windows_10-2022_09_10.json",
					"Windows_11-2022_10_10.json",
				}

				t.Run("should return what bulletin needs to be downloaded", func(t *testing.T) {
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.ElementsMatch(t, toDownload, []msrc_io.SecurityBulletinName{
						"Windows_10-2022_10_10.json",
					})
					require.ElementsMatch(t, toDelete, []msrc_io.SecurityBulletinName{
						"Windows_10-2022_09_10.json",
					})
				})
			})

			t.Run("up to date local bulletins", func(t *testing.T) {
				local := []msrc_io.SecurityBulletinName{
					"Windows_10-2022_10_10.json",
					"Windows_11-2022_10_10.json",
				}

				t.Run("should do nothing", func(t *testing.T) {
					toDownload, toDelete := bulletinsDelta(os, local, remote)

					require.Empty(t, toDownload)
					require.Empty(t, toDelete)
				})
			})
		})

		t.Run("no Win OS provided", func(t *testing.T) {
			os := []fleet.OperatingSystem{
				{
					Name:          "CentOS",
					Version:       "8.0.0",
					Platform:      "rhel",
					KernelVersion: "5.10.76-linuxkit",
				},
			}
			local := []msrc_io.SecurityBulletinName{"Windows_11-2022_10_10.json"}
			remote := []msrc_io.SecurityBulletinName{"Windows_10-2022_10_10.json"}

			t.Run("nothing to download, nothing to delete", func(t *testing.T) {
				toDownload, toDelete := bulletinsDelta(os, local, remote)
				require.Empty(t, toDownload)
				require.Empty(t, toDelete)
			})
		})

		t.Run("no OS provided", func(t *testing.T) {
			var os []fleet.OperatingSystem
			t.Run("no local bulletins", func(t *testing.T) {
				var local []msrc_io.SecurityBulletinName

				t.Run("returns all remote", func(t *testing.T) {
					remote := []msrc_io.SecurityBulletinName{
						"Windows_10-2022_10_10.json",
						"Windows_11-2022_10_10.json",
					}

					toDownload, toDelete := bulletinsDelta(os, local, remote)
					require.ElementsMatch(t, toDownload, remote)
					require.Empty(t, toDelete)
				})
			})
		})
	})
}
