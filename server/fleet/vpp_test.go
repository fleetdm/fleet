package fleet

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateAppleAppConfiguration(t *testing.T) {
	const wrapHead = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
`
	wrap := func(body string) string { return wrapHead + body + "\n</plist>" }

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

	const appConfig = `<dict>
	<key>ServerURL</key>
	<string>https://app.example.com/enroll?device=$FLEET_VAR_HOST_UUID&amp;source=fleet</string>
	<key>UserName</key>
	<string>${FLEET_VAR_HOST_END_USER_IDP_USERNAME}</string>
	<key>WelcomeMessage</key>
	<string><![CDATA[Welcome to <Example> & friends]]></string>
	<key>DisableAnalytics</key>
	<true/>
	<key>AllowScreenshots</key>
	<false/>
	<key>SyncIntervalMinutes</key>
	<integer>15</integer>
	<key>BrandingOpacity</key>
	<real>0.85</real>
	<key>LicenseExpiry</key>
	<date>2026-01-01T00:00:00Z</date>
	<key>Payload</key>
	<data>
	TWFuYWdlZCBhcHAgY29uZmlndXJhdGlvbiB0ZXN0
	IHBheWxvYWQgZm9yIEZsZWV0IGRldmljZXMu
	</data>
	<key>Network</key>
	<dict>
		<key>Proxy</key>
		<dict>
			<key>Host</key>
			<string>proxy.example.com</string>
			<key>Port</key>
			<integer>8080</integer>
		</dict>
	</dict>
	<key>Accounts</key>
	<array>
		<dict>
			<key>Type</key>
			<string>exchange</string>
			<key>Server</key>
			<string>mail.example.com</string>
		</dict>
		<dict>
			<key>Type</key>
			<string>vpn</string>
			<key>OnDemand</key>
			<true/>
		</dict>
	</array>
</dict>`

	var large strings.Builder
	large.WriteString("<dict>\n")
	for i := range 2000 {
		fmt.Fprintf(&large, "\t<key>Key%d</key>\n\t<string>Value %d</string>\n", i, i)
	}
	large.WriteString("</dict>")

	deep := "<string>leaf</string>"
	for i := range 30 {
		if i%2 == 0 {
			deep = "<array>" + deep + "</array>"
		} else {
			deep = "<dict><key>K</key>" + deep + "</dict>"
		}
	}
	deep = "<dict><key>Root</key>" + deep + "</dict>"

	const prettyConfig = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ServerURL</key>
	<string>https://app.example.com</string>
	<key>SyncIntervalMinutes</key>
	<integer>15</integer>
	<key>RetryOffset</key>
	<integer>-3</integer>
	<key>BrandingOpacity</key>
	<real>0.85</real>
	<key>LicenseExpiry</key>
	<date>2026-01-01T00:00:00Z</date>
	<key>Payload</key>
	<data>TWFuYWdlZCBhcHAgY29uZmlndXJhdGlvbg==</data>
	<key>DisableAnalytics</key>
	<true/>
	<key>AllowScreenshots</key>
	<false/>
	<key>Proxies</key>
	<array>
		<dict>
			<key>Host</key>
			<string>proxy.example.com</string>
			<key>Port</key>
			<integer>8080</integer>
		</dict>
	</array>
</dict>
</plist>`

	inDict := func(value string) string { return "<dict><key>K</key>" + value + "</dict>" }

	type testCase struct {
		name    string
		input   string
		wantErr bool
		errSub  string
	}
	cases := []testCase{
		{name: "empty", input: ""},
		{name: "bare dict fragment", input: fragment},
		{name: "full plist document", input: fullDoc},
		{name: "nested dict and array of dicts", input: nested},
		{name: "garbage non-XML", input: "not a plist", wantErr: true, errSub: "must be an XML plist"},
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
			errSub:  "must be an XML plist",
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
			name:    "sibling root hiding a disallowed var is rejected",
			input:   `<dict><key>K</key><string>safe</string></dict><dict><key>X</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`,
			wantErr: true,
			errSub:  "is not allowed after the root element",
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

		{
			name:    "text in dict without a key",
			input:   `<dict>no key</dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unexpected text inside <dict>",
		},
		{
			name:    "extra trailing end tag",
			input:   `<dict><key>a</key><string>b</string></dict></dict>`,
			wantErr: true,
			errSub:  "invalid plist: XML syntax error on line 1: unexpected end element </dict>",
		},
		{
			name:    "extra trailing end tag in full wrapper",
			input:   wrap(`<dict><key>a</key><string>b</string></dict></dict>`),
			wantErr: true,
			errSub:  "invalid plist: XML syntax error on line 4: element <plist> closed by </dict>",
		},
		{
			name:    "sibling dict after root",
			input:   `<dict/><dict/>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <dict> is not allowed after the root element",
		},
		{
			name:    "two values in plist",
			input:   `<plist><dict/><dict/></plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <plist> must contain exactly one value",
		},
		{
			name:    "text after root",
			input:   `<dict/>junk`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unexpected text outside the root element",
		},
		{
			name:    "processing instruction after root",
			input:   `<dict/><?foo bar?>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: processing instruction <?foo?> is not allowed",
		},
		{
			name:    "key followed by key",
			input:   `<dict><key>a</key><key>b</key><string>x</string></dict>`,
			wantErr: true,
			errSub:  `invalid plist: line 1: key "a" has no value`,
		},
		{
			name:    "key at the end of a dict",
			input:   `<dict><key>a</key></dict>`,
			wantErr: true,
			errSub:  `invalid plist: line 1: key "a" has no value`,
		},
		{
			name:    "element inside string",
			input:   `<dict><key>a</key><string>x<dict/>y</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <dict> is not allowed inside <string>",
		},
		{
			name:    "element inside key",
			input:   `<dict><key>a<string/></key><string>b</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <string> is not allowed inside <key>",
		},
		{
			name:    "text inside true",
			input:   `<dict><key>a</key><true>yes</true></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <true> must be empty",
		},
		{
			name:    "whitespace inside true",
			input:   `<dict><key>a</key><true> </true></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <true> must be empty",
		},
		{
			name:    "nested plist as dict value",
			input:   `<dict><key>a</key><plist><dict/></plist></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <plist> is not allowed inside <dict>",
		},
		{
			name:    "key inside array",
			input:   `<dict><key>a</key><array><key>b</key></array></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <key> is not allowed inside <array>",
		},
		{
			name:    "namespaced dict",
			input:   `<x:dict xmlns:x="urn:x"><key>a</key><string>b</string></x:dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: element <dict> must not have a namespace",
		},
		{
			name:    "root prefix bound to the empty namespace",
			input:   `<plist:dict xmlns:plist=""><key>a</key><string>b</string></plist:dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: element <plist:dict> must not have a namespace prefix",
		},
		{
			name:    "key prefix bound to the empty namespace",
			input:   "<dict>\n<a:key xmlns:a=\"\">a</a:key><string>b</string></dict>",
			wantErr: true,
			errSub:  "invalid plist: line 2: element <a:key> must not have a namespace prefix",
		},
		{
			name:    "non-breaking space between elements",
			input:   "<dict>\u00a0<key>a</key><string>b</string></dict>",
			wantErr: true,
			errSub:  "invalid plist: line 1: unexpected text inside <dict>",
		},
		{
			name:    "UTF-8 byte order mark",
			input:   "\xef\xbb\xbf<dict/>",
			wantErr: true,
			errSub:  "invalid plist: remove the UTF-8 byte order mark",
		},
		{
			name:    "comment after plist end tag",
			input:   wrap(`<dict/>`) + "\n<!-- trailing -->",
			wantErr: true,
			errSub:  "invalid plist: line 6: comments are not allowed before <plist> or after </plist>",
		},
		{
			name: "comment between XML declaration and DOCTYPE",
			input: `<?xml version="1.0" encoding="UTF-8"?>
<!-- header -->
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict/></plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 3: comments are not allowed before <!DOCTYPE>",
		},
		{
			name: "comment between DOCTYPE and plist",
			input: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- header -->
<plist version="1.0"><dict/></plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 4: comments are not allowed before <plist> or after </plist>",
		},
		{
			name:    "space in plist end tag",
			input:   `<plist version="1.0"><dict/></plist >`,
			wantErr: true,
			errSub:  "invalid plist: line 1: write the end tag as </plist>",
		},
		{
			name:    "angle bracket in plist attribute",
			input:   `<plist version=">"><dict/></plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: the <plist> start tag must not contain '>' in an attribute value",
		},
		{
			name:    "DOCTYPE with an internal subset",
			input:   `<!DOCTYPE plist [<!ENTITY x "y">]><dict/>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: only a plain <!DOCTYPE plist ...> declaration is allowed before the root element",
		},
		{
			name:    "comment hiding '>' inside DOCTYPE",
			input:   `<!DOCTYPE plist <!-- > -->><dict/>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: only a plain <!DOCTYPE plist ...> declaration is allowed before the root element",
		},
		{
			name:    "whitespace character reference after plist end tag",
			input:   `<plist><dict/></plist>&#32;`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unexpected text outside the root element",
		},
		{
			name:    "whitespace CDATA after plist end tag",
			input:   `<plist><dict/></plist><![CDATA[ ]]>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unexpected text outside the root element",
		},
		{
			name:    "DOCTYPE after root",
			input:   `<dict/><!DOCTYPE plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: only a plain <!DOCTYPE plist ...> declaration is allowed before the root element",
		},
		{
			name:    "comment inside string",
			input:   `<dict><key>a</key><string>x<!-- c -->y</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: comments are not allowed inside <string>",
		},
		{
			name:    "unknown element as dict value",
			input:   `<dict><key>a</key><foo/></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unknown element <foo>",
		},
		{
			name:    "unknown element inside string",
			input:   `<dict><key>a</key><string><foo/></string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: unknown element <foo>",
		},
		{
			name: "comment before DOCTYPE",
			input: `<!-- exported -->
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict/></plist>`,
			wantErr: true,
			errSub:  "invalid plist: line 2: comments are not allowed before <!DOCTYPE>",
		},
		{
			name: "comment before XML declaration",
			input: `<!-- exported -->
<?xml version="1.0" encoding="UTF-8"?>
<dict/>`,
			wantErr: true,
			errSub:  "invalid plist: line 2: the XML declaration must come first",
		},
		{
			name:    "numeric reference longer than 8 digits in string",
			input:   `<dict><key>K</key><string>&#0000000049;</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <string> has a character reference devices can't read",
		},
		{
			name:    "reference to half of a UTF-16 surrogate pair",
			input:   `<dict><key>K</key><string>Hi &#xD83D;&#xDE00;</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <string> has a character reference devices can't read",
		},
		{
			name:    "decimal reference to a surrogate half",
			input:   `<dict><key>K</key><string>&#55357;</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <string> has a character reference devices can't read",
		},
		{
			name:    "bad reference in a multi-line string reports its own line",
			input:   "<dict>\n\t<key>K</key>\n\t<string>first\nsecond &#xD800;\nthird</string>\n</dict>",
			wantErr: true,
			errSub:  "invalid plist: line 4: <string> has a character reference devices can't read",
		},
		{
			name:    "stray text in a pretty-printed dict reports its own line",
			input:   "<dict>\n\t<key>A</key>\n\t<string>a</string>\n\toops\n\n</dict>",
			wantErr: true,
			errSub:  "invalid plist: line 4: unexpected text inside <dict>",
		},
		{
			name:    "'>' in an attribute of a value element",
			input:   `<dict note=">"><key>a</key><string>b</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: the <dict> start tag must not contain '>' in an attribute value",
		},
		{
			name:  "plain attribute on a value element",
			input: `<dict note="managed"><key>a</key><string xml:space="preserve"> b </string></dict>`,
		},
		{
			name:  "nesting at the limit",
			input: strings.Repeat("<dict><key>k</key>", 99) + "<string>leaf</string>" + strings.Repeat("</dict>", 99),
		},
		{
			name:    "nesting past the limit",
			input:   strings.Repeat("<dict><key>k</key>", 100) + "<string>leaf</string>" + strings.Repeat("</dict>", 100),
			wantErr: true,
			errSub:  "invalid plist: line 1: nesting deeper than 100 levels is not allowed",
		},
		{
			name:    "hex reference longer than 8 digits in key",
			input:   `<dict><key>&#x000000041;</key><string>v</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: line 1: <key> has a character reference devices can't read",
		},
		{
			name:    "uppercase hex reference",
			input:   `<dict><key>K</key><string>&#X41;</string></dict>`,
			wantErr: true,
			errSub:  "invalid plist: XML syntax error on line 1: invalid character entity",
		},
		{
			name:  "references devices read, and a reference-like CDATA",
			input: `<dict><key>K&amp;&#x41;</key><string>&lt;&gt;&quot;&apos;&#65;&#x1F600;<![CDATA[&#X41;]]></string></dict>`,
		},
		{
			name:    "key without a value reports the key's line",
			input:   "<dict>\n\t<key>Orphan</key>\n\n\n\t<key>B</key>\n\t<string>b</string>\n</dict>",
			wantErr: true,
			errSub:  `invalid plist: line 2: key "Orphan" has no value`,
		},
		{
			name:    "key without a value at the end of a dict reports the key's line",
			input:   "<dict>\n\t<key>Orphan</key>\n\n</dict>",
			wantErr: true,
			errSub:  `invalid plist: line 2: key "Orphan" has no value`,
		},
		{
			name:    "integer with trailing space",
			input:   inDict(`<integer>8080 </integer>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <integer> value "8080 " is not valid`,
		},
		{
			name:    "integer on its own line",
			input:   "<dict>\n\t<key>SyncIntervalMinutes</key>\n\t<integer>\n  15\n</integer>\n</dict>",
			wantErr: true,
			errSub:  `invalid plist: line 3: <integer> value "\n  15\n" is not valid`,
		},
		{
			name:    "empty integer",
			input:   inDict(`<integer></integer>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <integer> value "" is not valid`,
		},
		{
			name:    "self-closing integer",
			input:   inDict(`<integer/>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <integer> value "" is not valid`,
		},
		{
			name:    "character reference in integer",
			input:   inDict(`<integer>&#49;</integer>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <integer> value "&#49;" is not valid`,
		},
		{
			name:    "self-closing real",
			input:   inDict(`<real/>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <real> value "" is not valid`,
		},
		{
			name:    "real with trailing space",
			input:   inDict(`<real>1.5 </real>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <real> value "1.5 " is not valid`,
		},
		{
			name:    "real with trailing non-breaking space",
			input:   inDict("<real>1.5\u00a0</real>"),
			wantErr: true,
			errSub:  `invalid plist: line 1: <real> value "1.5\u00a0" is not valid`,
		},
		{
			name:    "hexadecimal real",
			input:   inDict(`<real>0x1p3</real>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <real> value "0x1p3" is not valid`,
		},
		{
			name:    "infinity after whitespace",
			input:   inDict(`<real> inf</real>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <real> value " inf" is not valid`,
		},
		{
			name:    "date with a UTC offset",
			input:   inDict(`<date>2026-01-01T00:00:00+05:00</date>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <date> value "2026-01-01T00:00:00+05:00" is not valid`,
		},
		{
			name:    "date with fractional seconds",
			input:   inDict(`<date>2026-01-01T00:00:00.5Z</date>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <date> value "2026-01-01T00:00:00.5Z" is not valid`,
		},
		{
			name:    "date with leading space",
			input:   inDict(`<date> 2026-01-01T00:00:00Z</date>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <date> value " 2026-01-01T00:00:00Z" is not valid`,
		},
		{
			name:    "self-closing data",
			input:   inDict(`<data/>`),
			wantErr: true,
			errSub:  "invalid plist: line 1: write an empty <data> as <data></data>",
		},
		{
			name:    "CDATA in data",
			input:   inDict(`<data><![CDATA[QUJD]]></data>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <data> value "<![CDATA[QUJD]]>" is not valid`,
		},
		{
			name:    "entity reference in data",
			input:   inDict(`<data>QU&amp;JD</data>`),
			wantErr: true,
			errSub:  `invalid plist: line 1: <data> value "QU&amp;JD" is not valid`,
		},
		{
			name:    "long invalid value is truncated in the error",
			input:   inDict("<data>" + strings.Repeat("QUJD", 40) + "!</data>"),
			wantErr: true,
			errSub:  `invalid plist: line 1: <data> value "` + strings.Repeat("QUJD", 16) + `"... is not valid`,
		},
		{
			name:    "hex integer: valid on devices but rejected by micromdm/plist (known library gap)",
			input:   inDict(`<integer>0x1F90</integer>`),
			wantErr: true,
			errSub:  `invalid plist: strconv.ParseUint: parsing "0x1F90": invalid syntax`,
		},

		{name: "real with leading whitespace", input: inDict(`<real>  1.5</real>`)},
		{name: "empty data", input: inDict(`<data></data>`)},
		{
			name: "number forms devices accept",
			input: `<dict>
	<key>A</key><integer>-42</integer>
	<key>B</key><integer> 7</integer>
	<key>C</key><integer>007</integer>
	<key>D</key><real>-1e3</real>
	<key>E</key><real>.5</real>
	<key>F</key><real>5.</real>
	<key>G</key><real>&#49;.5</real>
	<key>H</key><real>-Infinity</real>
	<key>I</key><real>NaN</real>
</dict>`,
		},
		{name: "pretty-printed managed app configuration", input: prettyConfig},
		{name: "comment before bare dict", input: "<!-- header -->\n<dict/>"},
		{name: "comment after bare dict", input: "<dict/>\n<!-- trailer -->"},
		{name: "comment inside dict", input: `<dict><!-- c --><key>a</key><!-- d --><string>b</string><!-- e --></dict>`},
		{name: "comments inside plist around the dict", input: wrap("<!-- before -->\n<dict/>\n<!-- after -->")},
		{
			name: "full wrapper with a comment before a bare dict",
			input: `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<!-- header -->
<dict><key>a</key><string>b</string></dict>`,
		},
		{name: "realistic managed app configuration", input: appConfig},
		{name: "2,000 keys", input: large.String()},
		{name: "30 levels of nesting", input: deep},
	}

	for _, text := range []string{"&#10;", "&#32;", "<![CDATA[ ]]>"} {
		for _, c := range []struct{ container, input string }{
			{"dict", "<dict>" + text + "<key>a</key><string>b</string></dict>"},
			{"array", "<dict><key>a</key><array>" + text + "<string>b</string></array></dict>"},
			{"plist", "<plist>" + text + "<dict/></plist>"},
		} {
			cases = append(cases, testCase{
				name:    fmt.Sprintf("%s inside %s", text, c.container),
				input:   c.input,
				wantErr: true,
				errSub:  fmt.Sprintf("invalid plist: line 1: unexpected text inside <%s>", c.container),
			})
		}
	}

	var wrapped []testCase
	for _, c := range cases {
		if !c.wantErr && strings.HasPrefix(c.input, "<dict") {
			wrapped = append(wrapped, testCase{name: c.name + " in full wrapper", input: wrap(c.input)})
		}
	}
	cases = append(cases, wrapped...)

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
