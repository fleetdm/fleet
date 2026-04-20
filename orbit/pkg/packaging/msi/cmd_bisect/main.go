package main

import (
	"bytes"
	"fmt"
	"os"
	msi "github.com/fleetdm/fleet/v4/orbit/pkg/packaging/msi"
)

func main() {
	type td = msi.TableData; type ts = msi.TableSchema; type cd = msi.ColumnDef
	base18 := getBase18()
	cols := []cd{{Name: "MyP", Type: msi.ColStrPK(72)}, {Name: "C2", Type: msi.ColStr(72)}, {Name: "C3", Type: msi.ColStr(72)}, {Name: "C4", Type: msi.ColStr(72)}}

	// Isolate each value from the failing test D
	// D had: {"p1", "ORBITROOT", "CreateFolder", "D:P(A;;FA;;;SY)"}
	
	// E: only "ORBITROOT" (shared with Directory table)
	test("testE_orbitroot.msi", base18, &td{Schema: ts{Name: "MyP", Columns: cols},
		Rows: [][]any{{"p1", "ORBITROOT", "val3", "val4"}}})
	
	// F: only "CreateFolder" (shared table name)
	test("testF_createfolder.msi", base18, &td{Schema: ts{Name: "MyP", Columns: cols},
		Rows: [][]any{{"p1", "val2", "CreateFolder", "val4"}}})
	
	// G: only SDDL string
	test("testG_sddl.msi", base18, &td{Schema: ts{Name: "MyP", Columns: cols},
		Rows: [][]any{{"p1", "val2", "val3", "D:P(A;;FA;;;SY)"}}})
	
	// H: "ORBITROOT" + "CreateFolder" but no SDDL
	test("testH_shared_strings.msi", base18, &td{Schema: ts{Name: "MyP", Columns: cols},
		Rows: [][]any{{"p1", "ORBITROOT", "CreateFolder", "val4"}}})
}

func test(name string, base18 []func() *msi.TableData, extra *msi.TableData) {
	pool := msi.NewStringPool()
	tables := make([]*msi.TableData, 0)
	for _, f := range base18 { tables = append(tables, f()) }
	tables = append(tables, extra)
	var streams []struct{ n string; d []byte }
	for _, t := range tables { data := msi.EncodeTableData(t, pool); if len(data) > 0 { streams = append(streams, struct{ n string; d []byte }{msi.MsiEncodeName(t.Schema.Name, true), data}) } }
	streams = append(streams, struct{ n string; d []byte }{msi.MsiEncodeName("_Tables", true), msi.EncodeTablesStream(tables, pool)})
	streams = append(streams, struct{ n string; d []byte }{msi.MsiEncodeName("_Columns", true), msi.EncodeColumnsStream(tables, pool)})
	streams = append(streams, struct{ n string; d []byte }{msi.MsiEncodeName("_StringPool", true), pool.EncodePool()})
	streams = append(streams, struct{ n string; d []byte }{msi.MsiEncodeName("_StringData", true), pool.EncodeData()})
	si := msi.NewSummaryInfo("Test", "Test", "x64", "1.0.0")
	streams = append(streams, struct{ n string; d []byte }{"\x05SummaryInformation", si.Encode()})
	cw := msi.NewCFBWriterForTest()
	for _, s := range streams { cw.AddStreamForTest(s.n, s.d) }
	var buf bytes.Buffer; cw.WriteToForTest(&buf)
	os.WriteFile("/Users/lucas/git/fleet/"+name, buf.Bytes(), 0o644)
	fmt.Printf("%s: %d bytes, %d strings\n", name, buf.Len(), pool.Count())
}

