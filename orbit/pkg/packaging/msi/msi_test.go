package msi

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sassoftware/relic/v8/lib/comdoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteMSI_EndToEnd(t *testing.T) {
	// Create a temporary root directory with test files.
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))

	// Create some test files.
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "secret.txt"), []byte("test-secret"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "osquery.flags"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "certs.pem"), []byte("test-certs"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "bin", "orbit", "windows", "stable"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, "bin", "orbit", "windows", "stable", "orbit.exe"),
		[]byte("fake-orbit-binary"),
		0o644,
	))

	opts := MSIOptions{
		ProductName:    "Fleet osquery",
		ProductVersion: "1.28.0",
		Manufacturer:   "Fleet Device Management (fleetdm.com)",
		UpgradeCode:    "{B681CB20-107E-428A-9B14-2D3C1AFED244}",
		Architecture:   "amd64",
		FleetURL:       "https://fleet.example.com",
		EnrollSecret:   "test-secret",
		OrbitChannel:   "stable",
		OsquerydChannel: "stable",
		DesktopChannel: "stable",
		OrbitPath:      `bin\orbit\windows\stable\orbit.exe`,
	}

	var buf bytes.Buffer
	err := WriteMSI(&buf, rootDir, opts)
	require.NoError(t, err)

	// Verify the output is a valid CFB.
	msiData := buf.Bytes()
	require.Greater(t, len(msiData), 512)

	// Check CFB magic.
	assert.Equal(t, byte(0xD0), msiData[0])
	assert.Equal(t, byte(0xCF), msiData[1])
	assert.Equal(t, byte(0x11), msiData[2])
	assert.Equal(t, byte(0xE0), msiData[3])

	// Read back with comdoc.
	reader := bytes.NewReader(msiData)
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	names := make(map[string]struct{})
	for _, e := range entries {
		if e.Type == comdoc.DirStream {
			names[e.Name()] = struct{}{}
		}
	}

	// Verify key streams exist.
	// Stream names are MSI-encoded, so we decode them to check.
	decodedNames := make(map[string]struct{})
	for name := range names {
		decoded := msiDecodeName(name)
		decodedNames[decoded] = struct{}{}
	}

	_, ok := decodedNames["Table._StringData"]
	assert.True(t, ok, "should have _StringData stream")
	_, ok = decodedNames["Table._StringPool"]
	assert.True(t, ok, "should have _StringPool stream")
	_, ok = decodedNames["Table._Columns"]
	assert.True(t, ok, "should have _Columns stream")
	_, ok = decodedNames["Table._Tables"]
	assert.True(t, ok, "should have _Tables stream")
	_, ok = decodedNames["Table.Property"]
	assert.True(t, ok, "should have Property stream")
	_, ok = decodedNames["Table.File"]
	assert.True(t, ok, "should have File stream")
	_, ok = decodedNames["Table.Directory"]
	assert.True(t, ok, "should have Directory stream")
	_, ok = decodedNames["Table.Component"]
	assert.True(t, ok, "should have Component stream")
	_, ok = decodedNames["Table.Feature"]
	assert.True(t, ok, "should have Feature stream")
	_, ok = decodedNames["Table.Media"]
	assert.True(t, ok, "should have Media stream")
	_, ok = decodedNames["Table.ServiceInstall"]
	assert.True(t, ok, "should have ServiceInstall stream")
	// Check non-table streams.
	_, ok = names["\x05SummaryInformation"]
	assert.True(t, ok, "should have SummaryInformation")
	_, ok = decodedNames["orbit.cab"]
	assert.True(t, ok, "should have embedded CAB (MSI-encoded)")
}

