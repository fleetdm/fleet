package mysql

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestVulnerabilities(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestListVulnerabilities", testListVulnerabilities},
		{"TestVulnerabilityWithOS", testVulnerabilityWithOS},
		{"TestVulnerabilityWithSoftware", testVulnerabilityWithSoftware},
		{"TestOSVersionsByCVE", testOSVersionsByCVE},
		{"TestSoftwareByCVE", testSoftwareByCVE},
		{"TestVulnerabilitiesPagination", testVulnerabilitiesPagination},
		{"TestVulnerabilitiesTeamFilter", testVulnerabilitiesTeamFilter},
		{"TestListVulnerabilitiesSort", testListVulnerabilitiesSort},
		{"TestVulnerabilitiesFilters", testVulnerabilitiesFilters},
		{"TestCountVulnerabilities", testCountVulnerabilities},
		{"TestInsertVulnerabilityCounts", testInsertVulnerabilityCounts},
		{"TestVulnerabilityHostCountBatchInserts", testVulnerabilityHostCountBatchInserts},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer TruncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListVulnerabilities(t *testing.T, ds *Datastore) {
	mockTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	opts := fleet.VulnListOptions{}
	list, _, err := ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Empty(t, list)

	// Insert Host Count
	insertStmt := `
		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES (?, ?, ?)
	`
	_, err = ds.writer(context.Background()).Exec(insertStmt, "CVE-2020-1234", 0, 10)
	require.NoError(t, err)
	_, err = ds.writer(context.Background()).Exec(insertStmt, "CVE-2020-1235", 0, 15)
	require.NoError(t, err)
	_, err = ds.writer(context.Background()).Exec(insertStmt, "CVE-2020-1236", 0, 20)
	require.NoError(t, err)

	list, _, err = ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 3)

	// insert OS Vuln
	_, err = ds.InsertOSVulnerabilities(context.Background(), []fleet.OSVulnerability{
		{
			OSID:              1,
			CVE:               "CVE-2020-1234",
			ResolvedInVersion: ptr.String("1.0.0"),
		},
		{
			OSID: 1,
			CVE:  "CVE-2020-1235",
		},
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert Software Vuln
	_, err = ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: 1,
		CVE:        "CVE-2020-1236",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	err = ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
	})
	require.NoError(t, err)

	expected := map[string]fleet.VulnerabilityWithMetadata{
		"CVE-2020-1234": {
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1234",
				CVSSScore:        ptr.Float64(7.5),
				EPSSProbability:  ptr.Float64(0.5),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1234",
			},
			HostsCount: 10,
			Source:     fleet.MSRCSource,
		},
		"CVE-2020-1235": {
			CVEMeta:    fleet.CVEMeta{CVE: "CVE-2020-1235"},
			HostsCount: 15,
			Source:     fleet.MSRCSource,
		},
		"CVE-2020-1236": {
			CVEMeta:    fleet.CVEMeta{CVE: "CVE-2020-1236"},
			HostsCount: 20,
			Source:     fleet.NVDSource,
		},
	}
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{IsEE: true})
	require.NoError(t, err)
	require.Len(t, list, 3)
	for _, vuln := range list {
		expectedVuln, ok := expected[vuln.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.CVEMeta, vuln.CVEMeta)
		require.Equal(t, expectedVuln.HostsCount, vuln.HostsCount)
	}

	// Test Fleet Free
	expected = map[string]fleet.VulnerabilityWithMetadata{
		"CVE-2020-1234": {
			CVEMeta:    fleet.CVEMeta{CVE: "CVE-2020-1234"},
			HostsCount: 10,
			Source:     fleet.MSRCSource,
		},
		"CVE-2020-1235": {
			CVEMeta:    fleet.CVEMeta{CVE: "CVE-2020-1235"},
			HostsCount: 15,
			Source:     fleet.MSRCSource,
		},
		"CVE-2020-1236": {
			CVEMeta:    fleet.CVEMeta{CVE: "CVE-2020-1236"},
			HostsCount: 20,
			Source:     fleet.NVDSource,
		},
	}
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Len(t, list, 3)
	for _, vuln := range list {
		expectedVuln, ok := expected[vuln.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.CVEMeta, vuln.CVEMeta)
		require.Equal(t, expectedVuln.HostsCount, vuln.HostsCount)
	}
}

