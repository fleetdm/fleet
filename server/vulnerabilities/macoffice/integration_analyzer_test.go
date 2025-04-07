package macoffice_test

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/stretchr/testify/require"
)

func TestIntegrationsAnalyzer(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	vulnPath := t.TempDir()
	releaseNotes := macoffice.ReleaseNotes{
		{
			Date:    time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.69.1 (Build 23011802)",
		},
		{
			Date:    time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.69.1 (Build 23011600)",
		},
		{
			Date:    time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.69 (Build 23010700)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21734"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21735"},
			},
		},
		{
			Date:    time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.68 (Build 22121100)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.Outlook, Vulnerability: "CVE-2022-44713"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-44692"},
			},
		},
		{
			Date:    time.Date(2022, 11, 15, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.67 (Build 22111300)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.Word, Vulnerability: "CVE-2022-41061"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-41107"},
			},
		},
		{
			Date:    time.Date(2022, 10, 31, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.66.2 (Build 22102801)",
		},
		{
			Date:    time.Date(2022, 10, 12, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.66.1 (Build 22101101)",
		},
		{
			Date:    time.Date(2022, 10, 11, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.66 (Build 22100900)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.Word, Vulnerability: "CVE-2022-41031"},
				{Product: macoffice.Word, Vulnerability: "CVE-2022-38048"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-41043"},
			},
		},
		{
			Date:    time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.65 (Build 22091101)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.PowerPoint, Vulnerability: "CVE-2022-37962"},
			},
		},
		{
			Date:    time.Date(2022, 8, 16, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.64 (Build 22081401)",
		},
		{
			Date:    time.Date(2022, 7, 15, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.63.1 (Build 22071301)",
		},
		{
			Date:    time.Date(2022, 7, 12, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.63 (Build 22070801)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-26934"},
			},
		},
	}
	require.NoError(t, releaseNotes.Serialize(time.Now(), vulnPath))

	ctx := context.Background()

	t.Run("no apps", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
		software := []fleet.Software{
			{
				Name:    "Bicrosoft Bord.app",
				Version: "16.69.1",
				Source:  "programs",
			},
		}

		_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
		require.NoError(t, err)
		vulns, err := macoffice.Analyze(ctx, ds, vulnPath, true)
		require.NoError(t, err)
		require.Empty(t, vulns)
	})

	t.Run("no office apps", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
		software := []fleet.Software{
			{
				Name:             "Bicrosoft Bord.app",
				Version:          "16.69.1",
				BundleIdentifier: "com.bicrosoft.Bord",
				Source:           "apps",
			},
		}

		_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
		require.NoError(t, err)
		vulns, err := macoffice.Analyze(ctx, ds, vulnPath, true)
		require.NoError(t, err)
		require.Empty(t, vulns)
	})

	t.Run("latest version", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		host := test.NewHost(t, ds, "host1", "", "host1key", "host1uuid", time.Now())
		software := []fleet.Software{
			{
				Name:             "Microsoft Word.app",
				Version:          "16.69.1",
				BundleIdentifier: "com.microsoft.Word",
				Source:           "apps",
			},
			{
				Name:             "Microsoft PowerPoint.app",
				Version:          "16.69.1",
				BundleIdentifier: "com.microsoft.Powerpoint",
				Source:           "apps",
			},
		}

		_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
		require.NoError(t, err)

		vulns, err := macoffice.Analyze(ctx, ds, vulnPath, true)
		require.NoError(t, err)
		require.Empty(t, vulns)
	})

	t.Run("vulnerable versions", func(t *testing.T) {
		defer mysql.TruncateTables(t, ds)
		host := test.NewHost(t, ds, "host2", "", "host2key", "host2uuid", time.Now())
		software := []fleet.Software{
			{
				Name:             "Microsoft Word.app",
				Version:          "16.65",
				BundleIdentifier: "com.microsoft.Word",
				Source:           "apps",
			},
			{
				Name:             "Microsoft PowerPoint.app",
				Version:          "16.65",
				BundleIdentifier: "com.microsoft.Powerpoint",
				Source:           "apps",
			},
		}

		_, err := ds.UpdateHostSoftware(context.Background(), host.ID, software)
		require.NoError(t, err)
		require.NoError(t, ds.LoadHostSoftware(context.Background(), host, false))

		var powerpoint fleet.HostSoftwareEntry
		var word fleet.HostSoftwareEntry
		for _, s := range host.HostSoftware.Software {
			if s.Name == "Microsoft PowerPoint.app" {
				powerpoint = s
			}

			if s.Name == "Microsoft Word.app" {
				word = s
			}
		}

		// These 'old' vulnerabilities should be cleared out...
		ok, err := ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: word.ID, CVE: "3000-3000",
		}, fleet.MacOfficeReleaseNotesSource)
		require.True(t, ok)
		require.NoError(t, err)

		ok, err = ds.InsertSoftwareVulnerability(ctx, fleet.SoftwareVulnerability{
			SoftwareID: powerpoint.ID, CVE: "4000-3000",
		}, fleet.MacOfficeReleaseNotesSource)
		require.True(t, ok)
		require.NoError(t, err)

		vulns, err := macoffice.Analyze(ctx, ds, vulnPath, true)
		require.NoError(t, err)

		expected := []fleet.SoftwareVulnerability{
			{SoftwareID: word.ID, CVE: "CVE-2023-21734"},
			{SoftwareID: word.ID, CVE: "CVE-2023-21735"},
			{SoftwareID: word.ID, CVE: "CVE-2022-44692"},
			{SoftwareID: word.ID, CVE: "CVE-2022-41061"},
			{SoftwareID: word.ID, CVE: "CVE-2022-41107"},
			{SoftwareID: word.ID, CVE: "CVE-2022-41031"},
			{SoftwareID: word.ID, CVE: "CVE-2022-38048"},
			{SoftwareID: word.ID, CVE: "CVE-2022-41043"},
			{SoftwareID: powerpoint.ID, CVE: "CVE-2023-21734"},
			{SoftwareID: powerpoint.ID, CVE: "CVE-2023-21735"},
			{SoftwareID: powerpoint.ID, CVE: "CVE-2022-44692"},
			{SoftwareID: powerpoint.ID, CVE: "CVE-2022-41107"},
			{SoftwareID: powerpoint.ID, CVE: "CVE-2022-41043"},
		}

		require.ElementsMatch(t, expected, vulns)

		stored, err := ds.ListSoftwareVulnerabilitiesByHostIDsSource(ctx, []uint{host.ID}, fleet.MacOfficeReleaseNotesSource)
		require.NoError(t, err)
		require.ElementsMatch(t, expected, stored[host.ID])
	})
}
