package fleethttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
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
		{"default", nil, true, DefaultTimeout},
		{"timeout", []ClientOpt{WithTimeout(time.Second)}, true, time.Second},
		{"notimeout", []ClientOpt{WithNoTimeout()}, true, 0},
		{"nofollow", []ClientOpt{WithFollowRedir(false)}, false, DefaultTimeout},
		{"tlsconfig", []ClientOpt{WithTLSClientConfig(&tls.Config{})}, true, DefaultTimeout},
		{"combined", []ClientOpt{
			WithTLSClientConfig(&tls.Config{}),
			WithTimeout(time.Second),
			WithFollowRedir(false),
		}, false, time.Second},
		{"notimeout wins over earlier timeout", []ClientOpt{
			WithTimeout(time.Second),
			WithNoTimeout(),
		}, true, 0},
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
			assert.Zero(t, tr.ResponseHeaderTimeout)
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

func setAllowList(t *testing.T, input string) {
	t.Helper()
	al, err := ParseNetworkAllowList(input)
	require.NoError(t, err)
	SetNetworkAllowList(al)
	t.Cleanup(func() { SetNetworkAllowList(nil) })
}

func TestPrivateNetworkBlockingWithAllowList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	tsURL, err := url.Parse(ts.URL)
	require.NoError(t, err)
	tsPort := tsURL.Port()

	t.Run("CIDR allow-list bypasses loopback blocking", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "127.0.0.0/8")

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("exact IP allow-list bypasses loopback blocking", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "127.0.0.1")

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("hostname allow-list bypasses blocking", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "localhost")

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get("http://localhost:" + tsPort)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("non-matching allow-list still blocks", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "192.168.1.0/24")

		_, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("IP allow-list with matching port bypasses blocking", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "127.0.0.1:"+tsPort)

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("IP allow-list with wrong port still blocks", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		setAllowList(t, "127.0.0.1:1")

		_, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("nil allow-list does not affect blocking", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		SetNetworkAllowList(nil)
		t.Cleanup(func() { SetNetworkAllowList(nil) })

		_, err := NewClient(WithTimeout(5 * time.Second)).Get(ts.URL)
		require.ErrorIs(t, err, ErrPrivateNetworkBlocked)
	})

	t.Run("IP allow-list via DNS covers both v4 and v6 loopback", func(t *testing.T) {
		setBlockingMode(t, BlockingFull)
		// localhost may resolve to 127.0.0.1, ::1, or both.
		setAllowList(t, "127.0.0.0/8, ::1")

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get("http://localhost:" + tsPort)
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

func TestCheckIPAllowed(t *testing.T) {
	cases := []struct {
		name      string
		ip        string
		port      int
		mode      NetworkBlockingMode
		allowList string // parsed via ParseNetworkAllowList; "" = nil
		wantErr   bool
	}{
		// Tier 1: always blocked.
		{"loopback blocked", "127.0.0.1", 80, BlockingFull, "", true},
		{"IMDS blocked", "169.254.169.254", 80, BlockingFull, "", true},
		{"IPv6 loopback blocked", "::1", 80, BlockingFull, "", true},

		// Tier 2: private networks, only in BlockingFull.
		{"RFC1918 blocked in full mode", "10.0.0.1", 80, BlockingFull, "", true},
		{"RFC1918 allowed in private-allowed mode", "10.0.0.1", 80, BlockingPrivateAllowed, "", false},

		// Public IPs pass.
		{"public IPv4", "8.8.8.8", 80, BlockingFull, "", false},
		{"public IPv6", "2001:4860:4860::8888", 80, BlockingFull, "", false},

		// Allow-list overrides tier 1.
		{"loopback allow-listed by CIDR", "127.0.0.1", 80, BlockingFull, "127.0.0.0/8", false},
		{"loopback allow-listed by exact IP", "127.0.0.1", 80, BlockingFull, "127.0.0.1", false},
		{"IMDS allow-listed", "169.254.169.254", 80, BlockingFull, "169.254.169.254", false},

		// Allow-list overrides tier 2.
		{"RFC1918 allow-listed", "10.0.0.1", 80, BlockingFull, "10.0.0.0/8", false},

		// Allow-list with port: matching vs non-matching.
		{"allow-listed IP correct port", "127.0.0.1", 8080, BlockingFull, "127.0.0.1:8080", false},
		{"allow-listed IP wrong port", "127.0.0.1", 443, BlockingFull, "127.0.0.1:8080", true},

		// NAT64: embedded IPv4 is allow-listed.
		{"NAT64 embedded loopback allow-listed", "64:ff9b::7f00:1", 80, BlockingFull, "127.0.0.0/8", false},
		{"NAT64 embedded loopback not allow-listed", "64:ff9b::7f00:1", 80, BlockingFull, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip, "bad test IP")

			var al *NetworkAllowList
			if c.allowList != "" {
				var err error
				al, err = ParseNetworkAllowList(c.allowList)
				require.NoError(t, err)
			}

			err := checkIPAllowed(net.IPAddr{IP: ip}, c.port, c.mode, al)
			if c.wantErr {
				assert.ErrorIs(t, err, ErrPrivateNetworkBlocked)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBlockedIPSkippedAllowedIPConnected(t *testing.T) {
	// Verify the per-IP skip semantic: when a hostname resolves to
	// multiple IPs and some are blocked, the dialer skips blocked
	// addresses and connects via an allowed one.
	const marker = "reached-good-ip"

	// Listen only on 127.0.0.1.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { //nolint:errcheck // closed by cleanup
		io.WriteString(w, marker) //nolint:errcheck
	}))

	setBlockingMode(t, BlockingFull)
	// Allow-list only 127.0.0.1, not ::1.  When localhost resolves to
	// both [::1, 127.0.0.1], ::1 is blocked (tier-1, not allow-listed)
	// and skipped; 127.0.0.1 is allow-listed and dialled.
	setAllowList(t, "127.0.0.1")

	resp, err := NewClient(WithTimeout(5 * time.Second)).Get(
		fmt.Sprintf("http://localhost:%d/", port),
	)
	require.NoError(t, err)
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.NoError(t, err)
	assert.Equal(t, marker, string(body))
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

func TestDialFallbackToNextIP(t *testing.T) {
	// Listen on a real port so we have one reachable address.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { ln.Close() })
	goodPort := ln.Addr().(*net.TCPAddr).Port

	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { //nolint:errcheck // closed by cleanup
		io.WriteString(w, "ok") //nolint:errcheck
	}))

	// Pick a port that nothing is listening on (connection refused).
	badLn, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	badPort := badLn.Addr().(*net.TCPAddr).Port
	badLn.Close() // close immediately so the port is free but refused

	dialFn := privateNetworkBlockingDialContext(&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 5 * time.Second,
	})

	t.Run("falls back to second IP on connection refused", func(t *testing.T) {
		// Simulate the dialer seeing two IPs — first is dead, second is good.
		// We do this by calling the dialer directly with the dead address,
		// showing it fails, then calling with the good one, showing it
		// succeeds. The real multi-IP path runs inside the dialer, so we
		// test that end-to-end by listening on two addresses.

		// Dead address should fail quickly.
		_, err := dialFn(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", badPort))
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrPrivateNetworkBlocked)

		// Good address should succeed.
		conn, err := dialFn(t.Context(), "tcp", fmt.Sprintf("127.0.0.1:%d", goodPort))
		require.NoError(t, err)
		conn.Close()
	})

	t.Run("multi-address fallback via dual-stack localhost", func(t *testing.T) {
		// Listen on 127.0.0.1 only. When blocking is off, "localhost"
		// typically resolves to both ::1 and 127.0.0.1. If ::1 is tried
		// first and nothing is listening there on this port, the dialer
		// should fall back to 127.0.0.1 where we ARE listening.
		setBlockingMode(t, BlockingDisabled)

		resp, err := NewClient(WithTimeout(5 * time.Second)).Get(
			fmt.Sprintf("http://localhost:%d/", goodPort),
		)
		require.NoError(t, err)
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		require.NoError(t, err)
		assert.Equal(t, "ok", string(body))
	})

	t.Run("context cancellation stops fallback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel() // already cancelled

		_, err := dialFn(ctx, "tcp", fmt.Sprintf("127.0.0.1:%d", goodPort))
		require.Error(t, err)
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

// sizeLimitedClients covers every way a size-limited client can be built.
var sizeLimitedClients = map[string]func(maxSize int64) *http.Client{
	"WithMaxResponseSize": func(maxSize int64) *http.Client {
		return NewClient(WithTimeout(5*time.Second), WithMaxResponseSize(maxSize))
	},
	"NewSizeLimitTransport": func(maxSize int64) *http.Client {
		cli := NewClient(WithTimeout(5 * time.Second))
		cli.Transport = NewSizeLimitTransport(maxSize)
		return cli
	},
	// Zero value: base is nil, so RoundTrip has to resolve one itself.
	"SizeLimitTransport literal": func(maxSize int64) *http.Client {
		cli := NewClient(WithTimeout(5 * time.Second))
		cli.Transport = &SizeLimitTransport{maxSizeBytes: maxSize}
		return cli
	},
}

func TestSizeLimitedClientBlocksPrivateNetworks(t *testing.T) {
	const marker = "loopback-only"
	const maxSize = 1 << 20

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, marker) //nolint:errcheck
	}))
	defer ts.Close()

	for name, newClient := range sizeLimitedClients {
		t.Run(name, func(t *testing.T) {
			// Control: with blocking off the listener is reachable, so the
			// assertions below cannot pass on an unreachable address.
			setBlockingMode(t, BlockingDisabled)
			resp, err := newClient(maxSize).Get(ts.URL)
			require.NoError(t, err)
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, marker, string(body))

			for _, mode := range []NetworkBlockingMode{BlockingFull, BlockingPrivateAllowed} {
				setBlockingMode(t, mode)
				_, err := newClient(maxSize).Get(ts.URL)
				require.ErrorIs(t, err, ErrPrivateNetworkBlocked, "mode %v", mode)
			}
		})
	}
}

