package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestVulnerabilities(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"TestListVulnerabilities", testListVulnerabilities},
		{"TestVulnerabilitiesPagination", testVulnerabilitiesPagination},
		{"TestVulnerabilitiesTeamFilter", testVulnerabilitiesTeamFilter},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.fn(t, ds)
			TruncateTables(t, ds)
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
	}, fleet.NVDSource)
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

	expected := []fleet.VulnerabilityWithMetadata{
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1234",
				CVSSScore:        ptr.Float64(7.5),
				EPSSProbability:  ptr.Float64(0.5),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1234",
			},
			ResolvedInVersion: ptr.String("1.0.0"),
			HostCount:         10,
		},
		{
			CVEMeta:           fleet.CVEMeta{CVE: "CVE-2020-1235"},
			ResolvedInVersion: nil,
			HostCount:         15,
		},
		{
			CVEMeta:           fleet.CVEMeta{CVE: "CVE-2020-1236"},
			ResolvedInVersion: nil,
			HostCount:         20,
		},
	}
	list, _, err = ds.ListVulnerabilities(context.Background(), opts)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.ElementsMatch(t, expected, list)
}

func testVulnerabilitiesPagination(t *testing.T, ds *Datastore) {
	_ = seedVulnerabilities(t, ds)

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
	_ = seedVulnerabilities(t, ds)

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
		require.Equal(t, checkCounts[vuln.CVE], int(vuln.HostCount), vuln.CVE)
	}
}

func seedVulnerabilities(t *testing.T, ds *Datastore) []fleet.VulnerabilityWithMetadata {
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
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1235",
		},
		{
			CVE:              "CVE-2020-1236",
			CVSSScore:        ptr.Float64(7.7),
			EPSSProbability:  ptr.Float64(0.52),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1236",
		},
		{
			CVE:              "CVE-2020-1237",
			CVSSScore:        ptr.Float64(7.8),
			EPSSProbability:  ptr.Float64(0.53),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1237",
		},
		{
			CVE:              "CVE-2020-1238",
			CVSSScore:        ptr.Float64(7.9),
			EPSSProbability:  ptr.Float64(0.54),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1238",
		},
		{
			CVE:              "CVE-2020-1239",
			CVSSScore:        ptr.Float64(8.0),
			EPSSProbability:  ptr.Float64(0.55),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2020-1239",
		},
		{
			CVE:              "CVE-2020-1240",
			CVSSScore:        ptr.Float64(8.1),
			EPSSProbability:  ptr.Float64(0.56),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
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
	_, err := ds.InsertOSVulnerabilities(context.Background(), osVulns, fleet.NVDSource)
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

	expected := []fleet.VulnerabilityWithMetadata{
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1234",
				CVSSScore:        ptr.Float64(7.5),
				EPSSProbability:  ptr.Float64(0.5),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1234",
			},
			ResolvedInVersion: ptr.String("1.0.0"),
			HostCount:         100,
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1235",
				CVSSScore:        ptr.Float64(7.6),
				EPSSProbability:  ptr.Float64(0.51),
				CISAKnownExploit: ptr.Bool(false),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1235",
			},
			ResolvedInVersion: ptr.String("1.0.1"),
			HostCount:         90,
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1236",
				CVSSScore:        ptr.Float64(7.7),
				EPSSProbability:  ptr.Float64(0.52),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1236",
			},
			ResolvedInVersion: nil, // No resolved version provided
			HostCount:         98,  // Host count for team 0
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1237",
				CVSSScore:        ptr.Float64(7.8),
				EPSSProbability:  ptr.Float64(0.53),
				CISAKnownExploit: ptr.Bool(false),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1237",
			},
			ResolvedInVersion: nil, // No resolved version provided
			HostCount:         0,   // No host count for team 1
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1238",
				CVSSScore:        ptr.Float64(7.9),
				EPSSProbability:  ptr.Float64(0.54),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1238",
			},
			ResolvedInVersion: ptr.String("1.0.0"),
			HostCount:         60,
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1239",
				CVSSScore:        ptr.Float64(8.0),
				EPSSProbability:  ptr.Float64(0.55),
				CISAKnownExploit: ptr.Bool(false),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1239",
			},
			ResolvedInVersion: ptr.String("1.0.1"),
			HostCount:         15,
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1240",
				CVSSScore:        ptr.Float64(8.1),
				EPSSProbability:  ptr.Float64(0.56),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2020-1240",
			},
			ResolvedInVersion: nil, // No resolved version provided
			HostCount:         54,  // Host count for team 0
		},
		{
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2020-1241",
				CVSSScore:        nil, // No CVSSScore provided
				EPSSProbability:  nil, // No EPSSProbability provided
				CISAKnownExploit: nil, // No CISAKnownExploit provided
				Published:        nil, // No Published date provided
				Description:      "",  // No Description provided
			},
			ResolvedInVersion: nil, // No resolved version provided
			HostCount:         14,
		},
	}

	return expected
}
