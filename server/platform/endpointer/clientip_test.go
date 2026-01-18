package endpointer

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to create headers with proper canonicalization.
// Pass in pairs of header name and value. For example:
// makeHeaders("X-Forwarded-For", "1.1.1.1", "X-Real-IP", "2.2.2.2")
func makeHeaders(kvs ...string) http.Header {
	h := http.Header{}
	for i := 0; i < len(kvs); i += 2 {
		h.Set(kvs[i], kvs[i+1])
	}
	return h
}

func TestNewClientIPStrategy(t *testing.T) {
	tests := []struct {
		name           string
		trustedProxies string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "empty uses legacy strategy",
			trustedProxies: "",
			wantErr:        false,
		},
		{
			name:           "none uses RemoteAddr strategy",
			trustedProxies: "none",
			wantErr:        false,
		},
		{
			name:           "None (case insensitive)",
			trustedProxies: "None",
			wantErr:        false,
		},
		{
			name:           "NONE (case insensitive)",
			trustedProxies: "NONE",
			wantErr:        false,
		},
		{
			name:           "True-Client-IP header",
			trustedProxies: "True-Client-IP",
			wantErr:        false,
		},
		{
			name:           "X-Real-IP header",
			trustedProxies: "X-Real-IP",
			wantErr:        false,
		},
		{
			name:           "CF-Connecting-IP header",
			trustedProxies: "CF-Connecting-IP",
			wantErr:        false,
		},
		{
			name:           "hop count 1",
			trustedProxies: "1",
			wantErr:        false,
		},
		{
			name:           "hop count 2",
			trustedProxies: "2",
			wantErr:        false,
		},
		{
			name:           "hop count 0 is invalid",
			trustedProxies: "0",
			wantErr:        true,
			errContains:    "hop count must be >= 1",
		},
		{
			name:           "single IP range",
			trustedProxies: "10.0.0.0/8",
			wantErr:        false,
		},
		{
			name:           "multiple IP ranges",
			trustedProxies: "10.0.0.0/8, 192.168.0.0/16, 172.16.0.0/12",
			wantErr:        false,
		},
		{
			name:           "single IP address",
			trustedProxies: "192.168.1.1",
			wantErr:        false,
		},
		{
			name:           "invalid IP range",
			trustedProxies: "not-an-ip",
			wantErr:        true,
			errContains:    "invalid trusted_proxies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy, err := NewClientIPStrategy(tt.trustedProxies)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
				return
			}
			require.NoError(t, err)
			require.NotNil(t, strategy)
		})
	}
}

func TestClientIPStrategy_Legacy(t *testing.T) {
	strategy, err := NewClientIPStrategy("")
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "uses True-Client-IP first",
			headers:    makeHeaders("True-Client-IP", "1.1.1.1"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "1.1.1.1",
		},
		{
			name:       "uses X-Real-IP second",
			headers:    makeHeaders("X-Real-IP", "2.2.2.2"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "2.2.2.2",
		},
		{
			name:       "uses leftmost X-Forwarded-For third",
			headers:    makeHeaders("X-Forwarded-For", "3.3.3.3, 4.4.4.4"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "3.3.3.3",
		},
		{
			name:       "falls back to RemoteAddr",
			headers:    http.Header{},
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
		{
			name:       "True-Client-IP takes precedence over X-Forwarded-For",
			headers:    makeHeaders("True-Client-IP", "1.1.1.1", "X-Forwarded-For", "3.3.3.3"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "1.1.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := strategy.ClientIP(tt.headers, tt.remoteAddr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}

func TestClientIPStrategy_None(t *testing.T) {
	strategy, err := NewClientIPStrategy("none")
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "ignores True-Client-IP",
			headers:    makeHeaders("True-Client-IP", "1.1.1.1"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
		{
			name:       "ignores X-Real-IP",
			headers:    makeHeaders("X-Real-IP", "2.2.2.2"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
		{
			name:       "ignores X-Forwarded-For",
			headers:    makeHeaders("X-Forwarded-For", "3.3.3.3, 4.4.4.4"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
		{
			name:       "uses RemoteAddr only",
			headers:    http.Header{},
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := strategy.ClientIP(tt.headers, tt.remoteAddr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}

func TestClientIPStrategy_SingleIPHeader(t *testing.T) {
	strategy, err := NewClientIPStrategy("True-Client-IP")
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "uses True-Client-IP when present",
			headers:    makeHeaders("True-Client-IP", "1.1.1.1"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "1.1.1.1",
		},
		{
			name:       "falls back to RemoteAddr when header missing",
			headers:    http.Header{},
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
		{
			name:       "ignores X-Forwarded-For",
			headers:    makeHeaders("X-Forwarded-For", "3.3.3.3"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := strategy.ClientIP(tt.headers, tt.remoteAddr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}

func TestClientIPStrategy_HopCount(t *testing.T) {
	strategy, err := NewClientIPStrategy("2")
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "extracts correct IP with 2 hops",
			headers:    makeHeaders("X-Forwarded-For", "1.1.1.1, 2.2.2.2, 3.3.3.3"),
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "2.2.2.2",
		},
		{
			name:       "falls back to RemoteAddr when header missing",
			headers:    http.Header{},
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := strategy.ClientIP(tt.headers, tt.remoteAddr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}

func TestClientIPStrategy_IPRanges(t *testing.T) {
	// Trust private IP ranges
	strategy, err := NewClientIPStrategy("10.0.0.0/8, 192.168.0.0/16")
	require.NoError(t, err)

	tests := []struct {
		name       string
		headers    http.Header
		remoteAddr string
		wantIP     string
	}{
		{
			name:       "extracts client IP skipping trusted proxies",
			headers:    makeHeaders("X-Forwarded-For", "1.1.1.1, 10.0.0.5, 192.168.1.1"),
			remoteAddr: "10.0.0.1:12345",
			wantIP:     "1.1.1.1",
		},
		{
			name:       "returns rightmost non-trusted IP",
			headers:    makeHeaders("X-Forwarded-For", "8.8.8.8, 1.1.1.1, 10.0.0.5"),
			remoteAddr: "10.0.0.1:12345",
			wantIP:     "1.1.1.1",
		},
		{
			name:       "falls back to RemoteAddr when header missing",
			headers:    http.Header{},
			remoteAddr: "9.9.9.9:12345",
			wantIP:     "9.9.9.9",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := strategy.ClientIP(tt.headers, tt.remoteAddr)
			assert.Equal(t, tt.wantIP, ip)
		})
	}
}