func TestSizeLimitedClientEnforcesLimit(t *testing.T) {
	const maxSize = 1024
	oversized := strings.Repeat("x", maxSize*2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/chunked" {
			// No Content-Length, so the limit can only be enforced while reading.
			w.Header().Set("Transfer-Encoding", "chunked")
		}
		io.WriteString(w, oversized) //nolint:errcheck
	}))
	defer ts.Close()

	for name, newClient := range sizeLimitedClients {
		t.Run(name, func(t *testing.T) {
			setBlockingMode(t, BlockingDisabled)

			_, err := newClient(maxSize).Get(ts.URL + "/known-length")
			require.ErrorIs(t, err, ErrMaxSizeExceeded)

			resp, err := newClient(maxSize).Get(ts.URL + "/chunked")
			require.NoError(t, err)
			require.EqualValues(t, -1, resp.ContentLength)
			_, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			var maxBytesErr *http.MaxBytesError
			require.ErrorAs(t, err, &maxBytesErr)
		})
	}
}

// roundTripFunc is deliberately not an *http.Transport, matching how tests
// stub http.DefaultTransport.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestSizeLimitedClientPreservesDefaultTransportMock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	for name, newClient := range sizeLimitedClients {
		t.Run(name, func(t *testing.T) {
			var mocked bool
			orig := http.DefaultTransport
			t.Cleanup(func() { http.DefaultTransport = orig })
			http.DefaultTransport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
				mocked = true
				return orig.RoundTrip(r)
			})

			resp, err := newClient(1 << 20).Get(ts.URL)
			require.NoError(t, err)
			resp.Body.Close()
			require.True(t, mocked, "mock round tripper must stay in the chain")
		})
	}
}