func testVulnerabilityWithOS(t *testing.T, ds *Datastore) {
	mockTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	v, err := ds.Vulnerability(ctx, "CVE-2020-1234", nil, false)
	require.Nil(t, v)
	require.Error(t, err)
	var nfe *notFoundError
	require.ErrorAs(t, err, &nfe)

	// Insert Host Count
	insertStmt := `
		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES (?, ?, ?), (?, ?, ?)
	`
	_, err = ds.writer(context.Background()).Exec(insertStmt,
		"CVE-2020-1234", 0, 10,
		"CVE-2020-1234", 1, 4,
	)
	require.NoError(t, err)

	// // insert OS Vuln
	_, err = ds.InsertOSVulnerabilities(context.Background(), []fleet.OSVulnerability{
		{
			OSID:              1,
			CVE:               "CVE-2020-1234",
			ResolvedInVersion: ptr.String("1.0.0"),
		},
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// // insert CVEMeta
	err = ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
	})
	require.NoError(t, err)

	expected := fleet.VulnerabilityWithMetadata{
		CVEMeta: fleet.CVEMeta{
			CVE: "CVE-2020-1234",
		},
		HostsCount: 10,
		Source:     fleet.MSRCSource,
	}

	// No CVSSScores
	v, err = ds.Vulnerability(ctx, "CVE-2020-1234", nil, false)
	require.NoError(t, err)
	require.Equal(t, expected.CVEMeta, v.CVEMeta)
	require.Equal(t, expected.HostsCount, v.HostsCount)
	require.Equal(t, expected.Source, v.Source)

	// Team 1
	expected.HostsCount = 4
	v, err = ds.Vulnerability(ctx, "CVE-2020-1234", ptr.Uint(1), false)
	require.NoError(t, err)
	require.Equal(t, expected.CVEMeta, v.CVEMeta)
	require.Equal(t, expected.HostsCount, v.HostsCount)
	require.Equal(t, expected.Source, v.Source)

	expected = fleet.VulnerabilityWithMetadata{
		CVEMeta: fleet.CVEMeta{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
		HostsCount: 10,
		Source:     fleet.MSRCSource,
	}

	// With CVSSScores
	v, err = ds.Vulnerability(ctx, "CVE-2020-1234", nil, true)
	require.NoError(t, err)
	require.Equal(t, expected.CVEMeta, v.CVEMeta)
	require.Equal(t, expected.HostsCount, v.HostsCount)
	require.Equal(t, expected.Source, v.Source)
}

func testVulnerabilityWithSoftware(t *testing.T, ds *Datastore) {
	mockTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx := context.Background()

	v, err := ds.Vulnerability(ctx, "CVE-2020-1234", nil, false)
	require.Nil(t, v)
	require.Error(t, err)
	var nfe *notFoundError
	require.ErrorAs(t, err, &nfe)

	// Insert Host Count
	insertStmt := `
		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES (?, ?, ?)
	`

	_, err = ds.writer(context.Background()).Exec(insertStmt, "CVE-2020-1234", 0, 10)
	require.NoError(t, err)

	// insert Software Vuln
	_, err = ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: 1,
		CVE:        "CVE-2020-1234",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	err = ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
	})
	require.NoError(t, err)

	// No CVSSScores
	expected := fleet.VulnerabilityWithMetadata{
		CVEMeta: fleet.CVEMeta{
			CVE: "CVE-2020-1234",
		},
		HostsCount: 10,
		Source:     fleet.NVDSource,
	}

	v, err = ds.Vulnerability(ctx, "CVE-2020-1234", nil, false)
	require.NoError(t, err)
	require.Equal(t, expected.CVEMeta, v.CVEMeta)
	require.Equal(t, expected.HostsCount, v.HostsCount)
	require.Equal(t, expected.Source, v.Source)

	// With CVSSScores
	expected = fleet.VulnerabilityWithMetadata{
		CVEMeta: fleet.CVEMeta{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
		HostsCount: 10,
		Source:     fleet.NVDSource,
	}

	v, err = ds.Vulnerability(ctx, "CVE-2020-1234", nil, true)
	require.NoError(t, err)
	require.Equal(t, expected.CVEMeta, v.CVEMeta)
	require.Equal(t, expected.HostsCount, v.HostsCount)
	require.Equal(t, expected.Source, v.Source)
}

