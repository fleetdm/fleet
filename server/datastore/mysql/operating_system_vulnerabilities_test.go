package mysql

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
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
		{"DeleteOSVulnerabilitiesEmpty", testDeleteOSVulnerabilitiesEmpty},
		{"DeleteOSVulnerabilities", testDeleteOSVulnerabilities},
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
		{HostID: 1, CVE: "cve-1", OSID: 1},
		{HostID: 1, CVE: "cve-3", OSID: 1},
		{HostID: 2, CVE: "cve-2", OSID: 1},
	}

	for _, v := range vulns {
		_, err := ds.writer(ctx).Exec(
			`INSERT INTO operating_system_vulnerabilities(host_id,operating_system_id,cve) VALUES (?,?,?)`,
			v.HostID, v.OSID, v.CVE,
		)
		require.NoError(t, err)
	}

	t.Run("none matching", func(t *testing.T) {
		actual, err := ds.ListOSVulnerabilities(ctx, []uint{3})
		require.NoError(t, err)
		require.Empty(t, actual)
	})

	t.Run("returns matching", func(t *testing.T) {
		expected := []fleet.OSVulnerability{
			{HostID: 1, CVE: "cve-1", OSID: 1},
			{HostID: 1, CVE: "cve-3", OSID: 1},
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