// TestClientTimeoutBehavior verifies the timeout options are wired to http.Client.Timeout and actually abort a slow response, rather
// than only being recorded on the struct.
func TestClientTimeoutBehavior(t *testing.T) {
	// The handler blocks until the test releases it, so the only thing that can end the request is the client timeout.
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})

	t.Run("timeout aborts a slow response", func(t *testing.T) {
		_, err := NewClient(WithTimeout(100 * time.Millisecond)).Get(srv.URL)
		require.Error(t, err)
		var netErr interface{ Timeout() bool }
		require.True(t, errors.As(err, &netErr) && netErr.Timeout(), "expected a timeout error, got %v", err)
	})

	t.Run("no timeout waits for the response", func(t *testing.T) {
		cli := NewClient(WithNoTimeout())
		assert.Zero(t, cli.Timeout)
		done := make(chan error, 1)
		go func() {
			resp, err := cli.Get(srv.URL)
			if resp != nil {
				resp.Body.Close()
			}
			done <- err
		}()
		select {
		case err := <-done:
			t.Fatalf("request should still be in flight, got %v", err)
		case <-time.After(300 * time.Millisecond):
		}
	})
}

func TestClientResponseHeaderTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	tlsSrv := httptest.NewTLSServer(handler)
	t.Cleanup(tlsSrv.Close)
	tlsOpt := WithTLSClientConfig(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec // test server

	for _, c := range []struct {
		name    string
		url     string
		opts    []ClientOpt
		wantErr bool
	}{
		{"no option waits", srv.URL, nil, false},
		{"option aborts", srv.URL, []ClientOpt{WithResponseHeaderTimeout(100 * time.Millisecond)}, true},
		{"option aborts with TLS config", tlsSrv.URL, []ClientOpt{tlsOpt, WithResponseHeaderTimeout(100 * time.Millisecond)}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			resp, err := NewClient(append(c.opts, WithNoTimeout())...).Get(c.url)
			if c.wantErr {
				require.ErrorContains(t, err, "timeout awaiting response headers")
				return
			}
			require.NoError(t, err)
			resp.Body.Close()
		})
	}
}

