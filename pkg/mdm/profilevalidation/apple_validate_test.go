package profilevalidation

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// plistDict wraps dictionary content in a plist document.
func plistDict(content string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>` + content + `</dict></plist>`
}

func TestDecodeAppleProfileKinds(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		{name: "string", value: `<string>x</string>`, want: "string"},
		{name: "fleet variable is a string", value: `<string>$FLEET_VAR_HOST_UUID</string>`, want: "string"},
		{name: "true", value: `<true/>`, want: "boolean"},
		{name: "false", value: `<false/>`, want: "boolean"},
		{name: "integer", value: `<integer>5</integer>`, want: "integer"},
		{name: "negative integer", value: `<integer>-1</integer>`, want: "integer"},
		{name: "real", value: `<real>1.5</real>`, want: "real"},
		{name: "date", value: `<date>2026-01-01T00:00:00Z</date>`, want: "date"},
		{name: "data", value: `<data>AQID</data>`, want: "data"},
		{name: "digicert placeholder blanked", value: `<data>$FLEET_VAR_DIGICERT_DATA_CA</data>`, want: "data"},
		{name: "braced digicert placeholder blanked", value: `<data>${FLEET_VAR_DIGICERT_DATA_CA}</data>`, want: "data"},
		{name: "array", value: `<array><string>a</string></array>`, want: "array"},
		{name: "dictionary", value: `<dict><key>a</key><true/></dict>`, want: "dictionary"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root, err := decodeAppleProfile([]byte(plistDict(`<key>k</key>` + c.value)))
			require.NoError(t, err)
			assert.Equal(t, c.want, kindOf(root["k"]))
		})
	}
}

func TestDecodeAppleProfileErrors(t *testing.T) {
	cases := []struct {
		name    string
		data    string
		wantErr string
	}{
		{name: "signed", data: "0\x82\x01\x00 binary pkcs7", wantErr: "signed profiles are not supported"},
		{name: "invalid xml", data: `<?xml version="1.0"?><plist><dict><key>k</key>`, wantErr: "decode profile"},
		{name: "root not a dictionary", data: `<?xml version="1.0"?><plist version="1.0"><array/></plist>`, wantErr: "root is array"},
		// Only expanded or DigiCert variables can appear where the element needs a number or base64.
		{name: "secret in integer element", data: plistDict(`<key>k</key><integer>$FLEET_SECRET_X</integer>`), wantErr: "decode profile"}, // we fail to parse it in the upload path as the XML parser sees a string "$FLEET_SECRET" instead of a number, so I kept the same here.
		{name: "non-digicert variable in data", data: plistDict(`<key>k</key><data>$FLEET_VAR_X</data>`), wantErr: "decode profile"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := decodeAppleProfile([]byte(c.data))
			require.ErrorContains(t, err, c.wantErr)
		})
	}
}

func TestDecodeAppleDeclarationKinds(t *testing.T) {
	root, err := decodeAppleDeclaration([]byte(`{
		"s": "x", "b": true, "i": 5, "neg": -1, "r": 1.5, "e": 1e3,
		"a": [], "d": {}, "n": null
	}`))
	require.NoError(t, err)
	want := map[string]string{
		"s": "string", "b": "boolean", "i": "integer", "neg": "integer", "r": "real", "e": "real",
		"a": "array", "d": "dictionary", "n": "null",
	}
	for k, kind := range want {
		assert.Equal(t, kind, kindOf(root[k]), k)
	}
}

func TestDecodeAppleDeclarationErrors(t *testing.T) {
	_, err := decodeAppleDeclaration([]byte(`{"Type": `))
	require.ErrorContains(t, err, "decode declaration")
	_, err = decodeAppleDeclaration([]byte(`[]`))
	require.ErrorContains(t, err, "root is array")
	_, err = decodeAppleDeclaration([]byte(`{"Type": "x"} garbage`))
	require.ErrorContains(t, err, "unexpected data after the top-level object")
	_, err = decodeAppleDeclaration([]byte(`{"Type": "x"} {"Type": "y"}`))
	require.ErrorContains(t, err, "unexpected data after the top-level object")
	_, err = decodeAppleDeclaration([]byte("{\"Type\": \"x\"}\n\t "))
	require.NoError(t, err, "trailing whitespace is fine")
}

func TestTypeAccepts(t *testing.T) {
	key := func(types ...AppleType) *AppleKey { return &AppleKey{Type: types} }
	timestamp := &AppleKey{Type: AppleTypes{AppleTypeString}, ValueType: "<timestamp>"}
	localTime := &AppleKey{Type: AppleTypes{AppleTypeString}, ValueType: "localtime"}
	url := &AppleKey{Type: AppleTypes{AppleTypeString}, ValueType: "<url>"}
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		key         *AppleKey
		value       any
		declaration bool
		want        bool
	}{
		{name: "string in string", key: key(AppleTypeString), value: "x", want: true},
		{name: "string in boolean", key: key(AppleTypeBoolean), value: "true"},
		{name: "boolean in boolean", key: key(AppleTypeBoolean), value: true, want: true},
		{name: "integer in integer", key: key(AppleTypeInteger), value: uint64(1), want: true},
		{name: "real in integer", key: key(AppleTypeInteger), value: 1.5, want: true},
		{name: "integer in real", key: key(AppleTypeReal), value: int64(-1), want: true},
		{name: "float32 in real", key: key(AppleTypeReal), value: float32(1.5), want: true},
		{name: "json integer in real", key: key(AppleTypeReal), value: json.Number("1"), declaration: true, want: true},
		{name: "string in integer", key: key(AppleTypeInteger), value: "1"},
		{name: "array in dictionary", key: key(AppleTypeDictionary), value: []any{}},
		{name: "dictionary in dictionary", key: key(AppleTypeDictionary), value: map[string]any{}, want: true},
		{name: "array in array", key: key(AppleTypeArray), value: []any{}, want: true},
		{name: "data in data", key: key(AppleTypeData), value: []byte{}, want: true},
		{name: "date in date", key: key(AppleTypeDate), value: date, want: true},

		{name: "list accepts integer", key: key(AppleTypeInteger, AppleTypeString), value: uint64(1), want: true},
		{name: "list accepts string", key: key(AppleTypeInteger, AppleTypeString), value: "x", want: true},
		{name: "list rejects boolean", key: key(AppleTypeInteger, AppleTypeString), value: true},

		{name: "any", key: key(AppleTypeAny), value: true, want: true},
		{name: "unknown type", key: key("set"), value: true, want: true},
		{name: "unknown type in list", key: key(AppleTypeInteger, "set"), value: true, want: true},
		{name: "no types", key: key(), value: true, want: true},

		{name: "date in timestamp string", key: timestamp, value: date, want: true},
		{name: "date in localtime string", key: localTime, value: date, want: true},
		{name: "date in url string", key: url, value: date},
		{name: "date in plain string", key: key(AppleTypeString), value: date},
		{name: "unknown valuetype accepts a date", key: &AppleKey{Type: AppleTypes{AppleTypeString}, ValueType: "<future>"}, value: date, want: true},
		{name: "unknown valuetype accepts a boolean", key: &AppleKey{Type: AppleTypes{AppleTypeString}, ValueType: "future"}, value: true, want: true},

		{name: "declaration string in data", key: key(AppleTypeData), value: "AQID", declaration: true, want: true},
		{name: "declaration string in date", key: key(AppleTypeDate), value: "2026-01-01T00:00:00Z", declaration: true, want: true},
		{name: "profile string in data", key: key(AppleTypeData), value: "AQID"},
		{name: "profile string in date", key: key(AppleTypeDate), value: "2026-01-01T00:00:00Z"},

		{name: "fleet variable in string", key: key(AppleTypeString), value: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME", want: true},
		{name: "fleet variable in integer", key: key(AppleTypeInteger), value: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"},
		{name: "fleet variable in data", key: key(AppleTypeData), value: "$FLEET_VAR_HOST_END_USER_IDP_USERNAME"},
		{name: "secret in integer", key: key(AppleTypeInteger), value: "$FLEET_SECRET_PORT", want: true},
		{name: "braced secret inside a string in boolean", key: key(AppleTypeBoolean), value: "x${FLEET_SECRET_ON}", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, typeAccepts(c.key, c.value, c.declaration))
		})
	}
}

// walkFixture is a payload type covering the walker's rules. %s is where a test can add an extra
// field the validator doesn't know.
const walkFixture = `{"keys": {
	"Flag": {"type": "boolean"},
	"Deferrals": {"type": "dictionary", "keys": {"MajorPeriodInDays": {"type": "integer"%s}}},
	"Services": {"type": "dictionary", "keys": {
		"Camera": {"type": "array", "items": {"type": "dictionary", "keys": {"Allowed": {"type": "boolean"}}}}
	}},
	"Open": {"type": "dictionary", "open": true, "keys": {"Known": {"type": "integer"}}},
	"Headers": {"type": "dictionary", "open": true, "values": {"type": "string"}},
	"Settings": {"type": "any"},
	"Set": {"type": "set"},
	"Plain": {"type": "dictionary"},
	"Future": {"type": "dictionary", "valuetype": "<future>", "keys": {"X": {"type": "integer"}}},
	"com.example.dotted": {"type": "integer"}
}}`

func walkProfile(t *testing.T, pt *ApplePayloadType, mode Mode, content string) []Finding {
	t.Helper()
	dict, err := decodeAppleProfile([]byte(plistDict(content)))
	require.NoError(t, err)
	w := &appleWalker{root: pt.Keys, mode: mode, payloadType: "com.example.test"}
	w.walkKeys(pt.Keys, pt.Open, pt.Values, dict, "")
	return w.findings
}

func TestAppleWalker(t *testing.T) {
	var pt ApplePayloadType
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, walkFixture, ""), &pt))

	mismatch := func(path string, expected []string, actual string) Finding {
		return Finding{Code: CodeTypeMismatch, PayloadType: "com.example.test", KeyPath: path, Expected: expected, Actual: actual}
	}
	unknown := func(path string) Finding {
		return Finding{Code: CodeUnknownKey, PayloadType: "com.example.test", KeyPath: path}
	}

	cases := []struct {
		name       string
		content    string
		lenient    []Finding
		strictOnly []Finding
	}{
		{name: "valid", content: `<key>Flag</key><true/><key>Deferrals</key><dict><key>MajorPeriodInDays</key><integer>3</integer></dict>`},
		{
			name:    "top-level mismatch",
			content: `<key>Flag</key><string>true</string>`,
			lenient: []Finding{mismatch("Flag", []string{"boolean"}, "string")},
		},
		{
			name:    "nested mismatch",
			content: `<key>Deferrals</key><dict><key>MajorPeriodInDays</key><string>3</string></dict>`,
			lenient: []Finding{mismatch("Deferrals.MajorPeriodInDays", []string{"integer"}, "string")},
		},
		{
			name: "array positions in the path",
			content: `<key>Services</key><dict><key>Camera</key><array>
				<dict><key>Allowed</key><true/></dict><dict><key>Allowed</key><string>no</string></dict>
			</array></dict>`,
			lenient: []Finding{mismatch("Services.Camera[1].Allowed", []string{"boolean"}, "string")},
		},
		{
			name:    "a mismatched dictionary isn't walked",
			content: `<key>Deferrals</key><string>soon</string>`,
			lenient: []Finding{mismatch("Deferrals", []string{"dictionary"}, "string")},
		},
		{
			name:    "dotted key name written as-is",
			content: `<key>com.example.dotted</key><true/>`,
			lenient: []Finding{mismatch("com.example.dotted", []string{"integer"}, "boolean")},
		},
		{
			name:    "open dictionary checks listed keys only",
			content: `<key>Open</key><dict><key>Known</key><string>x</string><key>Anything</key><true/></dict>`,
			lenient: []Finding{mismatch("Open.Known", []string{"integer"}, "string")},
		},
		{
			name:    "open dictionary values are checked, names never",
			content: `<key>Headers</key><dict><key>X-Good</key><string>a</string><key>X-Bad</key><integer>1</integer></dict>`,
			lenient: []Finding{mismatch("Headers.X-Bad", []string{"string"}, "integer")},
		},
		{name: "any stops the walk", content: `<key>Settings</key><dict><key>Deep</key><dict><key>X</key><true/></dict></dict>`},
		{name: "unrecognized type accepts anything", content: `<key>Set</key><array><true/><string>x</string></array>`},
		{name: "dictionary without keys isn't walked", content: `<key>Plain</key><dict><key>Anything</key><true/></dict>`},
		{name: "unrecognized valuetype accepts anything", content: `<key>Future</key><dict><key>X</key><string>x</string><key>Y</key><true/></dict>`},
		{
			name:       "unknown keys are strict only",
			content:    `<key>Flagg</key><true/><key>Deferrals</key><dict><key>Typo</key><true/></dict>`,
			strictOnly: []Finding{unknown("Deferrals.Typo"), unknown("Flagg")},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.lenient, walkProfile(t, &pt, ModeLenient, c.content))
			strict := append(slices.Clone(c.lenient), c.strictOnly...)
			slices.SortStableFunc(strict, func(a, b Finding) int { return strings.Compare(a.KeyPath, b.KeyPath) })
			got := walkProfile(t, &pt, ModeStrict, c.content)
			if len(strict) == 0 {
				strict = nil
			}
			assert.Equal(t, strict, got)
		})
	}
}

func TestAppleWalkerIgnoresUnknownSchemaFields(t *testing.T) {
	var plain, extra ApplePayloadType
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, walkFixture, ""), &plain))
	require.NoError(t, json.Unmarshal(fmt.Appendf(nil, walkFixture, `, "requiredif": {"Flag": true}, "range": {"min": 1}`), &extra))
	for _, content := range []string{
		`<key>Deferrals</key><dict><key>MajorPeriodInDays</key><integer>0</integer></dict>`,
		`<key>Deferrals</key><dict><key>MajorPeriodInDays</key><true/></dict>`,
	} {
		for _, mode := range []Mode{ModeLenient, ModeStrict} {
			assert.Equal(t, walkProfile(t, &plain, mode, content), walkProfile(t, &extra, mode, content))
		}
	}
}

func TestAppleWalkerRef(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)
	layout := notNil(t, s.AppleProfile("com.apple.homescreenlayout"))

	// A wrong-type Type in a folder nested two folders deep, reached through "ref": "Pages[]".
	folder := func(inner string) string {
		return `<dict><key>Type</key><string>Folder</string><key>Pages</key><array><array>` + inner + `</array></array></dict>`
	}
	content := `<key>Pages</key><array><array>` + folder(folder(`<dict><key>Type</key><integer>1</integer></dict>`)) + `</array></array>`
	for _, mode := range []Mode{ModeLenient, ModeStrict} {
		assert.Equal(t, []Finding{{
			Code: CodeTypeMismatch, PayloadType: "com.example.test", KeyPath: "Pages[0][0].Pages[0][0].Pages[0][0].Type",
			Expected: []string{"string"}, Actual: "integer",
		}}, walkProfile(t, layout, mode, content))
	}
}

func TestResolveAppleRefRealSchema(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)
	// Every ref in the embedded schema must resolve, or the walker silently stops checking there.
	var walk func(root map[string]*AppleKey, k *AppleKey, path string)
	var refs int
	walk = func(root map[string]*AppleKey, k *AppleKey, path string) {
		if k == nil {
			return
		}
		if k.Ref != "" {
			refs++
			assert.NotNil(t, resolveAppleRef(root, k.Ref), "%s ref %s", path, k.Ref)
		}
		for name, c := range k.Keys {
			walk(root, c, path+"."+name)
		}
		walk(root, k.Items, path+"[]")
		walk(root, k.Values, path+".ANY")
	}
	for _, types := range []map[string]*ApplePayloadType{s.AppleProfiles, s.AppleDeclarations} {
		for name, pt := range types {
			for key, k := range pt.Keys {
				walk(pt.Keys, k, name+":"+key)
			}
		}
	}
	assert.Positive(t, refs)
}

func TestResolveAppleRef(t *testing.T) {
	var pt ApplePayloadType
	require.NoError(t, json.Unmarshal([]byte(`{"keys": {
		"a.b": {"type": "dictionary", "keys": {"c": {"type": "array", "items": {"type": "integer"}}}},
		"a": {"type": "dictionary", "keys": {"b": {"type": "string"}}}
	}}`), &pt))
	assert.Equal(t, AppleTypes{AppleTypeInteger}, notNil(t, resolveAppleRef(pt.Keys, "a.b.c[]")).Type)
	assert.Equal(t, AppleTypes{AppleTypeDictionary}, notNil(t, resolveAppleRef(pt.Keys, "a.b")).Type)
	assert.Nil(t, resolveAppleRef(pt.Keys, "a.b.d"))
	assert.Nil(t, resolveAppleRef(pt.Keys, "a[]"))
	assert.Nil(t, resolveAppleRef(pt.Keys, "missing"))
}

// mobileconfig wraps payloads in a profile with a valid envelope.
func mobileconfig(payloads ...string) []byte {
	return []byte(plistDict(`<key>PayloadType</key><string>Configuration</string>
		<key>PayloadVersion</key><integer>1</integer>
		<key>PayloadUUID</key><string>0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0</string>
		<key>PayloadIdentifier</key><string>com.example.profile</string>
		<key>PayloadDisplayName</key><string>Example</string>
		<key>PayloadContent</key><array>` + strings.Join(payloads, "") + `</array>`))
}

// payload is a payload dictionary with a valid envelope plus body.
func payload(payloadType, body string) string {
	return `<dict><key>PayloadType</key><string>` + payloadType + `</string>
		<key>PayloadVersion</key><integer>1</integer>
		<key>PayloadUUID</key><string>a1b2c3d4-e5f6-a7b8-c9d0-e1f2a3b4c5d6</string>
		<key>PayloadIdentifier</key><string>com.example.payload</string>` + body + `</dict>`
}

func TestValidateAppleProfile(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)

	const restrictions = "com.apple.applicationaccess"
	mismatch := func(payloadType, path string, expected []string, actual string) Finding {
		return Finding{Code: CodeTypeMismatch, PayloadType: payloadType, KeyPath: path, Expected: expected, Actual: actual}
	}
	mcxPreferences := payload("com.apple.ManagedClient.preferences", `<key>PayloadContent</key><dict>
		<key>com.example.app</key><dict><key>Forced</key><array><dict>
			<key>mcx_preference_settings</key><dict><key>Anything</key><integer>1</integer><key>Nested</key><dict/></dict>
		</dict></array></dict>
	</dict>`)

	cases := []struct {
		name     string
		profile  []byte
		lenient  []Finding
		strict   []Finding
		sameBoth bool // strict reports exactly what lenient does
	}{
		{
			name:     "boolean holding a string",
			profile:  mobileconfig(payload(restrictions, `<key>allowAirDrop</key><string>false</string>`)),
			lenient:  []Finding{mismatch(restrictions, "allowAirDrop", []string{"boolean"}, "string")},
			sameBoth: true,
		},
		{
			name:     "integer holding a real",
			profile:  mobileconfig(payload(restrictions, `<key>enforcedSoftwareUpdateDelay</key><real>3.5</real>`)),
			sameBoth: true,
		},
		{
			name:    "invented payload type",
			profile: mobileconfig(payload("com.apple.example", `<key>Anything</key><true/>`)),
			strict:  []Finding{{Code: CodeUnknownType, PayloadType: "com.apple.example"}},
		},
		{
			name:    "misspelled restrictions key",
			profile: mobileconfig(payload(restrictions, `<key>allowAirDropp</key><false/>`)),
			strict:  []Finding{{Code: CodeUnknownKey, PayloadType: restrictions, KeyPath: "allowAirDropp"}},
		},
		{
			name:    "ManagedClient preferences",
			profile: mobileconfig(mcxPreferences),
			strict:  []Finding{{Code: CodeUnknownType, PayloadType: "com.apple.ManagedClient.preferences"}},
		},
		{
			name: "nested documented key",
			profile: mobileconfig(payload("com.apple.extensiblesso",
				`<key>PlatformSSO</key><dict><key>AllowDeviceIdentifiersInAttestation</key><string>yes</string></dict>`)),
			lenient:  []Finding{mismatch("com.apple.extensiblesso", "PlatformSSO.AllowDeviceIdentifiersInAttestation", []string{"boolean"}, "string")},
			sameBoth: true,
		},
		{
			name:     "fleet variable in a string slot",
			profile:  mobileconfig(payload(restrictions, `<key>PayloadDisplayName</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>`)),
			sameBoth: true,
		},
		{
			name: "fleet variable in integer and data slots",
			profile: mobileconfig(
				payload(restrictions, `<key>enforcedSoftwareUpdateDelay</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>`),
				payload("com.apple.security.pkcs12", `<key>PayloadContent</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>`),
			),
			lenient: []Finding{
				mismatch(restrictions, "enforcedSoftwareUpdateDelay", []string{"integer"}, "string"),
				mismatch("com.apple.security.pkcs12", "PayloadContent", []string{"data"}, "string"),
			},
			sameBoth: true,
		},
		{
			name:     "digicert placeholder in a data slot",
			profile:  mobileconfig(payload("com.apple.security.pkcs12", `<key>PayloadContent</key><data>$FLEET_VAR_DIGICERT_DATA_CA</data>`)),
			sameBoth: true,
		},
		{
			name:     "secret in an integer slot",
			profile:  mobileconfig(payload(restrictions, `<key>enforcedSoftwareUpdateDelay</key><string>${FLEET_SECRET_DELAY}</string>`)),
			sameBoth: true,
		},
		{
			name: "seeded allow entries",
			profile: mobileconfig(
				payload("com.google.Chrome", `<key>HomepageLocation</key><string>https://fleetdm.com</string>`),
				payload("us.zoom.config", `<key>nogoogle</key><integer>1</integer>`),
				payload("com.apple.universalcontrol", `<key>Disable</key><true/>`),
			),
			sameBoth: true,
		},
		{
			name: "no platform checks",
			profile: mobileconfig(
				payload("com.apple.airplay.security", `<key>AccessType</key><string>ANY</string><key>SecurityType</key><string>PASSCODE_ONCE</string>`),
				payload(restrictions, `<key>allowAppClips</key><false/>`),
			),
			sameBoth: true,
		},
		{
			name: "envelope problems",
			profile: []byte(plistDict(`<key>PayloadType</key><string>Configuration</string>
				<key>PayloadUUID</key><string>not-a-uuid</string>
				<key>PayloadIdentifier</key><string>$FLEET_VAR_HOST_UUID</string>
				<key>PayloadContent</key><array><dict>
					<key>PayloadType</key><string>` + restrictions + `</string>
					<key>PayloadUUID</key><string>${FLEET_SECRET_UUID}</string>
					<key>PayloadIdentifier</key><string>restrictions</string>
				</dict></array>`)),
			strict: []Finding{
				{Code: CodeMissingPayloadVersion, PayloadType: TopLevelPayloadType, KeyPath: "PayloadVersion"},
				{Code: CodeMalformedUUID, PayloadType: TopLevelPayloadType, KeyPath: "PayloadUUID"},
				{Code: CodeMissingPayloadVersion, PayloadType: restrictions, KeyPath: "PayloadVersion"},
				{Code: CodeMalformedIdentifier, PayloadType: restrictions, KeyPath: "PayloadIdentifier"},
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.ValidateAppleProfile(c.profile, ModeLenient)
			require.NoError(t, err)
			assert.Equal(t, c.lenient, got)

			want := c.strict
			if c.sameBoth {
				want = c.lenient
			}
			got, err = s.ValidateAppleProfile(c.profile, ModeStrict)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
}

func TestValidateAppleProfileOverrides(t *testing.T) {
	embedded, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)
	s := *embedded
	s.AppleProfiles = maps.Clone(embedded.AppleProfiles)
	// As the generator writes allow entries: open, source override, and keys only when listed.
	s.AppleProfiles["com.example.allowed"] = &ApplePayloadType{Source: AppleSourceOverride, Open: true, Override: "custom domain"}
	s.AppleProfiles["com.example.typed"] = &ApplePayloadType{
		Source: AppleSourceOverride, Open: true, Override: "custom domain",
		Keys: map[string]*AppleKey{"Enabled": {Type: AppleTypes{AppleTypeBoolean}}},
	}
	mcx := *notNil(t, s.AppleProfiles["com.apple.ManagedClient.preferences"])
	mcx.Override = "allowed for this test"
	s.AppleProfiles["com.apple.ManagedClient.preferences"] = &mcx

	profile := mobileconfig(
		payload("com.example.allowed", `<key>Anything</key><dict><key>Deep</key><true/></dict>`),
		payload("com.example.typed", `<key>Enabled</key><string>yes</string><key>Other</key><integer>1</integer>`),
		payload("com.apple.ManagedClient.preferences", `<key>PayloadContent</key><dict/>`),
	)
	for _, mode := range []Mode{ModeLenient, ModeStrict} {
		got, err := s.ValidateAppleProfile(profile, mode)
		require.NoError(t, err)
		assert.Equal(t, []Finding{{
			Code: CodeTypeMismatch, PayloadType: "com.example.typed", KeyPath: "Enabled", Expected: []string{"boolean"}, Actual: "string",
		}}, got)
	}
}

