package fleethttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestClient(t *testing.T) {
	cases := []struct {
		name        string
		opts        []ClientOpt
		nilRedirect bool
		timeout     time.Duration
	}{
		{"default", nil, true, 0},
		{"timeout", []ClientOpt{WithTimeout(time.Second)}, true, time.Second},
		{"nofollow", []ClientOpt{WithFollowRedir(false)}, false, 0},
		{"tlsconfig", []ClientOpt{WithTLSClientConfig(&tls.Config{})}, true, 0},
		{"combined", []ClientOpt{
			WithTLSClientConfig(&tls.Config{}),
			WithTimeout(time.Second),
			WithFollowRedir(false),
		}, false, time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli := NewClient(c.opts...)
			require.IsType(t, &otelhttp.Transport{}, cli.Transport, "outer transport should be otelhttp")
			// Inspect the inner (base) transport wrapped by otelhttp via unsafe since the rt field is unexported.
			rtField := reflect.ValueOf(cli.Transport).Elem().FieldByName("rt")
			inner := *(*http.RoundTripper)(unsafe.Pointer(rtField.UnsafeAddr())) //nolint:gosec
			// All clients use a custom transport with the private network blocking DialContext.
			assert.IsType(t, &http.Transport{}, inner, "inner transport should be a custom *http.Transport") //nolint:gocritic
			if c.nilRedirect {
				assert.Nil(t, cli.CheckRedirect)
			} else {
				assert.NotNil(t, cli.CheckRedirect)
			}
			assert.Equal(t, c.timeout, cli.Timeout)
		})
	}
}

func TestTransport(t *testing.T) {
	defaultTLSConf := http.DefaultTransport.(*http.Transport).TLSClientConfig

	cases := []struct {
		name       string
		opts       []TransportOpt
		defaultTLS bool
	}{
		{"default", nil, true},
		{"tlsconf", []TransportOpt{WithTLSConfig(&tls.Config{})}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := NewTransport(c.opts...)
			if c.defaultTLS {
				assert.Equal(t, defaultTLSConf, tr.TLSClientConfig)
			} else {
				assert.NotEqual(t, defaultTLSConf, tr.TLSClientConfig)
			}
			assert.NotNil(t, tr.Proxy)
			assert.NotNil(t, tr.DialContext)
		})
	}
}

func TestAlwaysBlockedIPs(t *testing.T) {
	// These IPs are always blocked, even with --allow_private_network_integrations.
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"169.254.169.254", true}, // AWS IMDS
		{"169.254.0.1", true},
		{"::1", true},            // IPv6 loopback
		{"fe80::1", true},        // IPv6 link-local
		{"8.8.8.8", false},       // public
		{"10.0.0.1", false},      // RFC 1918 -- not in always-blocked
		{"192.168.1.1", false},   // RFC 1918 -- not in always-blocked
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip)
			assert.Equal(t, c.blocked, ipInCIDRs(ip, alwaysBlockedCIDRs))
		})
	}
}

func TestPrivateNetworkCIDRs(t *testing.T) {
	// These IPs are blocked when private network blocking is enabled.
	cases := []struct {
		ip      string
		private bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"192.168.1.1", true},
		{"0.0.0.0", true},
		{"fc00::1", true},        // IPv6 unique local
		{"8.8.8.8", false},       // public
		{"1.1.1.1", false},       // public
		{"172.32.0.1", false},    // just outside 172.16.0.0/12
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip)
			assert.Equal(t, c.private, ipInCIDRs(ip, privateNetworkCIDRs))
		})
	}
}

func TestPrivateNetworkBlocking(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Run("blocked when enabled", func(t *testing.T) {
		SetBlockPrivateNetworks(true)
		defer SetBlockPrivateNetworks(false)
		client := NewClient(WithTimeout(5 * time.Second))
		// localhost is always blocked (loopback), so this tests the always-blocked tier.
		_, err := client.Get(ts.URL)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("loopback blocked even with allow flag", func(t *testing.T) {
		// Private network blocking is off, but loopback is always blocked.
		SetBlockPrivateNetworks(false)
		client := NewClient(WithTimeout(5 * time.Second))
		_, err := client.Get(ts.URL)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})
}

func TestHostnamesMatch(t *testing.T) {
	tests := []struct {
		name          string
		inputA        string
		inputB        string
		expectedMatch bool
		expectError   bool
	}{
		{
			name:          "ValidHostnamesMatch",
			inputA:        "https://www.example.com/path",
			inputB:        "http://www.example.com:80",
			expectedMatch: true,
			expectError:   false,
		},
		{
			name:          "ValidHostnamesDoNotMatch",
			inputA:        "https://www.example.com",
			inputB:        "https://sub.example.com",
			expectedMatch: false,
			expectError:   false,
		},
		{
			name:          "InvalidURLA",
			inputA:        "ht tp://foo.com",
			inputB:        "https://www.example.com",
			expectedMatch: false,
			expectError:   true,
		},
		{
			name:          "InvalidURLB",
			inputA:        "https://www.example.com",
			inputB:        "ht tp://foo.com",
			expectedMatch: false,
			expectError:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matched, err := HostnamesMatch(test.inputA, test.inputB)

			if test.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedMatch, matched)

			}
		})
	}
}
