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
	"github.com/stretchr/testify/require"
)

func (s *integrationEnterpriseTestSuite) TestLinuxOSVulns() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
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

	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, testOS))
	var osinfo struct {
		ID          uint `db:"id"`
		OSVersionID uint `db:"os_version_id"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osinfo,
			`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Greater(t, osinfo.ID, uint(0))

	software := []fleet.Software{
		{Name: "linux-image-6.11.0-19-generic", Version: "6.11.0-19.19", Source: "deb_packages"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	var hostResponse getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResponse)

	// assertSoftware := func(t *testing.T, software []fleet.HostSoftwareEntry, contains *fleet.Software) {
	// 	t.Helper()
	// 	var found bool
	// 	for _, s := range software {
	// 		if s.Name == contains.Name {
	// 			found = true
	// 			assert.Equal(t, s.Name, contains.Name)
	// 			assert.Equal(t, s.Version, contains.Version)
	// 			assert.Equal(t, s.Source, contains.Source)
	// 			assert.Equal(t, s.ExtensionID, contains.ExtensionID)
	// 			assert.Equal(t, s.Browser, contains.Browser)
	// 			assert.Equal(t, s.GenerateCPE, contains.GenerateCPE)
	// 			assert.Len(t, contains.Vulnerabilities, len(s.Vulnerabilities))
	// 			for i, vuln := range s.Vulnerabilities {
	// 				assert.Equal(t, vuln.CVE, contains.Vulnerabilities[i].CVE)
	// 				assert.Equal(t, vuln.DetailsLink, contains.Vulnerabilities[i].DetailsLink)
	// 			}
	// 		}
	// 	}
	// 	if !found {
	// 		t.Fatalf("software not found")
	// 	}
	// }

	// expectedSoft2 := &fleet.Software{
	// 	Name:        "bar",
	// 	Version:     "0.0.3",
	// 	Source:      "apps",
	// 	ExtensionID: "xyz",
	// 	Browser:     "chrome",
	// 	GenerateCPE: "somecpe",
	// 	Vulnerabilities: fleet.Vulnerabilities{
	// 		{
	// 			CVE:         "cve-123-123-132",
	// 			DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-123-123-132",
	// 		},
	// 	},
	// }

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, fmt.Sprintf("%s %s", testOS.Name, testOS.Version), osVersionsResp.OSVersions[0].Name)
	require.Equal(t, testOS.Name, osVersionsResp.OSVersions[0].NameOnly)
	require.Equal(t, testOS.Version, osVersionsResp.OSVersions[0].Version)
	require.Equal(t, testOS.Platform, osVersionsResp.OSVersions[0].Platform)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 1)
}