func testVulnerabilitiesPagination(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	opts := fleet.VulnListOptions{
		ListOptions: fleet.ListOptions{
			Page:    0,
			PerPage: 5,
		},
	}

	list, meta, err := ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 5)
	require.NotNil(t, meta)
	require.False(t, meta.HasPreviousResults)
	require.True(t, meta.HasNextResults)

	opts.Page = 1
	list, meta, err = ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.NotNil(t, meta)
	require.True(t, meta.HasPreviousResults)
	require.False(t, meta.HasNextResults)
}

func testVulnerabilitiesTeamFilter(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	opts := fleet.VulnListOptions{
		TeamID: 1,
	}

	list, _, err := ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 6)

	checkCounts := map[string]int{
		"CVE-2020-1234": 20,
		"CVE-2020-1235": 19,
		"CVE-2020-1236": 18,
		"CVE-2020-1238": 16,
		"CVE-2020-1239": 15,
		// No team host counts for CVE-2020-1240
		"CVE-2020-1241": 14,
	}

	for _, vuln := range list {
		require.Equal(t, checkCounts[vuln.CVE], int(vuln.HostsCount), vuln.CVE)
	}
}

func testListVulnerabilitiesSort(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	opts := fleet.VulnListOptions{
		IsEE: true,
		ListOptions: fleet.ListOptions{
			Page:           0,
			PerPage:        5,
			OrderKey:       "cve",
			OrderDirection: fleet.OrderDescending,
		},
	}

	list, _, err := ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 5)
	require.Equal(t, "CVE-2020-1241", list[0].CVE)
	require.Equal(t, "CVE-2020-1239", list[1].CVE)
	require.Equal(t, "CVE-2020-1238", list[2].CVE)
	require.Equal(t, "CVE-2020-1237", list[3].CVE)
	require.Equal(t, "CVE-2020-1236", list[4].CVE)

	opts.OrderKey = "published"
	opts.OrderDirection = fleet.OrderAscending
	list, _, err = ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 5)
	require.Equal(t, "CVE-2020-1241", list[0].CVE) // NULL dates are sorted first
	require.Equal(t, "CVE-2020-1234", list[1].CVE)
	require.Equal(t, "CVE-2020-1236", list[2].CVE)
	require.Equal(t, "CVE-2020-1235", list[3].CVE)
	require.Equal(t, "CVE-2020-1237", list[4].CVE)
}

func testVulnerabilitiesFilters(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	// Test KnownExploit filter
	opts := fleet.VulnListOptions{
		IsEE:         true,
		KnownExploit: true,
	}
	list, _, err := ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)

	require.Len(t, list, 3)
	expected := []string{"CVE-2020-1234", "CVE-2020-1236", "CVE-2020-1238"}
	for _, vuln := range list {
		require.Contains(t, expected, vuln.CVE)
	}

	// Test CVE LIKE filter
	opts = fleet.VulnListOptions{
		ListOptions: fleet.ListOptions{
			MatchQuery: "2020-1234",
		},
	}
	list, _, err = ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "CVE-2020-1234", list[0].CVE)
}

