package fleet

import (
	"encoding/json"
	"fmt"
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

func TestVPPAppStoreAppMarshalJSON(t *testing.T) {
	plist := []byte(`<dict><key>K</key><string>v</string></dict>`)
	androidConfig := []byte(`{"a":1}`)

	cases := []struct {
		name              string
		app               VPPAppStoreApp
		wantConfigPresent bool
		wantConfigOnWire  any
	}{
		{
			name:              "iOS plist emitted as JSON string",
			app:               VPPAppStoreApp{VPPAppID: VPPAppID{Platform: IOSPlatform}, Configuration: plist},
			wantConfigPresent: true,
			wantConfigOnWire:  string(plist),
		},
		{
			name:              "iPadOS plist emitted as JSON string",
			app:               VPPAppStoreApp{VPPAppID: VPPAppID{Platform: IPadOSPlatform}, Configuration: plist},
			wantConfigPresent: true,
			wantConfigOnWire:  string(plist),
		},
		{
			name:              "Android raw JSON emitted as JSON object",
			app:               VPPAppStoreApp{VPPAppID: VPPAppID{Platform: AndroidPlatform}, Configuration: androidConfig},
			wantConfigPresent: true,
			wantConfigOnWire:  map[string]any{"a": float64(1)},
		},
		{
			name:              "iOS with nil Configuration omits field",
			app:               VPPAppStoreApp{VPPAppID: VPPAppID{Platform: IOSPlatform}},
			wantConfigPresent: false,
		},
		{
			name:              "Android with nil Configuration omits field",
			app:               VPPAppStoreApp{VPPAppID: VPPAppID{Platform: AndroidPlatform}},
			wantConfigPresent: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(c.app)
			require.NoError(t, err)

			var parsed map[string]any
			require.NoError(t, json.Unmarshal(data, &parsed))

			cfg, has := parsed["configuration"]
			require.Equal(t, c.wantConfigPresent, has)
			if c.wantConfigPresent {
				require.Equal(t, c.wantConfigOnWire, cfg)
			}
		})
	}
}

func TestVPPAppStoreAppUnmarshalJSON(t *testing.T) {
	plist := []byte(`<dict><key>K</key><string>v</string></dict>`)
	androidConfig := []byte(`{"a":1}`)

	cases := []struct {
		name       string
		wireJSON   string
		wantConfig []byte
	}{
		{
			name:       "iOS plist string decodes to raw bytes",
			wireJSON:   fmt.Sprintf(`{"platform":"ios","configuration":%q}`, string(plist)),
			wantConfig: plist,
		},
		{
			name:       "iPadOS plist string decodes to raw bytes",
			wireJSON:   fmt.Sprintf(`{"platform":"ipados","configuration":%q}`, string(plist)),
			wantConfig: plist,
		},
		{
			name:       "Android JSON object decodes to raw bytes",
			wireJSON:   fmt.Sprintf(`{"platform":"android","configuration":%s}`, string(androidConfig)),
			wantConfig: androidConfig,
		},
		{
			name:       "missing configuration field leaves nil",
			wireJSON:   `{"platform":"ios"}`,
			wantConfig: nil,
		},
		{
			name:       "iOS configuration null leaves nil",
			wireJSON:   `{"platform":"ios","configuration":null}`,
			wantConfig: nil,
		},
		{
			name:       "Android configuration null leaves nil",
			wireJSON:   `{"platform":"android","configuration":null}`,
			wantConfig: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got VPPAppStoreApp
			require.NoError(t, json.Unmarshal([]byte(c.wireJSON), &got))
			require.Equal(t, c.wantConfig, got.Configuration)
		})
	}
}

func TestVPPAppStoreAppJSONRoundTrip(t *testing.T) {
	plist := []byte(`<dict><key>K</key><string>v</string></dict>`)
	androidConfig := []byte(`{"a":1}`)

	cases := []VPPAppStoreApp{
		{VPPAppID: VPPAppID{AdamID: "1", Platform: IOSPlatform}, Configuration: plist},
		{VPPAppID: VPPAppID{AdamID: "2", Platform: IPadOSPlatform}, Configuration: plist},
		{VPPAppID: VPPAppID{AdamID: "3", Platform: AndroidPlatform}, Configuration: androidConfig},
		{VPPAppID: VPPAppID{AdamID: "4", Platform: IOSPlatform}},
		{VPPAppID: VPPAppID{AdamID: "5", Platform: AndroidPlatform}},
	}

	for _, original := range cases {
		t.Run(string(original.Platform)+"_"+original.AdamID, func(t *testing.T) {
			data, err := json.Marshal(original)
			require.NoError(t, err)

			var roundTripped VPPAppStoreApp
			require.NoError(t, json.Unmarshal(data, &roundTripped))

			require.Equal(t, original.Configuration, roundTripped.Configuration)
			require.Equal(t, original.Platform, roundTripped.Platform)
			require.Equal(t, original.AdamID, roundTripped.AdamID)
		})
	}
}
