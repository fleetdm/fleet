package osquery

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFleetFlagsAcceptGzip(t *testing.T) {
	u, err := url.Parse("https://fleet.example.com")
	require.NoError(t, err)

	tests := []struct {
		version  string
		wantFlag bool
	}{
		{"5.20.0", false},
		{"5.19.0-foobar", false},
		{"5.21.0", true},
		{"5.21.1", true},
		{"5.21.0-24-g9e10d95ae", true},
		{"5.22.0", true},
		{"4.0.0", false},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			flags := FleetFlags(tt.version, u)
			found := false
			for _, f := range flags {
				if f == "--tls_accept_gzip=true" {
					found = true
					break
				}
			}
			assert.Equal(t, tt.wantFlag, found, "version %s flag presence mismatch", tt.version)
		})
	}
}