func testCountVulnerabilities(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	// global count
	count, err := ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Equal(t, uint(7), count)

	// global count with exploit filter
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{KnownExploit: true})
	require.NoError(t, err)
	require.Equal(t, uint(3), count)

	// global count with match query
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{ListOptions: fleet.ListOptions{MatchQuery: "2020-1234"}})
	require.NoError(t, err)
	require.Equal(t, uint(1), count)

	// team count
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: 1})
	require.NoError(t, err)
	require.Equal(t, uint(6), count)

	// team count with exploit filter
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: 1, KnownExploit: true})
	require.NoError(t, err)
	require.Equal(t, uint(3), count)

	// team count with match query
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: 1, ListOptions: fleet.ListOptions{MatchQuery: "2020-1234"}})
	require.NoError(t, err)
	require.Equal(t, uint(1), count)

	// team count with exploit filter and match query
	count, err = ds.CountVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: 1, KnownExploit: true, ListOptions: fleet.ListOptions{MatchQuery: "2020-1234"}})
	require.NoError(t, err)
	require.Equal(t, uint(1), count)
}

func testInsertVulnerabilityCounts(t *testing.T, ds *Datastore) {
	windowsOS := fleet.OperatingSystem{
		Name:     "Windows 11 Pro",
		Version:  "10.0.22000.3007",
		Arch:     "x86_64",
		Platform: "windows",
	}

	macOS := fleet.OperatingSystem{
		Name:     "macOS",
		Version:  "14.1.2",
		Arch:     "arm64",
		Platform: "darwin",
	}

	// create Windows host1
	host1 := test.NewHost(t, ds, "host1", "192.168.0.1", "1111", "1111", time.Now())

	err := ds.UpdateHostOperatingSystem(context.Background(), host1.ID, windowsOS)
	require.NoError(t, err)

	// assert no vulns
	list, _, err := ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Empty(t, list)

	// insert Windows OS vulnerability
	_, err = ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: 1,
		CVE:  "CVE-2020-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected := []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 1},
	}
	assertHostCounts(t, globalExpected, list)

	// add host 2 with same OS
	host2 := test.NewHost(t, ds, "host2", "192.168.0.2", "2222", "2222", time.Now())
	err = ds.UpdateHostOperatingSystem(context.Background(), host2.ID, windowsOS)
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2},
	}
	assertHostCounts(t, globalExpected, list)

	// add 1 macOS host
	host3 := test.NewHost(t, ds, "host3", "192.168.0.3", "3333", "3333", time.Now())
	err = ds.UpdateHostOperatingSystem(context.Background(), host3.ID, macOS)
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// assert no new vulns
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	assertHostCounts(t, globalExpected, list)

	// add macos vulnerability
	_, err = ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: 2,
		CVE:  "CVE-2020-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2}, // windows vuln
		{CVE: "CVE-2020-1235", HostCount: 1}, // macos vuln
	}
	assertHostCounts(t, globalExpected, list)

	// add software vuln to host 1
	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, []fleet.Software{
		{
			Name:    "Chrome",
			Version: "1.0.0",
		},
	})
	require.NoError(t, err)
	_, err = ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: 1,
		CVE:        "CVE-2020-1236",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2}, // windows vuln
		{CVE: "CVE-2020-1235", HostCount: 1}, // macos vuln
		{CVE: "CVE-2020-1236", HostCount: 1}, // software vuln
	}
	assertHostCounts(t, globalExpected, list)

	// move host 1 to team 1
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID})
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// global counts should not change
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	assertHostCounts(t, globalExpected, list)

	// assert team counts
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team1.ID})
	require.NoError(t, err)
	team1expected := []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 1}, // windows vuln
		{CVE: "CVE-2020-1236", HostCount: 1}, // software vuln
	}
	assertHostCounts(t, team1expected, list)

	// add 5 macos hosts (4-9) to team2
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	for i := 4; i < 9; i++ {
		host := test.NewHost(t, ds, fmt.Sprintf("host%d", i+4), fmt.Sprintf("192.168.0.%d", i+4), fmt.Sprintf("%d", i+4444), fmt.Sprintf("%d", i+4444), time.Now())
		err = ds.UpdateHostOperatingSystem(context.Background(), host.ID, macOS)
		require.NoError(t, err)
		err = ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{host.ID})
		require.NoError(t, err)
	}

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// global counts should not change
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2},
		{CVE: "CVE-2020-1235", HostCount: 6}, // + 5 macos hosts
		{CVE: "CVE-2020-1236", HostCount: 1},
	}
	assertHostCounts(t, globalExpected, list)

	// team1 counts should not change
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team1.ID})
	require.NoError(t, err)
	assertHostCounts(t, team1expected, list)

	// team2 counts
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team2.ID})
	require.NoError(t, err)
	team2expected := []hostCount{
		{CVE: "CVE-2020-1235", HostCount: 5}, // macos vuln
	}
	assertHostCounts(t, team2expected, list)

	// patch team2 hosts
	macOSPatched := fleet.OperatingSystem{
		Name:     "macOS",
		Version:  "14.2",
		Arch:     "arm64",
		Platform: "darwin",
	}
	for i := 4; i < 9; i++ {
		err = ds.UpdateHostOperatingSystem(context.Background(), uint(i), macOSPatched)
		require.NoError(t, err)
	}

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// no change to team1 counts
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team1.ID})
	require.NoError(t, err)
	assertHostCounts(t, team1expected, list)

	// no vulns in team2
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team2.ID})
	require.NoError(t, err)
	require.Len(t, list, 0)

	// global counts reduced
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2},
		{CVE: "CVE-2020-1235", HostCount: 1}, // -5 macos hosts
		{CVE: "CVE-2020-1236", HostCount: 1},
	}
	assertHostCounts(t, globalExpected, list)

	// patch software vuln
	_, err = ds.UpdateHostSoftware(context.Background(), host1.ID, []fleet.Software{})
	require.NoError(t, err)

	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// global counts reduced
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	globalExpected = []hostCount{
		{CVE: "CVE-2020-1234", HostCount: 2},
		{CVE: "CVE-2020-1235", HostCount: 1},
		// CVE-2020-1236 removed
	}
	assertHostCounts(t, globalExpected, list)
}

