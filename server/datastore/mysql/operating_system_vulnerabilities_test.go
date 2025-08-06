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

	cves, err := ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", false)
	require.NoError(t, err)
	require.Empty(t, cves)

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
	cves, err = ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", false)
	require.NoError(t, err)

	expected := []string{"CVE-2021-1234", "CVE-2021-1235"}
	require.Len(t, cves, 2)
	for _, cve := range cves {
		require.Contains(t, expected, cve.CVE)
		require.Greater(t, cve.CreatedAt, time.Now().Add(-time.Hour)) // assert non-zero time
	}

	// test with CVS meta
	cves, err = ds.ListVulnsByOsNameAndVersion(ctx, "Microsoft Windows 11 Pro 21H2", "10.0.22000.795", true)
	require.NoError(t, err)
	require.Len(t, cves, 2)

	require.Equal(t, cveMeta[0].CVE, cves[0].CVE)
	require.Equal(t, &cveMeta[0].CVSSScore, cves[0].CVSSScore)
	require.Equal(t, &cveMeta[0].EPSSProbability, cves[0].EPSSProbability)
	require.Equal(t, &cveMeta[0].CISAKnownExploit, cves[0].CISAKnownExploit)
	require.Equal(t, cveMeta[0].Published, *cves[0].CVEPublished)
	require.Equal(t, cveMeta[0].Description, **cves[0].Description)
	require.Equal(t, cveMeta[1].CVE, cves[1].CVE)
	require.Equal(t, &cveMeta[1].CVSSScore, cves[1].CVSSScore)
	require.Equal(t, &cveMeta[1].EPSSProbability, cves[1].EPSSProbability)
	require.Equal(t, &cveMeta[1].CISAKnownExploit, cves[1].CISAKnownExploit)
	require.Equal(t, cveMeta[1].Published, *cves[1].CVEPublished)
	require.Equal(t, cveMeta[1].Description, **cves[1].Description)
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
	err = ds.DeleteOutOfDateOSVulnerabilities(ctx, fleet.NVDSource, 2*time.Hour)
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
			require.NoError(t, ds.ReconcileSoftwareTitles(ctx))
			require.NoError(t, ds.SyncHostsSoftwareTitles(ctx, time.Now()))

			if !tt.team {
				// Insert some fake counts, this should be ignored in ListKernelsByOS
				ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, "INSERT IGNORE INTO software_host_counts (software_id, hosts_count, team_id, global_stats) VALUES (?, ?, ?, ?)", tt.host.Software[0].ID, 999, 0, true)
					return err
				})
			}
			kernels, err := ds.ListKernelsByOS(ctx, os.OSVersionID, &teamID)
			require.NoError(t, err)

			require.Len(t, kernels, len(tt.software))

			for _, kernel := range kernels {
				expectedVulns, ok := tt.vulnsByKernelVersion[kernel.Version]
				require.True(t, ok)
				require.ElementsMatchf(t, expectedVulns, kernel.Vulnerabilities, "unexpected vulnerabilities for kernel %s", kernel.Version)
				require.Equal(t, kernel.HostsCount, uint(1))
			}

			cves, err := ds.ListVulnsByOsNameAndVersion(ctx, os.Name, os.Version, false)
			require.NoError(t, err)
			require.Len(t, cves, len(tt.vulns))

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, "not_found", false)
			require.NoError(t, err)
			require.Empty(t, cves)

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, os.Version, true)
			require.NoError(t, err)
			require.Len(t, cves, len(tt.vulns))

			cves, err = ds.ListVulnsByOsNameAndVersion(ctx, os.Name, "not_found", true)
			require.NoError(t, err)
			require.Empty(t, cves)

		})
	}
}
