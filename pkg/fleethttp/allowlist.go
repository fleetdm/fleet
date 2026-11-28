package fleethttp

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// NetworkAllowList is a parsed set of IP addresses, CIDR networks, DNS
// names, and wildcard DNS patterns that should bypass SSRF network
// blocking. Use ParseNetworkAllowList to create one from a
// comma-separated configuration string.
type NetworkAllowList struct {
	cidrs     []cidrEntry
	ips       []ipEntry
	hosts     []hostEntry
	wildcards []wildcardEntry
}

type cidrEntry struct {
	network *net.IPNet
}

type ipEntry struct {
	ip   net.IP
	port int // 0 = any port
}

type hostEntry struct {
	host string // lowercase, no trailing dot
	port int    // 0 = any port
}

type wildcardEntry struct {
	suffix string // lowercase, e.g. ".example.com"
	port   int    // 0 = any port
}

// ParseNetworkAllowList parses a comma-separated list of allow-list
// entries. Each entry is one of:
//
//   - An IP address: 192.168.1.1, ::1, [::1]
//   - An IP address with port: 192.168.1.1:8080, [::1]:443
//   - A CIDR network: 10.0.0.0/8, fc00::/7
//   - A DNS name: jira.internal.example.com
//   - A DNS name with port: jira.internal.example.com:8080
//   - A wildcard DNS pattern: *.example.com
//   - A wildcard DNS pattern with port: *.example.com:443
//
// Whitespace around entries is trimmed. Empty entries (e.g. from a
// trailing comma) are skipped. Returns an error if any entry is
// malformed.
func ParseNetworkAllowList(input string) (*NetworkAllowList, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return &NetworkAllowList{}, nil
	}

	al := &NetworkAllowList{}
	for _, raw := range strings.Split(input, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}
		if err := al.addEntry(entry); err != nil {
			return nil, fmt.Errorf("invalid allow-list entry %q: %w", entry, err)
		}
	}
	return al, nil
}

func (al *NetworkAllowList) addEntry(entry string) error {
	switch {
	case strings.Contains(entry, "/"):
		return al.addCIDR(entry)
	case strings.HasPrefix(entry, "["):
		return al.addBracketedIPv6(entry)
	case strings.HasPrefix(entry, "*"):
		return al.addWildcard(entry)
	case strings.Count(entry, ":") > 1:
		// Multiple colons without a leading bracket → bare IPv6.
		return al.addBareIPv6(entry)
	case strings.Contains(entry, ":"):
		// Exactly one colon → IPv4:port or hostname:port.
		return al.addHostPort(entry)
	default:
		// No colon, no slash, no bracket → bare IPv4 or DNS name.
		if ip := net.ParseIP(entry); ip != nil {
			al.ips = append(al.ips, ipEntry{ip: ip})
			return nil
		}
		if err := validateDNSName(entry); err != nil {
			return err
		}
		al.hosts = append(al.hosts, hostEntry{host: strings.ToLower(strings.TrimSuffix(entry, "."))})
		return nil
	}
}

func (al *NetworkAllowList) addCIDR(entry string) error {
	_, network, err := net.ParseCIDR(entry)
	if err != nil {
		return fmt.Errorf("invalid CIDR: %w", err)
	}
	al.cidrs = append(al.cidrs, cidrEntry{network: network})
	return nil
}

func (al *NetworkAllowList) addBracketedIPv6(entry string) error {
	closeBracket := strings.Index(entry, "]")
	if closeBracket < 0 {
		return errors.New("missing closing bracket in IPv6 address")
	}

	ipStr := entry[1:closeBracket]
	rest := entry[closeBracket+1:]

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IPv6 address: %q", ipStr)
	}

	if rest == "" {
		al.ips = append(al.ips, ipEntry{ip: ip})
		return nil
	}

	if !strings.HasPrefix(rest, ":") {
		return fmt.Errorf("unexpected characters after IPv6 address: %q", rest)
	}

	port, err := parsePort(rest[1:])
	if err != nil {
		return err
	}
	al.ips = append(al.ips, ipEntry{ip: ip, port: port})
	return nil
}

func (al *NetworkAllowList) addBareIPv6(entry string) error {
	ip := net.ParseIP(entry)
	if ip == nil {
		return fmt.Errorf("invalid IPv6 address: %q", entry)
	}
	al.ips = append(al.ips, ipEntry{ip: ip})
	return nil
}

func (al *NetworkAllowList) addWildcard(entry string) error {
	if !strings.HasPrefix(entry, "*.") || len(entry) < 3 {
		return errors.New("wildcard must be in the form *.domain.tld")
	}
	rest := entry[2:] // strip "*."

	host, port, err := splitDNSPort(rest)
	if err != nil {
		return err
	}
	if err := validateDNSName(host); err != nil {
		return fmt.Errorf("invalid wildcard domain: %w", err)
	}
	al.wildcards = append(al.wildcards, wildcardEntry{
		suffix: "." + strings.ToLower(strings.TrimSuffix(host, ".")),
		port:   port,
	})
	return nil
}

