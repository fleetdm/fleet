package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestLinuxOSVulns() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo3.local",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "ubuntu",
		OSVersion:       "Ubuntu 22.04",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	testOS := fleet.OperatingSystem{Name: "Ubuntu", Version: "24.10", Arch: "x86_64", KernelVersion: "6.11.0-29-generic", Platform: "ubuntu"}

	require.NoError(t, s.ds.UpdateHostOperatingSystem(ctx, host.ID, testOS))
	var osinfo struct {
		ID          uint `db:"id"`
		OSVersionID uint `db:"os_version_id"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &osinfo,
			`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Greater(t, osinfo.ID, uint(0))
	require.Greater(t, osinfo.OSVersionID, uint(0))

	software := []fleet.Software{
		{Name: "linux-image-6.11.0-19-generic", Version: "6.11.0-19.19", Source: "deb_packages", IsKernel: true},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	soft1 := host.Software[0]

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(ctx, cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))
	soft1 = host.Software[0]

	vuln1 := fleet.SoftwareVulnerability{
		SoftwareID: soft1.ID,
		CVE:        "cve-123-123-132",
	}
	vuln2 := fleet.SoftwareVulnerability{
		SoftwareID: soft1.ID,
		CVE:        "cve-123-123-133",
	}
	vuln3 := fleet.SoftwareVulnerability{
		SoftwareID: soft1.ID,
		CVE:        "cve-123-123-134",
	}
	for _, vuln := range []fleet.SoftwareVulnerability{vuln1, vuln2, vuln3} {
		inserted, err := s.ds.InsertSoftwareVulnerability(
			ctx, vuln, fleet.NVDSource,
		)
		require.NoError(t, err)
		require.True(t, inserted)
	}

	// Aggregate OS versions
	require.NoError(t, s.ds.UpdateOSVersions(ctx))
	require.NoError(t, s.ds.UpdateOSVersions(ctx))
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, time.Now()))

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	assert.Len(t, osVersionsResp.OSVersions, 1)
	assert.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	assert.Equal(t, fmt.Sprintf("%s %s", testOS.Name, testOS.Version), osVersionsResp.OSVersions[0].Name)
	assert.Equal(t, testOS.Name, osVersionsResp.OSVersions[0].NameOnly)
	assert.Equal(t, testOS.Version, osVersionsResp.OSVersions[0].Version)
	assert.Equal(t, testOS.Platform, osVersionsResp.OSVersions[0].Platform)
	assert.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 3)

	// Test entity endpoint
	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osVersionsResp.OSVersions[0].OSVersionID), nil, http.StatusOK, &osVersionResp, "team_id", fmt.Sprintf("%d", 0))
	assert.Len(t, osVersionResp.OSVersion.Kernels, 1)
	assert.Equal(t, osVersionResp.OSVersion.Kernels[0].Version, software[0].Version)
	assert.Equal(t, osVersionResp.OSVersion.Kernels[0].HostsCount, uint(1))
	assert.Equal(t, osVersionResp.OSVersion.Kernels[0].Vulnerabilities, []string{vuln1.CVE, vuln2.CVE, vuln3.CVE})

}
