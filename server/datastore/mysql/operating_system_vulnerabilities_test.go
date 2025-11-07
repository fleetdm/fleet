package mysql

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOperatingSystemVulnerabilities(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListOSVulnerabilitiesEmpty", testListOSVulnerabilitiesByOSEmpty},
		{"ListOSVulnerabilities", testListOSVulnerabilitiesByOS},
		{"ListVulnsByOsNameAndVersion", testListVulnsByOsNameAndVersion},
		{"InsertOSVulnerabilities", testInsertOSVulnerabilities},
		{"InsertSingleOSVulnerability", testInsertOSVulnerability},
		{"DeleteOSVulnerabilitiesEmpty", testDeleteOSVulnerabilitiesEmpty},
		{"DeleteOSVulnerabilities", testDeleteOSVulnerabilities},
		{"DeleteOutOfDateOSVulnerabilities", testDeleteOutOfDateOSVulnerabilities},
		{"TestListKernelsByOS", testListKernelsByOS},
		{"TestKernelVulnsHostCount", testKernelVulnsHostCount},
		{"RefreshOSVersionVulnerabilities", testRefreshOSVersionVulnerabilities},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListOSVulnerabilitiesByOSEmpty(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	actual, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
	require.NoError(t, err)
	require.Empty(t, actual)
}

func testListOSVulnerabilitiesByOS(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3")},
		{CVE: "cve-3", OSID: 1, ResolvedInVersion: ptr.String("10.14.2")},
		{CVE: "cve-2", OSID: 1, ResolvedInVersion: ptr.String("8.123.1")},
		{CVE: "cve-1", OSID: 2, ResolvedInVersion: ptr.String("1.2.3")},
		{CVE: "cve-5", OSID: 2, ResolvedInVersion: ptr.String("10.14.2")},
	}

	for _, v := range vulns {
		_, err := ds.InsertOSVulnerability(ctx, v, fleet.MSRCSource)
		require.NoError(t, err)
	}

	t.Run("returns matching", func(t *testing.T) {
		expected := []fleet.OSVulnerability{
			{CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3"), Source: fleet.MSRCSource},
			{CVE: "cve-3", OSID: 1, ResolvedInVersion: ptr.String("10.14.2"), Source: fleet.MSRCSource},
			{CVE: "cve-2", OSID: 1, ResolvedInVersion: ptr.String("8.123.1"), Source: fleet.MSRCSource},
		}

		actual, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
		require.NoError(t, err)
		require.ElementsMatch(t, expected, actual)
	})
}

func testListVulnsByOsNameAndVersion(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	seedOS := []fleet.OperatingSystem{
		{
			Name:           "Microsoft Windows 11 Pro 21H2",
			Version:        "10.0.22000.795",
			Arch:           "64-bit",
			KernelVersion:  "10.0.22000.795",
			Platform:       "windows",
			DisplayVersion: "21H2",
		},
		{
			Name:           "Microsoft Windows 11 Pro 21H2",
			Version:        "10.0.22000.795",
			Arch:           "ARM 64-bit",
			KernelVersion:  "10.0.22000.795",
			Platform:       "windows",
			DisplayVersion: "21H2",
		},
		{
			Name:           "Microsoft Windows 11 Pro 22H2",
			Version:        "10.0.22621.890",
			Arch:           "64-bit",
			KernelVersion:  "10.0.22621.890",
			Platform:       "windows",
			DisplayVersion: "22H2",
		},
	}

	var dbOS []fleet.OperatingSystem
	for _, seed := range seedOS {
		os, err := newOperatingSystemDB(context.Background(), ds.writer(context.Background()), seed)
		require.NoError(t, err)
		dbOS = append(dbOS, *os)
	}

	cves, err := ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", false, nil, nil)
	require.NoError(t, err)
	require.Empty(t, cves.Vulnerabilities)

	mockTime := time.Date(2024, time.January, 18, 10, 0, 0, 0, time.UTC)

	cveMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        ptr.Float64(9.7),
			EPSSProbability:  ptr.Float64(4.2),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "A bad vulnerability",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        ptr.Float64(9.8),
			EPSSProbability:  ptr.Float64(0.1),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "A worse vulnerability",
		},
		{
			CVE:              "CVE-2021-1236",
			CVSSScore:        ptr.Float64(9.8),
			EPSSProbability:  ptr.Float64(0.1),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "A terrible vulnerability",
		},
	}

	err = ds.InsertCVEMeta(ctx, cveMeta)
	require.NoError(t, err)

	// add CVEs for each OS with different architectures
	vulns := []fleet.OSVulnerability{
		{CVE: "CVE-2021-1234", OSID: dbOS[0].ID, ResolvedInVersion: ptr.String("1.2.3")},
		{CVE: "CVE-2021-1235", OSID: dbOS[1].ID, ResolvedInVersion: ptr.String("10.14.2")},
		{CVE: "CVE-2021-1236", OSID: dbOS[2].ID, ResolvedInVersion: ptr.String("103.2.1")},
	}

	_, err = ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)

	// push other vulns into the past to ensure "SELECT DISTINCT" wouldn't deduplicate properly
	_, err = ds.writer(ctx).ExecContext(ctx, "UPDATE operating_system_vulnerabilities SET created_at = NOW() - INTERVAL 5 SECOND")
	require.NoError(t, err)

	_, err = ds.InsertOSVulnerabilities(ctx, []fleet.OSVulnerability{
		{CVE: "CVE-2021-1234", OSID: dbOS[1].ID, ResolvedInVersion: ptr.String("1.2.3")}, // same OS, different arch
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// test without CVS meta
	cves, err = ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", false, nil, nil)
	require.NoError(t, err)

	expected := []string{"CVE-2021-1234", "CVE-2021-1235"}
	require.Len(t, cves.Vulnerabilities, 2)
	for _, cve := range cves.Vulnerabilities {
		require.Contains(t, expected, cve.CVE)
		require.Greater(t, cve.CreatedAt, time.Now().Add(-time.Hour)) // assert non-zero time
	}

	// test with CVS meta
	cves, err = ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", true, nil, nil)
	require.NoError(t, err)
	require.Len(t, cves.Vulnerabilities, 2)

	require.Equal(t, cveMeta[0].CVE, cves.Vulnerabilities[0].CVE)
	require.NotNil(t, cves.Vulnerabilities[0].CVSSScore)
	require.Equal(t, *cveMeta[0].CVSSScore, **cves.Vulnerabilities[0].CVSSScore)
	require.NotNil(t, cves.Vulnerabilities[0].EPSSProbability)
	require.Equal(t, *cveMeta[0].EPSSProbability, **cves.Vulnerabilities[0].EPSSProbability)
	require.NotNil(t, cves.Vulnerabilities[0].CISAKnownExploit)
	require.Equal(t, *cveMeta[0].CISAKnownExploit, **cves.Vulnerabilities[0].CISAKnownExploit)
	require.NotNil(t, cves.Vulnerabilities[0].CVEPublished)
	require.Equal(t, *cveMeta[0].Published, **cves.Vulnerabilities[0].CVEPublished)
	require.NotNil(t, cves.Vulnerabilities[0].Description)
	require.Equal(t, cveMeta[0].Description, **cves.Vulnerabilities[0].Description)
	require.Equal(t, cveMeta[1].CVE, cves.Vulnerabilities[1].CVE)
	require.NotNil(t, cves.Vulnerabilities[1].CVSSScore)
	require.Equal(t, *cveMeta[1].CVSSScore, **cves.Vulnerabilities[1].CVSSScore)
	require.NotNil(t, cves.Vulnerabilities[1].EPSSProbability)
	require.Equal(t, *cveMeta[1].EPSSProbability, **cves.Vulnerabilities[1].EPSSProbability)
	require.NotNil(t, cves.Vulnerabilities[1].CISAKnownExploit)
	require.Equal(t, *cveMeta[1].CISAKnownExploit, **cves.Vulnerabilities[1].CISAKnownExploit)
	require.NotNil(t, cves.Vulnerabilities[1].CVEPublished)
	require.Equal(t, *cveMeta[1].Published, **cves.Vulnerabilities[1].CVEPublished)
	require.NotNil(t, cves.Vulnerabilities[1].Description)
	require.Equal(t, cveMeta[1].Description, **cves.Vulnerabilities[1].Description)
}

func testInsertOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1},
		{CVE: "cve-3", OSID: 1},
		{CVE: "cve-2", OSID: 1},
	}

	c, err := ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.Equal(t, int64(3), c)

	expected := []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1, Source: fleet.MSRCSource},
		{CVE: "cve-3", OSID: 1, Source: fleet.MSRCSource},
		{CVE: "cve-2", OSID: 1, Source: fleet.MSRCSource},
	}

	actual, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
	require.NoError(t, err)
	require.ElementsMatch(t, expected, actual)
}

