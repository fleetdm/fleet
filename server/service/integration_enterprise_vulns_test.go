package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestLinuxOSVulns() {
	t := s.T()
	ctx := context.Background()

	kernel1 := fleet.Software{Name: "linux-image-6.11.0-9-generic", Version: "6.11.0-9.9", Source: "deb_packages", IsKernel: true}
	kernel2 := fleet.Software{Name: "linux-image-7.11.0-10-generic", Version: "7.11.0-10.10", Source: "deb_packages", IsKernel: true}
	kernel3 := fleet.Software{Name: "linux-image-8.11.0-11-generic", Version: "8.11.0-11.11", Source: "deb_packages", IsKernel: true}
	software := []fleet.Software{
		kernel1,
		kernel2,
		kernel3, // this one will have 0 vulns
	}

	cases := []struct {
		name                 string
		host                 *fleet.Host
		software             []fleet.Software
		vulns                []fleet.SoftwareVulnerability
		vulnsByKernelVersion map[string][]string
		os                   fleet.OperatingSystem
	}{
		{
			name:  "ubuntu",
			host:  test.NewHost(t, s.ds, "host_ubuntu2410", "", "hostkey_ubuntu2410", "hostuuid_ubuntu2410", time.Now(), test.WithPlatform("linux")),
			vulns: []fleet.SoftwareVulnerability{{CVE: "CVE-2025-0001"}, {CVE: "CVE-2025-0002"}, {CVE: "CVE-2025-0003"}},
			vulnsByKernelVersion: map[string][]string{
				kernel1.Version: {"CVE-2025-0001", "CVE-2025-0002"},
				kernel2.Version: {"CVE-2025-0003"},
				kernel3.Version: nil,
			},
			software: software,
			os:       fleet.OperatingSystem{Name: "Ubuntu", Version: "24.10", Arch: "x86_64", KernelVersion: "6.11.0-9-generic", Platform: "ubuntu"},
		},
		{
			name:     "amazon linux",
			host:     test.NewHost(t, s.ds, "host_amzn2023", "", "hostkey_amzn2023", "hostuuid_amzn2023", time.Now(), test.WithPlatform("fedora")),
			software: []fleet.Software{{Name: "kernel", Version: "6.1.144", Arch: "x86_64", Source: "rpm_packages", IsKernel: true}},
			vulns:    []fleet.SoftwareVulnerability{{CVE: "CVE-2025-0006"}},
			vulnsByKernelVersion: map[string][]string{
				"6.1.144": {"CVE-2025-0006"},
			},
			os: fleet.OperatingSystem{Name: "Amazon Linux", Version: "2023.0.0", Arch: "x86_64", KernelVersion: "6.1.144-170.251.amzn2023.x86_64", Platform: "amzn"},
		},
		{
			name:     "RHEL",
			host:     test.NewHost(t, s.ds, "host_fedora41", "", "hostkey_fedora41", "hostuuid_fedora41", time.Now(), test.WithPlatform("rhel")),
			software: []fleet.Software{{Name: "kernel-core", Version: "6.11.4", Arch: "aarch64", Source: "rpm_packages", IsKernel: true}},
			vulns:    []fleet.SoftwareVulnerability{{CVE: "CVE-2025-0007"}},
			vulnsByKernelVersion: map[string][]string{
				"6.11.4": {"CVE-2025-0007"},
			},
			os: fleet.OperatingSystem{Name: "Fedora Linux", Version: "41.0.0", Arch: "aarch64", KernelVersion: "6.11.4-301.fc41.aarch64", Platform: "rhel"},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			require.NoError(t, s.ds.UpdateHostOperatingSystem(ctx, tt.host.ID, tt.os))
			var osinfo struct {
				ID          uint `db:"id"`
				OSVersionID uint `db:"os_version_id"`
			}
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				return sqlx.GetContext(ctx, q, &osinfo,
					`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
					tt.os.Name, tt.os.Version, tt.os.Arch, tt.os.KernelVersion, tt.os.Platform)
			})
			require.Greater(t, osinfo.ID, uint(0))
			require.Greater(t, osinfo.OSVersionID, uint(0))

			_, err := s.ds.UpdateHostSoftware(ctx, tt.host.ID, tt.software)
			require.NoError(t, err)
			require.NoError(t, s.ds.LoadHostSoftware(ctx, tt.host, false))

			softwareIDByVersion := make(map[string]uint)
			for _, s := range tt.host.Software {
				softwareIDByVersion[s.Version] = s.ID
			}

			cpes := []fleet.SoftwareCPE{{SoftwareID: tt.host.Software[0].ID, CPE: "somecpe"}}
			_, err = s.ds.UpsertSoftwareCPEs(ctx, cpes)
			require.NoError(t, err)

			// Reload software so that GeneratedCPEID is set.
			require.NoError(t, s.ds.LoadHostSoftware(ctx, tt.host, false))

			var vulnsToInsert []fleet.SoftwareVulnerability
			for k, v := range tt.vulnsByKernelVersion {
				for _, s := range v {
					vulnsToInsert = append(vulnsToInsert, fleet.SoftwareVulnerability{
						SoftwareID: softwareIDByVersion[k],
						CVE:        s,
					})
				}
			}

			for _, v := range vulnsToInsert {
				_, err = s.ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
				require.NoError(t, err)
			}

			// Aggregate OS versions
			require.NoError(t, s.ds.UpdateOSVersions(ctx))
			require.NoError(t, s.ds.UpdateOSVersions(ctx))
			require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
			require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
			require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
			require.NoError(t, s.ds.InsertKernelSoftwareMapping(ctx))

			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				mysql.DumpTable(t, q, "kernels")
				return nil
			})

			var osVersionsResp osVersionsResponse
			s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
			var osVersion *fleet.OSVersion
			for _, os := range osVersionsResp.OSVersions {
				if os.Version == tt.os.Version {
					osVersion = &os
					break
				}
			}

			assert.Equal(t, 1, osVersion.HostsCount)
			assert.Equal(t, fmt.Sprintf("%s %s", tt.os.Name, tt.os.Version), osVersion.Name)
			assert.Equal(t, tt.os.Name, osVersion.NameOnly)
			assert.Equal(t, tt.os.Version, osVersion.Version)
			assert.Equal(t, tt.os.Platform, osVersion.Platform)
			assert.Len(t, osVersion.Vulnerabilities, len(tt.vulns))

			// Test entity endpoint
			var osVersionResp getOSVersionResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osVersion.OSVersionID), nil, http.StatusOK, &osVersionResp, "team_id", fmt.Sprintf("%d", 0))
			assert.Len(t, osVersionResp.OSVersion.Kernels, len(tt.software))
			// Make sure the ordering is the same
			sort.Slice(osVersionResp.OSVersion.Kernels, func(i, j int) bool {
				return osVersionResp.OSVersion.Kernels[i].Version < osVersionResp.OSVersion.Kernels[j].Version
			})
			sort.Slice(tt.software, func(i, j int) bool {
				return tt.software[i].Version < tt.software[j].Version
			})
			for i, k := range osVersionResp.OSVersion.Kernels {
				assert.Equal(t, tt.software[i].Version, k.Version)
				assert.Equal(t, uint(1), k.HostsCount)
				assert.ElementsMatch(t, tt.vulnsByKernelVersion[k.Version], k.Vulnerabilities)
			}

		})
	}

}