// testVulnerabilityHostCountBatchInserts tests the ability to insert a large
// number of vulnerabilities in a single batch insert
// to keep this test fast, we only insert 5 hosts
func testVulnerabilityHostCountBatchInserts(t *testing.T, ds *Datastore) {
	// create 5 hosts
	hosts := make([]*fleet.Host, 5)
	for i := 0; i < 5; i++ {
		hosts[i] = test.NewHost(t, ds, fmt.Sprintf("host%d", i), fmt.Sprintf("192.168.0.%d", i), fmt.Sprintf("%d", i+1000), fmt.Sprintf("%d", i+1000), time.Now())
	}

	// add 2 hosts to team 1
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		err = ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[i].ID})
		require.NoError(t, err)
	}

	// create 200 OS vulns
	osVulns := make([]fleet.OSVulnerability, 200)
	for i := 0; i < 200; i++ {
		osVulns[i] = fleet.OSVulnerability{
			OSID: 1,
			CVE:  fmt.Sprintf("CVE-2020-%d", i),
		}
	}

	// create 200 software vulns
	softwareVulns := make([]fleet.SoftwareVulnerability, 200)
	for i := 0; i < 200; i++ {
		softwareVulns[i] = fleet.SoftwareVulnerability{
			SoftwareID: 1,
			CVE:        fmt.Sprintf("CVE-2021-%d", i),
		}
	}

	// insert OS vulns
	_, err = ds.InsertOSVulnerabilities(context.Background(), osVulns, fleet.NVDSource)
	require.NoError(t, err)

	// insert software vulns
	for _, vuln := range softwareVulns {
		_, err = ds.InsertSoftwareVulnerability(context.Background(), vuln, fleet.NVDSource)
		require.NoError(t, err)
	}

	// update host OS
	for i := 0; i < 5; i++ {
		err = ds.UpdateHostOperatingSystem(context.Background(), hosts[i].ID, fleet.OperatingSystem{
			Name:     "Windows 11 Pro",
			Version:  "10.0.22000.3007",
			Arch:     "x86_64",
			Platform: "windows",
		})
		require.NoError(t, err)
	}

	// update host software
	for i := 0; i < 5; i++ {
		_, err = ds.UpdateHostSoftware(context.Background(), hosts[i].ID, []fleet.Software{
			{
				Name:    "Chrome",
				Version: "1.0.0",
			},
		})
		require.NoError(t, err)
	}

	// update host counts
	err = ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// assert host counts
	list, _, err := ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{})
	require.NoError(t, err)
	require.Len(t, list, 400)
	for _, vuln := range list {
		require.Equal(t, uint(5), vuln.HostsCount)
	}

	// assert team counts
	list, _, err = ds.ListVulnerabilities(context.Background(), fleet.VulnListOptions{TeamID: team1.ID})
	require.NoError(t, err)
	require.Len(t, list, 400)
	for _, vuln := range list {
		require.Equal(t, uint(2), vuln.HostsCount)
	}
}

