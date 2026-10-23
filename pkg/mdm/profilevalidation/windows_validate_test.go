package profilevalidation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeWindowsProfile(t *testing.T) {
	items, err := decodeWindowsProfile([]byte(`
<Replace>
	<CmdID>1</CmdID>
	<Item>
		<Meta><Format xmlns="syncml:metinf">int</Format></Meta>
		<Target><LocURI> ./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength </LocURI></Target>
		<Data>8</Data>
	</Item>
</Replace>
<Add>
	<Item>
		<Meta><Format>chr</Format></Meta>
		<Target><LocURI>./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome/HomepageLocation</LocURI></Target>
		<Data><![CDATA[<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>]]></Data>
	</Item>
</Add>
<Atomic>
	<Exec>
		<Item><Target><LocURI>./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll</LocURI></Target></Item>
	</Exec>
	<Replace>
		<Item>
			<Target><LocURI>./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall</LocURI></Target>
			<Data></Data>
		</Item>
		<Item><Source><LocURI>./Vendor/MSFT/NoTarget</LocURI></Source><Data>x</Data></Item>
		<Item>
			<Target><LocURI>./Vendor/MSFT/Example/Xml</LocURI></Target>
			<Data><Rules><Rule id="1"/></Rules></Data>
		</Item>
	</Replace>
</Atomic>`))
	require.NoError(t, err)
	assert.Equal(t, []windowsItem{
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", MetaFormat: "int", Data: new("8")},
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome/HomepageLocation", MetaFormat: "chr", Data: new(`<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>`)},
		{LocURI: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll"},
		{LocURI: "./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", Data: new("")},
		// Raw child elements are already well-formed; only the element's own text is kept.
		{LocURI: "./Vendor/MSFT/Example/Xml", Data: new("")},
	}, items)
}

func TestDecodeWindowsProfileEdges(t *testing.T) {
	// A body that's only a secret placeholder expands at apply time; it has nothing to check.
	items, err := decodeWindowsProfile([]byte(`$FLEET_SECRET_WINDOWS_PROFILE`))
	require.NoError(t, err)
	assert.Empty(t, items)

	for name, data := range map[string]string{
		"unclosed":   `<Replace><Item><Target><LocURI>./Vendor/MSFT/X</LocURI></Target></Item>`,
		"mismatched": `<Replace><Item></Replace></Item>`,
		"bad entity": `<Replace><Item><Data>&nope;</Data></Item></Replace>`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := decodeWindowsProfile([]byte(data))
			require.Error(t, err)
		})
	}
}

func TestWindowsDataAccepts(t *testing.T) {
	f := func(formats ...WindowsFormat) WindowsFormats { return formats }
	cases := []struct {
		name    string
		formats WindowsFormats
		data    string
		want    bool
	}{
		{name: "int", formats: f(WindowsFormatInt), data: "5", want: true},
		{name: "negative int", formats: f(WindowsFormatInt), data: "-1", want: true},
		{name: "int with whitespace", formats: f(WindowsFormatInt), data: "\n  5\n", want: true},
		{name: "int with letters", formats: f(WindowsFormatInt), data: "abc"},
		{name: "int with real", formats: f(WindowsFormatInt), data: "1.5"},
		{name: "empty int", formats: f(WindowsFormatInt), data: "  ", want: true},

		{name: "bool true", formats: f(WindowsFormatBool), data: "true", want: true},
		{name: "bool false", formats: f(WindowsFormatBool), data: "False", want: true},
		{name: "bool 0", formats: f(WindowsFormatBool), data: "0", want: true},
		{name: "bool 1", formats: f(WindowsFormatBool), data: "1", want: true},
		{name: "bool 2", formats: f(WindowsFormatBool), data: "2"},
		{name: "bool yes", formats: f(WindowsFormatBool), data: "yes"},

		{name: "b64", formats: f(WindowsFormatB64), data: "Zmxl\n ZXQ=", want: true},
		{name: "invalid b64", formats: f(WindowsFormatB64), data: "not base64!"},

		{name: "xml", formats: f(WindowsFormatXML), data: `<a><b x="1"/></a>`, want: true},
		{name: "unclosed xml", formats: f(WindowsFormatXML), data: `<a><b>`},
		{name: "mismatched xml", formats: f(WindowsFormatXML), data: `<a></b>`},

		{name: "chr", formats: f(WindowsFormatChr), data: "anything", want: true},
		{name: "bin", formats: f(WindowsFormatBin), data: "anything", want: true},
		{name: "time", formats: f(WindowsFormatTime), data: "anything", want: true},
		{name: "float", formats: f(WindowsFormatFloat), data: "anything", want: true},
		{name: "unknown format", formats: f("future"), data: "anything", want: true},
		{name: "no format", data: "anything", want: true},

		{name: "list with chr accepts text", formats: f(WindowsFormatInt, WindowsFormatChr), data: "abc", want: true},
		{name: "list with chr accepts int", formats: f(WindowsFormatInt, WindowsFormatChr), data: "5", want: true},
		{name: "list of checked formats", formats: f(WindowsFormatInt, WindowsFormatBool), data: "abc"},

		{name: "variable in chr", formats: f(WindowsFormatChr), data: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME", want: true},
		{name: "variable in xml", formats: f(WindowsFormatXML), data: "<a>$FLEET_VAR_HOST_UUID</a>", want: true},
		{name: "variable in int", formats: f(WindowsFormatInt), data: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"},
		{name: "variable in bool", formats: f(WindowsFormatBool), data: "${FLEET_VAR_HOST_UUID}"},
		{name: "variable in b64", formats: f(WindowsFormatB64), data: "$FLEET_VAR_HOST_UUID"},
		{name: "secret in int", formats: f(WindowsFormatInt), data: "$FLEET_SECRET_TIMEOUT", want: true},
		{name: "secret in b64", formats: f(WindowsFormatB64), data: "${FLEET_SECRET_CERT}", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, windowsDataAccepts(c.formats, c.data))
		})
	}
}

