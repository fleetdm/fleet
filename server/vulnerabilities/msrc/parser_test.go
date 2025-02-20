package msrc

import (
	"compress/bzip2"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/parsed"
	msrcxml "github.com/fleetdm/fleet/v4/server/vulnerabilities/msrc/xml"
	"github.com/stretchr/testify/require"
)

func extractXMLFixtureFile(t *testing.T, src, dst string) {
	srcF, err := os.Open(src)
	require.NoError(t, err)
	defer srcF.Close()
	dstF, err := os.Create(dst)
	require.NoError(t, err)
	defer dstF.Close()
	r := bzip2.NewReader(srcF)
	// ignoring "G110: Potential DoS vulnerability via decompression bomb", as this is test code.
	_, err = io.Copy(dstF, r) //nolint:gosec
	require.NoError(t, err)
}

func TestParser(t *testing.T) {
	xmlSrcPath := filepath.Join("..", "testdata", "msrc-2022-may.xml.bz2")
	xmlDstPath := filepath.Join(t.TempDir(), "msrc-2022-may.xml")

	extractXMLFixtureFile(t, xmlSrcPath, xmlDstPath)
	f, err := os.Open(xmlDstPath)
	require.NoError(t, err)

	// Parse XML
	xmlResult, err := parseXML(f)
	f.Close()
	require.NoError(t, err)

	// All the products we expect to see after marshaling, grouped by their product name.
	expectedProducts := map[string]parsed.Products{
		"Windows 10": {
			"11568": parsed.NewProductFromFullName("Windows 10 Version 1809 for 32-bit Systems"),
			"11569": parsed.NewProductFromFullName("Windows 10 Version 1809 for x64-based Systems"),
			"11570": parsed.NewProductFromFullName("Windows 10 Version 1809 for ARM64-based Systems"),
			"11712": parsed.NewProductFromFullName("Windows 10 Version 1909 for 32-bit Systems"),
			"11713": parsed.NewProductFromFullName("Windows 10 Version 1909 for x64-based Systems"),
			"11714": parsed.NewProductFromFullName("Windows 10 Version 1909 for ARM64-based Systems"),
			"11896": parsed.NewProductFromFullName("Windows 10 Version 21H1 for x64-based Systems"),
			"11897": parsed.NewProductFromFullName("Windows 10 Version 21H1 for ARM64-based Systems"),
			"11898": parsed.NewProductFromFullName("Windows 10 Version 21H1 for 32-bit Systems"),
			"11800": parsed.NewProductFromFullName("Windows 10 Version 20H2 for x64-based Systems"),
			"11801": parsed.NewProductFromFullName("Windows 10 Version 20H2 for 32-bit Systems"),
			"11802": parsed.NewProductFromFullName("Windows 10 Version 20H2 for ARM64-based Systems"),
			"11929": parsed.NewProductFromFullName("Windows 10 Version 21H2 for 32-bit Systems"),
			"11930": parsed.NewProductFromFullName("Windows 10 Version 21H2 for ARM64-based Systems"),
			"11931": parsed.NewProductFromFullName("Windows 10 Version 21H2 for x64-based Systems"),
			"10729": parsed.NewProductFromFullName("Windows 10 for 32-bit Systems"),
			"10735": parsed.NewProductFromFullName("Windows 10 for x64-based Systems"),
			"10852": parsed.NewProductFromFullName("Windows 10 Version 1607 for 32-bit Systems"),
			"10853": parsed.NewProductFromFullName("Windows 10 Version 1607 for x64-based Systems"),
		},
		"Windows Server 2019": {
			"11571": parsed.NewProductFromFullName("Windows Server 2019"),
			"11572": parsed.NewProductFromFullName("Windows Server 2019  (Server Core installation)"),
		},
		"Windows Server 2022": {
			"11923": parsed.NewProductFromFullName("Windows Server 2022"),
			"11924": parsed.NewProductFromFullName("Windows Server 2022 (Server Core installation)"),
		},
		"Windows Server": {
			"11803": parsed.NewProductFromFullName("Windows Server, version 20H2 (Server Core Installation)"),
		},
		"Windows 11": {
			"11926": parsed.NewProductFromFullName("Windows 11 for x64-based Systems"),
			"11927": parsed.NewProductFromFullName("Windows 11 for ARM64-based Systems"),
		},
		"Windows Server 2016": {
			"10816": parsed.NewProductFromFullName("Windows Server 2016"),
			"10855": parsed.NewProductFromFullName("Windows Server 2016  (Server Core installation)"),
		},
		"Windows 8.1": {
			"10481": parsed.NewProductFromFullName("Windows 8.1 for 32-bit systems"),
			"10482": parsed.NewProductFromFullName("Windows 8.1 for x64-based systems"),
		},
		"Windows RT 8.1": {
			"10484": parsed.NewProductFromFullName("Windows RT 8.1"),
		},
		"Windows Server 2012": {
			"10378": parsed.NewProductFromFullName("Windows Server 2012"),
			"10379": parsed.NewProductFromFullName("Windows Server 2012 (Server Core installation)"),
		},
		"Windows Server 2012 R2": {
			"10483": parsed.NewProductFromFullName("Windows Server 2012 R2"),
			"10543": parsed.NewProductFromFullName("Windows Server 2012 R2 (Server Core installation)"),
		},
		"Windows 7": {
			"10047": parsed.NewProductFromFullName("Windows 7 for 32-bit Systems Service Pack 1"),
			"10048": parsed.NewProductFromFullName("Windows 7 for x64-based Systems Service Pack 1"),
		},
		"Windows Server 2008": {
			"9312":  parsed.NewProductFromFullName("Windows Server 2008 for 32-bit Systems Service Pack 2"),
			"10287": parsed.NewProductFromFullName("Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)"),
			"9318":  parsed.NewProductFromFullName("Windows Server 2008 for x64-based Systems Service Pack 2"),
			"9344":  parsed.NewProductFromFullName("Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation)"),
		},
		"Windows Server 2008 R2": {
			"10051": parsed.NewProductFromFullName("Windows Server 2008 R2 for x64-based Systems Service Pack 1"),
			"10049": parsed.NewProductFromFullName("Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)"),
		},
	}

	// All the products we expect to see in the parsed XML file, grouped by product name.
	expectedXMLProducts := map[string]parsed.Products{
		"Windows 10": {
			"11568": parsed.Product("Windows 10 Version 1809 for 32-bit Systems"),
			"11569": parsed.Product("Windows 10 Version 1809 for x64-based Systems"),
			"11570": parsed.Product("Windows 10 Version 1809 for ARM64-based Systems"),
			"11712": parsed.Product("Windows 10 Version 1909 for 32-bit Systems"),
			"11713": parsed.Product("Windows 10 Version 1909 for x64-based Systems"),
			"11714": parsed.Product("Windows 10 Version 1909 for ARM64-based Systems"),
			"11896": parsed.Product("Windows 10 Version 21H1 for x64-based Systems"),
			"11897": parsed.Product("Windows 10 Version 21H1 for ARM64-based Systems"),
			"11898": parsed.Product("Windows 10 Version 21H1 for 32-bit Systems"),
			"11800": parsed.Product("Windows 10 Version 20H2 for x64-based Systems"),
			"11801": parsed.Product("Windows 10 Version 20H2 for 32-bit Systems"),
			"11802": parsed.Product("Windows 10 Version 20H2 for ARM64-based Systems"),
			"11929": parsed.Product("Windows 10 Version 21H2 for 32-bit Systems"),
			"11930": parsed.Product("Windows 10 Version 21H2 for ARM64-based Systems"),
			"11931": parsed.Product("Windows 10 Version 21H2 for x64-based Systems"),
			"10729": parsed.Product("Windows 10 for 32-bit Systems"),
			"10735": parsed.Product("Windows 10 for x64-based Systems"),
			"10852": parsed.Product("Windows 10 Version 1607 for 32-bit Systems"),
			"10853": parsed.Product("Windows 10 Version 1607 for x64-based Systems"),
		},
		"Windows Server 2019": {
			"11571": parsed.Product("Windows Server 2019"),
			"11572": parsed.Product("Windows Server 2019  (Server Core installation)"),
		},
		"Windows Server 2022": {
			"11923": parsed.Product("Windows Server 2022"),
			"11924": parsed.Product("Windows Server 2022 (Server Core installation)"),
		},
		"Windows Server": {
			"11803": parsed.Product("Windows Server, version 20H2 (Server Core Installation)"),
		},
		"Windows 11": {
			"11926": parsed.Product("Windows 11 for x64-based Systems"),
			"11927": parsed.Product("Windows 11 for ARM64-based Systems"),
		},
		"Windows Server 2016": {
			"10816": parsed.Product("Windows Server 2016"),
			"10855": parsed.Product("Windows Server 2016  (Server Core installation)"),
		},
		"Windows 8.1": {
			"10481": parsed.Product("Windows 8.1 for 32-bit systems"),
			"10482": parsed.Product("Windows 8.1 for x64-based systems"),
		},
		"Windows RT 8.1": {
			"10484": parsed.Product("Windows RT 8.1"),
		},
		"Windows Server 2012": {
			"10378": parsed.Product("Windows Server 2012"),
			"10379": parsed.Product("Windows Server 2012 (Server Core installation)"),
		},
		"Windows Server 2012 R2": {
			"10483": parsed.Product("Windows Server 2012 R2"),
			"10543": parsed.Product("Windows Server 2012 R2 (Server Core installation)"),
		},
		"Windows 7": {
			"10047": parsed.Product("Windows 7 for 32-bit Systems Service Pack 1"),
			"10048": parsed.Product("Windows 7 for x64-based Systems Service Pack 1"),
		},
		"Windows Server 2008": {
			"9312":  parsed.Product("Windows Server 2008 for 32-bit Systems Service Pack 2"),
			"10287": parsed.Product("Windows Server 2008 for 32-bit Systems Service Pack 2 (Server Core installation)"),
			"9318":  parsed.Product("Windows Server 2008 for x64-based Systems Service Pack 2"),
			"9344":  parsed.Product("Windows Server 2008 for x64-based Systems Service Pack 2 (Server Core installation)"),
		},
		"Windows Server 2008 R2": {
			"10051": parsed.Product("Windows Server 2008 R2 for x64-based Systems Service Pack 1"),
			"10049": parsed.Product("Windows Server 2008 R2 for x64-based Systems Service Pack 1 (Server Core installation)"),
		},
	}

	expectedCVEs := map[string][]string{
		"Windows 10": {
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-23279",
			"CVE-2022-29142",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29140",
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
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-22016",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
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
			"CVE-2022-29137",
			"CVE-2022-29139",
		},
		"Windows Server 2019": {
			"CVE-2022-26927",
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-29142",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29140",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-24466",
			"CVE-2022-26913",
			"CVE-2022-26925",
			"CVE-2022-26926",
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
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-29139",
		},
		"Windows Server 2022": {
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-23279",
			"CVE-2022-29142",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29140",
			"CVE-2022-21972",
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
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-22016",
			"CVE-2022-29102",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29106",
			"CVE-2022-29112",
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
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-22017",
			"CVE-2022-26940",
			"CVE-2022-29139",
		},
		"Windows Server": {
			"CVE-2022-24466",
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-23279",
			"CVE-2022-29142",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29140",
			"CVE-2022-21972",
			"CVE-2022-22713",
			"CVE-2022-23270",
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
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-29139",
		},
		"Windows 11": {
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-23279",
			"CVE-2022-29116",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29140",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-24466",
			"CVE-2022-26913",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26927",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-22016",
			"CVE-2022-29103",
			"CVE-2022-29104",
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
			"CVE-2022-29137",
			"CVE-2022-22017",
			"CVE-2022-26940",
			"CVE-2022-29139",
		},
		"Windows Server 2016": {
			"CVE-2022-29137",
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-24466",
			"CVE-2022-26925",
			"CVE-2022-26926",
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
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-29139",
			"CVE-2022-29140",
		},
		"Windows 8.1": {
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
			"CVE-2022-29112",
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29137",
			"CVE-2022-29139",
		},
		"Windows RT 8.1": {
			"CVE-2022-26934",
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26933",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
			"CVE-2022-29112",
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29137",
			"CVE-2022-29139",
		},
		"Windows Server 2012": {
			"CVE-2022-26936",
			"CVE-2022-30190",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26937",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29102",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
			"CVE-2022-29112",
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-29139",
		},
		"Windows Server 2012 R2": {
			"CVE-2022-30190",
			"CVE-2022-26923",
			"CVE-2022-29150",
			"CVE-2022-29151",
			"CVE-2022-29122",
			"CVE-2022-29120",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26930",
			"CVE-2022-26931",
			"CVE-2022-26933",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26937",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29102",
			"CVE-2022-29103",
			"CVE-2022-29104",
			"CVE-2022-29105",
			"CVE-2022-29112",
			"CVE-2022-29114",
			"CVE-2022-29115",
			"CVE-2022-29125",
			"CVE-2022-29126",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29134",
			"CVE-2022-29135",
			"CVE-2022-29137",
			"CVE-2022-29138",
			"CVE-2022-29123",
			"CVE-2022-29139",
			"CVE-2022-26936",
		},
		"Windows 7": {
			"CVE-2022-29105",
			"CVE-2022-30190",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26931",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29103",
			"CVE-2022-29112",
			"CVE-2022-29115",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29137",
			"CVE-2022-29139",
		},
		"Windows Server 2008": {
			"CVE-2022-29115",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26931",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-26937",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-29103",
			"CVE-2022-29112",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29137",
			"CVE-2022-29139",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
		},
		"Windows Server 2008 R2": {
			"CVE-2022-30190",
			"CVE-2022-21972",
			"CVE-2022-23270",
			"CVE-2022-26925",
			"CVE-2022-26926",
			"CVE-2022-26931",
			"CVE-2022-26934",
			"CVE-2022-26935",
			"CVE-2022-26936",
			"CVE-2022-26937",
			"CVE-2022-22011",
			"CVE-2022-22012",
			"CVE-2022-22013",
			"CVE-2022-22014",
			"CVE-2022-22015",
			"CVE-2022-29103",
			"CVE-2022-29112",
			"CVE-2022-29115",
			"CVE-2022-29127",
			"CVE-2022-29128",
			"CVE-2022-29129",
			"CVE-2022-29130",
			"CVE-2022-29132",
			"CVE-2022-29137",
			"CVE-2022-29139",
			"CVE-2022-29141",
			"CVE-2022-22019",
			"CVE-2022-29121",
			"CVE-2022-30138",
			"CVE-2022-29105",
		},
	}

	// A random vulnerability ("CVE-2022-29137")
	expectedVulns := map[string]map[string]parsed.Vulnerability{
		"Windows 10": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"11568": true,
					"11569": true,
					"11570": true,
					"11712": true,
					"11713": true,
					"11714": true,
					"11896": true,
					"11897": true,
					"11898": true,
					"11800": true,
					"11801": true,
					"11802": true,
					"11929": true,
					"11930": true,
					"11931": true,
					"10729": true,
					"10735": true,
					"10852": true,
					"10853": true,
				},
				RemediatedBy: map[uint]bool{
					5013941: true,
					5013952: true,
					5013942: true,
					5013963: true,
					5013945: true,
				},
			},
		},
		"Windows Server 2019": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"11571": true,
					"11572": true,
				},
				RemediatedBy: map[uint]bool{
					5013941: true,
				},
			},
		},

		"Windows Server 2022": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"11923": true,
					"11924": true,
				},
				RemediatedBy: map[uint]bool{
					5013944: true,
				},
			},
		},

		"Windows Server": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"11803": true,
				},
				RemediatedBy: map[uint]bool{
					5013942: true,
				},
			},
		},

		"Windows Server 2008": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"9312":  true,
					"10287": true,
					"9318":  true,
					"9344":  true,
				},
				RemediatedBy: map[uint]bool{
					5014010: true,
					5014006: true,
				},
			},
		},

		"Windows Server 2008 R2": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10051": true,
					"10049": true,
				},
				RemediatedBy: map[uint]bool{
					5014012: true,
					5013999: true,
				},
			},
		},

		"Windows Server 2012": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10378": true,
					"10379": true,
				},
				RemediatedBy: map[uint]bool{
					5014017: true,
					5014018: true,
				},
			},
		},

		"Windows Server 2012 R2": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10483": true,
					"10543": true,
				},
				RemediatedBy: map[uint]bool{
					5014011: true,
					5014001: true,
				},
			},
		},

		"Windows 7": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10047": true,
					"10048": true,
				},
				RemediatedBy: map[uint]bool{
					5014012: true,
					5013999: true,
				},
			},
		},

		"Windows Server 2016": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10816": true,
					"10855": true,
				},
				RemediatedBy: map[uint]bool{
					5013952: true,
				},
			},
		},

		"Windows 11": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"11926": true,
					"11927": true,
				},
				RemediatedBy: map[uint]bool{
					5013943: true,
				},
			},
		},

		"Windows RT 8.1": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10484": true,
				},
				RemediatedBy: map[uint]bool{
					5014025: true,
				},
			},
		},

		"Windows 8.1": {
			"CVE-2022-29137": {
				PublishedEpoch: ptr.Int64(1652169600),
				ProductIDs: map[string]bool{
					"10481": true,
					"10482": true,
				},
				RemediatedBy: map[uint]bool{
					5014011: true,
					5014001: true,
				},
			},
		},
	}

	// A random vulnerability ("CVE-2022-29137")
	expectedVendorFixes := map[string]map[uint]parsed.VendorFix{
		"Windows 10": {
			5013941: {
				FixedBuilds: []string{"10.0.17763.2928"},
				ProductIDs: map[string]bool{
					"11568": true,
					"11569": true,
					"11570": true,
				},
				Supersedes: ptr.Uint(5012647),
			},
			5013952: {
				FixedBuilds: []string{"10.0.14393.5125"},
				ProductIDs: map[string]bool{
					"10852": true,
					"10853": true,
				},
				Supersedes: ptr.Uint(5012596),
			},
			5013942: {
				FixedBuilds: []string{"10.0.19043.1706", "10.0.19042.1706", "10.0.19044.1706"},
				ProductIDs: map[string]bool{
					"11896": true,
					"11897": true,
					"11898": true,
					"11929": true,
					"11800": true,
					"11801": true,
					"11802": true,
					"11930": true,
					"11931": true,
				},
				Supersedes: ptr.Uint(5012599),
			},
			5013963: {
				FixedBuilds: []string{"10.0.10240.19297"},
				ProductIDs: map[string]bool{
					"10729": true,
					"10735": true,
				},
				Supersedes: ptr.Uint(5012653),
			},

			5013945: {
				FixedBuilds: []string{"10.0.18363.2274"},
				ProductIDs: map[string]bool{
					"11712": true,
					"11713": true,
					"11714": true,
				},
				Supersedes: ptr.Uint(5012591),
			},
		},
		"Windows Server 2019": {
			5013941: {
				FixedBuilds: []string{"10.0.17763.2928"},
				ProductIDs: map[string]bool{
					"11571": true,
					"11572": true,
				},
				Supersedes: ptr.Uint(5012647),
			},
		},

		"Windows Server 2022": {
			5013944: {
				FixedBuilds: []string{"10.0.20348.707"},
				ProductIDs: map[string]bool{
					"11923": true,
					"11924": true,
				},
				Supersedes: ptr.Uint(5012604),
			},
		},

		"Windows Server": {
			5013942: {
				FixedBuilds: []string{"10.0.19042.1706"},
				ProductIDs: map[string]bool{
					"11803": true,
				},
				Supersedes: ptr.Uint(5012599),
			},
		},

		"Windows Server 2008": {
			5014010: {
				FixedBuilds: []string{"6.0.6003.21481"},
				ProductIDs: map[string]bool{
					"9312":  true,
					"10287": true,
					"9318":  true,
					"9344":  true,
				},
				Supersedes: ptr.Uint(5012658),
			},
			5014006: {
				FixedBuilds: []string{"6.0.6003.21481"},
				ProductIDs: map[string]bool{
					"9312":  true,
					"10287": true,
					"9318":  true,
					"9344":  true,
				},
			},
		},

		"Windows Server 2008 R2": {
			5014012: {
				FixedBuilds: []string{"6.1.7601.25954"},
				ProductIDs: map[string]bool{
					"10051": true,
					"10049": true,
				},
				Supersedes: ptr.Uint(5012626),
			},
			5013999: {
				FixedBuilds: []string{"6.1.7601.25954"},
				ProductIDs: map[string]bool{
					"10051": true,
					"10049": true,
				},
			},
		},

		"Windows Server 2012": {
			5014017: {
				FixedBuilds: []string{"6.2.9200.23714"},
				ProductIDs: map[string]bool{
					"10378": true,
					"10379": true,
				},
				Supersedes: ptr.Uint(5012650),
			},
			5014018: {
				FixedBuilds: []string{"6.2.9200.23714"},
				ProductIDs: map[string]bool{
					"10378": true,
					"10379": true,
				},
			},
		},

		"Windows Server 2012 R2": {
			5014011: {
				FixedBuilds: []string{"6.3.9600.20371"},
				ProductIDs: map[string]bool{
					"10483": true,
					"10543": true,
				},
				Supersedes: ptr.Uint(5012670),
			},
			5014001: {
				FixedBuilds: []string{"6.3.9600.20365"},
				ProductIDs: map[string]bool{
					"10483": true,
					"10543": true,
				},
			},
		},

		"Windows 7": {
			5014012: {
				FixedBuilds: []string{"6.1.7601.25954"},
				ProductIDs: map[string]bool{
					"10047": true,
					"10048": true,
				},
				Supersedes: ptr.Uint(5012626),
			},
			5013999: {
				FixedBuilds: []string{"6.1.7601.25954"},
				ProductIDs: map[string]bool{
					"10047": true,
					"10048": true,
				},
			},
		},

		"Windows Server 2016": {
			5013952: {
				FixedBuilds: []string{"10.0.14393.5125"},
				ProductIDs: map[string]bool{
					"10816": true,
					"10855": true,
				},
			},
		},

		"Windows 11": {
			5013943: {
				FixedBuilds: []string{"10.0.22000.675"},
				ProductIDs: map[string]bool{
					"11926": true,
					"11927": true,
				},
				Supersedes: ptr.Uint(5012592),
			},
		},

		"Windows RT 8.1": {
			5014025: {
				FixedBuilds: []string{"6.3.9600.20367"},
				ProductIDs: map[string]bool{
					"10484": true,
				},
			},
		},

		"Windows 8.1": {
			5014011: {
				FixedBuilds: []string{"6.3.9600.20371"},
				ProductIDs: map[string]bool{
					"10481": true,
					"10482": true,
				},
				Supersedes: ptr.Uint(5012670),
			},
			5014001: {
				FixedBuilds: []string{"6.3.9600.20365"},
				ProductIDs: map[string]bool{
					"10481": true,
					"10482": true,
				},
			},
		},
	}

	t.Run("ParseFeed", func(t *testing.T) {
		t.Run("errors out if file does not exists", func(t *testing.T) {
			_, err := ParseFeed("asdcv")
			require.Error(t, err)
		})
	})

	t.Run("mapToSecurityBulletins", func(t *testing.T) {
		bulletins, err := mapToSecurityBulletins(xmlResult)
		require.NoError(t, err)

		t.Run("should map the vendor fixes entries correctly", func(t *testing.T) {
			for pName, vF := range expectedVendorFixes {
				bulletin := bulletins[pName]

				for KBID, fix := range vF {
					sut := bulletin.VendorFixes[KBID]
					require.Equal(t, fix.FixedBuilds, sut.FixedBuilds, pName, KBID)
					require.Equal(t, fix.ProductIDs, sut.ProductIDs, pName, KBID)
					// We want to check that either both are nil or that both are not nil
					require.False(t, (fix.Supersedes == nil || sut.Supersedes == nil) && !(fix.Supersedes == nil || sut.Supersedes == nil), pName, KBID)
					if fix.Supersedes != nil {
						require.Equal(t, *fix.Supersedes, *sut.Supersedes, pName, KBID)
					}
				}
			}
		})

		t.Run("should map the vulnerability entries correctly", func(t *testing.T) {
			for pName, v := range expectedVulns {
				bulletin := bulletins[pName]

				for cve, vuln := range v {
					sut := bulletin.Vulnerabities[cve]
					require.Equal(t, *vuln.PublishedEpoch, *sut.PublishedEpoch, pName)
					require.Equal(t, vuln.RemediatedBy, sut.RemediatedBy, pName)
					require.Equal(t, vuln.ProductIDs, sut.ProductIDs, pName)
				}
			}
		})

		t.Run("should have one bulletin per product", func(t *testing.T) {
			var expected []string
			for p := range expectedProducts {
				expected = append(expected, p)
			}

			var actual []string
			for _, g := range bulletins {
				actual = append(actual, g.ProductName)
			}

			require.Len(t, bulletins, len(expected))
			require.ElementsMatch(t, expected, actual)
		})

		t.Run("each bulletin should have the right products", func(t *testing.T) {
			for _, g := range bulletins {
				require.Equal(t, expectedProducts[g.ProductName], g.Products, g.ProductName)
			}
		})

		t.Run("each bulletin should have the right vulnerabilities", func(t *testing.T) {
			for _, g := range bulletins {
				var actual []string
				for v := range g.Vulnerabities {
					actual = append(actual, v)
				}
				require.ElementsMatch(t, actual, expectedCVEs[g.ProductName], g.ProductName)
			}
		})
	})

	t.Run("parseXML", func(t *testing.T) {
		t.Run("only windows products are included", func(t *testing.T) {
			var expected []msrcxml.Product
			for _, grp := range expectedXMLProducts {
				for pID, pFn := range grp {
					expected = append(
						expected,
						msrcxml.Product{ProductID: pID, FullName: string(pFn)},
					)
				}
			}

			var actual []msrcxml.Product
			for _, v := range xmlResult.WinProducts {
				actual = append(actual, v)
			}
			require.ElementsMatch(t, actual, expected)
		})

		t.Run("only CVEs for windows products are included", func(t *testing.T) {
			expected := make(map[string]bool)
			for _, p := range expectedCVEs {
				for _, v := range p {
					expected[v] = true
				}
			}
			actual := make(map[string]bool)
			for _, v := range xmlResult.WinVulnerabities {
				actual[v.CVE] = true
			}
			require.Equal(t, expected, actual)
		})

		t.Run("scores are parsed correctly", func(t *testing.T) {
			// Check the score of a random CVE (CVE-2022-24466)
			for _, v := range xmlResult.WinVulnerabities {
				if v.CVE == "CVE-2022-24466" {
					require.Equal(t, 4.1, v.Score)
				}
			}
		})

		t.Run("the revision history is parsed correctly", func(t *testing.T) {
			// Check the revision history of a random CVE (CVE-2022-29114)
			for _, v := range xmlResult.WinVulnerabities {
				if v.CVE == "CVE-2022-29114" {
					require.Len(t, v.Revisions, 1)
					require.Equal(t, "2022-05-10T08:00:00", v.Revisions[0].Date)
					require.Equal(t, "<p>Information published.</p>\n", v.Revisions[0].Description)
				}
			}
		})

		t.Run("the remediations are parsed correctly", func(t *testing.T) {
			// Check the remediations of a random CVE (CVE-2022-29126)
			expectedRemediations := []msrcxml.VulnerabilityRemediation{
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.17763.2928",
					ProductIDs:      []string{"11568", "11569", "11570", "11571", "11572"},
					Description:     "5013941",
					Supercedence:    "5012647",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013941",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11568", "11569", "11570", "11571", "11572"},
					Description: "5013941",
					URL:         "https://support.microsoft.com/help/5013941",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.18363.2274",
					ProductIDs:      []string{"11712", "11713", "11714"},
					Description:     "5013945",
					Supercedence:    "5012591",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013945",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.19043.1706",
					ProductIDs:      []string{"11896", "11897", "11898", "11929"},
					Description:     "5013942",
					Supercedence:    "5012599",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013942",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"},
					Description: "5013942",
					URL:         "https://support.microsoft.com/help/5013942",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.20348.707",
					ProductIDs:      []string{"11923", "11924"},
					Description:     "5013944",
					Supercedence:    "5012604",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013944",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11923", "11924"},
					Description: "5013944",
					URL:         "https://support.microsoft.com/help/5013944",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.19042.1706",
					ProductIDs:      []string{"11800", "11801", "11802", "11803"},
					Description:     "5013942",
					Supercedence:    "5012599",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013942",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"},
					Description: "5013942",
					URL:         "https://support.microsoft.com/help/5013942",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.22000.675",
					ProductIDs:      []string{"11926", "11927"},
					Description:     "5013943",
					Supercedence:    "5012592",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013943",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11926", "11927"},
					Description: "5013943",
					URL:         "https://support.microsoft.com/help/5013943",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.19044.1706",
					ProductIDs:      []string{"11930", "11931"},
					Description:     "5013942",
					Supercedence:    "5012599",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013942",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"11896", "11897", "11898", "11800", "11801", "11802", "11803", "11929", "11930", "11931"},
					Description: "5013942",
					URL:         "https://support.microsoft.com/help/5013942",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.10240.19297",
					ProductIDs:      []string{"10729", "10735"},
					Description:     "5013963",
					Supercedence:    "5012653",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013963",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "10.0.14393.5125",
					ProductIDs:      []string{"10852", "10853", "10816", "10855"},
					Description:     "5013952",
					Supercedence:    "5012596",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5013952",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"10852", "10853", "10816", "10855"},
					Description: "5013952",
					URL:         "https://support.microsoft.com/help/5013952",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "6.3.9600.20371",
					ProductIDs:      []string{"10481", "10482", "10483", "10543"},
					Description:     "5014011",
					Supercedence:    "5012670",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5014011",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"10481", "10482", "10483", "10543"},
					Description: "5014011",
					URL:         "https://support.microsoft.com/help/5014011",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "6.3.9600.20365",
					ProductIDs:      []string{"10481", "10482", "10483", "10543"},
					Description:     "5014001",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5014001",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"10481", "10482", "10483", "10543"},
					Description: "5014001",
					URL:         "https://support.microsoft.com/help/5014001",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "6.3.9600.20367",
					ProductIDs:      []string{"10484"},
					Description:     "5014025",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5014025",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "6.2.9200.23714",
					ProductIDs:      []string{"10378", "10379"},
					Description:     "5014017",
					Supercedence:    "5012650",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5014017",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"10378", "10379"},
					Description: "5014017",
					URL:         "https://support.microsoft.com/help/5014017",
				},
				{
					Type:            "Vendor Fix",
					FixedBuild:      "6.2.9200.23714",
					ProductIDs:      []string{"10378", "10379"},
					Description:     "5014018",
					RestartRequired: "Yes",
					URL:             "https://catalog.update.microsoft.com/v7/site/Search.aspx?q=KB5014018",
				},
				{
					Type:        "Known Issue",
					ProductIDs:  []string{"10378", "10379"},
					Description: "5014018",
					URL:         "https://support.microsoft.com/help/5014018",
				},
			}
			for _, v := range xmlResult.WinVulnerabities {
				if v.CVE == "CVE-2022-29126" {
					require.Len(t, v.Remediations, len(expectedRemediations))
					require.ElementsMatch(t, v.Remediations, expectedRemediations)
				}
			}
		})
	})
}