func TestValidateAppleProfileErrors(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)
	_, err = s.ValidateAppleProfile([]byte(plistDict(`<key>PayloadType</key><string>com.apple.applicationaccess</string>`)), ModeLenient)
	require.ErrorContains(t, err, "invalid PayloadType: com.apple.applicationaccess")
	_, err = s.ValidateAppleProfile([]byte("0\x82 signed"), ModeStrict)
	require.ErrorContains(t, err, "signed profiles are not supported")
}

func TestValidateAppleDeclaration(t *testing.T) {
	s, err := Parse(profileValidationSchemaRaw)
	require.NoError(t, err)

	const lockType = "com.apple.configuration.passcode.settings"
	declaration := func(declarationType, payload string) []byte {
		return []byte(`{"Type": "` + declarationType + `", "Identifier": "com.example.declaration", "Payload": ` + payload + `}`)
	}

	cases := []struct {
		name     string
		data     []byte
		lenient  []Finding
		strict   []Finding
		sameBoth bool
	}{
		{
			name:     "valid",
			data:     declaration(lockType, `{"RequirePasscode": true, "MinimumLength": 8}`),
			sameBoth: true,
		},
		{
			name: "wrong type in Payload",
			data: declaration(lockType, `{"RequirePasscode": "yes", "MinimumLength": 8.5}`),
			lenient: []Finding{{
				Code: CodeTypeMismatch, PayloadType: lockType, KeyPath: "RequirePasscode", Expected: []string{"boolean"}, Actual: "string",
			}},
			sameBoth: true,
		},
		{
			name:   "unknown Type",
			data:   declaration("com.apple.configuration.example", `{"Anything": 1}`),
			strict: []Finding{{Code: CodeUnknownType, PayloadType: "com.apple.configuration.example"}},
		},
		{
			name: "unknown keys",
			data: []byte(`{"Type": "` + lockType + `", "Identifier": "x", "ServerToken": "t", "Payload": {"RequirePasscod": true}}`),
			strict: []Finding{
				{Code: CodeUnknownKey, PayloadType: lockType, KeyPath: "ServerToken"},
				{Code: CodeUnknownKey, PayloadType: lockType, KeyPath: "RequirePasscod"},
			},
		},
		{
			name: "envelope key of the wrong type",
			data: []byte(`{"Type": "` + lockType + `", "Identifier": 1, "Payload": []}`),
			lenient: []Finding{
				{Code: CodeTypeMismatch, PayloadType: lockType, KeyPath: "Identifier", Expected: []string{"string"}, Actual: "integer"},
				{Code: CodeTypeMismatch, PayloadType: lockType, KeyPath: "Payload", Expected: []string{"dictionary"}, Actual: "array"},
			},
			sameBoth: true,
		},
		{
			name:     "JSON string in a data slot",
			data:     declaration("com.apple.configuration.extensible-sso", `{"PlatformSSO": {"AccessKey": {"ReaderGroupIdentifier": "AQID"}}}`),
			sameBoth: true,
		},
		{
			name:     "secret in an integer slot",
			data:     declaration(lockType, `{"MinimumLength": "$FLEET_SECRET_LENGTH"}`),
			sameBoth: true,
		},
		{
			name:     "activation",
			data:     declaration("com.apple.activation.simple", `{"StandardConfigurations": ["com.example.declaration"]}`),
			sameBoth: true,
		},
		{
			name: "activation with a wrong item",
			data: declaration("com.apple.activation.simple", `{"StandardConfigurations": ["a", 1]}`),
			lenient: []Finding{{
				Code: CodeTypeMismatch, PayloadType: "com.apple.activation.simple", KeyPath: "StandardConfigurations[1]",
				Expected: []string{"string"}, Actual: "integer",
			}},
			sameBoth: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := s.ValidateAppleDeclaration(c.data, ModeLenient)
			require.NoError(t, err)
			assert.Equal(t, c.lenient, got)

			want := c.strict
			if c.sameBoth {
				want = c.lenient
			}
			got, err = s.ValidateAppleDeclaration(c.data, ModeStrict)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}

	_, err = s.ValidateAppleDeclaration([]byte(`not json`), ModeLenient)
	require.ErrorContains(t, err, "decode declaration")
}
