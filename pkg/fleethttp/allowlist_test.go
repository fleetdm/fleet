package fleethttp

import (
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNetworkAllowList(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		cidrs     int
		ips       int
		hosts     int
		wildcards int
	}{
		{"empty string", "", 0, 0, 0, 0},
		{"whitespace only", "   ", 0, 0, 0, 0},
		{"trailing comma", "10.0.0.1,", 0, 1, 0, 0},
		{"leading comma", ",10.0.0.1", 0, 1, 0, 0},
		{"multiple commas", "10.0.0.1,,10.0.0.2", 0, 2, 0, 0},

		// IP addresses
		{"bare IPv4", "192.168.1.1", 0, 1, 0, 0},
		{"IPv4 with port", "192.168.1.1:8080", 0, 1, 0, 0},
		{"bare IPv6", "::1", 0, 1, 0, 0},
		{"bare IPv6 full", "fe80::1", 0, 1, 0, 0},
		{"bracketed IPv6", "[::1]", 0, 1, 0, 0},
		{"bracketed IPv6 with port", "[::1]:443", 0, 1, 0, 0},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", 0, 1, 0, 0},

		// CIDRs
		{"IPv4 CIDR", "10.0.0.0/8", 1, 0, 0, 0},
		{"IPv6 CIDR", "fc00::/7", 1, 0, 0, 0},
		{"single-host CIDR", "192.168.1.1/32", 1, 0, 0, 0},

		// DNS names
		{"simple hostname", "example.com", 0, 0, 1, 0},
		{"hostname with port", "example.com:8080", 0, 0, 1, 0},
		{"subdomain", "jira.internal.example.com", 0, 0, 1, 0},
		{"trailing dot FQDN", "example.com.", 0, 0, 1, 0},
		{"single label", "localhost", 0, 0, 1, 0},
		{"underscore label", "_dmarc.example.com", 0, 0, 1, 0},

		// Wildcards
		{"wildcard", "*.example.com", 0, 0, 0, 1},
		{"wildcard with port", "*.example.com:443", 0, 0, 0, 1},
		{"deep wildcard", "*.internal.example.com", 0, 0, 0, 1},

		// Mixed
		{"mixed entries", "10.0.0.0/8, jira.internal:8080, *.example.com, [::1]:443, 192.168.1.1", 1, 2, 1, 1},
		{"whitespace around entries", "  10.0.0.0/8 , example.com , *.foo.bar  ", 1, 0, 1, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			al, err := ParseNetworkAllowList(c.input)
			require.NoError(t, err)
			require.NotNil(t, al)
			assert.Len(t, al.cidrs, c.cidrs, "cidrs")
			assert.Len(t, al.ips, c.ips, "ips")
			assert.Len(t, al.hosts, c.hosts, "hosts")
			assert.Len(t, al.wildcards, c.wildcards, "wildcards")
		})
	}
}

func TestParseNetworkAllowListInvalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string // substring expected in error message
	}{
		{"garbage characters", "!!!", "invalid character"},
		{"spaces in hostname", "exam ple.com", "invalid character"},
		{"invalid CIDR prefix length", "10.0.0.0/33", "invalid CIDR"},
		{"CIDR with port suffix", "10.0.0.0/8:80", "invalid CIDR"},
		{"port too high", "example.com:99999", "port number out of range"},
		{"port zero", "example.com:0", "port number out of range"},
		{"port non-numeric", "example.com:abc", "invalid port number"},
		{"negative port", "example.com:-1", "port number out of range"},
		{"bare wildcard star", "*", "wildcard must be"},
		{"wildcard no dot", "*example.com", "wildcard must be"},
		{"wildcard double star", "**.example.com", "wildcard must be"},
		{"wildcard empty domain", "*.", "wildcard must be"},
		{"hyphen start label", "-example.com", "cannot start or end with a hyphen"},
		{"hyphen end label", "example-.com", "cannot start or end with a hyphen"},
		{"missing bracket", "[::1", "missing closing bracket"},
		{"bad bracketed content", "[not-ipv6]:443", "invalid IPv6 address"},
		{"junk after bracket", "[::1]junk", "unexpected characters"},
		{"empty label", "example..com", "invalid DNS label length"},
		{"bare colon garbage", "::not-an-ip", "invalid IPv6 address"},
		{"DNS name too long", longDNS(254), "DNS name exceeds 253 characters"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseNetworkAllowList(c.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), c.want)
		})
	}
}