func testOSVersionsByCVE(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	// global
	osv, _, err := ds.OSVersionsByCVE(context.Background(), "CVE-2020-1238", nil)
	require.NoError(t, err)

	expected := []fleet.VulnerableOS{
		{
			OSVersion: fleet.OSVersion{
				Name:        "Microsoft Windows 11 Enterprise 22H2 10.0.22621.2715",
				NameOnly:    "Microsoft Windows 11 Enterprise 22H2",
				OSVersionID: 1,
				Version:     "10.0.22621.2715",
				Platform:    "windows",
				HostsCount:  10,
			},
			ResolvedInVersion: ptr.String("1.0.0"),
		},
	}

	require.Len(t, osv, 1)
	require.Equal(t, osv[0].OSVersion, expected[0].OSVersion)

	// team 1
	expected[0].OSVersion.HostsCount = 4
	osv, _, err = ds.OSVersionsByCVE(context.Background(), "CVE-2020-1238", ptr.Uint(1))
	require.NoError(t, err)
	require.Len(t, osv, 1)
	require.Equal(t, osv[0].OSVersion, expected[0].OSVersion)

	// team 2
	expected[0].OSVersion.HostsCount = 3
	osv, _, err = ds.OSVersionsByCVE(context.Background(), "CVE-2020-1238", ptr.Uint(2))
	require.NoError(t, err)
	require.Len(t, osv, 1)
	require.Equal(t, osv[0].OSVersion, expected[0].OSVersion)
}

func testSoftwareByCVE(t *testing.T, ds *Datastore) {
	seedVulnerabilities(t, ds)

	// global
	software, _, err := ds.SoftwareByCVE(context.Background(), "CVE-2020-1234", nil)
	require.NoError(t, err)

	expected := &fleet.VulnerableSoftware{
		ID:                1,
		Name:              "Chrome",
		Version:           "1.0.0",
		Source:            "programs",
		HostsCount:        5,
		GenerateCPE:       "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*",
		ResolvedInVersion: ptr.String("1.0.0"),
	}

	require.Len(t, software, 1)
	require.Equal(t, expected, software[0])

	// team 1
	expected.HostsCount = 4
	software, _, err = ds.SoftwareByCVE(context.Background(), "CVE-2020-1234", ptr.Uint(1))
	require.NoError(t, err)
	require.Len(t, software, 1)
	require.Equal(t, expected, software[0])

	// team 2
	expected.HostsCount = 1
	software, _, err = ds.SoftwareByCVE(context.Background(), "CVE-2020-1234", ptr.Uint(2))
	require.NoError(t, err)
	require.Len(t, software, 1)
	require.Equal(t, expected, software[0])
}

func assertHostCounts(t *testing.T, expected []hostCount, actual []fleet.VulnerabilityWithMetadata) {
	t.Helper()
	require.Len(t, actual, len(expected))
	for i, vuln := range actual {
		require.Equal(t, expected[i].CVE, vuln.CVE)
		require.Equal(t, expected[i].HostCount, vuln.HostsCount)
	}
}