// TestWriteMSI_FullRoundTrip generates a full 18-table MSI and validates
// every table's column definitions and data by reading it back.
func TestWriteMSI_FullRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))

	// Create test files matching a realistic orbit layout.
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "secret.txt"), []byte("test-secret"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "osquery.flags"), []byte(""), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "bin", "orbit", "windows", "stable"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, "bin", "orbit", "windows", "stable", "orbit.exe"),
		[]byte("fake-orbit"),
		0o644,
	))

	opts := MSIOptions{
		ProductName:    "Fleet osquery",
		ProductVersion: "1.28.0",
		Manufacturer:   "Fleet Device Management (fleetdm.com)",
		UpgradeCode:    "{B681CB20-107E-428A-9B14-2D3C1AFED244}",
		Architecture:   "amd64",
		FleetURL:       "https://fleet.example.com",
		EnrollSecret:   "test-secret",
		OrbitChannel:   "stable",
		OsquerydChannel: "stable",
		DesktopChannel: "stable",
		OrbitPath:      `bin\orbit\windows\stable\orbit.exe`,
	}

	var buf bytes.Buffer
	require.NoError(t, WriteMSI(&buf, rootDir, opts))

	// Read back with comdoc.
	msiData := buf.Bytes()
	reader := bytes.NewReader(msiData)
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	// Collect all streams by decoded name.
	streamData := make(map[string][]byte)
	for _, e := range entries {
		if e.Type != comdoc.DirStream {
			continue
		}
		decoded := msiDecodeName(e.Name())
		name := decoded
		if len(decoded) > 6 && decoded[:6] == "Table." {
			name = decoded[6:]
		}
		r, readErr := doc.ReadStream(e)
		require.NoError(t, readErr, "reading stream %s", name)
		var content bytes.Buffer
		_, err := content.ReadFrom(r)
		require.NoError(t, err)
		streamData[name] = content.Bytes()
	}

	// Decode string pool.
	allStrings, err := testDecodeStrings(
		bytes.NewReader(streamData["_StringData"]),
		bytes.NewReader(streamData["_StringPool"]),
	)
	require.NoError(t, err)
	require.NotEmpty(t, allStrings)

	// Decode _Tables stream — list of uint16 string pool indices.
	tablesRaw := streamData["_Tables"]
	require.True(t, len(tablesRaw)%2 == 0, "_Tables stream size must be even")
	tableCount := len(tablesRaw) / 2
	tableNames := make([]string, tableCount)
	for i := range tableCount {
		idx := binary.LittleEndian.Uint16(tablesRaw[i*2:])
		require.Greater(t, int(idx), 0, "table index must be > 0")
		require.LessOrEqual(t, int(idx), len(allStrings), "table index out of range")
		tableNames[i] = allStrings[idx-1]
	}

	// Verify expected tables are present.
	expectedTables := []string{
		"Component", "CreateFolder",
		"Directory", "Environment", "Feature", "FeatureComponents",
		"File", "InstallExecuteSequence", "Media",
		"MsiServiceConfigFailureActions", "Property",
		"Registry", "ServiceControl", "ServiceInstall", "Upgrade",
	}
	assert.Equal(t, expectedTables, tableNames, "tables should be sorted alphabetically")

	// Decode _Columns stream — 4 columns in column-major order.
	columnsRaw := streamData["_Columns"]
	require.True(t, len(columnsRaw)%8 == 0, "_Columns stream size must be divisible by 8")
	colRowCount := len(columnsRaw) / 8

	// Read column-major: all Table IDs, then all Numbers, then all Name IDs, then all Types.
	colTableIDs := make([]uint16, colRowCount)
	colNumbers := make([]uint16, colRowCount)
	colNameIDs := make([]uint16, colRowCount)
	colTypes := make([]uint16, colRowCount)
	for i := range colRowCount {
		colTableIDs[i] = binary.LittleEndian.Uint16(columnsRaw[i*2:])
	}
	off := colRowCount * 2
	for i := range colRowCount {
		colNumbers[i] = binary.LittleEndian.Uint16(columnsRaw[off+i*2:])
	}
	off += colRowCount * 2
	for i := range colRowCount {
		colNameIDs[i] = binary.LittleEndian.Uint16(columnsRaw[off+i*2:])
	}
	off += colRowCount * 2
	for i := range colRowCount {
		colTypes[i] = binary.LittleEndian.Uint16(columnsRaw[off+i*2:])
	}

	// Build column definitions per table from _Columns.
	type colInfo struct {
		number uint16
		name   string
		typ    uint16
	}
	tableColumns := make(map[string][]colInfo)
	for i := range colRowCount {
		tblIdx := colTableIDs[i]
		require.Greater(t, int(tblIdx), 0, "column table index must be > 0")
		require.LessOrEqual(t, int(tblIdx), len(allStrings), "column table index %d out of range", tblIdx)
		tblName := allStrings[tblIdx-1]

		nameIdx := colNameIDs[i]
		require.Greater(t, int(nameIdx), 0, "column name index must be > 0")
		require.LessOrEqual(t, int(nameIdx), len(allStrings), "column name index %d out of range", nameIdx)
		colName := allStrings[nameIdx-1]

		// Number and Type are XOR'd shorts.
		colNum := colNumbers[i] ^ 0x8000
		colTyp := colTypes[i] ^ 0x8000

		tableColumns[tblName] = append(tableColumns[tblName], colInfo{
			number: colNum,
			name:   colName,
			typ:    colTyp,
		})
	}

	// Verify each table listed in _Tables has columns in _Columns.
	for _, tbl := range tableNames {
		cols, ok := tableColumns[tbl]
		require.True(t, ok, "table %s should have columns in _Columns", tbl)
		require.NotEmpty(t, cols, "table %s should have at least one column", tbl)

		// Verify column numbers are sequential 1..N.
		for j, col := range cols {
			assert.Equal(t, uint16(j+1), col.number, "table %s column %d number mismatch", tbl, j)
		}
	}

	// Verify specific table schemas.
	// Property: Property (s72 PK), Value (l255)
	propCols := tableColumns["Property"]
	require.Len(t, propCols, 2, "Property should have 2 columns")
	assert.Equal(t, "Property", propCols[0].name)
	assert.Equal(t, "Value", propCols[1].name)

	// Verify each table's data stream exists and has valid size.
	for _, tbl := range tableNames {
		data, ok := streamData[tbl]
		require.True(t, ok, "table %s should have a data stream", tbl)

		cols := tableColumns[tbl]
		// Calculate expected bytes per row.
		var bytesPerRow int
		for _, col := range cols {
			baseType := col.typ & 0x0F00
			if baseType >= 0x0800 { // String types (0x0900, 0x0D00, 0x0F00)
				bytesPerRow += 2 // uint16 string pool index
			} else if col.typ&0xFF == 4 { // Long (4-byte int)
				bytesPerRow += 4
			} else { // Short (2-byte int)
				bytesPerRow += 2
			}
		}
		require.Greater(t, bytesPerRow, 0, "table %s bytesPerRow should be > 0", tbl)
		require.True(t, len(data)%bytesPerRow == 0,
			"table %s data size %d not divisible by row size %d", tbl, len(data), bytesPerRow)
		rowCount := len(data) / bytesPerRow
		require.Greater(t, rowCount, 0, "table %s should have at least 1 row", tbl)

		// Validate all string pool indices in the data are within range.
		dataOffset := 0
		for _, col := range cols {
			baseType := col.typ & 0x0F00
			if baseType >= 0x0800 { // String type
				for r := range rowCount {
					idx := binary.LittleEndian.Uint16(data[dataOffset+r*2:])
					// 0 = null (valid for nullable), otherwise must be in pool range.
					if idx > 0 {
						assert.LessOrEqual(t, int(idx), len(allStrings),
							"table %s col %s row %d: pool index %d out of range (pool size %d)",
							tbl, col.name, r, idx, len(allStrings))
					}
				}
				dataOffset += rowCount * 2
			} else if col.typ&0xFF == 4 { // Long
				dataOffset += rowCount * 4
			} else { // Short
				dataOffset += rowCount * 2
			}
		}
	}

	// Verify specific data: Property table should contain ProductName.
	propData := streamData["Property"]
	propCols2 := tableColumns["Property"]
	var propBytesPerRow int
	for _, col := range propCols2 {
		baseType := col.typ & 0x0F00
		if baseType >= 0x0800 {
			propBytesPerRow += 2
		} else if col.typ&0xFF == 4 {
			propBytesPerRow += 4
		} else {
			propBytesPerRow += 2
		}
	}
	propRowCount := len(propData) / propBytesPerRow
	// Read the Property column (first column, string type).
	propNames := make(map[string]string)
	for r := range propRowCount {
		keyIdx := binary.LittleEndian.Uint16(propData[r*2:])
		valIdx := binary.LittleEndian.Uint16(propData[propRowCount*2+r*2:])
		if keyIdx > 0 && int(keyIdx) <= len(allStrings) && valIdx > 0 && int(valIdx) <= len(allStrings) {
			propNames[allStrings[keyIdx-1]] = allStrings[valIdx-1]
		}
	}
	assert.Equal(t, "Fleet osquery", propNames["ProductName"])
	assert.Equal(t, "1.28.0", propNames["ProductVersion"])
	assert.Equal(t, "{B681CB20-107E-428A-9B14-2D3C1AFED244}", propNames["UpgradeCode"])

}