func testInsertOSVulnerability(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := fleet.OSVulnerability{
		CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3"),
	}

	vulnsUpdate := fleet.OSVulnerability{
		CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.4"),
	}

	vulnNoCVE := fleet.OSVulnerability{
		OSID: 1, ResolvedInVersion: ptr.String("1.2.4"),
	}

	// Inserting a vulnerability with no CVE should not insert anything
	didInsert, err := ds.InsertOSVulnerability(ctx, vulnNoCVE, fleet.MSRCSource)
	require.Error(t, err)
	require.False(t, didInsert)

	// Inserting a vulnerability with a CVE should insert
	didInsert, err = ds.InsertOSVulnerability(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.True(t, didInsert)

	// Inserting the same vulnerability should not insert, but update
	didInsert, err = ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	assert.False(t, didInsert)

	// Inserting the exact same vulnerability again may or may not change updated_at, but qualifies as an update
	didInsert, err = ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	assert.False(t, didInsert)

	// simulate vuln in the past to make sure updated_at gets set
	_, err = ds.writer(ctx).ExecContext(ctx, "UPDATE operating_system_vulnerabilities SET updated_at = NOW() - INTERVAL 5 MINUTE WHERE operating_system_id = 1")
	require.NoError(t, err)

	// Inserting the exact same vulnerability again will update again, as we need to bump updated_at
	didInsert, err = ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	assert.False(t, didInsert)

	// make sure the update happened
	var recentRows uint
	require.NoError(t, sqlx.Get(ds.writer(ctx), &recentRows, "SELECT COUNT(*) FROM operating_system_vulnerabilities WHERE operating_system_id = 1 AND updated_at > NOW() - INTERVAL 5 SECOND"))
	require.Equal(t, uint(1), recentRows)

	expected := vulnsUpdate
	expected.Source = fleet.MSRCSource

	list1, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
	require.NoError(t, err)
	require.Len(t, list1, 1)
	require.Equal(t, expected, list1[0])
}

func testDeleteOSVulnerabilitiesEmpty(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1},
		{CVE: "cve-1", OSID: 1},
		{CVE: "cve-3", OSID: 1},
		{CVE: "cve-2", OSID: 1},
	}

	err := ds.DeleteOSVulnerabilities(ctx, vulns)
	require.NoError(t, err)
}

func testDeleteOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1},
		{CVE: "cve-2", OSID: 1},
		{CVE: "cve-3", OSID: 1},
	}

	c, err := ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.Equal(t, int64(3), c)

	toDelete := []fleet.OSVulnerability{
		{CVE: "cve-2", OSID: 1},
	}

	err = ds.DeleteOSVulnerabilities(ctx, toDelete)
	require.NoError(t, err)

	actual, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
	require.NoError(t, err)
	require.ElementsMatch(t, []fleet.OSVulnerability{
		{CVE: "cve-1", OSID: 1, Source: fleet.MSRCSource},
		{CVE: "cve-3", OSID: 1, Source: fleet.MSRCSource},
	}, actual)
}

func testDeleteOutOfDateOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	yesterday := time.Now().Add(-3 * time.Hour).Format("2006-01-02 15:04:05")

	oldVuln := fleet.OSVulnerability{
		CVE: "cve-1", OSID: 1,
	}

	newVuln := fleet.OSVulnerability{
		CVE: "cve-2", OSID: 1,
	}

	_, err := ds.InsertOSVulnerability(ctx, oldVuln, fleet.NVDSource)
	require.NoError(t, err)

	_, err = ds.writer(ctx).ExecContext(ctx, "UPDATE operating_system_vulnerabilities SET updated_at = ?", yesterday)
	require.NoError(t, err)

	_, err = ds.InsertOSVulnerability(ctx, newVuln, fleet.NVDSource)
	require.NoError(t, err)

	// Delete out of date vulns
	err = ds.DeleteOutOfDateOSVulnerabilities(ctx, fleet.NVDSource, time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)

	actual, err := ds.ListOSVulnerabilitiesByOS(ctx, 1)
	require.NoError(t, err)
	require.Len(t, actual, 1)
	require.ElementsMatch(t, []fleet.OSVulnerability{newVuln}, actual)
}