// longDNS generates a DNS name of exactly n bytes using repeated "a." labels.
func longDNS(n int) string {
	// "a." is 2 bytes per label, so n/2 repetitions of "a." gives n bytes.
	s := ""
	for len(s) < n {
		if s != "" {
			s += "."
		}
		s += "a"
	}
	return s
}

func TestParseNetworkAllowListPortStorage(t *testing.T) {
	t.Run("IP without port stores port 0", func(t *testing.T) {
		al, err := ParseNetworkAllowList("192.168.1.1")
		require.NoError(t, err)
		require.Len(t, al.ips, 1)
		assert.Equal(t, 0, al.ips[0].port)
	})

	t.Run("IP with port stores it", func(t *testing.T) {
		al, err := ParseNetworkAllowList("192.168.1.1:8080")
		require.NoError(t, err)
		require.Len(t, al.ips, 1)
		assert.Equal(t, 8080, al.ips[0].port)
	})

	t.Run("bracketed IPv6 with port stores it", func(t *testing.T) {
		al, err := ParseNetworkAllowList("[::1]:443")
		require.NoError(t, err)
		require.Len(t, al.ips, 1)
		assert.Equal(t, 443, al.ips[0].port)
	})

	t.Run("host without port stores port 0", func(t *testing.T) {
		al, err := ParseNetworkAllowList("example.com")
		require.NoError(t, err)
		require.Len(t, al.hosts, 1)
		assert.Equal(t, 0, al.hosts[0].port)
	})

	t.Run("host with port stores it", func(t *testing.T) {
		al, err := ParseNetworkAllowList("example.com:9090")
		require.NoError(t, err)
		require.Len(t, al.hosts, 1)
		assert.Equal(t, 9090, al.hosts[0].port)
	})

	t.Run("wildcard without port stores port 0", func(t *testing.T) {
		al, err := ParseNetworkAllowList("*.example.com")
		require.NoError(t, err)
		require.Len(t, al.wildcards, 1)
		assert.Equal(t, 0, al.wildcards[0].port)
	})

	t.Run("wildcard with port stores it", func(t *testing.T) {
		al, err := ParseNetworkAllowList("*.example.com:443")
		require.NoError(t, err)
		require.Len(t, al.wildcards, 1)
		assert.Equal(t, 443, al.wildcards[0].port)
	})
}

func TestParseNetworkAllowListCaseNormalization(t *testing.T) {
	al, err := ParseNetworkAllowList("Example.COM, *.Internal.EXAMPLE.com")
	require.NoError(t, err)
	require.Len(t, al.hosts, 1)
	assert.Equal(t, "example.com", al.hosts[0].host)
	require.Len(t, al.wildcards, 1)
	assert.Equal(t, ".internal.example.com", al.wildcards[0].suffix)
}

func TestNetworkAllowListMatchesIP(t *testing.T) {
	al, err := ParseNetworkAllowList("10.0.0.0/8, fc00::/7, 192.168.1.1, 172.16.0.5:8080")
	require.NoError(t, err)

	cases := []struct {
		name  string
		ip    string
		port  int
		match bool
	}{
		// CIDR matches (any port).
		{"IPv4 in CIDR", "10.1.2.3", 80, true},
		{"IPv4 in CIDR different port", "10.1.2.3", 443, true},
		{"IPv6 in CIDR", "fc00::1", 80, true},
		{"IPv4 outside CIDR", "11.0.0.1", 80, false},
		{"IPv6 outside CIDR", "2001:db8::1", 80, false},

		// Exact IP, no port restriction.
		{"exact IP any port", "192.168.1.1", 9999, true},
		{"exact IP port zero", "192.168.1.1", 0, true},
		{"wrong IP", "192.168.1.2", 80, false},

		// Exact IP with port restriction.
		{"IP port matches", "172.16.0.5", 8080, true},
		{"IP port mismatch", "172.16.0.5", 443, false},
		{"IP port zero vs restricted", "172.16.0.5", 0, false},

		// IPv4-mapped IPv6 matches IPv4 entry.
		{"v4-mapped matches v4 entry", "::ffff:192.168.1.1", 80, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ip := net.ParseIP(c.ip)
			require.NotNil(t, ip, "bad test IP: %s", c.ip)
			assert.Equal(t, c.match, al.MatchesIP(ip, c.port))
		})
	}
}

