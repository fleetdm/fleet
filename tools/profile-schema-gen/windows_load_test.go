package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ddfFile wraps nodes in a DDF document with both namespaces, as Microsoft writes them.
func ddfFile(nodes string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE MgmtTree PUBLIC " -//OMA//DTD-DM-DDF 1.2//EN" "http://www.openmobilealliance.org/tech/DTD/DM_DDF-V1_2.dtd"[<?oma-dm-ddf-ver supported-versions="1.2"?>]>
<MgmtTree xmlns:MSFT="http://schemas.microsoft.com/MobileDevice/DM">
  <VerDTD>1.2</VerDTD>
  <MSFT:Diagnostics />
` + nodes + `
</MgmtTree>`
}

func ddfLeafNode(name, format string) string {
	return `<Node><NodeName>` + name + `</NodeName><DFProperties><AccessType><Get /></AccessType><DFFormat><` + format + ` /></DFFormat>
	<MSFT:Applicability><MSFT:OsBuildVersion>10.0.10240</MSFT:OsBuildVersion></MSFT:Applicability></DFProperties></Node>`
}

func ddfInteriorNode(name, path, children string) string {
	p := ""
	if path != "" {
		p = "<Path>" + path + "</Path>"
	}
	return `<Node><NodeName>` + name + `</NodeName>` + p + `<DFProperties><AccessType><Get /></AccessType><DFFormat><node /></DFFormat></DFProperties>` + children + `</Node>`
}

func writeDDFs(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	return dir
}

func leafURIs(leaves []ddfLeaf) []string {
	uris := make([]string, 0, len(leaves))
	for _, l := range leaves {
		uris = append(uris, l.LocURI)
	}
	return uris
}

func TestLoadWindows(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	dir := writeDDFs(t, map[string]string{
		// Policy-style: <Path> on the top-level node, nested areas.
		"Policy_DeviceLock.xml": "\xef\xbb\xbf" + ddfFile(ddfInteriorNode("Policy", "./Device/Vendor/MSFT",
			ddfInteriorNode("Config", "", ddfInteriorNode("DeviceLock", "",
				ddfLeafNode("MaxInactivityTimeDeviceLock", "int")+ddfLeafNode("DevicePasswordEnabled", "int"))))),
		// Firewall-style: bare ./Vendor root and a dynamic segment.
		"Firewall.XML": ddfFile(ddfInteriorNode("Firewall", "./Vendor/MSFT",
			ddfInteriorNode("FirewallRules", "", ddfInteriorNode("", "",
				ddfInteriorNode("Action", "", ddfLeafNode("Type", "int")))+
				// An interior node with no children isn't a setting.
				ddfInteriorNode("Empty", "", "")))),
		"readme.txt": "not a DDF",
	})

	leaves, err := loadWindows(t.Context(), logger, dir)
	require.NoError(t, err)
	require.Len(t, leaves, 3)
	assert.Equal(t, []string{
		"./Device/Vendor/MSFT/Policy/Config/DeviceLock/DevicePasswordEnabled",
		"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock",
		"./Vendor/MSFT/Firewall/FirewallRules/*/Action/Type",
	}, leafURIs(leaves))
	assert.Equal(t, "Policy_DeviceLock.xml", leaves[0].File)
	assert.Equal(t, "Firewall.XML", leaves[2].File)
	assert.Equal(t, "int", leaves[0].Props.Format.value())

	again, err := loadWindows(t.Context(), logger, dir)
	require.NoError(t, err)
	assert.Equal(t, leaves, again)
}

func TestLoadWindowsWhitespaceAndRootLeaf(t *testing.T) {
	// Names and paths are trimmed; a top-level leaf with only a Path still gets one.
	dir := writeDDFs(t, map[string]string{
		"a.xml": ddfFile(ddfInteriorNode(" Area ", " ./User/Vendor/MSFT ", ddfLeafNode("\n  Setting\n", "chr"))),
		"b.xml": ddfFile(`<Node><NodeName>Top</NodeName><Path>./DevDetail</Path><DFProperties><DFFormat><chr /></DFFormat></DFProperties></Node>`),
	})
	leaves, err := loadWindows(t.Context(), slog.New(slog.DiscardHandler), dir)
	require.NoError(t, err)
	assert.Equal(t, []string{"./DevDetail/Top", "./User/Vendor/MSFT/Area/Setting"}, leafURIs(leaves))
}

func TestLoadWindowsErrors(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	cases := []struct {
		name    string
		files   map[string]string
		wantErr string
	}{
		{
			name: "same LocURI in two files",
			files: map[string]string{
				"a.xml": ddfFile(ddfInteriorNode("X", "./Device/Vendor/MSFT", ddfLeafNode("Y", "int"))),
				"b.xml": ddfFile(ddfInteriorNode("X", "./Device/Vendor/MSFT", ddfLeafNode("Y", "int"))),
			},
			wantErr: "./Device/Vendor/MSFT/X/Y is defined in both a.xml and b.xml",
		},
		{name: "no files", files: map[string]string{"readme.txt": "x"}, wantErr: "no DDF files in"},
		{name: "invalid XML", files: map[string]string{"bad.xml": "<MgmtTree><Node>"}, wantErr: "parse bad.xml"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadWindows(t.Context(), logger, writeDDFs(t, c.files))
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}
