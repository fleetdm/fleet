package fleet

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAppleAppConfiguration(t *testing.T) {
	const fragment = `<dict>
	<key>ServerURL</key>
	<string>https://example.com</string>
	<key>EnableTelemetry</key>
	<true/>
</dict>`

	const fullDoc = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ServerURL</key>
	<string>https://example.com</string>
</dict>
</plist>`

	const nested = `<dict>
	<key>Outer</key>
	<dict>
		<key>Inner</key>
		<string>value</string>
	</dict>
	<key>List</key>
	<array>
		<dict>
			<key>K</key>
			<string>v</string>
		</dict>
	</array>
</dict>`

	cases := []struct {
		name    string
		input   string
		wantErr bool
		errSub  string
	}{
		{name: "empty", input: ""},
		{name: "bare dict fragment", input: fragment},
		{name: "full plist document", input: fullDoc},
		{name: "nested dict and array of dicts", input: nested},
		{name: "garbage non-XML", input: "not a plist", wantErr: true, errSub: "invalid plist"},
		{name: "malformed XML unclosed tag", input: "<dict><key>foo</key><string>bar", wantErr: true, errSub: "invalid plist"},
		{name: "root is array", input: `<array><string>x</string></array>`, wantErr: true, errSub: "invalid plist"},
		{name: "root is string", input: `<string>oops</string>`, wantErr: true, errSub: "invalid plist"},
		{
			name:  "allowed variable",
			input: `<dict><key>HostID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`,
		},
		{
			name:  "allowed variable with braces",
			input: `<dict><key>HostID</key><string>${FLEET_VAR_HOST_UUID}</string></dict>`,
		},
		{
			name:  "multiple allowed variables in one string",
			input: `<dict><key>K</key><string>https://x/$FLEET_VAR_HOST_UUID/$FLEET_VAR_HOST_HARDWARE_SERIAL</string></dict>`,
		},
		{
			name:    "credential variable not allowed in app config",
			input:   `<dict><key>K</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "unknown variable name",
			input:   `<dict><key>K</key><string>$FLEET_VAR_BOGUS_NAME</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_BOGUS_NAME",
		},
		{
			name: "all standard plist value types accepted",
			input: `<dict>
	<key>S</key><string>val</string>
	<key>I</key><integer>42</integer>
	<key>R</key><real>3.14</real>
	<key>T</key><true/>
	<key>F</key><false/>
	<key>D</key><data>YWJj</data>
	<key>A</key><array><string>x</string><integer>1</integer></array>
</dict>`,
		},
		{
			name:    "ASCII control character in string value",
			input:   "<dict><key>K</key><string>x\x01y</string></dict>",
			wantErr: true,
			errSub:  "invalid plist",
		},
		{
			name:    "json null token",
			input:   "null",
			wantErr: true,
			errSub:  "invalid plist",
		},
		{
			name:    "hex-entity-encoded $ does not bypass disallow list",
			input:   `<dict><key>K</key><string>&#x24;FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "decimal-entity-encoded $ does not bypass disallow list",
			input:   `<dict><key>K</key><string>&#36;FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "disallowed variable inside a CDATA section is caught",
			input:   `<dict><key>K</key><string><![CDATA[$FLEET_VAR_NDES_SCEP_CHALLENGE]]></string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "disallowed variable nested inside an array is caught",
			input:   `<dict><key>K</key><array><dict><key>Inner</key><string>$FLEET_VAR_BOGUS</string></dict></array></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_BOGUS",
		},
		{
			name:    "disallowed variable used as a key is caught",
			input:   `<dict><key>$FLEET_VAR_BOGUS</key><string>x</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_BOGUS",
		},
		{
			name:    "duplicate key bypass: disallowed var hidden by last-wins map semantics",
			input:   `<dict><key>K</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string><key>K</key><string>safe_value</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "trailing sibling bypass: disallowed var in element dropped by parser",
			input:   `<dict><key>K</key><string>safe</string></dict><dict><key>X</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`,
			wantErr: true,
			errSub:  "unsupported variable $FLEET_VAR_NDES_SCEP_CHALLENGE",
		},
		{
			name:    "openstep dict format rejected",
			input:   `{ServerURL = "https://x.com";}`,
			wantErr: true,
			errSub:  "must be an XML plist",
		},
		{
			name:    "binary plist rejected",
			input:   "bplist00\xd1\x01\x02Q1Q2\x08\x0b\r\x00\x00\x00\x00\x00\x00\x01\x01\x00\x00\x00\x00\x00\x00\x00\x03\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x0f",
			wantErr: true,
			errSub:  "must be an XML plist",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := ValidateAppleAppConfiguration([]byte(c.input))
			if c.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.errSub)
				var iae *InvalidArgumentError
				require.ErrorAs(t, err, &iae)
				return
			}
			require.NoError(t, err)
		})
	}
}
