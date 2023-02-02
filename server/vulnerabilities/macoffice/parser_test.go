package macoffice

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReleaseHTML(t *testing.T) {
	html := ""

	expected := []OfficeRelease{
		{
			Date:    "January 19, 2023",
			Version: "Version 16.69.1 (Build 23011802)",
		},
		{
			Date:    "January 17, 2023",
			Version: "Version 16.69.1 (Build 23011600)",
		},
		{
			Date:    "January 10, 2023",
			Version: "Version 16.69 (Build 23010700)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2023-21734"},
				{Product: OfficeSuite, Vulnerability: "CVE-2023-21734"},
			},
		},
		{
			Date:    "December 13, 2022",
			Version: "Version 16.68 (Build 22121100)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2022-44692"},
				{Product: Outlook, Vulnerability: "CVE-2022-44713"},
			},
		},
		{
			Date:    "November 15, 2022",
			Version: "Version 16.67 (Build 22111300)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2022-41061"},
				{Product: OfficeSuite, Vulnerability: "CVE-2022-41107"},
			},
		},
		{
			Date:    "October 31, 2022",
			Version: "Version 16.66.2 (Build 22102801)",
		},
		{
			Date:    "October 12, 2022",
			Version: "Version 16.66.1 (Build 22101101)",
		},
		{
			Date:    "October 11, 2022",
			Version: "Version 16.66 (Build 22100900)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2022-41031"},
				{Product: Word, Vulnerability: "CVE-2022-38048"},
				{Product: OfficeSuite, Vulnerability: "CVE-2022-41043"},
			},
		},
		{
			Date:    "September 13, 2022",
			Version: "Version 16.65 (Build 22091101)",
			SecurityUpdates: []SecurityUpdate{
				{Product: PowerPoint, Vulnerability: "CVE-2022-37962"},
			},
		},
		{
			Date:    "August 16, 2022",
			Version: "Version 16.64 (Build 22081401)",
		},
		{
			Date:    "July 15, 2022",
			Version: "Version 16.63.1 (Build 22071401)",
		},
		{
			Date:    "July 12, 2022",
			Version: "Version 16.63 (Build 22070801)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2022-26934"},
			},
		},
		{
			Date:    "June 14, 2022",
			Version: "Version 16.62 (Build 22061100)",
		},
		{
			Date:    "May 23, 2022",
			Version: "Version 16.61.1 (Build 22052000)",
		},
		{
			Date:    "May 10, 2022",
			Version: "Version 16.61 (Build 22050700)",
		},
		{
			Date:    "April 12, 2022",
			Version: "Version 16.60 (Build 22041000)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2022-26901"},
				{Product: Excel, Vulnerability: "CVE-2022-24473"},
			},
		},
		{
			Date:    "March 15, 2022",
			Version: "Version 16.59 (Build 22031300)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2022-24511"},
			},
		},

		{
			Date:    "February 16, 2022",
			Version: "Version 16.58 (Build 22021501)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2022-22716"},
				{Product: OfficeSuite, Vulnerability: "CVE-2022-22003"},
			},
		},
		{
			Date:    "January 13, 2022",
			Version: "Version 16.57 (Build 22011101)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2022-21841"},
				{Product: OfficeSuite, Vulnerability: "CVE-2022-21840"},
			},
		},

		{
			Date:    "January 13, 2022",
			Version: "Version 16.57 (Build 22011101)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2022-21841"},
				{Product: OfficeSuite, Vulnerability: "CVE-2022-21840"},
			},
		},
		{
			Date:    "November 16, 2021",
			Version: "Version 16.55 (Build 21111400)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-40442"},
				{Product: Excel, Vulnerability: "CVE-2021-42292"},
			},
		},
		{
			Date:    "October 12, 2021",
			Version: "Version 16.54 (Build 21101001)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-40474"},
				{Product: Excel, Vulnerability: "CVE-2021-40485"},
				{Product: OfficeSuite, Vulnerability: "CVE-2021-40454"},
			},
		},
		{
			Date:    "September 14, 2021",
			Version: "Version 16.53 (Build 21091200)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-38655"},
				{Product: OfficeSuite, Vulnerability: "CVE-2021-38650"},
			},
		},
		{
			Date:    "August 10, 2021",
			Version: "Version 16.52 (Build 21080801)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2021-36941"},
			},
		},
		{
			Date:    "July 13, 2021",
			Version: "Version 16.51 (Build 21071101)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-34501"},
			},
		},
		{
			Date:    "June 15, 2021",
			Version: "Version 16.50 (Build 21061301)",
			SecurityUpdates: []SecurityUpdate{
				{Product: OfficeSuite, Vulnerability: "CVE-2021-31941"},
				{Product: OfficeSuite, Vulnerability: "CVE-2021-31940"},
			},
		},
		{
			Date:    "May 11, 2021",
			Version: "Version 16.49 (Build 21050901)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-31177"},
			},
		},
		{
			Date:    "April 13, 2021",
			Version: "Version 16.48 (Build 21041102)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-28451"},
				{Product: Excel, Vulnerability: "CVE-2021-28456"},
				{Product: Word, Vulnerability: "CVE-2021-28453"},
			},
		},
		{
			Date:    "March 16, 2021",
			Version: "Version 16.47 (Build 21031401)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-27054"},
				{Product: Excel, Vulnerability: "CVE-2021-27057"},
			},
		},
		{
			Date:    "February 16, 2021",
			Version: "Version 16.46 (Build 21021202)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-24067"},
				{Product: Excel, Vulnerability: "CVE-2021-24069"},
			},
		},
		{
			Date:    "January 13, 2021",
			Version: "Version 16.45 (Build 21011103)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2021-1714"},
				{Product: Excel, Vulnerability: "CVE-2021-1713"},
				{Product: Word, Vulnerability: "CVE-2021-1716"},
				{Product: Word, Vulnerability: "CVE-2021-1715"},
			},
		},
		{
			Date:    "December 15, 2020",
			Version: "Version 16.44 (Build 20121301)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-17123"},
				{Product: Excel, Vulnerability: "CVE-2020-17126"},
				{Product: Excel, Vulnerability: "CVE-2020-17128"},
				{Product: Outlook, Vulnerability: "CVE-2020-17119"},
				{Product: PowerPoint, Vulnerability: "CVE-2020-17124"},
			},
		},
		{
			Date:    "November 10, 2020",
			Version: "Version 16.43 (Build 20110804)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-17067"},
			},
		},
		{
			Date:    "October 13, 2020",
			Version: "Version 16.42 (Build 20101102)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-16929"},
				{Product: Word, Vulnerability: "CVE-2020-16933"},
				{Product: OfficeSuite, Vulnerability: "CVE-2020-16918"},
			},
		},
		{
			Date:    "September 15, 2020",
			Version: "Version 16.41 (Build 20091302)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-1224"},
				{Product: Word, Vulnerability: "CVE-2020-1218"},
				{Product: Word, Vulnerability: "CVE-2020-1338"},
				{Product: OfficeSuite, Vulnerability: "CVE-2020-1193"},
				{Product: OfficeSuite, Vulnerability: "CVE-2020-16855"},
			},
		},
		{
			Date:    "August 11, 2020",
			Version: "Version 16.40 (Build 20081000)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-1495"},
				{Product: Excel, Vulnerability: "CVE-2020-1498"},
				{Product: Word, Vulnerability: "CVE-2020-1503"},
				{Product: Word, Vulnerability: "CVE-2020-1583"},
			},
		},
		{
			Date:    "July 14, 2020",
			Version: "Version 16.39 (Build 20071300)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2020-1342"},
				{Product: Word, Vulnerability: "CVE-2020-1445"},
				{Product: Word, Vulnerability: "CVE-2020-1446"},
				{Product: Word, Vulnerability: "CVE-2020-1447"},
				{Product: OfficeSuite, Vulnerability: "CVE-2020-1409"},
			},
		},
		{
			Date:    "June 16, 2020",
			Version: "Version 16.38 (Build 20061401)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-1225"},
				{Product: Excel, Vulnerability: "CVE-2020-1226"},
				{Product: Outlook, Vulnerability: "CVE-2020-1229"},
				{Product: OfficeSuite, Vulnerability: "CVE-2020-1321"},
			},
		},
		{
			Date:    "May 12, 2020",
			Version: "Version 16.37 (Build 20051002)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-0901"},
			},
		},
		{
			Date:    "April 14, 2020",
			Version: "Version 16.36 (Build 20041300)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2020-0980"},
			},
		},
		{
			Date:    "March 10, 2020",
			Version: "Version 16.35 (Build 20030802)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2020-0850"},
				{Product: Word, Vulnerability: "CVE-2020-0851"},
				{Product: Word, Vulnerability: "CVE-2020-0855"},
				{Product: Word, Vulnerability: "CVE-2020-0892"},
			},
		},
		{
			Date:    "February 11, 2020",
			Version: "Version 16.34 (Build 20020900)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-0759"},
			},
		},
		{
			Date:    "January 14, 2020",
			Version: "Version 16.33 (Build 20011301)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2020-0650"},
				{Product: Excel, Vulnerability: "CVE-2020-0651"},
			},
		},
		{
			Date:    "December 10, 2019",
			Version: "Version 16.32 (Build 19120802)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2019-1464"},
				{Product: PowerPoint, Vulnerability: "CVE-2019-1462"},
			},
		},
		{
			Date:    "November 12, 2019",
			Version: "Version 16.31 (Build 19111002)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2019-1446"},
				{Product: Excel, Vulnerability: "CVE-2019-1448"},
				{Product: Excel, Vulnerability: "CVE-2019-1457"},
			},
		},
		{
			Date:    "October 15, 2019",
			Version: "Version 16.30 (Build 19101301)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2019-1327"},
				{Product: Excel, Vulnerability: "CVE-2019-1331"},
			},
		},
		{
			Date:    "September 18, 2019",
			Version: "Version 16.29.1 (Build 19091700)",
		},
		{
			Date:    "September 10, 2019",
			Version: "Version 16.29 (Build 19090802)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2019-1263"},
				{Product: Excel, Vulnerability: "CVE-2019-1297"},
			},
		},
		{
			Date:    "August 13, 2019 release",
			Version: "Version 16.28 (Build 19081202)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2019-1201"},
				{Product: Word, Vulnerability: "CVE-2019-1205"},
				{Product: OfficeSuite, Vulnerability: "CVE-2019-1148"},
				{Product: OfficeSuite, Vulnerability: "CVE-2019-1149"},
				{Product: OfficeSuite, Vulnerability: "CVE-2019-1151"},
				{Product: OfficeSuite, Vulnerability: "CVE-2019-1153"},
			},
		},
		{
			Date:    "July 16, 2019 release",
			Version: "Version 16.27 (Build 19071500)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Excel, Vulnerability: "CVE-2019-1110"},
				{Product: Excel, Vulnerability: "CVE-2019-1111"},
				{Product: Outlook, Vulnerability: "CVE-2019-1084"},
			},
		},
		{
			Date:    "June 11, 2019 release",
			Version: "Version 16.26 (Build 19060901)",
			SecurityUpdates: []SecurityUpdate{
				{Product: Word, Vulnerability: "CVE-2019-1034"},
				{Product: Word, Vulnerability: "CVE-2019-1035"},
			},
		},
		// Releases after this point are in a table format
	}
	actual, err := ParseReleaseHTML(html)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}