// TestWriteMSI_GenerateForWindows creates a test MSI at /tmp/WriteMSI_fixed.msi
// for manual Windows validation. Run with: go test -run TestWriteMSI_GenerateForWindows -v
func TestWriteMSI_GenerateForWindows(t *testing.T) {
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))

	// Realistic orbit file layout.
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "secret.txt"), []byte("test-secret"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "osquery.flags"), []byte("--flagfile=osquery.flags"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "fleet.pem"), []byte("CERT"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "installer_utils.ps1"), []byte("# PowerShell utils"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(rootDir, "bin", "orbit", "windows", "stable"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(rootDir, "bin", "orbit", "windows", "stable", "orbit.exe"),
		[]byte("MZ-fake-orbit-binary-for-testing"),
		0o644,
	))

	opts := MSIOptions{
		ProductName:    "Fleet osquery",
		ProductVersion: "1.28.0",
		Manufacturer:   "Fleet Device Management (fleetdm.com)",
		UpgradeCode:    "{B681CB20-107E-428A-9B14-2D3C1AFED244}",
		Architecture:   "amd64",
		FleetURL:       "https://fleet.example.com",
		EnrollSecret:   "test-secret",
		FleetCertificate: "fleet.pem",
		OrbitChannel:   "stable",
		OsquerydChannel: "stable",
		DesktopChannel: "stable",
		OrbitPath:      `bin\orbit\windows\stable\orbit.exe`,
	}

	var buf bytes.Buffer
	require.NoError(t, WriteMSI(&buf, rootDir, opts))

	path := "/tmp/WriteMSI_fixed.msi"
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
	t.Logf("Created %s (%d bytes)", path, buf.Len())
}

