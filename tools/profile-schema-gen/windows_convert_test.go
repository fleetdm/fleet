package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/profilevalidation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ddfSetting is a leaf node with the given format and extra DFProperties elements.
func ddfSetting(name, format, props string) string {
	return `<Node><NodeName>` + name + `</NodeName><DFProperties><AccessType><Replace /></AccessType>
		<Description>ignored</Description><DFFormat><` + format + ` /></DFFormat><Occurrence><One /></Occurrence>
		<Scope><Permanent /></Scope><DFType><MIME /></DFType>` + props + `</DFProperties></Node>`
}

func allowed(valueType, body string) string {
	return `<MSFT:AllowedValues ValueType="` + valueType + `">` + body + `</MSFT:AllowedValues>`
}

func enums(values ...string) string {
	var b strings.Builder
	for _, v := range values {
		b.WriteString(`<MSFT:Enum><MSFT:Value>` + v + `</MSFT:Value><MSFT:ValueDescription>d</MSFT:ValueDescription></MSFT:Enum>`)
	}
	return b.String()
}

func convertDDFs(t *testing.T, files map[string]string) (*profilevalidation.ProfileValidationSchema, string) {
	t.Helper()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	entries, err := loadWindows(t.Context(), logger, writeDDFs(t, files))
	require.NoError(t, err)
	var schema profilevalidation.ProfileValidationSchema
	convertWindows(t.Context(), logger, entries, &schema)
	return &schema, logs.String()
}

func TestConvertWindows(t *testing.T) {
	const xsd = `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema"><xs:element name="a"/></xs:schema>`
	settings := ddfSetting("Enum", "int", allowed("ENUM", enums("0", "1", "2"))) +
		ddfSetting("Flag", "int", allowed("Flag", enums("0x0", "0x1", "0x4"))) +
		ddfSetting("Range", "int", allowed("Range", `<MSFT:Value>[0-999]</MSFT:Value>`)) +
		ddfSetting("NegativeRange", "int", allowed("Range", `<MSFT:Value>[(-1)-100]</MSFT:Value>`)) +
		ddfSetting("CommaRange", "int", allowed("Range", `<MSFT:Value>[0,1]</MSFT:Value>`)) +
		ddfSetting("BadRange", "int", allowed("Range", `<MSFT:Value>0 to 9</MSFT:Value>`)) +
		ddfSetting("RegEx", "chr", allowed("RegEx", `<MSFT:Value>^\d+$</MSFT:Value>`)) +
		ddfSetting("XSD1", "chr", allowed("XSD", `<MSFT:Value><![CDATA[`+xsd+`]]></MSFT:Value>`)) +
		ddfSetting("XSD2", "chr", allowed("XSD", `<MSFT:Value><![CDATA[`+xsd+`]]></MSFT:Value>`)) +
		ddfSetting("ADMX", "chr", allowed("ADMX", `<MSFT:AdmxBacked Area="a" Name="n" File="f.admx" />`)) +
		ddfSetting("None", "chr", allowed("None", ``)) +
		ddfSetting("JSON", "chr", allowed("JSON", `<MSFT:Value>{}</MSFT:Value>`)) +
		ddfSetting("List", "chr", allowed("ENUM", enums("a", "b")+`<MSFT:List Delimiter="|" />`)) +
		ddfSetting("Future", "chr", allowed("YAML", `<MSFT:Value>x</MSFT:Value>`)) +
		ddfSetting("Future2", "chr", allowed("YAML", `<MSFT:Value>y</MSFT:Value>`)) +
		ddfSetting("Builds", "int", `<MSFT:Applicability><MSFT:OsBuildVersion>10.0.19041, 10.0.18362.1, 99.9.99999</MSFT:OsBuildVersion></MSFT:Applicability>`) +
		ddfSetting("Deprecated", "int", `<MSFT:Deprecated OsBuildDeprecated="10.0.22000" />`) +
		ddfSetting("DeprecatedNoBuild", "int", `<MSFT:Deprecated />`) +
		ddfSetting("NewProp", "int", `<MSFT:SomethingNew>1</MSFT:SomethingNew><MSFT:GpMapping GpEnglishName="x" GpAreaPath="y" />`) +
		ddfSetting("NewFormat", "decimal128", ``)
	schema, logs := convertDDFs(t, map[string]string{
		"device.xml": ddfFile(ddfInteriorNode("Area", "./Device/Vendor/MSFT", settings)),
		"user.xml":   ddfFile(ddfInteriorNode("Area", "./User/Vendor/MSFT", ddfSetting("Setting", "bool", ""))),
	})

	node := func(name string) *profilevalidation.WindowsNode {
		n := schema.Windows[strings.TrimSuffix("./Device/Vendor/MSFT/Area/"+name, "/")]
		require.NotNil(t, n, name)
		return n
	}
	int1 := profilevalidation.WindowsFormats{profilevalidation.WindowsFormatInt}
	device := profilevalidation.WindowsScopeDevice

	assert.Equal(t, &profilevalidation.WindowsNode{Format: int1, Scope: device, Enum: []string{"0", "1", "2"}}, node("Enum"))
	assert.Equal(t, []string{"0x0", "0x1", "0x4"}, node("Flag").Flag)
	assert.Equal(t, &profilevalidation.Range{Min: new(0.0), Max: new(999.0)}, node("Range").Range)
	assert.Equal(t, &profilevalidation.Range{Min: new(-1.0), Max: new(100.0)}, node("NegativeRange").Range)
	assert.Equal(t, &profilevalidation.Range{Min: new(0.0), Max: new(1.0)}, node("CommaRange").Range)
	assert.Nil(t, node("BadRange").Range, "an unparsed range stays unconstrained")
	assert.Contains(t, logs, `level=WARN msg="unrecognized DDF range; leaving the node unconstrained" loc_uri=./Device/Vendor/MSFT/Area/BadRange`)
	assert.Equal(t, `^\d+$`, node("RegEx").RegEx)

	// Identical XSDs are stored once, keyed by their sha256.
	require.Len(t, schema.WindowsXSDs, 1)
	assert.Equal(t, sha256Hex([]byte(xsd)), node("XSD1").XSD)
	assert.Equal(t, node("XSD1").XSD, node("XSD2").XSD)
	assert.Equal(t, xsd, schema.WindowsXSDs[node("XSD1").XSD])

	assert.True(t, node("ADMX").ADMXBacked)
	for _, name := range []string{"None", "JSON", "Future"} {
		n := node(name)
		assert.Empty(t, n.Enum, name)
		assert.Empty(t, n.RegEx, name)
		assert.Nil(t, n.Range, name)
	}
	assert.Equal(t, []string{"a", "b"}, node("List").Enum)
	assert.Equal(t, "|", node("List").Delimiter)
	assert.Equal(t, []string{"10.0.19041", "10.0.18362.1", "99.9.99999"}, node("Builds").MinBuild)
	assert.Equal(t, &profilevalidation.WindowsDeprecated{Build: "10.0.22000"}, node("Deprecated").Deprecated)
	assert.Equal(t, &profilevalidation.WindowsDeprecated{}, node("DeprecatedNoBuild").Deprecated)
	assert.Equal(t, profilevalidation.WindowsFormats{"decimal128"}, node("NewFormat").Format, "unknown formats are carried")

	// Unknown upstream fields are reported once each; known ones (GpMapping) aren't.
	assert.Equal(t, 1, strings.Count(logs, `msg="unrecognized DDF allowed-values type; nodes left unconstrained" value_type=YAML first_loc_uri=./Device/Vendor/MSFT/Area/Future`))
	assert.Contains(t, logs, `msg="unrecognized DDF property; ignored" property=SomethingNew first_loc_uri=./Device/Vendor/MSFT/Area/NewProp`)
	assert.NotContains(t, logs, "property=GpMapping")
	assert.NotContains(t, logs, "property=Description")

	user := notNil(t, schema.Windows["./User/Vendor/MSFT/Area/Setting"])
	assert.Equal(t, profilevalidation.WindowsScopeUser, user.Scope)
	area := node("")
	assert.Equal(t, profilevalidation.WindowsFormats{profilevalidation.WindowsFormatNode}, area.Format)
	assert.Equal(t, device, area.Scope)
}