func TestNetworkAllowListMatchesHost(t *testing.T) {
	al, err := ParseNetworkAllowList("jira.internal, scep.corp:8080, *.example.com, *.secure.io:443")
	require.NoError(t, err)

	cases := []struct {
		name  string
		host  string
		port  int
		match bool
	}{
		// Exact hostname, no port restriction.
		{"exact match", "jira.internal", 80, true},
		{"exact match different port", "jira.internal", 443, true},
		{"case insensitive", "Jira.INTERNAL", 80, true},
		{"trailing dot", "jira.internal.", 80, true},
		{"no match", "other.internal", 80, false},

		// Exact hostname with port restriction.
		{"host port matches", "scep.corp", 8080, true},
		{"host port mismatch", "scep.corp", 443, false},

		// Wildcard, no port restriction.
		{"wildcard single level", "sub.example.com", 80, true},
		{"wildcard multi level", "a.b.example.com", 80, true},
		{"wildcard case insensitive", "Sub.Example.COM", 80, true},
		{"wildcard no match on base", "example.com", 80, false},
		{"wildcard no match unrelated", "example.org", 80, false},

		// Wildcard with port restriction.
		{"wildcard port matches", "app.secure.io", 443, true},
		{"wildcard port mismatch", "app.secure.io", 80, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.match, al.MatchesHost(c.host, c.port))
		})
	}
}

func TestNetworkAllowListMatchesURL(t *testing.T) {
	al, err := ParseNetworkAllowList("jira.internal, *.example.com:443, 10.0.0.0/8, 192.168.1.1:8080")
	require.NoError(t, err)

	cases := []struct {
		name   string
		rawURL string
		match  bool
	}{
		// DNS host, port inferred from scheme.
		{"https infers 443", "https://jira.internal/path", true},
		{"http infers 80", "http://jira.internal/path", true},

		// Wildcard with port restriction, scheme inference.
		{"wildcard https matches 443", "https://sub.example.com/api", true},
		{"wildcard http does not match 443", "http://sub.example.com/api", false},
		{"wildcard explicit 443", "https://sub.example.com:443/api", true},
		{"wildcard explicit 8080", "https://sub.example.com:8080/api", false},

		// IP host in URL → falls through to MatchesIP.
		{"IP in CIDR via URL", "http://10.5.5.5:80/hook", true},
		{"IP with port match via URL", "http://192.168.1.1:8080/hook", true},
		{"IP with port mismatch via URL", "http://192.168.1.1:9090/hook", false},
		{"IP not in list via URL", "http://172.16.0.1:80/hook", false},

		// Unknown scheme → port 0.
		{"unknown scheme no port", "ftp://jira.internal/file", true},
		{"unknown scheme with explicit port", "ftp://jira.internal:21/file", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			u, err := url.Parse(c.rawURL)
			require.NoError(t, err)
			assert.Equal(t, c.match, al.MatchesURL(u))
		})
	}
}

func TestNetworkAllowListNilAndEmpty(t *testing.T) {
	t.Run("nil receiver", func(t *testing.T) {
		var al *NetworkAllowList
		assert.False(t, al.MatchesIP(net.ParseIP("10.0.0.1"), 80))
		assert.False(t, al.MatchesHost("example.com", 80))
		u, _ := url.Parse("https://example.com")
		assert.False(t, al.MatchesURL(u))
		assert.True(t, al.IsEmpty())
	})

	t.Run("empty allow list", func(t *testing.T) {
		al, err := ParseNetworkAllowList("")
		require.NoError(t, err)
		assert.True(t, al.IsEmpty())
		assert.False(t, al.MatchesIP(net.ParseIP("10.0.0.1"), 80))
		assert.False(t, al.MatchesHost("example.com", 80))
	})

	t.Run("non-empty is not empty", func(t *testing.T) {
		al, err := ParseNetworkAllowList("10.0.0.0/8")
		require.NoError(t, err)
		assert.False(t, al.IsEmpty())
	})
}

func TestDefaultPort(t *testing.T) {
	assert.Equal(t, 80, DefaultPort("http"))
	assert.Equal(t, 443, DefaultPort("https"))
	assert.Equal(t, 80, DefaultPort("HTTP"))
	assert.Equal(t, 443, DefaultPort("HTTPS"))
	assert.Equal(t, 0, DefaultPort("ftp"))
	assert.Equal(t, 0, DefaultPort(""))
}
