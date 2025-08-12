package macoffice_test

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/stretchr/testify/require"
)

func TestReleaseNote(t *testing.T) {
	t.Run("#CollectVulnerabilities", func(t *testing.T) {
		sut := macoffice.ReleaseNote{
			Date:    time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.69 (Build 23010700)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21734"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21735"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21734"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2023-21735"},
				{Product: macoffice.Word, Vulnerability: "CVE-2022-41061"},
				{Product: macoffice.Outlook, Vulnerability: "CVE-2022-44713"},
				{Product: macoffice.WholeSuite, Vulnerability: "CVE-2022-41107"},
			},
		}

		expected := []string{
			"CVE-2023-21734",
			"CVE-2023-21735",
			"CVE-2022-41061",
			"CVE-2022-41107",
		}

		require.ElementsMatch(t, expected, sut.CollectVulnerabilities(macoffice.Word))
	})

	t.Run("#CmpVersion", func(t *testing.T) {
		softwareVer := "16.69.1"
		t.Run("when the same", func(t *testing.T) {
			testCases := []macoffice.ReleaseNote{
				{
					Date:    time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.69.1 (Build 23011802)",
				},
				{
					Date:    time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.69.1 (Build 23011600)",
				},
			}
			for _, tCase := range testCases {
				require.Equal(t, 0, tCase.CmpVersion(softwareVer))
			}
		})

		t.Run("when release version is older than", func(t *testing.T) {
			testCases := []macoffice.ReleaseNote{
				{
					Date:    time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.69 (Build 23010700)",
				},
				{
					Date:    time.Date(2022, 12, 13, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.68 (Build 22121100)",
				},
			}
			for _, tCase := range testCases {
				require.Equal(t, -1, tCase.CmpVersion(softwareVer))
			}
		})

		t.Run("when release version is newer than", func(t *testing.T) {
			testCases := []macoffice.ReleaseNote{
				{
					Date:    time.Date(2023, 1, 19, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.70.1 (Build 23011802)",
				},
				{
					Date:    time.Date(2023, 1, 17, 0, 0, 0, 0, time.UTC),
					Version: "Version 16.69.2 (Build 23011600)",
				},
			}
			for _, tCase := range testCases {
				require.Equal(t, 1, tCase.CmpVersion(softwareVer))
			}
		})
	})

	t.Run("#OfficeProductFromBundleId", func(t *testing.T) {
		testCases := []struct {
			bundle string
			pType  macoffice.ProductType
			notOk  bool
		}{
			{
				bundle: "com.parallels.winapp.a5c41f715c1b8a880253846c025624e9.c23ed995b43c4ce1bd8d7ead2fa634fa",
				notOk:  true,
			},
			{
				bundle: "com.microsoft.teams",
				notOk:  true,
			},
			{
				bundle: "com.microsoft.Powerpoint",
				pType:  macoffice.PowerPoint,
			},
			{
				bundle: "com.microsoft.Word",
				pType:  macoffice.Word,
			},
			{
				bundle: "com.microsoft.Excel",
				pType:  macoffice.Excel,
			},
			{
				bundle: "com.microsoft.onenote.mac",
				pType:  macoffice.OneNote,
			},
			{
				// TODO: Check if this is the right bundle
				bundle: "com.microsoft.outlook",
				pType:  macoffice.Outlook,
			},
		}

		for _, tc := range testCases {
			r, ok := macoffice.OfficeProductFromBundleId(tc.bundle)
			if tc.notOk {
				require.False(t, ok)
			}
			require.Equal(t, tc.pType, r)
		}
	})
}

func TestBuildNumber(t *testing.T) {
	testCases := []struct {
		Date     time.Time
		Version  string
		Expected string
	}{
		{
			Date:     time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95 (Build 25030928)",
			Expected: "25030928",
		},
		{
			Date:     time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95.1 (Build 25031528)",
			Expected: "25031528",
		},
		{
			Date:     time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95.1",
			Expected: "",
		},
	}
	for _, tCase := range testCases {
		releaseNote := macoffice.ReleaseNote{
			Date:    tCase.Date,
			Version: tCase.Version,
		}
		require.Equal(t, tCase.Expected, releaseNote.BuildNumber(), "Expected %q for %q", tCase.Expected, tCase.Version)
	}
}

func TestShortVersionFormat(t *testing.T) {
	testCases := []struct {
		Date     time.Time
		Version  string
		Expected string
	}{
		{
			Date:     time.Date(2025, 3, 11, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95 (Build 25030928)",
			Expected: "16.95",
		},
		{
			Date:     time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95.1 (Build 25031528)",
			Expected: "16.95.1",
		},
		{
			Date:     time.Date(2025, 3, 18, 0, 0, 0, 0, time.UTC),
			Version:  "Version 16.95.1",
			Expected: "",
		},
	}
	for _, tCase := range testCases {
		releaseNote := macoffice.ReleaseNote{
			Date:    tCase.Date,
			Version: tCase.Version,
		}
		require.Equal(t, tCase.Expected, releaseNote.ShortVersionFormat(), "Expected %q for %q", tCase.Expected, tCase.Version)
	}
}
