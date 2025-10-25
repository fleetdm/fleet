package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
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
			host:  test.NewHost(t, s.ds, "host_ubuntu2410", "", "hostkey_ubuntu2410", "hostuuid_ubuntu2410", time.Now(), test.WithPlatform("ubuntu")),
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

			// Entity endpoint kernels field should be empty
			require.NoError(t, s.ds.UpdateOSVersions(ctx))
			resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, "team_id", fmt.Sprintf("%d", 0))
			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Contains(t, string(bodyBytes), `"kernels": []`)

			// Aggregate OS versions
			require.NoError(t, s.ds.UpdateOSVersions(ctx))
			require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
			require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
			require.NoError(t, s.ds.InsertKernelSoftwareMapping(ctx))

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
			require.NotNil(t, osVersionResp.OSVersion.Kernels)
			kernels := *osVersionResp.OSVersion.Kernels
			assert.Len(t, kernels, len(tt.software))
			// Make sure the ordering is the same
			sort.Slice(kernels, func(i, j int) bool {
				return kernels[i].Version < kernels[j].Version
			})
			sort.Slice(tt.software, func(i, j int) bool {
				return tt.software[i].Version < tt.software[j].Version
			})
			for i, k := range kernels {
				assert.Equal(t, tt.software[i].Version, k.Version)
				assert.Equal(t, uint(1), k.HostsCount)
				assert.ElementsMatch(t, tt.vulnsByKernelVersion[k.Version], k.Vulnerabilities)
			}
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestOSVersionsMaxVulnerabilities() {
	t := s.T()
	ctx := t.Context()

	// Shared setup - create a host with an OS that has many vulnerabilities
	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "1"),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            uuid.NewString(),
		Hostname:        t.Name() + "foo.local",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	// Set the OS version with a kernel version
	require.NoError(t, s.ds.UpdateHostOperatingSystem(ctx, host.ID, fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "22.04.1 LTS",
		Platform:      "ubuntu",
		Arch:          "x86_64",
		KernelVersion: "5.15.0-1001",
	}))

	// Create kernel software with many vulnerabilities
	software := []fleet.Software{
		{Name: "linux-image-5.15.0-1001-generic", Version: "5.15.0-1001", Source: "deb_packages", IsKernel: true},
		{Name: "linux-image-5.15.0-1002-generic", Version: "5.15.0-1002", Source: "deb_packages", IsKernel: true},
	}

	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	cpes := make([]fleet.SoftwareCPE, 0, len(software))
	for _, sw := range host.Software {
		cpes = append(cpes, fleet.SoftwareCPE{
			SoftwareID: sw.ID,
			CPE:        fmt.Sprintf("cpe:2.3:a:linux:kernel:%s:*:*:*:*:*:*:*", sw.Version),
		})
	}
	_, err = s.ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	// Reload software
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	// Insert multiple vulnerabilities for each software
	vulns := []string{"CVE-2024-0001", "CVE-2024-0002", "CVE-2024-0003", "CVE-2024-0004", "CVE-2024-0005", "CVE-2024-0006", "CVE-2024-0007", "CVE-2024-0008"}
	for _, sw := range host.Software {
		for _, cve := range vulns {
			_, err = s.ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
				SoftwareID: sw.ID,
				CVE:        cve,
			}, fleet.NVDSource)
			require.NoError(t, err)
		}
	}

	// Update OS versions table
	require.NoError(t, s.ds.UpdateOSVersions(ctx))
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	require.NoError(t, s.ds.InsertKernelSoftwareMapping(ctx))

	// Get the OS version ID for entity endpoint tests
	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	var osVersionID uint
	for _, os := range osVersionsResp.OSVersions {
		if os.Version == "22.04.1 LTS" {
			osVersionID = os.OSVersionID
			break
		}
	}
	require.NotZero(t, osVersionID, "Should find Ubuntu 22.04.1 LTS OS version ID")

	t.Run("aggregate endpoint", func(t *testing.T) {
		// Test 1: Request without max_vulnerabilities should return all vulnerabilities
		var resp osVersionsResponse
		s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &resp)
		var osVersion *fleet.OSVersion
		for _, os := range resp.OSVersions {
			if os.Version == "22.04.1 LTS" {
				osVersion = &os
				break
			}
		}
		require.NotNil(t, osVersion, "Should find Ubuntu 22.04.1 LTS")
		assert.Equal(t, len(vulns), len(osVersion.Vulnerabilities), "Should return all vulnerabilities when max_vulnerabilities is not specified")
		assert.Equal(t, len(vulns), osVersion.VulnerabilitiesCount, "Count should match total vulnerabilities")

		// Test 2: Request with max_vulnerabilities=3 should return only 3 vulnerabilities
		s.DoJSON("GET", "/api/latest/fleet/os_versions?max_vulnerabilities=3", nil, http.StatusOK, &resp)
		osVersion = nil
		for _, os := range resp.OSVersions {
			if os.Version == "22.04.1 LTS" {
				osVersion = &os
				break
			}
		}
		require.NotNil(t, osVersion, "Should find Ubuntu 22.04.1 LTS")
		assert.Equal(t, 3, len(osVersion.Vulnerabilities), "Should return only 3 vulnerabilities when max_vulnerabilities=3")
		assert.Equal(t, len(vulns), osVersion.VulnerabilitiesCount, "Count should still show total vulnerabilities")

		// Test 3: Request with max_vulnerabilities=0 should return empty array with count
		s.DoJSON("GET", "/api/latest/fleet/os_versions?max_vulnerabilities=0", nil, http.StatusOK, &resp)
		osVersion = nil
		for _, os := range resp.OSVersions {
			if os.Version == "22.04.1 LTS" {
				osVersion = &os
				break
			}
		}
		require.NotNil(t, osVersion, "Should find Ubuntu 22.04.1 LTS")
		assert.Equal(t, 0, len(osVersion.Vulnerabilities), "Should return 0 vulnerabilities when max_vulnerabilities=0")
		assert.Equal(t, len(vulns), osVersion.VulnerabilitiesCount, "Count should still show total vulnerabilities")

		// Test 4: Request with max_vulnerabilities=-1 should return error
		res := s.Do("GET", "/api/latest/fleet/os_versions?max_vulnerabilities=-1", nil, http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "max_vulnerabilities must be >= 0")
	})

	t.Run("entity endpoint", func(t *testing.T) {
		// Test 1: Request without max_vulnerabilities should return all vulnerabilities
		var resp getOSVersionResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osVersionID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.OSVersion)
		assert.Equal(t, len(vulns), len(resp.OSVersion.Vulnerabilities), "Should return all vulnerabilities when max_vulnerabilities is not specified")
		assert.Equal(t, len(vulns), resp.OSVersion.VulnerabilitiesCount, "Count should match total vulnerabilities")

		// Test 2: Request with max_vulnerabilities=3 should return only 3 vulnerabilities
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d?max_vulnerabilities=3", osVersionID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.OSVersion)
		assert.Equal(t, 3, len(resp.OSVersion.Vulnerabilities), "Should return only 3 vulnerabilities when max_vulnerabilities=3")
		assert.Equal(t, len(vulns), resp.OSVersion.VulnerabilitiesCount, "Count should still show total vulnerabilities")

		// Test 3: Request with max_vulnerabilities=0 should return empty array with count
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d?max_vulnerabilities=0", osVersionID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.OSVersion)
		assert.Equal(t, 0, len(resp.OSVersion.Vulnerabilities), "Should return 0 vulnerabilities when max_vulnerabilities=0")
		assert.Equal(t, len(vulns), resp.OSVersion.VulnerabilitiesCount, "Count should still show total vulnerabilities")

		// Test 4: Request with max_vulnerabilities=-1 should return error
		res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d?max_vulnerabilities=-1", osVersionID), nil, http.StatusUnprocessableEntity)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "max_vulnerabilities must be >= 0")
	})
}