func seedVulnerabilities(t *testing.T, ds *Datastore) {
	// insert 20 hosts
	var hostids []uint
	for i := 0; i < 20; i++ {
		host := test.NewHost(t, ds, fmt.Sprintf("host%d", i), fmt.Sprintf("192.168.0.%d", i), fmt.Sprintf("%d", i+1000), fmt.Sprintf("%d", i+1000), time.Now())
		hostids = append(hostids, host.ID)
	}

	// update 15 hosts to windows
	for i := 0; i < 10; i++ {
		err := ds.UpdateHostOperatingSystem(context.Background(), hostids[i], fleet.OperatingSystem{
			Name:     "Microsoft Windows 11 Enterprise 22H2",
			Version:  "10.0.22621.2715",
			Arch:     "x86_64",
			Platform: "windows",
		})
		require.NoError(t, err)
	}

	// update 5 hosts to macOS
	for i := 10; i < 15; i++ {
		err := ds.UpdateHostOperatingSystem(context.Background(), hostids[i], fleet.OperatingSystem{
			Name:     "macOS",
			Version:  "14.1.2",
			Arch:     "arm64",
			Platform: "darwin",
		})
		require.NoError(t, err)
	}

	// move 4 windows hosts to team 1
	team1, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(context.Background(), &team1.ID, hostids[:4])
	require.NoError(t, err)

	// move 3 windows hosts to team 2
	team2, err := ds.NewTeam(context.Background(), &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	err = ds.AddHostsToTeam(context.Background(), &team2.ID, hostids[4:7])
	require.NoError(t, err)

	// move 1 macOS host to team 2
	err = ds.AddHostsToTeam(context.Background(), &team2.ID, []uint{hostids[10]})
	require.NoError(t, err)

	err = ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	// State:
	// 10 global windows hosts
	// 5 global macOS hosts
	// 4 windows hosts in team 1
	// 3 windows hosts in team 2
	// 1 macOS host in team 2

	// add software to 5 windows hosts
	// affects:
	// 5 global windows hosts
	// 4 windows hosts in team 1
	// 1 windows host in team 2
	for i := 0; i < 5; i++ {
		_, err = ds.UpdateHostSoftware(context.Background(), hostids[i], []fleet.Software{
			{
				Name:    "Chrome",
				Version: "1.0.0",
				Source:  "programs",
			},
		})
		require.NoError(t, err)
	}

	_, err = ds.UpsertSoftwareCPEs(context.Background(), []fleet.SoftwareCPE{
		{
			SoftwareID: 1,
			CPE:        "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*",
		},
	})
	require.NoError(t, err)

	err = ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)

	softwareVulns := []fleet.SoftwareVulnerability{
		{
			SoftwareID:        1,
			CVE:               "CVE-2020-1234",
			ResolvedInVersion: ptr.String("1.0.0"),
		},
		{
			SoftwareID:        1,
			CVE:               "CVE-2020-1235",
			ResolvedInVersion: ptr.String("1.0.1"),
		},
		{
			SoftwareID: 2,
			CVE:        "CVE-2020-1236",
		},
		{
			SoftwareID: 2,
			CVE:        "CVE-2020-1237",
		},
	}

	osVulns := []fleet.OSVulnerability{
		{
			OSID:              1,
			CVE:               "CVE-2020-1238",
			ResolvedInVersion: ptr.String("1.0.0"),
		},
		{
			OSID:              1,
			CVE:               "CVE-2020-1239",
			ResolvedInVersion: ptr.String("1.0.1"),
		},
		{
			OSID: 2,
			CVE:  "CVE-2020-1240",
		},
		{
			OSID: 2,
			CVE:  "CVE-2020-1241",
		},
	}

	mockTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	cveMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2020-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1234",
		},
		{
			CVE:              "CVE-2020-1235",
			CVSSScore:        ptr.Float64(7.6),
			EPSSProbability:  ptr.Float64(0.51),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime.Add(time.Hour * 2)),
			Description:      "Test CVE 2020-1235",
		},
		{
			CVE:              "CVE-2020-1236",
			CVSSScore:        ptr.Float64(7.7),
			EPSSProbability:  ptr.Float64(0.52),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime.Add(time.Hour * 1)),
			Description:      "Test CVE 2020-1236",
		},
		{
			CVE:              "CVE-2020-1237",
			CVSSScore:        ptr.Float64(7.8),
			EPSSProbability:  ptr.Float64(0.53),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime.Add(time.Hour * 3)),
			Description:      "Test CVE 2020-1237",
		},
		{
			CVE:              "CVE-2020-1238",
			CVSSScore:        ptr.Float64(7.9),
			EPSSProbability:  ptr.Float64(0.54),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime.Add(time.Hour * 4)),
			Description:      "Test CVE 2020-1238",
		},
		{
			CVE:              "CVE-2020-1239",
			CVSSScore:        ptr.Float64(8.0),
			EPSSProbability:  ptr.Float64(0.55),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime.Add(time.Hour * 5)),
			Description:      "Test CVE 2020-1239",
		},
		{
			CVE:              "CVE-2020-1240",
			CVSSScore:        ptr.Float64(8.1),
			EPSSProbability:  ptr.Float64(0.56),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime.Add(time.Hour * 6)),
			Description:      "Test CVE 2020-1240",
		},
		// CVE-2020-1241 ommited to test null values
	}

	vulnHostCount := []struct {
		cve       string
		teamID    uint
		hostCount int
	}{
		{
			cve:       "CVE-2020-1234",
			teamID:    0,
			hostCount: 100,
		},
		{
			cve:       "CVE-2020-1234",
			teamID:    1,
			hostCount: 20,
		},
		{
			cve:       "CVE-2020-1235",
			teamID:    0,
			hostCount: 90,
		},
		{
			cve:       "CVE-2020-1235",
			teamID:    1,
			hostCount: 19,
		},
		{
			cve:       "CVE-2020-1236",
			teamID:    0,
			hostCount: 80,
		},
		{
			cve:       "CVE-2020-1236",
			teamID:    1,
			hostCount: 18,
		},
		{
			cve:       "CVE-2020-1237",
			teamID:    0,
			hostCount: 70,
		},
		// no team 1 host count for CVE-2020-1237
		{
			cve:       "CVE-2020-1238",
			teamID:    0,
			hostCount: 60,
		},
		{
			cve:       "CVE-2020-1238",
			teamID:    1,
			hostCount: 16,
		},
		{
			cve:       "CVE-2020-1239",
			teamID:    0,
			hostCount: 50,
		},
		{
			cve:       "CVE-2020-1239",
			teamID:    1,
			hostCount: 15,
		},
		// no host counts for CVE-2020-1240
		{
			cve:       "CVE-2020-1241",
			teamID:    0,
			hostCount: 40,
		},
		{
			cve:       "CVE-2020-1241",
			teamID:    1,
			hostCount: 14,
		},
	}

	// Insert OS Vuln
	_, err = ds.InsertOSVulnerabilities(context.Background(), osVulns, fleet.NVDSource)
	require.NoError(t, err)

	// Insert Software Vuln
	for _, vuln := range softwareVulns {
		_, err = ds.InsertSoftwareVulnerability(context.Background(), vuln, fleet.NVDSource)
		require.NoError(t, err)
	}

	// Insert CVEMeta
	err = ds.InsertCVEMeta(context.Background(), cveMeta)
	require.NoError(t, err)

	// Insert Host Count
	insertStmt := `
		INSERT INTO vulnerability_host_counts (cve, team_id, host_count)
		VALUES (?, ?, ?)
	`
	for _, vuln := range vulnHostCount {
		_, err = ds.writer(context.Background()).Exec(insertStmt, vuln.cve, vuln.teamID, vuln.hostCount)
		require.NoError(t, err)
	}
}