func getBase18() []func() *msi.TableData {
	type td = msi.TableData; type ts = msi.TableSchema; type cd = msi.ColumnDef
	return []func() *td{
		func() *td { return &td{Schema: ts{Name: "Property", Columns: []cd{{Name: "Property", Type: msi.ColStrPK(72)}, {Name: "Value", Type: msi.ColStrL(255)}}}, Rows: [][]any{{"ProductCode", "{12345678-0000-0000-0000-000000000000}"}, {"ProductName", "Test"}, {"ProductVersion", "1.0.0"}, {"ProductLanguage", "1033"}, {"Manufacturer", "Test"}, {"UpgradeCode", "{B681CB20-107E-428A-9B14-2D3C1AFED244}"}}} },
		func() *td { return &td{Schema: ts{Name: "Directory", Columns: []cd{{Name: "Directory", Type: msi.ColStrPK(72)}, {Name: "Directory_Parent", Type: msi.ColStrN(72)}, {Name: "DefaultDir", Type: msi.ColStrL(255)}}}, Rows: [][]any{{"TARGETDIR", nil, "SourceDir"}, {"PF64", "TARGETDIR", "PF64"}, {"ORBITROOT", "PF64", "Orbit"}}} },
		func() *td { return &td{Schema: ts{Name: "Component", Columns: []cd{{Name: "Component", Type: msi.ColStrPK(72)}, {Name: "ComponentId", Type: msi.ColStrN(38)}, {Name: "Directory_", Type: msi.ColStr(72)}, {Name: "Attributes", Type: 0x0104}, {Name: "Condition", Type: msi.ColStrN(255)}, {Name: "KeyPath", Type: msi.ColStrN(72)}}}, Rows: [][]any{{"C1", "{11111111-1111-1111-1111-111111111111}", "ORBITROOT", int32(0), nil, nil}}} },
		func() *td { return &td{Schema: ts{Name: "Feature", Columns: []cd{{Name: "Feature", Type: msi.ColStrPK(38)}, {Name: "Feature_Parent", Type: msi.ColStrN(38)}, {Name: "Title", Type: msi.ColStrLN(64)}, {Name: "Description", Type: msi.ColStrLN(255)}, {Name: "Display", Type: 0x1502}, {Name: "Level", Type: 0x0502}, {Name: "Directory_", Type: msi.ColStrN(72)}, {Name: "Attributes", Type: 0x0502}}}, Rows: [][]any{{"F1", nil, "Test", nil, int16(0), int16(1), "ORBITROOT", int16(0)}}} },
		func() *td { return &td{Schema: ts{Name: "FeatureComponents", Columns: []cd{{Name: "Feature_", Type: msi.ColStrPK(38)}, {Name: "Component_", Type: msi.ColStrPK(72)}}}, Rows: [][]any{{"F1", "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "File", Columns: []cd{{Name: "File", Type: msi.ColStrPK(72)}, {Name: "Component_", Type: msi.ColStr(72)}, {Name: "FileName", Type: msi.ColStrL(255)}, {Name: "FileSize", Type: 0x0104}, {Name: "Version", Type: msi.ColStrN(72)}, {Name: "Language", Type: msi.ColStrN(20)}, {Name: "Attributes", Type: 0x1502}, {Name: "Sequence", Type: 0x0502}}}, Rows: [][]any{{"f1", "C1", "test.txt", int32(5), nil, nil, int16(0), int16(1)}}} },
		func() *td { return &td{Schema: ts{Name: "Media", Columns: []cd{{Name: "DiskId", Type: 0x2502}, {Name: "LastSequence", Type: 0x0502}, {Name: "DiskPrompt", Type: msi.ColStrLN(64)}, {Name: "Cabinet", Type: msi.ColStrN(255)}, {Name: "VolumeLabel", Type: msi.ColStrN(32)}, {Name: "Source", Type: msi.ColStrN(72)}}}, Rows: [][]any{{int16(1), int16(1), nil, "#test.cab", nil, nil}}} },
		func() *td { return &td{Schema: ts{Name: "ServiceInstall", Columns: []cd{{Name: "ServiceInstall", Type: msi.ColStrPK(72)}, {Name: "Name", Type: msi.ColStr(255)}, {Name: "DisplayName", Type: msi.ColStrLN(255)}, {Name: "ServiceType", Type: 0x0104}, {Name: "StartType", Type: 0x0104}, {Name: "ErrorControl", Type: 0x0104}, {Name: "LoadOrderGroup", Type: msi.ColStrN(255)}, {Name: "Dependencies", Type: msi.ColStrN(255)}, {Name: "StartName", Type: msi.ColStrN(255)}, {Name: "Password", Type: msi.ColStrN(255)}, {Name: "Arguments", Type: msi.ColStrN(255)}, {Name: "Component_", Type: msi.ColStr(72)}, {Name: "Description", Type: msi.ColStrLN(255)}}}, Rows: [][]any{{"svc1", "Svc", "Svc", int32(16), int32(2), int32(0), nil, nil, "LocalSystem", nil, "--t", "C1", "D"}}} },
		func() *td { return &td{Schema: ts{Name: "ServiceControl", Columns: []cd{{Name: "ServiceControl", Type: msi.ColStrPK(72)}, {Name: "Name", Type: msi.ColStrL(255)}, {Name: "Event", Type: 0x0502}, {Name: "Arguments", Type: msi.ColStrLN(255)}, {Name: "Wait", Type: 0x1502}, {Name: "Component_", Type: msi.ColStr(72)}}}, Rows: [][]any{{"sc1", "Svc", int16(147), nil, int16(1), "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "CustomAction", Columns: []cd{{Name: "Action", Type: msi.ColStrPK(72)}, {Name: "Type", Type: 0x0502}, {Name: "Source", Type: msi.ColStrN(72)}, {Name: "Target", Type: msi.ColStrN(255)}}}, Rows: [][]any{{"CA1", int16(51), "p1", "cmd"}}} },
		func() *td { return &td{Schema: ts{Name: "InstallExecuteSequence", Columns: []cd{{Name: "Action", Type: msi.ColStrPK(72)}, {Name: "Condition", Type: msi.ColStrN(255)}, {Name: "Sequence", Type: 0x1502}}}, Rows: [][]any{{"CostInitialize", nil, int16(800)}, {"CostFinalize", nil, int16(1000)}, {"InstallValidate", nil, int16(1400)}, {"InstallInitialize", nil, int16(1500)}, {"InstallFinalize", nil, int16(6600)}}} },
		func() *td { return &td{Schema: ts{Name: "Upgrade", Columns: []cd{{Name: "UpgradeCode", Type: msi.ColStrPK(72)}, {Name: "VersionMin", Type: msi.ColStrN(20)}, {Name: "VersionMax", Type: msi.ColStrN(20)}, {Name: "Language", Type: msi.ColStrN(255)}, {Name: "Attributes", Type: 0x0104}, {Name: "Remove", Type: msi.ColStrN(255)}, {Name: "ActionProperty", Type: msi.ColStr(72)}}}, Rows: [][]any{{"{B681CB20-107E-428A-9B14-2D3C1AFED244}", "1.0.0", nil, nil, int32(256), nil, "WIX_UP"}}} },
		func() *td { return &td{Schema: ts{Name: "Registry", Columns: []cd{{Name: "Registry", Type: msi.ColStrPK(72)}, {Name: "Root", Type: 0x0502}, {Name: "Key", Type: msi.ColStrL(255)}, {Name: "Name", Type: msi.ColStrLN(255)}, {Name: "Value", Type: msi.ColStrLN(255)}, {Name: "Component_", Type: msi.ColStr(72)}}}, Rows: [][]any{{"r1", int16(2), `SOFTWARE\T`, "Path", "[ORBITROOT]", "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "CreateFolder", Columns: []cd{{Name: "Directory_", Type: msi.ColStrPK(72)}, {Name: "Component_", Type: msi.ColStrPK(72)}}}, Rows: [][]any{{"ORBITROOT", "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "Environment", Columns: []cd{{Name: "Environment", Type: msi.ColStrPK(72)}, {Name: "Name", Type: msi.ColStrL(255)}, {Name: "Value", Type: msi.ColStrLN(255)}, {Name: "Component_", Type: msi.ColStr(72)}}}, Rows: [][]any{{"e1", "=-TV", "t", "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "MsiServiceConfigFailureActions", Columns: []cd{{Name: "MsiServiceConfigFailureActions", Type: msi.ColStrPK(72)}, {Name: "Name", Type: msi.ColStr(255)}, {Name: "Event", Type: 0x0502}, {Name: "ResetPeriod", Type: 0x1104}, {Name: "RebootMessage", Type: msi.ColStrLN(255)}, {Name: "Command", Type: msi.ColStrLN(255)}, {Name: "Actions", Type: msi.ColStrN(255)}, {Name: "DelayActions", Type: msi.ColStrN(255)}, {Name: "Component_", Type: msi.ColStr(72)}}}, Rows: [][]any{{"sf1", "Svc", int16(1), int32(86400), nil, nil, "1/1000", "1000", "C1"}}} },
		func() *td { return &td{Schema: ts{Name: "RegLocator", Columns: []cd{{Name: "Signature_", Type: msi.ColStrPK(72)}, {Name: "Root", Type: 0x0502}, {Name: "Key", Type: msi.ColStr(255)}, {Name: "Name", Type: msi.ColStrN(255)}, {Name: "Type", Type: 0x1502}}}, Rows: [][]any{{"PW", int16(2), `SOFTWARE\M`, "Path", int16(2)}}} },
		func() *td { return &td{Schema: ts{Name: "AppSearch", Columns: []cd{{Name: "Property", Type: msi.ColStrPK(72)}, {Name: "Signature_", Type: msi.ColStrPK(72)}}}, Rows: [][]any{{"PWSH", "PW"}}} },
	}
}