func testListKernelsByOS(t *testing.T, ds *Datastore) {
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
		team                 bool
		host                 *fleet.Host
		software             []fleet.Software
		vulns                []fleet.SoftwareVulnerability
		vulnsByKernelVersion map[string][]string
		os                   fleet.OperatingSystem
	}{
		{
			name:  "ubuntu no team",
			team:  false,
			host:  test.NewHost(t, ds, "host_ubuntu2410", "", "hostkey_ubuntu2410", "hostuuid_ubuntu2410", time.Now(), test.WithPlatform("linux")),
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
			name:     "ubuntu with team",
			team:     true,
			host:     test.NewHost(t, ds, "host_ubuntu2404", "", "hostkey_ubuntu2404", "hostuuid_ubuntu2404", time.Now(), test.WithPlatform("linux")),
			software: software[1:],
			vulns:    []fleet.SoftwareVulnerability{{CVE: "CVE-2025-0004"}, {CVE: "CVE-2025-0005"}, {CVE: "CVE-2025-0003"}}, // Note the overlap; kernel2 has 0003 from the previous test
			vulnsByKernelVersion: map[string][]string{
				kernel2.Version: {"CVE-2025-0004", "CVE-2025-0005", "CVE-2025-0003"},
				kernel3.Version: nil,
			},
			os: fleet.OperatingSystem{Name: "Ubuntu", Version: "24.04", Arch: "x86_64", KernelVersion: "6.11.0-9-generic", Platform: "ubuntu"},
		},
		{
			name:     "amazon linux with team",
			team:     true,
			host:     test.NewHost(t, ds, "host_amzn2023", "", "hostkey_amzn2023", "hostuuid_amzn2023", time.Now(), test.WithPlatform("fedora")),
			software: []fleet.Software{{Name: "kernel", Version: "6.1.144", Arch: "x86_64", Source: "rpm_packages", IsKernel: true}},
			vulns:    []fleet.SoftwareVulnerability{{CVE: "CVE-2025-0006"}},
			vulnsByKernelVersion: map[string][]string{
				"6.1.144": {"CVE-2025-0006"},
			},
			os: fleet.OperatingSystem{Name: "Amazon Linux", Version: "2023.0.0", Arch: "x86_64", KernelVersion: "6.1.144-170.251.amzn2023.x86_64", Platform: "amzn"},
		},
		{
			name:     "RHEL with team",
			team:     true,
			host:     test.NewHost(t, ds, "host_fedora41", "", "hostkey_fedora41", "hostuuid_fedora41", time.Now(), test.WithPlatform("rhel")),
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
			var teamID uint
			if tt.team {
				team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1_" + tt.name})
				require.NoError(t, err)
				require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{tt.host.ID})))
				teamID = team1.ID
			}

			require.NoError(t, ds.UpdateHostOperatingSystem(ctx, tt.host.ID, tt.os))

			os, err := ds.GetHostOperatingSystem(ctx, tt.host.ID)
			require.NoError(t, err)

			_, err = ds.UpdateHostSoftware(ctx, tt.host.ID, tt.software)
			require.NoError(t, err)
			require.NoError(t, ds.LoadHostSoftware(ctx, tt.host, false))

			// Sort the host software by name to enforce a deterministic order
			sort.Slice(tt.host.Software, func(i, j int) bool {
				return tt.host.Software[i].Name < tt.host.Software[j].Name
			})

			softwareIDByVersion := make(map[string]uint)
			for _, s := range tt.host.Software {
				softwareIDByVersion[s.Version] = s.ID
			}

			cpes := []fleet.SoftwareCPE{
				{SoftwareID: tt.host.Software[0].ID, CPE: "somecpe"},
			}
			_, err = ds.UpsertSoftwareCPEs(ctx, cpes)
			require.NoError(t, err)
			require.NoError(t, ds.LoadHostSoftware(ctx, tt.host, false))

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
				_, err = ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
				require.NoError(t, err)
			}
			require.NoError(t, ds.LoadHostSoftware(ctx, tt.host, false))

			require.NoError(t, ds.UpdateOSVersions(ctx))
			require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
			require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))
			require.NoError(t, ds.InsertKernelSoftwareMapping(ctx))

			kernels, err := ds.ListKernelsByOS(ctx, os.OSVersionID, &teamID)
			require.NoError(t, err)

			require.Len(t, kernels, len(tt.software))

			for _, kernel := range kernels {
				expectedVulns, ok := tt.vulnsByKernelVersion[kernel.Version]
				require.True(t, ok)
				require.ElementsMatchf(t, expectedVulns, kernel.Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernel.Version)
				require.Equal(t, kernel.HostsCount, uint(1))
			}

			expectedSet := make(map[string]struct{})
			for _, v := range tt.vulns {
				expectedSet[v.CVE] = struct{}{}
			}

			cves, err := ds.ListVulnsByOsNameAndVersion(ctx, os.Name, os.Version, false, &teamID, nil)
			require.NoError(t, err)
			for _, g := range cves.Vulnerabilities {
				_, ok := expectedSet[g.CVE]
				assert.Truef(t, ok, "got unexpected CVE: %s", g.CVE)
			}

			assert.Len(t, cves.Vulnerabilities, len(tt.vulns))

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, "not_found", false, nil, nil)
			require.NoError(t, err)
			require.Empty(t, cves.Vulnerabilities)

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, os.Version, true, nil, nil)
			require.NoError(t, err)
			require.Len(t, cves.Vulnerabilities, len(tt.vulns))
			for _, g := range cves.Vulnerabilities {
				_, ok := expectedSet[g.CVE]
				assert.True(t, ok)
			}

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, "not_found", true, nil, nil)
			require.NoError(t, err)
			require.Empty(t, cves.Vulnerabilities)
		})
	}
}

