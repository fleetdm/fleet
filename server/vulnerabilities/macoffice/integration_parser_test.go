package macoffice_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/stretchr/testify/require"
)

var expected = []macoffice.ReleaseNote{
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
	{
		Date:    time.Date(2022, 6, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.62 (Build 22061100)",
	},
	{
		Date:    time.Date(2022, 5, 23, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.61.1 (Build 22052000)",
	},
	{
		Date:    time.Date(2022, 5, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.61 (Build 22050700)",
	},
	{
		Date:    time.Date(2022, 4, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.60 (Build 22041000)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2022-26901"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2022-24473"},
		},
	},
	{
		Date:    time.Date(2022, 3, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.59 (Build 22031300)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2022-24511"},
		},
	},

	{
		Date:    time.Date(2022, 2, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.58 (Build 22021501)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2022-22716"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-22003"},
		},
	},
	{
		Date:    time.Date(2022, 1, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.57 (Build 22011101)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-21841"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-21840"},
		},
	},

	{
		Date:    time.Date(2021, 12, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.56 (Build 21121100)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2021-43875"},
		},
	},
	{
		Date:    time.Date(2021, 11, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.55 (Build 21111400)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-40442"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-42292"},
		},
	},
	{
		Date:    time.Date(2021, 10, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.54 (Build 21101001)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-40474"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-40485"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2021-40454"},
		},
	},
	{
		Date:    time.Date(2021, 9, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.53 (Build 21091200)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-38655"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2021-38650"},
		},
	},
	{
		Date:    time.Date(2021, 8, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.52 (Build 21080801)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2021-36941"},
		},
	},
	{
		Date:    time.Date(2021, 7, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.51 (Build 21071101)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-34501"},
		},
	},
	{
		Date:    time.Date(2021, 6, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.50 (Build 21061301)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2021-31941"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2021-31940"},
		},
	},
	{
		Date:    time.Date(2021, 5, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.49 (Build 21050901)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-31177"},
		},
	},
	{
		Date:    time.Date(2021, 4, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.48 (Build 21041102)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-28451"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-28456"},
			{Product: macoffice.Word, Vulnerability: "CVE-2021-28453"},
		},
	},
	{
		Date:    time.Date(2021, 3, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.47 (Build 21031401)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-27054"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-27057"},
		},
	},
	{
		Date:    time.Date(2021, 2, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.46 (Build 21021202)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-24067"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-24069"},
		},
	},
	{
		Date:    time.Date(2021, 1, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.45 (Build 21011103)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-1714"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2021-1713"},
			{Product: macoffice.Word, Vulnerability: "CVE-2021-1716"},
			{Product: macoffice.Word, Vulnerability: "CVE-2021-1715"},
		},
	},
	{
		Date:    time.Date(2020, 12, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.44 (Build 20121301)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-17123"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-17126"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-17128"},
			{Product: macoffice.Outlook, Vulnerability: "CVE-2020-17119"},
			{Product: macoffice.PowerPoint, Vulnerability: "CVE-2020-17124"},
		},
	},
	{
		Date:    time.Date(2020, 11, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.43 (Build 20110804)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-17067"},
		},
	},
	{
		Date:    time.Date(2020, 10, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.42 (Build 20101102)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-16929"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-16933"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2020-16918"},
		},
	},
	{
		Date:    time.Date(2020, 9, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.41 (Build 20091302)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-1224"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1218"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1338"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2020-1193"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2020-16855"},
		},
	},
	{
		Date:    time.Date(2020, 8, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.40 (Build 20081000)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-1495"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-1498"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1503"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1583"},
		},
	},
	{
		Date:    time.Date(2020, 7, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.39 (Build 20071300)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1342"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1445"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1446"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-1447"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2020-1409"},
		},
	},
	{
		Date:    time.Date(2020, 6, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.38 (Build 20061401)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-1225"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-1226"},
			{Product: macoffice.Outlook, Vulnerability: "CVE-2020-1229"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2020-1321"},
		},
	},
	{
		Date:    time.Date(2020, 5, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.37 (Build 20051002)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-0901"},
		},
	},
	{
		Date:    time.Date(2020, 4, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.36 (Build 20041300)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2020-0980"},
		},
	},
	{
		Date:    time.Date(2020, 3, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.35 (Build 20030802)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2020-0850"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-0851"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-0855"},
			{Product: macoffice.Word, Vulnerability: "CVE-2020-0892"},
		},
	},
	{
		Date:    time.Date(2020, 2, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.34 (Build 20020900)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-0759"},
		},
	},
	{
		Date:    time.Date(2020, 1, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.33 (Build 20011301)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-0650"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2020-0651"},
		},
	},
	{
		Date:    time.Date(2019, 12, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.32 (Build 19120802)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1464"},
			{Product: macoffice.PowerPoint, Vulnerability: "CVE-2019-1462"},
		},
	},
	{
		Date:    time.Date(2019, 11, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.31 (Build 19111002)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1446"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1448"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1457"},
		},
	},
	{
		Date:    time.Date(2019, 10, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.30 (Build 19101301)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1327"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1331"},
		},
	},
	{
		Date:    time.Date(2019, 9, 18, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.29.1 (Build 19091700)",
	},
	{
		Date:    time.Date(2019, 9, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.29 (Build 19090802)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1263"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1297"},
		},
	},
	{
		Date:    time.Date(2019, 8, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.28 (Build 19081202)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2019-1201"},
			{Product: macoffice.Word, Vulnerability: "CVE-2019-1205"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2019-1148"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2019-1149"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2019-1151"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2019-1153"},
		},
	},
	{
		Date:    time.Date(2019, 7, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.27 (Build 19071500)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1110"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-1111"},
			{Product: macoffice.Outlook, Vulnerability: "CVE-2019-1084"},
		},
	},
	{
		Date:    time.Date(2019, 6, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.26 (Build 19060901)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2019-1034"},
			{Product: macoffice.Word, Vulnerability: "CVE-2019-1035"},
		},
	},
	// Releases after this point are in a table format
	{
		Date:    time.Date(2019, 5, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.25 (Build 19051201)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2019-0953"},
		},
	},
	{
		Date:    time.Date(2019, 4, 29, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.24.1 (Build 19042400)",
	},
	{
		Date:    time.Date(2019, 4, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.24 (Build 19041401)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-0828"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2019-0822"},
		},
	},
	{
		Date: time.Date(2019, 3, 27, 0, 0, 0, 0, time.UTC),
	},
	{
		Date: time.Date(2019, 3, 14, 0, 0, 0, 0, time.UTC),
	},
	{
		Date: time.Date(2019, 3, 12, 0, 0, 0, 0, time.UTC),
	},
	{
		Date: time.Date(2019, 2, 26, 0, 0, 0, 0, time.UTC),
	},
	{
		Date:    time.Date(2019, 2, 20, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.22.1 (Build 19022000)",
	},
	{
		Date:    time.Date(2019, 2, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.22.0 (Build 19021100)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2019-0669"},
		},
	},
	{
		Date:    time.Date(2019, 1, 24, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.21.0 (Build 19011700)",
	},
	{
		Date:    time.Date(2019, 1, 23, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.21.0 (Build 190102303)",
	},
	{
		Date:    time.Date(2019, 1, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.21.0 (Build 190101500)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2019-0561"},
			{Product: macoffice.Word, Vulnerability: "CVE-2019-0585"},
		},
	},
	{
		Date:    time.Date(2018, 12, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.20.0 (Build 18120801)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8597"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8627"},
			{Product: macoffice.PowerPoint, Vulnerability: "CVE-2018-8628"},
		},
	},
	{
		Date:    time.Date(2018, 11, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.19.0 (Build 18110915)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8574"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8577"},
		},
	},
	{
		Date:    time.Date(2018, 10, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.18.0 (Build 18101400)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2018-8432"},
		},
	},
	{
		Date:    time.Date(2018, 9, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.17.0 (Build 18090901)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8429"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8331"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2018-8332"},
		},
	},
	{
		Date:    time.Date(2018, 8, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.16.0 (Build 18081201)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8375"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8382"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2018-8412"},
		},
	},
	{
		Date:    time.Date(2018, 7, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.15.0 (Build 18070902)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2018-8281"},
		},
	},
	{
		Date:    time.Date(2018, 6, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.14.1 (Build 18061302)",
	},
	{
		Date:    time.Date(2018, 6, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.14.0 (Build 18061000)",
	},
	{
		Date:    time.Date(2018, 5, 24, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.13.1 (Build 18052304)",
	},
	{
		Date:    time.Date(2018, 5, 23, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.13.1 (Build 18052203)",
	},
	{
		Date:    time.Date(2018, 5, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.13.0 (Build 18051301)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8147"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-8162"},
			{Product: macoffice.PowerPoint, Vulnerability: "CVE-2018-8176"},
		},
	},
	{
		Date:    time.Date(2018, 4, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.12.0 (Build 18041000)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-1029"},
		},
	},
	{
		Date:    time.Date(2018, 3, 19, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.11.1 (Build 18031900)",
	},
	{
		Date:    time.Date(2018, 3, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.11.0 (Build 18031100)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2018-0907"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2018-0919"},
		},
	},
	{
		Date:    time.Date(2018, 2, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.10.0 (Build 18021001)",
	},
	{
		Date:    time.Date(2018, 1, 26, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.9.1 (Build 18012504)",
	},
	{
		Date:    time.Date(2018, 1, 18, 0, 0, 0, 0, time.UTC),
		Version: "Version 16.9.0 (Build 18011602)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2018-0792"},
			{Product: macoffice.Word, Vulnerability: "CVE-2018-0794"},
			{Product: macoffice.Outlook, Vulnerability: "CVE-2018-0793"},
		},
	},
	{
		Date: time.Date(2017, 12, 17, 0, 0, 0, 0, time.UTC),
	},
	{
		Date:    time.Date(2017, 12, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.41.0 (Build 17120500)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.PowerPoint, Vulnerability: "CVE-2017-11934"},
		},
	},
	{
		Date:    time.Date(2017, 11, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.40.0 (Build 17110800)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2017-11877"},
		},
	},
	{
		Date:    time.Date(2017, 10, 10, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.39.0 (Build 17101000)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2017-11825"},
		},
	},
	{
		Date:    time.Date(2017, 9, 12, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.38.0 (Build 17090200)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Excel, Vulnerability: "CVE-2017-8631"},
			{Product: macoffice.Excel, Vulnerability: "CVE-2017-8632"},
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2017-8676"},
		},
	},
	{
		Date:    time.Date(2017, 8, 15, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.37.0 (Build 17081500)",
	},
	{
		Date:    time.Date(2017, 7, 21, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.36.1 (Build 17072101)",
	},
	{
		Date:    time.Date(2017, 7, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.36.0 (Build 17070201)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2017-8501"},
		},
	},
	{
		Date:    time.Date(2017, 6, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.35.0 (Build 17061600)",
	},
	{
		Date:    time.Date(2017, 6, 13, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.35.0 (Build 17061000)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.Word, Vulnerability: "CVE-2017-8509"},
		},
	},
	{
		Date:    time.Date(2017, 5, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.34.0 (Build 17051500)",
		SecurityUpdates: []macoffice.SecurityUpdate{
			{Product: macoffice.WholeSuite, Vulnerability: "CVE-2017-0254"},
		},
	},
	{
		Date: time.Date(2017, 5, 9, 0, 0, 0, 0, time.UTC),
	},
	{
		Date:    time.Date(2017, 4, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.33.0 (Build 17040900)",
	},
	{
		Date:    time.Date(2017, 3, 14, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.32.0 (Build 17030901)",
	},
	{
		Date:    time.Date(2017, 2, 16, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.31.0 (Build 17021600)",
	},
	{
		Date:    time.Date(2017, 1, 11, 0, 0, 0, 0, time.UTC),
		Version: "Version 15.30.0 (Build 17010700)",
	},
}

func TestIntegrationsParseReleaseHTML(t *testing.T) {
	nettest.Run(t)

	res, err := http.Get(macoffice.RelNotesURL)
	require.NoError(t, err)
	defer res.Body.Close()

	actual, err := macoffice.ParseReleaseHTML(res.Body)
	require.NoError(t, err)
	require.NotEmpty(t, actual)

	t.Run("should parse Dates", func(t *testing.T) {
		expectedDates := make([]time.Time, 0, len(expected))
		for _, e := range expected {
			expectedDates = append(expectedDates, e.Date)
		}

		actualDates := make([]time.Time, 0, len(actual))
		for _, a := range actual {
			actualDates = append(actualDates, a.Date)
		}

		require.Subset(t, actualDates, expectedDates)
	})

	t.Run("should parse release versions", func(t *testing.T) {
		expectedVersions := make([]string, 0, len(expected))
		for _, e := range expected {
			expectedVersions = append(expectedVersions, e.Version)
		}

		actualVersions := make([]string, 0, len(actual))
		for _, a := range actual {
			actualVersions = append(actualVersions, a.Version)
		}

		require.Subset(t, actualVersions, expectedVersions)
	})

	t.Run("should parse security updates", func(t *testing.T) {
		expectedUpdates := make([]macoffice.SecurityUpdate, 0, len(expected))
		for _, e := range expected {
			expectedUpdates = append(expectedUpdates, e.SecurityUpdates...)
		}

		actualUpdates := make([]macoffice.SecurityUpdate, 0, len(actual))
		for _, a := range actual {
			actualUpdates = append(actualUpdates, a.SecurityUpdates...)
		}

		require.Subset(t, actualUpdates, expectedUpdates)
	})
}
