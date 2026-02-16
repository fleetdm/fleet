package fleethttp

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noopResolver returns a known public IP so that CheckURLForSSRF reaches the
// IP-range check without triggering real DNS lookups.
func noopResolver(_ context.Context, _ string) ([]string, error) {
	return []string{"93.184.216.34"}, nil // example.com
}

func TestCheckURLForSSRFBlockedLiteralIPs(t *testing.T) {
	t.Parallel()

	blocked := []string{
		"http://127.0.0.1/mscep/mscep.dll",
		"http://127.255.255.255/path",
		"http://10.0.0.1/admin",
		"http://10.255.255.255/admin",
		"http://172.16.0.1/admin",
		"http://172.31.255.255/admin",
		"http://192.168.0.1/admin",
		"http://192.168.255.255/admin",
		"http://169.254.169.254/latest/meta-data/",
		"http://169.254.0.1/whatever",
		"http://100.64.0.1/admin",
		"http://100.127.255.255/admin",
		"http://0.0.0.0/path",
		"http://[::1]/path",
		"http://[fe80::1]/path",
		"http://[fc00::1]/path",
		"http://[fdff::1]/path",
	}

	for _, u := range blocked {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			err := CheckURLForSSRF(context.Background(), u, noopResolver)
			require.Error(t, err, "expected SSRF block for %s", u)
			var ssrfErr *SSRFError
			assert.True(t, errors.As(err, &ssrfErr), "expected SSRFError for %s, got %T: %v", u, err, err)
		})
	}
}

func TestCheckURLForSSRFAllowedPublicIPs(t *testing.T) {
	t.Parallel()

	allowed := []string{
		"https://ndes.corp.example.com/mscep/mscep.dll",
		"https://93.184.216.34/path", // example.com
		"http://8.8.8.8/path",        // Google DNS
		"https://1.1.1.1/path",       // Cloudflare DNS
	}

	for _, u := range allowed {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			err := CheckURLForSSRF(context.Background(), u, noopResolver)
			assert.NoError(t, err, "expected no SSRF block for %s", u)
		})
	}
}

func TestCheckURLForSSRFDNSResolutionBlocked(t *testing.T) {
	t.Parallel()

	// Simulate a hostname that resolves to a private IP
	privateResolver := func(_ context.Context, _ string) ([]string, error) {
		return []string{"192.168.1.100"}, nil
	}

	err := CheckURLForSSRF(context.Background(), "https://attacker-controlled.example.com/admin", privateResolver)
	require.Error(t, err)
	var ssrfErr *SSRFError
	assert.True(t, errors.As(err, &ssrfErr))
	assert.Equal(t, net.ParseIP("192.168.1.100").String(), ssrfErr.IP.String())
}

func TestCheckURLForSSRFMetadataEndpoints(t *testing.T) {
	t.Parallel()

	metadataURLs := []string{
		"http://169.254.169.254/latest/meta-data/iam/security-credentials/",
		"http://169.254.169.254/metadata/instance?api-version=2021-02-01",
	}

	for _, u := range metadataURLs {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			err := CheckURLForSSRF(context.Background(), u, noopResolver)
			require.Error(t, err)
			var ssrfErr *SSRFError
			assert.True(t, errors.As(err, &ssrfErr))
		})
	}
}

func TestCheckURLForSSRFBadScheme(t *testing.T) {
	t.Parallel()

	err := CheckURLForSSRF(context.Background(), "file:///etc/passwd", noopResolver)
	require.Error(t, err)
	assert.NotErrorIs(t, err, (*SSRFError)(nil))
}

