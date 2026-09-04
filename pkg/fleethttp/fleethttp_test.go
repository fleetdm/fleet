package fleethttp

import (
	"crypto/tls"
	"fmt"
	"io"
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

func TestParseCIDRs(t *testing.T) {
	t.Run("valid CIDRs", func(t *testing.T) {
		result := parseCIDRs([]string{"10.0.0.0/8", "192.168.0.0/16"})
		require.Len(t, result, 2)
		assert.True(t, result[0].Contains(net.ParseIP("10.0.0.1")))
		assert.False(t, result[0].Contains(net.ParseIP("11.0.0.1")))
		assert.True(t, result[1].Contains(net.ParseIP("192.168.1.1")))
		assert.False(t, result[1].Contains(net.ParseIP("192.169.1.1")))
	})

	t.Run("empty list", func(t *testing.T) {
		result := parseCIDRs([]string{})
		assert.Empty(t, result)
	})

	t.Run("invalid CIDR panics", func(t *testing.T) {
		assert.Panics(t, func() {
			parseCIDRs([]string{"not-a-cidr"})
		})
	})
}

func TestIpInCIDRs(t *testing.T) {
	cidrs := parseCIDRs([]string{"10.0.0.0/8", "172.16.0.0/12"})

	cases := []struct {
		ip    string
		match bool
	}{
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"172.16.0.1", true},
		{"172.31.255.255", true},
		{"172.32.0.1", false},
		{"192.168.1.1", false},
		{"8.8.8.8", false},
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			assert.Equal(t, c.match, ipInCIDRs(net.ParseIP(c.ip), cidrs))
		})
	}
}

func TestAlwaysBlockedIPs(t *testing.T) {
	// These IPs are always blocked, even with --allow_private_network_integrations.
	cases := []struct {
		ip      string
		blocked bool
	}{
		{"0.0.0.0", true}, // unspecified; connects to loopback
		{"127.0.0.1", true},
		{"127.0.0.2", true},
		{"169.254.169.254", true}, // AWS IMDS
		{"169.254.0.1", true},
		{"::", true},           // IPv6 unspecified; connects to loopback
		{"::127.0.0.1", true},  // deprecated IPv4-compatible form
		{"::1", true},          // IPv6 loopback
		{"fe80::1", true},      // IPv6 link-local
		{"8.8.8.8", false},     // public
		{"10.0.0.1", false},    // RFC 1918 -- not in always-blocked
		{"192.168.1.1", false}, // RFC 1918 -- not in always-blocked
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip)
			assert.Equal(t, c.blocked, ipInCIDRs(ip, alwaysBlockedCIDRs))
		})
	}
}

func TestNAT64EmbeddedIPv4(t *testing.T) {
	// A NAT64 address reaches the IPv4 address it carries, so the embedded
	// address decides whether it is blocked. The prefix itself carries public
	// IPv4 traffic on IPv6-only networks and must stay reachable.
	cases := []struct {
		ip       string
		embedded string
	}{
		{"64:ff9b::7f00:1", "127.0.0.1"},          // loopback
		{"64:ff9b::a9fe:a9fe", "169.254.169.254"}, // cloud IMDS
		{"64:ff9b::a00:1", "10.0.0.1"},            // RFC 1918
		{"64:ff9b::808:808", "8.8.8.8"},           // public
		{"2001:4860:4860::8888", ""},              // not NAT64
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip)
			got := embeddedIPv4(ip)
			if c.embedded == "" {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, c.embedded, got.String())
		})
	}

	blocked := func(t *testing.T, target string, mode NetworkBlockingMode) error {
		t.Helper()
		setBlockingMode(t, mode)
		_, err := NewClient(WithTimeout(3 * time.Second)).Get(target)
		return err
	}
	t.Run("embedded internal address is blocked", func(t *testing.T) {
		for _, ip := range []string{"64:ff9b::7f00:1", "64:ff9b::a9fe:a9fe"} {
			err := blocked(t, "http://["+ip+"]:80/", BlockingPrivateAllowed)
			require.ErrorIs(t, err, ErrPrivateNetworkBlocked, ip)
		}
		err := blocked(t, "http://[64:ff9b::a00:1]:80/", BlockingFull)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})
	t.Run("embedded public address is not blocked", func(t *testing.T) {
		// Reaching it may fail for unrelated reasons; it must not be the guard.
		err := blocked(t, "http://[64:ff9b::808:808]:80/", BlockingFull)
		if err != nil {
			assert.NotErrorIs(t, err, ErrPrivateNetworkBlocked)
		}
	})
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
		{"fc00::1", true},     // IPv6 unique local
		{"0.0.0.0", false},    // always-blocked instead, so not listed here
		{"8.8.8.8", false},    // public
		{"1.1.1.1", false},    // public
		{"172.32.0.1", false}, // just outside 172.16.0.0/12
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip)
			assert.Equal(t, c.private, ipInCIDRs(ip, privateNetworkCIDRs))
		})
	}
}

