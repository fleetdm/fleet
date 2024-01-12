package mysql

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestOperatingSystemVulnerabilities(t *testing.T) {
	ds := CreateMySQLDS(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"ListOSVulnerabilitiesEmpty", testListOSVulnerabilitiesEmpty},
		{"ListOSVulnerabilities", testListOSVulnerabilities},
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

func testListOSVulnerabilitiesEmpty(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	actual, err := ds.ListOSVulnerabilities(ctx, []uint{4})
	require.NoError(t, err)
	require.Empty(t, actual)
}

func testListOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3")},
		{HostID: 1, CVE: "cve-3", OSID: 1, ResolvedInVersion: ptr.String("10.14.2")},
		{HostID: 2, CVE: "cve-2", OSID: 1, ResolvedInVersion: ptr.String("8.123.1")},
	}

	for _, v := range vulns {
		_, err := ds.InsertOSVulnerability(ctx, v, fleet.MSRCSource)
		require.NoError(t, err)
	}

	t.Run("none matching", func(t *testing.T) {
		actual, err := ds.ListOSVulnerabilities(ctx, []uint{3})
		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("returns matching", func(t *testing.T) {
		expected := []fleet.OSVulnerability{
			{HostID: 1, CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3")},
			{HostID: 1, CVE: "cve-3", OSID: 1, ResolvedInVersion: ptr.String("10.14.2")},
		}

		actual, err := ds.ListOSVulnerabilities(ctx, []uint{1})
		require.NoError(t, err)
		require.ElementsMatch(t, expected, actual)
	})
}

func testInsertOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
		{HostID: 2, CVE: "cve-2", OSID: 1},
	}

	c, err := ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.Equal(t, int64(3), c)

	expected := []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
	}

	actual, err := ds.ListOSVulnerabilities(ctx, []uint{1})
	require.NoError(t, err)
	require.ElementsMatch(t, expected, actual)
}

func testInsertOSVulnerability(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := fleet.OSVulnerability{
		HostID: 1, CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.3"),
	}

	vulnsUpdate := fleet.OSVulnerability{
		HostID: 1, CVE: "cve-1", OSID: 1, ResolvedInVersion: ptr.String("1.2.4"),
	}

	vulnNoCVE := fleet.OSVulnerability{
		HostID: 1, OSID: 1,
	}

	// Inserting a vulnerability with no CVE should not insert anything
	didInsert, err := ds.InsertOSVulnerability(ctx, vulnNoCVE, fleet.MSRCSource)
	require.Error(t, err)
	require.False(t, didInsert)

	// Inserting a vulnerability with a CVE should insert
	didInsert, err = ds.InsertOSVulnerability(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.True(t, didInsert)

	// Inserting the same vulnerability should not insert
	didInsert, err = ds.InsertOSVulnerability(ctx, vulnsUpdate, fleet.MSRCSource)
	require.NoError(t, err)
	require.Equal(t, false, didInsert)

	list1, err := ds.ListOSVulnerabilities(ctx, []uint{1})
	require.NoError(t, err)
	require.Len(t, list1, 1)
	require.Equal(t, vulnsUpdate, list1[0])
}

func testDeleteOSVulnerabilitiesEmpty(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
		{HostID: 2, CVE: "cve-2", OSID: 1},
	}

	err := ds.DeleteOSVulnerabilities(ctx, vulns)
	require.NoError(t, err)
}

func testDeleteOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()

	vulns := []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
		{HostID: 2, CVE: "cve-2", OSID: 1},
	}

	c, err := ds.InsertOSVulnerabilities(ctx, vulns, fleet.MSRCSource)
	require.NoError(t, err)
	require.Equal(t, int64(3), c)

	toDelete := []fleet.OSVulnerability{
		{HostID: 2, CVE: "cve-2", OSID: 1},
	}

	err = ds.DeleteOSVulnerabilities(ctx, toDelete)
	require.NoError(t, err)

	actual, err := ds.ListOSVulnerabilities(ctx, []uint{1})
	require.NoError(t, err)
	require.ElementsMatch(t, []fleet.OSVulnerability{
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
	}, actual)

	actual, err = ds.ListOSVulnerabilities(ctx, []uint{2})
	require.NoError(t, err)
	require.Empty(t, actual)
}

func testDeleteOutOfDateOSVulnerabilities(t *testing.T, ds *Datastore) {
	ctx := context.Background()
	yesterday := time.Now().Add(-3 * time.Hour).Format("2006-01-02 15:04:05")

	oldVuln := fleet.OSVulnerability{
		HostID: 1, CVE: "cve-1", OSID: 1,
	}

	newVuln := fleet.OSVulnerability{
		HostID: 1, CVE: "cve-2", OSID: 1,
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

	actual, err := ds.ListOSVulnerabilities(ctx, []uint{1})
	require.NoError(t, err)
	require.Len(t, actual, 1)
	require.ElementsMatch(t, []fleet.OSVulnerability{newVuln}, actual)
}