func TestCheckURLForSSRFResolverError(t *testing.T) {
	t.Parallel()

	failResolver := func(_ context.Context, _ string) ([]string, error) {
		return nil, errors.New("simulated DNS failure")
	}
	err := CheckURLForSSRF(context.Background(), "https://cant-resolve.example.com/admin", failResolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving host")
}

func TestCheckURLForSSRF_UnparseableAddressFailsClosed(t *testing.T) {
	t.Parallel()

	// A custom resolver returning a non-IP string will be blocked
	badResolver := func(_ context.Context, _ string) ([]string, error) {
		return []string{"not-an-ip"}, nil
	}
	err := CheckURLForSSRF(context.Background(), "https://example.com/admin", badResolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a valid IP")
}

func TestCheckURLForSSRFMultipleResolutions(t *testing.T) {
	t.Parallel()

	mixedResolver := func(_ context.Context, _ string) ([]string, error) {
		return []string{"93.184.216.34", "10.0.0.1"}, nil
	}
	err := CheckURLForSSRF(context.Background(), "https://mixed.example.com/admin", mixedResolver)
	require.Error(t, err)
	var ssrfErr *SSRFError
	assert.True(t, errors.As(err, &ssrfErr))
	assert.Equal(t, net.ParseIP("10.0.0.1").String(), ssrfErr.IP.String())
}

func TestCheckURLForSSRFIPv4MappedBypass(t *testing.T) {
	t.Parallel()

	// An attacker could supply an IPv4-mapped IPv6 address like ::ffff:192.168.1.1
	// to reach a private IPv4 host while bypassing the IPv4 blocklist check.
	blocked := []string{
		"http://[::ffff:192.168.1.1]/admin",     // RFC 1918 private
		"http://[::ffff:127.0.0.1]/admin",       // Loopback
		"http://[::ffff:169.254.169.254]/admin", // Link-local metadata
		"http://[::ffff:10.0.0.1]/admin",        // RFC 1918 private
	}
	for _, u := range blocked {
		t.Run(u, func(t *testing.T) {
			t.Parallel()
			err := CheckURLForSSRF(context.Background(), u, noopResolver)
			require.Error(t, err, "expected SSRF block for IPv4-mapped %s", u)
			var ssrfErr *SSRFError
			assert.True(t, errors.As(err, &ssrfErr), "expected SSRFError for %s, got %T: %v", u, err, err)
		})
	}
}

func TestCheckURLForSSRFSSRFErrorMessage(t *testing.T) {
	err := CheckURLForSSRF(context.Background(), "http://127.0.0.1/admin", noopResolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "blocked IP address")
	assert.Contains(t, err.Error(), "127.0.0.1")
}

// noopDial is used as the dial parameter so tests never open real sockets.
func noopDial(_ context.Context, _, _ string) (net.Conn, error) {
	return nil, errors.New("no-op dial: connection not attempted in tests")
}

// staticResolver returns a fixed list of IPs for any host.
func staticResolver(ips ...string) func(ctx context.Context, host string) ([]string, error) {
	return func(_ context.Context, _ string) ([]string, error) {
		return ips, nil
	}
}

func TestSSRFDialContextBlocksPrivateIPs(t *testing.T) {
	t.Parallel()

	blocked := []struct {
		addr string
		ip   string
	}{
		{"127.0.0.1:80", "127.0.0.1"},
		{"10.0.0.1:443", "10.0.0.1"},
		{"172.16.0.1:8080", "172.16.0.1"},
		{"192.168.1.1:443", "192.168.1.1"},
		{"169.254.169.254:80", "169.254.169.254"},
	}

	for _, tc := range blocked {
		t.Run(tc.addr, func(t *testing.T) {
			t.Parallel()
			dial := SSRFDialContext(nil, staticResolver(tc.ip), noopDial)
			conn, err := dial(context.Background(), "tcp", tc.addr)
			require.Error(t, err, "expected dial to be blocked for %s", tc.addr)
			assert.Nil(t, conn)
			var ssrfErr *SSRFError
			assert.True(t, errors.As(err, &ssrfErr), "expected SSRFError for %s, got %T: %v", tc.addr, err, err)
			assert.Equal(t, net.ParseIP(tc.ip).String(), ssrfErr.IP.String())
		})
	}
}

func TestSSRFDialContextAllowsPublicIPs(t *testing.T) {
	t.Parallel()

	publicIPs := []string{
		"93.184.216.34",
		"8.8.8.8",
		"1.1.1.1",
	}

	for _, publicIP := range publicIPs {
		t.Run(publicIP, func(t *testing.T) {
			t.Parallel()
			dial := SSRFDialContext(nil, staticResolver(publicIP), noopDial)
			_, err := dial(context.Background(), "tcp", publicIP+":80")
			var ssrfErr *SSRFError
			assert.False(t, errors.As(err, &ssrfErr), "public IP %s should not be SSRF-blocked", publicIP)
		})
	}
}

func TestSSRFDialContextBlocksMixedResolution(t *testing.T) {
	t.Parallel()

	// Simulates DNS rebinding: resolver returns one public and one private IP.
	dial := SSRFDialContext(nil, staticResolver("93.184.216.34", "192.168.1.100"), noopDial)

	_, err := dial(context.Background(), "tcp", "attacker.example.com:443")
	require.Error(t, err)
	var ssrfErr *SSRFError
	assert.True(t, errors.As(err, &ssrfErr))
	assert.Equal(t, net.ParseIP("192.168.1.100").String(), ssrfErr.IP.String())
}

func TestSSRFDialContextResolverError(t *testing.T) {
	t.Parallel()

	failResolver := func(_ context.Context, _ string) ([]string, error) {
		return nil, errors.New("simulated DNS failure")
	}
	dial := SSRFDialContext(nil, failResolver, noopDial)
	_, err := dial(context.Background(), "tcp", "cant-resolve.example.com:443")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolving")
}

func TestSSRFDialContextNilsUseDefaults(t *testing.T) {
	t.Parallel()

	dial := SSRFDialContext(nil, nil, nil)
	require.NotNil(t, dial)
}

func TestNewTransportHasSSRFDialContext(t *testing.T) {
	t.Parallel()

	tr := NewTransport()
	require.NotNil(t, tr.DialContext, "NewTransport() must set DialContext for SSRF protection")
}