func TestNewGithubClientAuthorization(t *testing.T) {
	var mu sync.Mutex
	gotAuth := map[string]string{}
	record := func(name string, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		gotAuth[name] = r.Header.Get("Authorization")
	}

	mirror := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("mirror", r)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(mirror.Close)
	api := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			record("api redirect", r)
			http.Redirect(w, r, mirror.URL+"/asset", http.StatusFound)
			return
		}
		record("api", r)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(api.Close)
	plainAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record("plain api", r)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(plainAPI.Close)

	// NewClient builds on http.DefaultTransport, so swap in one that trusts the
	// test servers' certificate (all httptest TLS servers share it).
	origTransport := http.DefaultTransport
	http.DefaultTransport = api.Client().Transport
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	// Stand the test server in for api.github.com.
	setAPIHost := func(t *testing.T, rawURL string) {
		u, err := url.Parse(rawURL)
		require.NoError(t, err)
		origHost := githubAPIHost
		githubAPIHost = u.Host
		t.Cleanup(func() { githubAPIHost = origHost })
	}
	setAPIHost(t, api.URL)

	cases := []struct {
		name         string
		testToken    string
		fleetToken   string
		expectedAuth string
	}{
		{name: "no token", expectedAuth: ""},
		{name: "test token", testToken: "test-tok", expectedAuth: "Bearer test-tok"},
		{name: "fleet token", fleetToken: "fleet-tok", expectedAuth: "Bearer fleet-tok"},
		{name: "test token wins", testToken: "test-tok", fleetToken: "fleet-tok", expectedAuth: "Bearer test-tok"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("NETWORK_TEST_GITHUB_TOKEN", c.testToken)
			t.Setenv("FLEET_VULNERABILITIES_GITHUB_TOKEN", c.fleetToken)
			// Ambient GitHub variables must never be picked up.
			t.Setenv("GITHUB_TOKEN", "ambient-github-token")
			t.Setenv("GH_TOKEN", "ambient-gh-token")
			mu.Lock()
			clear(gotAuth)
			mu.Unlock()

			cli := NewGithubClient()
			for _, u := range []string{api.URL, mirror.URL, api.URL + "/redirect"} {
				resp, err := cli.Get(u)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
			}

			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, map[string]string{
				"api":          c.expectedAuth,
				"api redirect": c.expectedAuth,
				// The token must never leave the GitHub API host, directly or via a redirect.
				"mirror": "",
			}, gotAuth)
		})
	}

	t.Run("plain HTTP to the API host", func(t *testing.T) {
		t.Setenv("FLEET_VULNERABILITIES_GITHUB_TOKEN", "fleet-tok")
		setAPIHost(t, plainAPI.URL)
		mu.Lock()
		clear(gotAuth)
		mu.Unlock()

		resp, err := NewGithubClient().Get(plainAPI.URL)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		mu.Lock()
		defer mu.Unlock()
		// The token must never be sent in plaintext.
		require.Equal(t, map[string]string{"plain api": ""}, gotAuth)
	})

	t.Run("request without headers", func(t *testing.T) {
		var gotAuth string
		base := roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotAuth = r.Header.Get("Authorization")
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Request: r}, nil
		})
		req := &http.Request{Method: http.MethodGet, URL: &url.URL{Scheme: "https", Host: githubAPIHost, Path: "/"}}

		resp, err := (&githubTokenTransport{token: "tok", base: base}).RoundTrip(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, "Bearer tok", gotAuth)
		require.Nil(t, req.Header, "the caller's request must not be modified")
	})
}
