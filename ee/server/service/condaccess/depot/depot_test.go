package depot

import (
	"crypto/x509"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractUUIDFromCert(t *testing.T) {
	cases := []struct {
		name string
		uris []string
		want string
	}{
		{"apple prefix", []string{"urn:device:apple:uuid:ABC-123"}, "ABC-123"},
		{"fleet prefix", []string{"urn:device:fleet:uuid:ABC-123"}, "ABC-123"},
		{"no URI", nil, ""},
		{"unknown prefix", []string{"urn:device:linux:uuid:ABC-123"}, ""},
		{"skips other URIs", []string{"https://example.com", "urn:device:fleet:uuid:ABC-123"}, "ABC-123"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			crt := &x509.Certificate{}
			for _, s := range c.uris {
				u, err := url.Parse(s)
				require.NoError(t, err)
				crt.URIs = append(crt.URIs, u)
			}
			require.Equal(t, c.want, extractUUIDFromCert(crt))
		})
	}
}