func TestConvertWindowsInheritsApplicability(t *testing.T) {
	applicability := func(builds string) string {
		return `<MSFT:Applicability><MSFT:OsBuildVersion>` + builds + `</MSFT:OsBuildVersion></MSFT:Applicability>`
	}
	interior := func(name, props, children string) string {
		return `<Node><NodeName>` + name + `</NodeName><DFProperties><DFFormat><node /></DFFormat>` + props + `</DFProperties>` + children + `</Node>`
	}
	tree := `<Node><NodeName>CSP</NodeName><Path>./Device/Vendor/MSFT</Path><DFProperties><DFFormat><node /></DFFormat>` +
		applicability("10.0.10240") + `</DFProperties>` +
		ddfSetting("FromRoot", "int", "") +
		interior("Newer", applicability("10.0.22000"), ddfSetting("FromCloserAncestor", "int", "")+
			ddfSetting("Own", "int", applicability("10.0.26100"))) +
		`</Node>`
	other := ddfFile(`<Node><NodeName>Bare</NodeName><Path>./Device/Vendor/MSFT</Path><DFProperties><DFFormat><node /></DFFormat></DFProperties>` +
		ddfSetting("NoBuild", "int", "") + `</Node>`)
	schema, _ := convertDDFs(t, map[string]string{"csp.xml": ddfFile(tree), "bare.xml": other})

	build := func(locURI string) []string {
		n := schema.Windows[locURI]
		require.NotNil(t, n, locURI)
		return n.MinBuild
	}
	assert.Equal(t, []string{"10.0.10240"}, build("./Device/Vendor/MSFT/CSP/FromRoot"))
	assert.Equal(t, []string{"10.0.22000"}, build("./Device/Vendor/MSFT/CSP/Newer/FromCloserAncestor"))
	assert.Equal(t, []string{"10.0.26100"}, build("./Device/Vendor/MSFT/CSP/Newer/Own"))
	assert.Equal(t, []string{"10.0.10240"}, build("./Device/Vendor/MSFT/CSP"))
	// Inheritance stays within a file's tree.
	assert.Nil(t, build("./Device/Vendor/MSFT/Bare/NoBuild"))
}