func TestWriteMSI_PropertyRoundTrip(t *testing.T) {
	// Generate an MSI and read back the Property table to verify product metadata.
	tmpDir := t.TempDir()
	rootDir := filepath.Join(tmpDir, "root")
	require.NoError(t, os.MkdirAll(rootDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(rootDir, "test.txt"), []byte("test"), 0o644))

	opts := MSIOptions{
		ProductName:    "Fleet osquery",
		ProductVersion: "1.28.0",
		Manufacturer:   "Fleet Device Management",
		UpgradeCode:    "{B681CB20-107E-428A-9B14-2D3C1AFED244}",
		Architecture:   "x64",
		OrbitChannel:   "stable",
		OsquerydChannel: "stable",
		DesktopChannel: "stable",
	}

	var buf bytes.Buffer
	require.NoError(t, WriteMSI(&buf, rootDir, opts))

	// Read back and extract property values using the same decoding as pkg/file/msi.go.
	reader := bytes.NewReader(buf.Bytes())
	doc, err := comdoc.ReadFile(reader)
	require.NoError(t, err)
	defer doc.Close()

	entries, err := doc.ListDir(nil)
	require.NoError(t, err)

	// Find and read the required streams.
	streamData := make(map[string][]byte)
	targetStreams := map[string]struct{}{
		"_StringData": {},
		"_StringPool": {},
		"_Columns":    {},
		"Property":    {},
	}

	for _, e := range entries {
		if e.Type != comdoc.DirStream {
			continue
		}
		decoded := msiDecodeName(e.Name())
		// Strip "Table." prefix.
		name := decoded
		if len(decoded) > 6 && decoded[:6] == "Table." {
			name = decoded[6:]
		}
		if _, isTarget := targetStreams[name]; isTarget {
			r, readErr := doc.ReadStream(e)
			require.NoError(t, readErr)
			var content bytes.Buffer
			_, err := content.ReadFrom(r)
			require.NoError(t, err)
			streamData[name] = content.Bytes()
		}
	}

	// Decode string pool.
	allStrings, err := testDecodeStrings(
		bytes.NewReader(streamData["_StringData"]),
		bytes.NewReader(streamData["_StringPool"]),
	)
	require.NoError(t, err)
	require.NotEmpty(t, allStrings)

	// Verify ProductName is in the string pool.
	assert.True(t, slices.Contains(allStrings, "Fleet osquery"), "ProductName 'Fleet osquery' should be in string pool")

	// Verify ProductVersion is in the string pool.
	assert.True(t, slices.Contains(allStrings, "1.28.0"), "ProductVersion '1.28.0' should be in string pool")
}
