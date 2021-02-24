package insecure

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	proxy, err := NewTLSProxy("localhost")
	require.NoError(t, err)
	assert.NotZero(t, proxy.Port)
}

func TestParseURL(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input     string
		shouldErr bool
		expected  *url.URL
	}{
		{
			input:    "localhost:8080",
			expected: &url.URL{Scheme: "https", Host: "localhost:8080"},
		},
		{
			input:    "https://localhost:8080",
			expected: &url.URL{Scheme: "https", Host: "localhost:8080"},
		},
		{
			input:     "http://localhost:8080",
			shouldErr: true,
		},
		{
			input:    "https://fleetdm.com/prefix",
			expected: &url.URL{Scheme: "https", Host: "fleetdm.com", Path: "/prefix"},
		},
		{
			input:    "fleetdm.com/prefix",
			expected: &url.URL{Scheme: "https", Host: "fleetdm.com", Path: "/prefix"},
		},
		{
			input:     " **foobar",
			shouldErr: true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.input, func(t *testing.T) {
			url, err := parseURL(tt.input)
			if tt.shouldErr {
				require.Error(t, err)
				return
			}

			assert.Equal(t, tt.expected, url)
		})
	}
}