func testKernelVulnsHostCount(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	host1 := test.NewHost(t, ds, "host_ubuntu2410", "", "hostkey_ubuntu2410", "hostuuid_ubuntu2410", time.Now(), test.WithPlatform("ubuntu"))
	host2 := test.NewHost(t, ds, "host_ubuntu2404", "", "hostkey_ubuntu2404", "hostuuid_ubuntu2404", time.Now(), test.WithPlatform("ubuntu"))
	host3 := test.NewHost(t, ds, "host_ubuntu2404_2", "", "hostkey_ubuntu2404_2", "hostuuid_ubuntu2404_2", time.Now(), test.WithPlatform("ubuntu"))

	// Same as host 2 and 3, but on a different team
	host4 := test.NewHost(t, ds, "host_ubuntu2404_3", "", "hostkey_ubuntu2404_3", "hostuuid_ubuntu2404_3", time.Now(), test.WithPlatform("ubuntu"))

	os1 := &fleet.OperatingSystem{Name: "Ubuntu", Version: "24.10", Arch: "x86_64", KernelVersion: "6.11.0-9-generic", Platform: "ubuntu"}
	os2 := &fleet.OperatingSystem{Name: "Ubuntu", Version: "24.04", Arch: "x86_64", KernelVersion: "6.11.0-9-generic", Platform: "ubuntu"}

	kernel := fleet.Software{Name: "linux-image-6.11.0-9-generic", Version: "6.11.0-9.9", Source: "deb_packages", IsKernel: true}

	team1, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1_" + t.Name()})
	require.NoError(t, err)

	team2, err := ds.NewTeam(ctx, &fleet.Team{Name: "team2_" + t.Name()})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host1.ID, host2.ID, host3.ID})))
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team2.ID, []uint{host4.ID})))

	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host1.ID, *os1))
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host2.ID, *os2))
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host3.ID, *os2))
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host4.ID, *os2))

	os1, err = ds.GetHostOperatingSystem(ctx, host1.ID)
	require.NoError(t, err)

	os2, err = ds.GetHostOperatingSystem(ctx, host2.ID)
	require.NoError(t, err)

	addKernelToHost := func(h *fleet.Host) {
		var vulnsToInsert []fleet.SoftwareVulnerability
		_, err = ds.UpdateHostSoftware(ctx, h.ID, []fleet.Software{kernel})
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(ctx, h, false))

		_, err = ds.UpsertSoftwareCPEs(ctx, []fleet.SoftwareCPE{{SoftwareID: h.Software[0].ID, CPE: "somecpe"}})
		require.NoError(t, err)

		for _, cve := range []string{"CVE-2025-0001", "CVE-2025-0002"} {
			vulnsToInsert = append(vulnsToInsert, fleet.SoftwareVulnerability{
				SoftwareID: h.Software[0].ID,
				CVE:        cve,
			})
		}

		for _, v := range vulnsToInsert {
			_, err = ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
			require.NoError(t, err)
		}
	}

	for _, h := range []*fleet.Host{host1, host2, host3, host4} {
		addKernelToHost(h)
	}

	for _, h := range []*fleet.Host{host1, host2, host3, host4} {
		require.NoError(t, ds.LoadHostSoftware(ctx, h, false))
	}

	updateMappings := func() {
		require.NoError(t, ds.UpdateOSVersions(ctx))
		require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
		require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))
		require.NoError(t, ds.InsertKernelSoftwareMapping(ctx))
	}

	updateMappings()

	expectedCVEs := []string{"CVE-2025-0001", "CVE-2025-0002"}

	kernels, err := ds.ListKernelsByOS(ctx, os1.OSVersionID, &team1.ID)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(1), kernels[0].HostsCount) // host1

	kernels, err = ds.ListKernelsByOS(ctx, os2.OSVersionID, &team1.ID)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	require.Equal(t, uint(2), kernels[0].HostsCount) // host2, host3

	kernels, err = ds.ListKernelsByOS(ctx, os2.OSVersionID, &team2.ID)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(1), kernels[0].HostsCount) // host4

	// "All teams" (aka team ID is nil)
	// For os2, should be 3 since it's on host2, host3, and host4
	kernels, err = ds.ListKernelsByOS(ctx, os2.OSVersionID, nil)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(3), kernels[0].HostsCount)

	// For os1, should be 1 since it's on host1
	kernels, err = ds.ListKernelsByOS(ctx, os1.OSVersionID, nil)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(1), kernels[0].HostsCount)

	// Add another host to team1, counts should update
	host5 := test.NewHost(t, ds, "host_ubuntu2404_4", "", "hostkey_ubuntu2404_4", "hostuuid_ubuntu2404_4", time.Now(), test.WithPlatform("ubuntu"))
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team1.ID, []uint{host5.ID})))
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host5.ID, *os2))
	addKernelToHost(host5)

	updateMappings()

	kernels, err = ds.ListKernelsByOS(ctx, os2.OSVersionID, &team1.ID)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(3), kernels[0].HostsCount) // host2, host3, host5

	// "All teams" (aka team ID is nil)
	// For os2, should be 4 since it's on host2, host3, host4, and now host5
	kernels, err = ds.ListKernelsByOS(ctx, os2.OSVersionID, nil)
	require.NoError(t, err)
	require.Len(t, kernels, 1)
	assert.ElementsMatchf(t, expectedCVEs, kernels[0].Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernels[0].Version)
	assert.Equal(t, uint(4), kernels[0].HostsCount)

	// Delete host 1. We should see the count for the kernel go down to 0.
	require.NoError(t, ds.DeleteHost(ctx, host1.ID))

	updateMappings()

	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		var count uint
		err := sqlx.GetContext(ctx, q, &count, "SELECT hosts_count FROM kernel_host_counts WHERE os_version_id = ?", os1.OSVersionID)
		require.NoError(t, err)
		assert.Zero(t, count)
		return nil
	})

	kernels, err = ds.ListKernelsByOS(ctx, os1.OSVersionID, nil)
	require.NoError(t, err)
	require.Empty(t, kernels)
}

func testRefreshOSVersionVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create test host and software
	host := test.NewHost(t, ds, "test-host", "", "testkey", "testuuid", time.Now(), test.WithPlatform("ubuntu"))
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))

	os := fleet.OperatingSystem{
		Name:          "Ubuntu",
		Version:       "22.04",
		Arch:          "x86_64",
		KernelVersion: "5.15.0-1",
		Platform:      "ubuntu",
	}
	require.NoError(t, ds.UpdateHostOperatingSystem(ctx, host.ID, os))

	osRecord, err := ds.GetHostOperatingSystem(ctx, host.ID)
	require.NoError(t, err)

	// Create Linux kernel software with vulnerabilities
	kernel := fleet.Software{
		Name:     "linux",
		Version:  "5.15.0-1",
		Source:   "programs",
		IsKernel: true,
	}
	_, err = ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{kernel})
	require.NoError(t, err)
	require.NoError(t, ds.LoadHostSoftware(ctx, host, false))

	// Add CVEs to the kernel
	vulns := []fleet.SoftwareVulnerability{
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2024-0001"},
		{SoftwareID: host.Software[0].ID, CVE: "CVE-2024-0002"},
	}
	for _, v := range vulns {
		_, err = ds.InsertSoftwareVulnerability(ctx, v, fleet.NVDSource)
		require.NoError(t, err)
	}

	// Update mappings to populate kernel_host_counts
	require.NoError(t, ds.UpdateOSVersions(ctx))
	require.NoError(t, ds.SyncHostsSoftware(ctx, time.Now()))
	require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))
	require.NoError(t, ds.InsertKernelSoftwareMapping(ctx))

	t.Run("populates per-team vulnerabilities", func(t *testing.T) {
		// Query per-team vulnerabilities
		type vuln struct {
			OSVersionID uint   `db:"os_version_id"`
			CVE         string `db:"cve"`
			TeamID      *uint  `db:"team_id"`
		}

		var teamVulns []vuln
		err := ds.writer(ctx).SelectContext(ctx, &teamVulns, `
			SELECT os_version_id, cve, team_id
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id = ? AND team_id IS NOT NULL
			ORDER BY cve
		`, osRecord.OSVersionID)
		require.NoError(t, err)
		require.Len(t, teamVulns, 2, "Should have 2 per-team Linux kernel vulnerabilities")
		require.Equal(t, osRecord.OSVersionID, teamVulns[0].OSVersionID)
		require.Equal(t, "CVE-2024-0001", teamVulns[0].CVE)
		require.NotNil(t, teamVulns[0].TeamID)
		require.Equal(t, team.ID, *teamVulns[0].TeamID)
		require.Equal(t, "CVE-2024-0002", teamVulns[1].CVE)
	})

	t.Run("populates all-teams aggregated vulnerabilities", func(t *testing.T) {
		// Query "all teams" vulnerabilities (team_id = NULL)
		type vuln struct {
			OSVersionID uint   `db:"os_version_id"`
			CVE         string `db:"cve"`
			TeamID      *uint  `db:"team_id"`
		}

		var allTeamsVulns []vuln
		err := ds.writer(ctx).SelectContext(ctx, &allTeamsVulns, `
			SELECT os_version_id, cve, team_id
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id = ? AND team_id IS NULL
			ORDER BY cve
		`, osRecord.OSVersionID)
		require.NoError(t, err)
		require.Len(t, allTeamsVulns, 2, "Should have 2 'all teams' aggregated vulnerabilities")
		require.Equal(t, osRecord.OSVersionID, allTeamsVulns[0].OSVersionID)
		require.Equal(t, "CVE-2024-0001", allTeamsVulns[0].CVE)
		require.Nil(t, allTeamsVulns[0].TeamID, "team_id should be NULL for 'all teams'")
		require.Equal(t, "CVE-2024-0002", allTeamsVulns[1].CVE)
	})

	t.Run("updates existing vulnerabilities on refresh", func(t *testing.T) {
		// Manually update resolved_in_version to verify UPDATE works
		_, err := ds.writer(ctx).ExecContext(ctx, `
			UPDATE operating_system_version_vulnerabilities
			SET resolved_in_version = '5.15.0-old'
			WHERE os_version_id = ? AND cve = 'CVE-2024-0001'
		`, osRecord.OSVersionID)
		require.NoError(t, err)

		// Update software_cve with new resolved_in_version
		_, err = ds.writer(ctx).ExecContext(ctx, `
			UPDATE software_cve
			SET resolved_in_version = '5.15.0-new'
			WHERE software_id = ? AND cve = 'CVE-2024-0001'
		`, host.Software[0].ID)
		require.NoError(t, err)

		// Refresh should update the value
		require.NoError(t, ds.refreshOSVersionVulnerabilities(ctx))

		// Verify the update
		var resolvedVersion string
		err = ds.writer(ctx).GetContext(ctx, &resolvedVersion, `
			SELECT resolved_in_version
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id = ? AND cve = 'CVE-2024-0001' AND team_id IS NOT NULL
			LIMIT 1
		`, osRecord.OSVersionID)
		require.NoError(t, err)
		require.Equal(t, "5.15.0-new", resolvedVersion)
	})

	t.Run("cleans up stale entries on refresh", func(t *testing.T) {
		// Verify we have vulnerabilities before cleanup
		var count int
		err := ds.writer(ctx).GetContext(ctx, &count, `
			SELECT COUNT(*)
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id = ?
		`, osRecord.OSVersionID)
		require.NoError(t, err)
		require.Greater(t, count, 0, "Should have vulnerabilities before cleanup")

		// Clear kernel_host_counts to simulate no active hosts with this kernel
		_, err = ds.writer(ctx).ExecContext(ctx, `TRUNCATE TABLE kernel_host_counts`)
		require.NoError(t, err)

		// Refresh should not error and should clean up all entries for this OS version
		require.NoError(t, ds.refreshOSVersionVulnerabilities(ctx))

		// Should have no vulnerabilities now (automatically cleaned up since they're stale)
		err = ds.writer(ctx).GetContext(ctx, &count, `
			SELECT COUNT(*)
			FROM operating_system_version_vulnerabilities
			WHERE os_version_id = ?
		`, osRecord.OSVersionID)
		require.NoError(t, err)
		require.Equal(t, 0, count, "All stale vulnerabilities should be cleaned up by refresh")
	})
}
