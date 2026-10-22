package profilevalidation

import (
	"encoding/json"
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
