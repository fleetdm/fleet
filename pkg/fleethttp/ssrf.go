package fleethttp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
)

var (
	// https://en.wikipedia.org/wiki/Reserved_IP_addresses
	IPV4_BLACKLIST = []string{
		"0.0.0.0/8",          // Current network (only valid as source address)
		"10.0.0.0/8",         // Private network
		"100.64.0.0/10",      // Shared Address Space
		"127.0.0.0/8",        // Loopback
		"169.254.0.0/16",     // Link-local
		"172.16.0.0/12",      // Private network
		"192.0.0.0/24",       // IETF Protocol Assignments
		"192.0.2.0/24",       // TEST-NET-1, documentation and examples
		"192.88.99.0/24",     // IPv6 to IPv4 relay (includes 2002::/16)
		"192.168.0.0/16",     // Private network
		"198.18.0.0/15",      // Network benchmark tests
		"198.51.100.0/24",    // TEST-NET-2, documentation and examples
		"203.0.113.0/24",     // TEST-NET-3, documentation and examples
		"224.0.0.0/4",        // IP multicast (former Class D network)
		"240.0.0.0/4",        // Reserved (former Class E network)
		"255.255.255.255/32", // Broadcast
	}

	IPV6_BLACKLIST = []string{
		"::1/128",        // Loopback
		"64:ff9b::/96",   // IPv4/IPv6 translation (RFC 6052)
		"64:ff9b:1::/48", // Local-use IPv4/IPv6 translation (RFC 8215)
		"100::/64",       // Discard prefix (RFC 6666)
		"2001::/32",      // Teredo tunneling
		"2001:10::/28",   // Deprecated (previously ORCHID)
		"2001:20::/28",   // ORCHIDv2
		"2001:db8::/32",  // Documentation and example source code
		"2002::/16",      // 6to4
		"3fff::/20",      // Documentation (RFC 9637, 2024)
		"5f00::/16",      // IPv6 Segment Routing (SRv6)
		"fc00::/7",       // Unique local address
		"fe80::/10",      // Link-local address
		"ff00::/8",       // Multicast
	}
)

var blockedCIDRs []*net.IPNet

func init() {
	for _, cidr := range append(IPV4_BLACKLIST, IPV6_BLACKLIST...) {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("fleethttp: invalid blocked CIDR %q: %v", cidr, err))
		}
		blockedCIDRs = append(blockedCIDRs, network)
	}
}

// isBlockedIP returns true when ip falls within any of the protected ranges.
func isBlockedIP(ip net.IP) bool {
	if ip4 := ip.To4(); ip4 != nil && len(ip) == net.IPv6len {
		ip = ip4
	}
	for _, network := range blockedCIDRs {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// SSRFError is returned when a URL resolves to a protected IP range.
type SSRFError struct {
	URL string
	IP  net.IP
}

func (e *SSRFError) Error() string {
	return fmt.Sprintf("URL %q resolves to a blocked address", e.URL)
}

func checkResolvedAddrs(ctx context.Context, host, rawURL string, resolver func(context.Context, string) ([]string, error)) error {
	addrs, err := resolver(ctx, host)
	if err != nil {
		return fmt.Errorf("resolving host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("host %q resolved to no addresses", host)
	}
	for _, addr := range addrs {
		h, _, err := net.SplitHostPort(addr)
		if err != nil {
			h = addr
		}
		ip := net.ParseIP(h)
		if ip == nil {
			return fmt.Errorf("resolved address %q for host %q is not a valid IP", h, host)
		}
		if isBlockedIP(ip) {
			return &SSRFError{URL: rawURL, IP: ip}
		}
	}
	return nil
}

// CheckURLForSSRF validates rawURL against SSRF attack vectors using a static blocklist.
func CheckURLForSSRF(ctx context.Context, rawURL string, resolver func(ctx context.Context, host string) ([]string, error)) error {
	if dev_mode.IsEnabled {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	scheme := parsed.Scheme
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URL scheme %q is not allowed; must be http or https", scheme)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return errors.New("URL has no host")
	}

	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedIP(ip) {
			return &SSRFError{URL: rawURL, IP: ip}
		}
		return nil
	}

	if resolver == nil {
		resolver = net.DefaultResolver.LookupHost
	}
	return checkResolvedAddrs(ctx, hostname, rawURL, resolver)
}

// SSRFDialContext returns a DialContext function that validates against SSRF attack vectors using a static blocklist.
func SSRFDialContext(
	base *net.Dialer,
	resolver func(ctx context.Context, host string) ([]string, error),
	dial func(ctx context.Context, network, addr string) (net.Conn, error),
) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if base == nil {
		base = &net.Dialer{}
	}
	if resolver == nil {
		resolver = net.DefaultResolver.LookupHost
	}
	if dial == nil {
		dial = base.DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if dev_mode.IsEnabled {
			return dial(ctx, network, addr)
		}

		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("ssrf dial: splitting host/port from %q: %w", addr, err)
		}

		if err := checkResolvedAddrs(ctx, host, net.JoinHostPort(host, port), resolver); err != nil {
			return nil, err
		}

		return dial(ctx, network, addr)
	}
}
