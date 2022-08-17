package msrc

import (
	"compress/bzip2"
	"io"
	"os"
	"path/filepath"
	"testing"

	msrc_input "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/input"
	"github.com/stretchr/testify/require"
)

func TestParser(t *testing.T) {
	t.Run("parseMSRCXML", func(t *testing.T) {
		srcPath := filepath.Join("..", "testdata", "msrc-2022-may.xml.bz2")
		dstPath := filepath.Join(t.TempDir(), "msrc-2022-may.xml")

		// Extract XML fixture
		srcF, err := os.Open(srcPath)
		require.NoError(t, err)
		defer srcF.Close()
		dstF, err := os.Create(dstPath)
		require.NoError(t, err)
		defer dstF.Close()
		r := bzip2.NewReader(srcF)
		// ignoring "G110: Potential DoS vulnerability via decompression bomb", as this is test code.
		_, err = io.Copy(dstF, r) //nolint:gosec
		require.NoError(t, err)

		// Parse XML
		fHandler, err := os.Open(dstPath)
		defer fHandler.Close()
		require.NoError(t, err)
		result, err := parseMSRCXML(fHandler)
		require.NoError(t, err)

		//-------------------------------------------
		// Check that only Windows Products are included
		expectedProducts := []msrc_input.ProductXML{
			{ProductID: "11568", FullName: "Windows 10 Version 1809 for 32-bit Systems"},
			{ProductID: "11569", FullName: "Windows 10 Version 1809 for x64-based Systems"},
			{ProductID: "11570", FullName: "Windows 10 Version 1809 for ARM64-based Systems"},
			{ProductID: "11571", FullName: "Windows Server 2019"},
			{ProductID: "11572", FullName: "Windows Server 2019  (Server Core installation)"},
			{ProductID: "11712", FullName: "Windows 10 Version 1909 for 32-bit Systems"},
			{ProductID: "11713", FullName: "Windows 10 Version 1909 for x64-based Systems"},
			{ProductID: "11714", FullName: "Windows 10 Version 1909 for ARM64-based Systems"},
			{ProductID: "11896", FullName: "Windows 10 Version 21H1 for x64-based Systems"},
			{ProductID: "11897", FullName: "Windows 10 Version 21H1 for ARM64-based Systems"},
			{ProductID: "11898", FullName: "Windows 10 Version 21H1 for 32-bit Systems"},
			{ProductID: "11923", FullName: "Windows Server 2022"},
			{ProductID: "11924", FullName: "Windows Server 2022 (Server Core installation)"},
			{ProductID: "11800", FullName: "Windows 10 Version 20H2 for x64-based Systems"},
			{ProductID: "11801", FullName: "Windows 10 Version 20H2 for 32-bit Systems"},
			{ProductID: "11802", FullName: "Windows 10 Version 20H2 for ARM64-based Systems"},
			{ProductID: "11803", FullName: "Windows Server, version 20H2 (Server Core Installation)"},
			{ProductID: "11926", FullName: "Windows 11 for x64-based Systems"},
			{ProductID: "11927", FullName: "Windows 11 for ARM64-based Systems"},
			{ProductID: "11929", FullName: "Windows 10 Version 21H2 for 32-bit Systems"},
			{ProductID: "11930", FullName: "Windows 10 Version 21H2 for ARM64-based Systems"},
			{ProductID: "11931", FullName: "Windows 10 Version 21H2 for x64-based Systems"},
			{ProductID: "10729", FullName: "Windows 10 for 32-bit Systems"},
			{ProductID: "10735", FullName: "Windows 10 for x64-based Systems"},
			{ProductID: "10852", FullName: "Windows 10 Version 1607 for 32-bit Systems"},
			{ProductID: "10853", FullName: "Windows 10 Version 1607 for x64-based Systems"},
			{ProductID: "10816", FullName: "Windows Server 2016"},
			{ProductID: "10855", FullName: "Windows Server 2016  (Server Core installation)"},
			{ProductID: "10481", FullName: "Windows 8.1 for 32-bit systems"},
			{ProductID: "10482", FullName: "Windows 8.1 for x64-based systems"},
			{ProductID: "10484", FullName: "Windows RT 8.1"},
			{ProductID: "10378", FullName: "Windows Server 2012"},
			{ProductID: "10379", FullName: "Windows Server 2012 (Server Core installation)"},
			{ProductID: "10483", FullName: "Windows Server 2012 R2"},
			{ProductID: "10543", FullName: "Windows Server 2012 R2 (Server Core installation)"},
			{ProductID: "10047", FullName: "Windows 7 for 32-bit Systems Service Pack 1"},
			{ProductID: "10048", FullName: "Windows 7 for x64-based Systems Service Pack 1"},
			{ProductID: "9312", FullName: "Windows Server 2008 for 32-bit Systems Service Pack 2"},
			{ProductID: "10287", FullName: "Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)"},
			{ProductID: "9318", FullName: "Windows Server 2008 for x64-based Systems Service Pack 2"},
			{ProductID: "9344", FullName: "Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation)"},
			{ProductID: "10051", FullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1"},
			{ProductID: "10049", FullName: "Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)"},
		}
		var actualProducts []msrc_input.ProductXML
		for _, v := range result.WinProducts {
			actualProducts = append(actualProducts, v)
		}
		require.ElementsMatch(t, actualProducts, expectedProducts)

		//-------------------------------------------------------
		// Check that only CVEs for Windows products are included.
		expectedCVEs := []string{
			"CVE-2022-21972",
			"CVE-2022-22713",
			"CVE-2022-23270",
			"CVE-2022-24466",
			"CVE-2022-26913",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26927",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26932",
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-26937",
			"CVE-2022-26938",
			"CVE-2022-26939",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-22016",
			"CVE-2022-29102",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
			"CVE-2022-29106",
			"CVE-2022-29112",
			"CVE-2022-29113",
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29131",
			"CVE-2022-29132",
			"CVE-2022-29133",
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29139",
			"CVE-2022-29140",
			"CVE-2022-29141",
			"CVE-2022-29142",
			"CVE-2022-22019",
			"CVE-2022-23279",
			"CVE-2022-26923",
			"CVE-2022-29116",
			"CVE-2022-29120",
			"CVE-2022-29121",
			"CVE-2022-29122",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-30138",
			"CVE-2022-30190",
			"CVE-2022-26940",
			"CVE-2022-22017",
			"CVE-2022-29123",
		}
		var actualCVEs []string
		for _, r := range result.WinVulnerabities {
			actualCVEs = append(actualCVEs, r.CVE)
		}
		require.ElementsMatch(t, expectedCVEs, actualCVEs)

		//--------------------------------------------------
		// Check the score of a random CVE (CVE-2022-24466)
		for _, v := range result.WinVulnerabities {
			if v.CVE == "CVE-2022-24466" {
				require.Equal(t, 4.1, v.Score)
			}
		}

		//------------------------------------------------------------
		// Check the revision history of a random CVE (CVE-2022-29114)
		for _, v := range result.WinVulnerabities {
			if v.CVE == "CVE-2022-29114" {
				require.Len(t, v.Revisions, 1)
				require.Equal(t, "2022-05-10T08:00:00", v.Revisions[0].Date)
				require.Equal(t, "<p>Information published.</p>\n", v.Revisions[0].Description)
			}
		}

		//-----------------------------------------------------------------
		// Check the remediations of a random CVE (CVE-2022-29126)
		expectedRemediations := []msrc_input.VulnerabilityRemediationXML{
			{Type: "Vendor Fix", FixedBuild: "10.0.17763.2928", ProductIDs: []string{"11568", "11569", "11570", "11571", "11572"}, Description: "5013941", Supercedence: "5012647", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11568", "11569", "11570", "11571", "11572"}, Description: "5013941"},
			{Type: "Vendor Fix", FixedBuild: "10.0.18363.2274", ProductIDs: []string{"11712", "11713", "11714"}, Description: "5013945", Supercedence: "5012591", RestartRequired: "Yes"},
			{Type: "Vendor Fix", FixedBuild: "10.0.19043.1706", ProductIDs: []string{"11896", "11897", "11898", "11929"}, Description: "5013942", Supercedence: "5012599", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"}, Description: "5013942"},
			{Type: "Vendor Fix", FixedBuild: "10.0.20348.707", ProductIDs: []string{"11923", "11924"}, Description: "5013944", Supercedence: "5012604", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11923", "11924"}, Description: "5013944"},
			{Type: "Vendor Fix", FixedBuild: "10.0.19042.1706", ProductIDs: []string{"11800", "11801", "11802", "11803"}, Description: "5013942", Supercedence: "5012599", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"}, Description: "5013942"},
			{Type: "Vendor Fix", FixedBuild: "10.0.22000.675", ProductIDs: []string{"11926", "11927"}, Description: "5013943", Supercedence: "5012592", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11926", "11927"}, Description: "5013943"},
			{Type: "Vendor Fix", FixedBuild: "10.0.19044.1706", ProductIDs: []string{"11930", "11931"}, Description: "5013942", Supercedence: "5012599", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"}, Description: "5013942"},
			{Type: "Vendor Fix", FixedBuild: "10.0.10240.19297", ProductIDs: []string{"10729", "10735"}, Description: "5013963", Supercedence: "5012653", RestartRequired: "Yes"},
			{Type: "Vendor Fix", FixedBuild: "10.0.14393.5125", ProductIDs: []string{"10852", "10853", "10816", "10855"}, Description: "5013952", Supercedence: "5012596", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"10852", "10853", "10816", "10855"}, Description: "5013952"},
			{Type: "Vendor Fix", FixedBuild: "6.3.9600.20371", ProductIDs: []string{"10481", "10482", "10483", "10543"}, Description: "5014011", Supercedence: "5012670", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"10481", "10482", "10483", "10543"}, Description: "5014011"},
			{Type: "Vendor Fix", FixedBuild: "6.3.9600.20365", ProductIDs: []string{"10481", "10482", "10483", "10543"}, Description: "5014001", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"10481", "10482", "10483", "10543"}, Description: "5014001"},
			{Type: "Vendor Fix", FixedBuild: "6.3.9600.20367", ProductIDs: []string{"10484"}, Description: "5014025", RestartRequired: "Yes"},
			{Type: "Vendor Fix", FixedBuild: "6.2.9200.23714", ProductIDs: []string{"10378", "10379"}, Description: "5014017", Supercedence: "5012650", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"10378", "10379"}, Description: "5014017"},
			{Type: "Vendor Fix", FixedBuild: "6.2.9200.23714", ProductIDs: []string{"10378", "10379"}, Description: "5014018", RestartRequired: "Yes"},
			{Type: "Known Issue", ProductIDs: []string{"10378", "10379"}, Description: "5014018"},
		}
		for _, v := range result.WinVulnerabities {
			if v.CVE == "CVE-2022-29126" {
				require.Len(t, v.Remediations, len(expectedRemediations))
				require.ElementsMatch(t, v.Remediations, expectedRemediations)
			}
		}
	})
}