func (al *NetworkAllowList) addHostPort(entry string) error {
	host, portStr, err := net.SplitHostPort(entry)
	if err != nil {
		return fmt.Errorf("invalid host:port: %w", err)
	}
	port, err := parsePort(portStr)
	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip != nil {
		al.ips = append(al.ips, ipEntry{ip: ip, port: port})
		return nil
	}
	if err := validateDNSName(host); err != nil {
		return err
	}
	al.hosts = append(al.hosts, hostEntry{
		host: strings.ToLower(strings.TrimSuffix(host, ".")),
		port: port,
	})
	return nil
}

// MatchesIP reports whether ip matches any IP address or CIDR entry in
// the allow list. CIDR entries match regardless of port. IP entries
// without a port match any port; entries with a port match only when
// port matches.
func (al *NetworkAllowList) MatchesIP(ip net.IP, port int) bool {
	if al == nil {
		return false
	}
	for _, entry := range al.cidrs {
		if entry.network.Contains(ip) {
			return true
		}
	}
	for _, entry := range al.ips {
		if entry.ip.Equal(ip) && (entry.port == 0 || entry.port == port) {
			return true
		}
	}
	return false
}

// MatchesHost reports whether host matches any DNS name or wildcard
// entry in the allow list. Matching is case-insensitive. Entries
// without a port match any port; entries with a port match only when
// port matches.
func (al *NetworkAllowList) MatchesHost(host string, port int) bool {
	if al == nil {
		return false
	}
	lower := strings.ToLower(strings.TrimSuffix(host, "."))
	for _, entry := range al.hosts {
		if entry.host == lower && (entry.port == 0 || entry.port == port) {
			return true
		}
	}
	for _, entry := range al.wildcards {
		if strings.HasSuffix(lower, entry.suffix) && (entry.port == 0 || entry.port == port) {
			return true
		}
	}
	return false
}

// MatchesURL reports whether the URL's host matches any entry in the
// allow list. When the URL has no explicit port, the default port for
// the scheme is inferred (http→80, https→443). If the URL's host is an
// IP address, IP and CIDR entries are checked in addition to DNS
// entries.
func (al *NetworkAllowList) MatchesURL(u *url.URL) bool {
	if al == nil {
		return false
	}
	host := u.Hostname()
	port := portFromURL(u)

	if al.MatchesHost(host, port) {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return al.MatchesIP(ip, port)
	}
	return false
}

// DefaultPort returns the standard port number for common URL schemes
// (http→80, https→443). Returns 0 for unrecognized schemes.
func DefaultPort(scheme string) int {
	switch strings.ToLower(scheme) {
	case "http":
		return 80
	case "https":
		return 443
	default:
		return 0
	}
}

func portFromURL(u *url.URL) int {
	if p := u.Port(); p != "" {
		port, err := strconv.Atoi(p)
		if err != nil || port < 1 || port > 65535 {
			return 0
		}
		return port
	}
	return DefaultPort(u.Scheme)
}

// IsEmpty reports whether the allow list contains no entries.
func (al *NetworkAllowList) IsEmpty() bool {
	if al == nil {
		return true
	}
	return len(al.cidrs) == 0 && len(al.ips) == 0 && len(al.hosts) == 0 && len(al.wildcards) == 0
}

// splitDNSPort splits a string that is known NOT to be an IPv6 address
// into a host and optional numeric port. Returns port 0 when no port is
// present.
func splitDNSPort(s string) (string, int, error) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return s, 0, nil
	}
	port, err := parsePort(s[i+1:])
	if err != nil {
		return "", 0, err
	}
	return s[:i], port, nil
}

func parsePort(s string) (int, error) {
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid port number: %q", s)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("port number out of range: %d", port)
	}
	return port, nil
}

func validateDNSName(name string) error {
	if len(name) == 0 {
		return errors.New("empty DNS name")
	}
	if len(name) > 253 {
		return errors.New("DNS name exceeds 253 characters")
	}
	trimmed := strings.TrimSuffix(name, ".")
	if trimmed == "" {
		return errors.New("DNS name is only a root dot")
	}
	for _, label := range strings.Split(trimmed, ".") {
		if err := validateDNSLabel(label); err != nil {
			return err
		}
	}
	return nil
}

func validateDNSLabel(label string) error {
	if len(label) == 0 || len(label) > 63 {
		return fmt.Errorf("invalid DNS label length: %q", label)
	}
	for i, c := range label {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9':
			// valid
		case c == '-':
			if i == 0 || i == len(label)-1 {
				return fmt.Errorf("DNS label %q cannot start or end with a hyphen", label)
			}
		case c == '_':
			// Underscores appear in SRV records and some real-world hostnames
			// (e.g. _dmarc.example.com). Allow them anywhere in the label.
		default:
			return fmt.Errorf("invalid character %q in DNS name", c)
		}
	}
	return nil
}