func TestResolveWindowsNode(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)
	node := func(key string) *WindowsNode {
		n := s.Windows[key]
		require.NotNil(t, n, key)
		return n
	}
	const chromeHomepage = "/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation"

	cases := []struct {
		name       string
		locURI     string
		want       *WindowsNode
		wrongScope bool
	}{
		{name: "bare firewall", locURI: "./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", want: node("./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall")},
		{name: "device firewall", locURI: "./Device/Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", want: node("./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall")},
		{name: "firewall rule name", locURI: "./Vendor/MSFT/Firewall/MdmStore/FirewallRules/Block Telnet/Action/Type", want: node("./Vendor/MSFT/Firewall/MdmStore/FirewallRules/*/Action/Type")},
		{name: "variable in dynamic segment", locURI: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll", want: node("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/*/Install/Enroll")},
		{name: "lowercase", locURI: "./device/vendor/msft/policy/config/devicelock/maxinactivitytimedevicelock", want: node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock")},
		{name: "bare policy", locURI: "./Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock", want: node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock")},
		{name: "user path for device-only policy", locURI: "./User/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock", want: node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock"), wrongScope: true},
		{name: "user path for bare firewall", locURI: "./User/Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", want: node("./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall"), wrongScope: true},
		{name: "device path for user-only policy", locURI: "./Device/Vendor/MSFT/Policy/Config/Education/PrinterNames", want: node("./User/Vendor/MSFT/Policy/Config/Education/PrinterNames"), wrongScope: true},
		{name: "template policy", locURI: "./Device" + chromeHomepage, want: node("./Device" + chromeHomepage)},
		{name: "template policy user", locURI: "./User" + chromeHomepage, want: node("./User" + chromeHomepage)},
		{name: "ADMX install item", locURI: "./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Chrome/Policy/ChromeAdmx", want: node("./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/*/*/*")},
		{name: "DMAcc", locURI: "./SyncML/DMAcc/MDMServer/Name", want: node("./SyncML/DMAcc/*/Name")},
		{name: "ingested ADMX area", locURI: "./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification"},
		{name: "policy typo", locURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/Typoo"},
		{name: "policy area alone", locURI: "./Device/Vendor/MSFT/Policy/Config/Unknown"},
		{name: "unknown CSP", locURI: "./Device/Vendor/MSFT/Unknown/Setting"},
		{name: "not rooted", locURI: "Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, wrongScope := s.resolveWindowsNode(c.locURI)
			assert.Same(t, c.want, got)
			assert.Equal(t, c.wrongScope, wrongScope)
		})
	}
}

func TestWindowsArea(t *testing.T) {
	for locURI, want := range map[string]string{
		"./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock":                    "DeviceLock",
		"./user/vendor/msft/policy/config/Start/HideSleep":                                             "Start",
		"./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation": "GoogleChrome~Policy~googlechrome~Startup",
		"./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/Chrome/Policy/ChromeAdmx":            "Policy",
		"./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall":                                 "Firewall",
		"./User/Vendor/MSFT/ClientCertificateInstall/SCEP/x/Install/Enroll":                            "ClientCertificateInstall",
		"./SyncML/DMAcc/MDMServer/Name":                                                                "DMAcc",
		"./DevDetail/Ext/Microsoft/DeviceName":                                                         "DevDetail",
		"./Vendor/MSFT":                                                                                "",
	} {
		assert.Equal(t, want, windowsArea(locURI), locURI)
	}
}
