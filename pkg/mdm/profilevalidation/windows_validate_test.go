package profilevalidation

import (
	"strings"
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
		<Item>
			<Target><LocURI>./Vendor/MSFT/Example/Mixed</LocURI></Target>
			<Data>5<x/></Data>
		</Item>
	</Replace>
</Atomic>`))
	require.NoError(t, err)
	assert.Equal(t, []windowsItem{
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", MetaFormat: "int", Data: new("8")},
		{LocURI: "./Device/Vendor/MSFT/Policy/Config/Chrome~Policy~googlechrome/HomepageLocation", MetaFormat: "chr", Data: new(`<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>`)},
		{LocURI: "./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/Enroll"},
		{LocURI: "./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", Data: new("")},
		// The text leaves child elements out, so their raw markup is kept too.
		{LocURI: "./Vendor/MSFT/Example/Xml", Data: new(""), DataXML: `<Rules><Rule id="1"/></Rules>`},
		{LocURI: "./Vendor/MSFT/Example/Mixed", Data: new("5"), DataXML: `5<x/>`},
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
		dataXML string
		want    bool
	}{
		{name: "int", formats: f(WindowsFormatInt), data: "5", want: true},
		{name: "negative int", formats: f(WindowsFormatInt), data: "-1", want: true},
		{name: "int with whitespace", formats: f(WindowsFormatInt), data: "\n  5\n", want: true},
		{name: "int with letters", formats: f(WindowsFormatInt), data: "abc"},
		{name: "int with real", formats: f(WindowsFormatInt), data: "1.5"},
		{name: "empty int", formats: f(WindowsFormatInt), data: ""},
		{name: "blank int", formats: f(WindowsFormatInt), data: "  "},
		{name: "int with child element", formats: f(WindowsFormatInt), data: "5", dataXML: "5<x/>"},

		{name: "bool true", formats: f(WindowsFormatBool), data: "true", want: true},
		{name: "bool false", formats: f(WindowsFormatBool), data: "False", want: true},
		{name: "bool 0", formats: f(WindowsFormatBool), data: "0", want: true},
		{name: "bool 1", formats: f(WindowsFormatBool), data: "1", want: true},
		{name: "bool 2", formats: f(WindowsFormatBool), data: "2"},
		{name: "bool yes", formats: f(WindowsFormatBool), data: "yes"},
		{name: "empty bool", formats: f(WindowsFormatBool), data: ""},

		{name: "b64", formats: f(WindowsFormatB64), data: "Zmxl\n ZXQ=", want: true},
		{name: "invalid b64", formats: f(WindowsFormatB64), data: "not base64!"},
		{name: "empty b64", formats: f(WindowsFormatB64), data: " \n "},

		{name: "xml", formats: f(WindowsFormatXML), data: `<a><b x="1"/></a>`, want: true},
		{name: "unclosed xml", formats: f(WindowsFormatXML), data: `<a><b>`},
		{name: "mismatched xml", formats: f(WindowsFormatXML), data: `<a></b>`},
		{name: "xml with prolog and comments", formats: f(WindowsFormatXML), data: `<?xml version="1.0"?><!-- c --><a/><!-- c -->`, want: true},
		{name: "plain text xml", formats: f(WindowsFormatXML), data: "just text"},
		{name: "comment-only xml", formats: f(WindowsFormatXML), data: "<!-- c -->"},
		{name: "two roots", formats: f(WindowsFormatXML), data: "<a/><b/>"},
		{name: "text after root", formats: f(WindowsFormatXML), data: "<a/>tail"},
		{name: "empty xml", formats: f(WindowsFormatXML), data: ""},
		{name: "raw child xml", formats: f(WindowsFormatXML), dataXML: `<Rules><Rule id="1"/></Rules>`, want: true},
		{name: "raw child xml two roots", formats: f(WindowsFormatXML), dataXML: "<a/><b/>"},
		{name: "raw child xml with text", formats: f(WindowsFormatXML), data: "5", dataXML: "5<x/>"},

		{name: "chr", formats: f(WindowsFormatChr), data: "anything", want: true},
		{name: "empty chr", formats: f(WindowsFormatChr), data: "", want: true},
		{name: "chr with child element", formats: f(WindowsFormatChr), dataXML: "<x/>", want: true},
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
		{name: "secret in raw child xml", formats: f(WindowsFormatXML), dataXML: "<a/>$FLEET_SECRET_RULES", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, windowsDataAccepts(c.formats, c.data, c.dataXML))
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
		{name: "renamed template install", locURI: "./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/CloudManagementEnrollmentToken", want: node("./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome/CloudManagementEnrollmentToken")},
		{name: "renamed template install user", locURI: "./User/Vendor/MSFT/Policy/Config/MyChrome~policy~GoogleChrome~Startup/homepagelocation", want: node("./User" + chromeHomepage)},
		// An older Chrome ADMX had RelaunchNotification directly under googlechrome.
		{name: "template policy in another category", locURI: "./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification"},
		{name: "unknown ADMX", locURI: "./Device/Vendor/MSFT/Policy/Config/Contoso~Policy~contoso/Setting"},
		{name: "policy typo", locURI: "./Device/Vendor/MSFT/Policy/Config/DeviceLock/Typoo"},
		{name: "policy area alone", locURI: "./Device/Vendor/MSFT/Policy/Config/Unknown"},
		{name: "unknown CSP", locURI: "./Device/Vendor/MSFT/Unknown/Setting"},
		// Devices accept a LocURI without "./", and so does the structural validator.
		{name: "not rooted", locURI: "Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", want: node("./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall")},
		{name: "not rooted device", locURI: "Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock", want: node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock")},
		{name: "not rooted user for device-only policy", locURI: "User/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock", want: node("./Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock"), wrongScope: true},
		// Lowercasing Ⱥ adds a byte, so an index into the lowercased LocURI would overrun it.
		{name: "unicode app name", locURI: "./Device/Vendor/MSFT/Policy/Config/" + strings.Repeat("Ⱥ", 64) + "~Policy~googlechrome~Startup/HomepageLocation", want: node("./Device" + chromeHomepage)},
		{name: "unicode app name unknown policy", locURI: "./Device/Vendor/MSFT/Policy/Config/" + strings.Repeat("Ⱥ", 64) + "~Policy~googlechrome/Nope"},
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
		"Device/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock":                      "DeviceLock",
		"Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall":                                   "Firewall",
	} {
		assert.Equal(t, want, windowsArea(locURI), locURI)
	}
}

// syncMLItem is a <Replace> item; format and data are left out when empty.
func syncMLItem(locURI, format, data string) string {
	item := "<Replace><Item>"
	if format != "" {
		item += `<Meta><Format xmlns="syncml:metinf">` + format + "</Format></Meta>"
	}
	item += "<Target><LocURI>" + locURI + "</LocURI></Target>"
	if data != "" {
		item += "<Data>" + data + "</Data>"
	}
	return item + "</Item></Replace>\n"
}

func TestValidateWindowsProfile(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)

	const (
		minLength   = "./Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength"
		inactivity  = "./User/Vendor/MSFT/Policy/Config/DeviceLock/MaxInactivityTimeDeviceLock"
		emailSync   = "./Device/Vendor/MSFT/Policy/Config/Accounts/DomainNamesForEmailSync"
		admxBacked  = "./Device/Vendor/MSFT/Policy/Config/ADMX_AdmPwd/POL_AdmPwd"
		unknown     = "./Device/Vendor/MSFT/Unknown/Setting"
		foreignADMX = "./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/RelaunchNotification"
	)
	typeMismatch := func(area, locURI string, expected ...string) Finding {
		return Finding{Code: CodeTypeMismatch, PayloadType: area, KeyPath: locURI, Expected: expected}
	}
	unknownKey := func(area, locURI string) Finding {
		return Finding{Code: CodeUnknownKey, PayloadType: area, KeyPath: locURI}
	}

	cases := []struct {
		name            string
		profile         string
		lenient, strict []Finding
	}{
		{
			name:    "int with letters",
			profile: syncMLItem(minLength, "", "abc"),
			lenient: []Finding{typeMismatch("DeviceLock", minLength, "int")},
			strict:  []Finding{typeMismatch("DeviceLock", minLength, "int")},
		},
		{name: "int without meta", profile: syncMLItem(minLength, "", "5")},
		{name: "bool accepts 1 on bare and device firewall", profile: syncMLItem("./Vendor/MSFT/Firewall/MdmStore/PublicProfile/EnableFirewall", "bool", "true") +
			syncMLItem("./Device/Vendor/MSFT/Firewall/MdmStore/PrivateProfile/EnableFirewall", "bool", "1")},
		{name: "ADMX-backed markup", profile: syncMLItem(admxBacked, "chr", `<![CDATA[<enabled/><data id="AdmPwd_PasswordLength" value="14"/>]]>`)},
		{name: "ADMX-backed escaped markup", profile: syncMLItem(admxBacked, "chr", `&lt;disabled/&gt;`)},
		{
			name:    "unknown LocURI",
			profile: syncMLItem(unknown, "int", "abc"),
			strict:  []Finding{unknownKey("Unknown", unknown)},
		},
		{
			name:    "ADMX policy not in a template",
			profile: syncMLItem(foreignADMX, "chr", "<enabled/>"),
			strict:  []Finding{unknownKey("chrome~Policy~googlechrome", foreignADMX)},
		},
		{name: "renamed template install", profile: syncMLItem("./Device/Vendor/MSFT/Policy/Config/chrome~Policy~googlechrome/CloudManagementEnrollmentToken", "chr", "<enabled/>")},
		{
			name:    "policy typo",
			profile: syncMLItem("./Device/Vendor/MSFT/Policy/Config/DeviceLock/Typoo", "int", "5"),
			strict:  []Finding{unknownKey("DeviceLock", "./Device/Vendor/MSFT/Policy/Config/DeviceLock/Typoo")},
		},
		{
			name:    "user path for device-only policy",
			profile: syncMLItem(inactivity, "int", "5"),
			strict:  []Finding{{Code: CodeWrongScope, PayloadType: "DeviceLock", KeyPath: inactivity, Expected: []string{"device"}, Actual: "user"}},
		},
		{
			name:    "wrong scope still checks data",
			profile: syncMLItem(inactivity, "int", "abc"),
			lenient: []Finding{typeMismatch("DeviceLock", inactivity, "int")},
			strict: []Finding{
				{Code: CodeWrongScope, PayloadType: "DeviceLock", KeyPath: inactivity, Expected: []string{"device"}, Actual: "user"},
				typeMismatch("DeviceLock", inactivity, "int"),
			},
		},
		{
			name:    "meta format disagrees",
			profile: syncMLItem(emailSync, "int", "example.com"),
			strict:  []Finding{{Code: CodeFormatMismatch, PayloadType: "Accounts", KeyPath: emailSync, Expected: []string{"chr"}, Actual: "int"}},
		},
		{name: "meta format case", profile: syncMLItem(emailSync, "CHR", "example.com")},
		{
			name: "chrome template policy and its ADMX install",
			profile: syncMLItem("./Device/Vendor/MSFT/Policy/ConfigOperations/ADMXInstall/GoogleChrome/Policy/GoogleChromeAdmx", "chr",
				`<![CDATA[<policyDefinitions revision="1.0" schemaVersion="1.0"></policyDefinitions>]]>`) +
				syncMLItem("./Device/Vendor/MSFT/Policy/Config/GoogleChrome~Policy~googlechrome~Startup/HomepageLocation", "chr",
					`<![CDATA[<enabled/><data id="HomepageLocation" value="https://fleetdm.com"/>]]>`),
		},
		{name: "variable in chr", profile: syncMLItem(emailSync, "chr", "$FLEET_VAR_HOST_END_USER_IDP_USERNAME")},
		{
			name:    "variable in int",
			profile: syncMLItem(minLength, "int", "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"),
			lenient: []Finding{typeMismatch("DeviceLock", minLength, "int")},
			strict:  []Finding{typeMismatch("DeviceLock", minLength, "int")},
		},
		{name: "secret in int", profile: syncMLItem(minLength, "int", "$FLEET_SECRET_MIN_LENGTH")},
		{name: "variable in dynamic segment", profile: syncMLItem("./User/Vendor/MSFT/ClientCertificateInstall/SCEP/$FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID/Install/KeyLength", "int", "2048")},
		{name: "secret-only body", profile: "$FLEET_SECRET_WINDOWS_PROFILE"},
		{
			name:    "empty int",
			profile: syncMLItem(minLength, "int", "") + "<Replace><Item><Target><LocURI>" + minLength + "</LocURI></Target><Data/></Item></Replace>",
			lenient: []Finding{typeMismatch("DeviceLock", minLength, "int")},
			strict:  []Finding{typeMismatch("DeviceLock", minLength, "int")},
		},
		{name: "empty chr", profile: "<Replace><Item><Target><LocURI>./Device/Vendor/MSFT/Policy/Config/UserRights/CreateToken</LocURI></Target><Data></Data></Item></Replace>"},
		{
			name:    "int with child element",
			profile: syncMLItem(minLength, "int", "5<x/>"),
			lenient: []Finding{typeMismatch("DeviceLock", minLength, "int")},
			strict:  []Finding{typeMismatch("DeviceLock", minLength, "int")},
		},
		{
			name:    "not rooted keeps the LocURI as written",
			profile: syncMLItem("Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", "int", "abc"),
			lenient: []Finding{typeMismatch("DeviceLock", "Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", "int")},
			strict:  []Finding{typeMismatch("DeviceLock", "Device/Vendor/MSFT/Policy/Config/DeviceLock/MinDevicePasswordLength", "int")},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.ValidateWindowsProfile([]byte(c.profile), ModeLenient)
			require.NoError(t, err)
			assert.Equal(t, c.lenient, got, "lenient")
			got, err = s.ValidateWindowsProfile([]byte(c.profile), ModeStrict)
			require.NoError(t, err)
			assert.Equal(t, c.strict, got, "strict")
		})
	}

	_, err = s.ValidateWindowsProfile([]byte(`<Replace><Item>`), ModeLenient)
	require.Error(t, err)
}

func TestValidateWindowsProfileFixture(t *testing.T) {
	node := func(formats ...WindowsFormat) *WindowsNode {
		return &WindowsNode{Format: formats, Scope: WindowsScopeDevice}
	}
	s := &ProfileValidationSchema{Windows: map[string]*WindowsNode{
		"./Device/Vendor/MSFT/Example/Float":  node(WindowsFormatFloat),
		"./Device/Vendor/MSFT/Example/IntChr": node(WindowsFormatInt, WindowsFormatChr),
		"./Device/Vendor/MSFT/Example/Future": node("future"),
		"./Device/Vendor/MSFT/Example/Any":    node(WindowsFormatAny),
	}}
	profile := syncMLItem("./Device/Vendor/MSFT/Example/Float", "float", "abc") +
		syncMLItem("./Device/Vendor/MSFT/Example/IntChr", "chr", "abc") +
		syncMLItem("./Device/Vendor/MSFT/Example/IntChr", "int", "5") +
		// Formats the validator doesn't know accept any Data and any Meta format.
		syncMLItem("./Device/Vendor/MSFT/Example/Future", "int", "abc") +
		syncMLItem("./Device/Vendor/MSFT/Example/Any", "int", "abc")
	got, err := s.ValidateWindowsProfile([]byte(profile), ModeStrict)
	require.NoError(t, err)
	assert.Empty(t, got)

	got, err = s.ValidateWindowsProfile([]byte(syncMLItem("./Device/Vendor/MSFT/Example/IntChr", "bool", "abc")), ModeStrict)
	require.NoError(t, err)
	assert.Equal(t, []Finding{{Code: CodeFormatMismatch, PayloadType: "Example", KeyPath: "./Device/Vendor/MSFT/Example/IntChr", Expected: []string{"int", "chr"}, Actual: "bool"}}, got)
}
