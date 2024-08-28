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
