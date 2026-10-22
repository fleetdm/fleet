package profilevalidation

import (
	"testing"

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
