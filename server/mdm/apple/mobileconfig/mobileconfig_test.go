package mobileconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXMLEscapeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// characters that should be escaped
		{"hello & world", "hello &amp; world"},
		{"this is a <test>", "this is a &lt;test&gt;"},
		{"\"quotes\" and 'single quotes'", "&#34;quotes&#34; and &#39;single quotes&#39;"},
		{"special chars: \t\n\r", "special chars: &#x9;&#xA;&#xD;"},
		// no special characters
		{"plain string", "plain string"},
		// string that already contains escaped characters
		{"already &lt;escaped&gt;", "already &amp;lt;escaped&amp;gt;"},
		// empty string
		{"", ""},
		// multiple special characters
		{"A&B<C>D\"'E\tF\nG\r", "A&amp;B&lt;C&gt;D&#34;&#39;E&#x9;F&#xA;G&#xD;"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			out, err := XMLEscapeString(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, out)
		})
	}
}

func TestHasPayloadType(t *testing.T) {
	build := func(payloadType string) Mobileconfig {
		return Mobileconfig(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadType</key>
			<string>` + payloadType + `</string>
			<key>PayloadIdentifier</key>
			<string>com.example.profile.cert</string>
			<key>PayloadDisplayName</key>
			<string>Test Cert</string>
			<key>PayloadUUID</key>
			<string>00000000-0000-0000-0000-000000000001</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Test Profile</string>
	<key>PayloadIdentifier</key>
	<string>com.example.profile</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>00000000-0000-0000-0000-000000000002</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`)
	}

	t.Run("ACME profile reports ACME payload", func(t *testing.T) {
		got, err := build(ACMEPayloadType).HasPayloadType(ACMEPayloadType)
		require.NoError(t, err)
		require.True(t, got)
	})

	t.Run("SCEP profile does not report ACME payload", func(t *testing.T) {
		got, err := build(SCEPPayloadType).HasPayloadType(ACMEPayloadType)
		require.NoError(t, err)
		require.False(t, got)
	})

	t.Run("SCEP profile reports SCEP payload", func(t *testing.T) {
		got, err := build(SCEPPayloadType).HasPayloadType(SCEPPayloadType)
		require.NoError(t, err)
		require.True(t, got)
	})
}