func setBlockingMode(t *testing.T, mode NetworkBlockingMode) {
	t.Helper()
	SetNetworkBlockingMode(mode)
	t.Cleanup(func() { SetNetworkBlockingMode(BlockingDisabled) })
}

func TestPrivateNetworkBlockingDialContext(t *testing.T) {
	// Start a test server on localhost (always-blocked: loopback).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	t.Run("loopback blocked when blocking enabled", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		client := NewClient(WithTimeout(5 * time.Second))
		_, err := client.Get(ts.URL)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
		assert.Contains(t, err.Error(), "127.0.0.1")
	})

	t.Run("loopback blocked even with allow_private_network flag", func(t *testing.T) {
		// Tier 1 (always-blocked) cannot be overridden by the flag.
		setBlockingMode(t, BlockingPrivateAllowed)
		client := NewClient(WithTimeout(5 * time.Second))
		_, err := client.Get(ts.URL)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("unspecified address cannot reach a loopback-only service", func(t *testing.T) {
		// Connecting to an unspecified address reaches services listening on
		// loopback, so it has to be blocked like loopback itself.
		const marker = "loopback-only"
		cases := []struct {
			bind        string
			unspecified string
		}{
			{"127.0.0.1:0", "0.0.0.0"},
			{"[::1]:0", "[::]"},
		}
		for _, c := range cases {
			ln, err := net.Listen("tcp", c.bind)
			if err != nil {
				t.Skipf("cannot listen on %s: %v", c.bind, err)
			}
			t.Cleanup(func() { ln.Close() })
			go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { //nolint:errcheck // closed by cleanup
				io.WriteString(w, marker) //nolint:errcheck
			}))
			url := fmt.Sprintf("http://%s:%d/", c.unspecified, ln.Addr().(*net.TCPAddr).Port)

			// Without blocking the address does reach the loopback-only
			// listener, so the assertions below test the guard rather than an
			// address that was unreachable anyway.
			setBlockingMode(t, BlockingDisabled)
			resp, err := NewClient(WithTimeout(5 * time.Second)).Get(url)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, marker, string(body))

			for _, mode := range []NetworkBlockingMode{BlockingFull, BlockingPrivateAllowed} {
				setBlockingMode(t, mode)
				_, err := NewClient(WithTimeout(5 * time.Second)).Get(url)
				require.ErrorIs(t, err, ErrPrivateNetworkBlocked, "%s in mode %v", url, mode)
			}
		}
	})

	t.Run("not blocked when blocking is not enabled", func(t *testing.T) {
		// Default state: blocking not enabled (tests, CLI).
		client := NewClient(WithTimeout(5 * time.Second))
		resp, err := client.Get(ts.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("public IP allowed when blocking enabled", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		client := NewClient(WithTimeout(5 * time.Second))
		// google.com is public -- should not be blocked (may fail for other
		// reasons in CI, so we only check it's not ErrPrivateNetworkBlocked).
		_, err := client.Get("https://google.com")
		if err != nil {
			assert.NotErrorIs(t, err, ErrPrivateNetworkBlocked)
		}
	})

	t.Run("error message includes hostname and IP", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		client := NewClient(WithTimeout(5 * time.Second))
		_, err := client.Get(ts.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "127.0.0.1 resolves to 127.0.0.1")
	})

	t.Run("invalid address returns error", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		dialFn := privateNetworkBlockingDialContext(&net.Dialer{Timeout: time.Second})
		_, err := dialFn(t.Context(), "tcp", "no-port")
		require.Error(t, err)
		// Should fail on SplitHostPort, not on blocking.
		assert.NotErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("unresolvable host returns error", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		dialFn := privateNetworkBlockingDialContext(&net.Dialer{Timeout: time.Second})
		_, err := dialFn(t.Context(), "tcp", "this-host-does-not-exist.invalid:443")
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("connects to resolved IP not hostname", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		dialFn := privateNetworkBlockingDialContext(&net.Dialer{Timeout: time.Second})
		_, err := dialFn(t.Context(), "tcp", "localhost:9999")
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
		assert.Contains(t, err.Error(), "localhost resolves to")
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
