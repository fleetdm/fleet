package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
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
		{"ListVulnssByOsNameAndVersion", testListVulnsByOsNameAndVersion},
		{"InsertOSVulnerabilities", testInsertOSVulnerabilities},
		{"InsertSingleOSVulnerability", testInsertOSVulnerability},
		{"DeleteOSVulnerabilitiesEmpty", testDeleteOSVulnerabilitiesEmpty},
		{"DeleteOSVulnerabilities", testDeleteOSVulnerabilities},
		{"DeleteOutOfDateOSVulnerabilities", testDeleteOutOfDateOSVulnerabilities},
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

	dbOS := []fleet.OperatingSystem{}
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
		{CVE: "CVE-2021-1234", OSID: dbOS[1].ID, ResolvedInVersion: ptr.String("1.2.3")}, // same OS, different arch
		{CVE: "CVE-2021-1235", OSID: dbOS[1].ID, ResolvedInVersion: ptr.String("10.14.2")},
		{CVE: "CVE-2021-1236", OSID: dbOS[2].ID, ResolvedInVersion: ptr.String("103.2.1")},
	}

	_, err = ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
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
	didInsertOrUpdate, err := ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	assert.True(t, didInsertOrUpdate)

	// Inserting the exact same vulnerability again should not insert and not update
	didInsertOrUpdate, err = ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	assert.False(t, didInsertOrUpdate)

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
