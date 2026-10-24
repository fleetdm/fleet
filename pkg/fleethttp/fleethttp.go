// Package fleethttp provides uniform creation and configuration of HTTP
// related types used throughout Fleet.
package fleethttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// NetworkBlockingMode controls how outbound HTTP connections are filtered.
type NetworkBlockingMode int32

const (
	// BlockingDisabled performs no filtering. This is the default for tests,
	// CLI tools, and any caller that doesn't go through fleet serve.
	BlockingDisabled NetworkBlockingMode = iota
	// BlockingFull blocks both the always-blocked tier (loopback, IMDS) and
	// private networks (RFC 1918, etc.). This is the production default.
	BlockingFull
	// BlockingPrivateAllowed blocks the always-blocked tier only. Private
	// networks are allowed for environments with on-prem integrations
	// (e.g. EJBCA, Jira, SCEP servers). Set via
	// --server_allow_private_network_integrations.
	BlockingPrivateAllowed
	// BlockingBypassAll performs no filtering at all. Used in dev mode, and
	// can also be set in production via --server_bypass_network_blocking as
	// an infra-level escape hatch for environments where egress is already
	// constrained by external infrastructure (e.g. a proxy or firewall) that
	// Fleet's own checks would otherwise conflict with. Disables SSRF
	// protection for every outbound integration request, not just the one
	// causing the conflict.
	BlockingBypassAll
)

// networkBlockingMode holds the current blocking mode. Default is
// BlockingDisabled so tests, CLI tools, and non-serve callers are unaffected.
var networkBlockingMode atomic.Int32

// SetNetworkBlockingMode sets the blocking mode. Called by fleet serve at startup.
func SetNetworkBlockingMode(mode NetworkBlockingMode) {
	networkBlockingMode.Store(int32(mode))
}

// networkAllowList holds the optional SSRF allow-list. When non-nil,
// hostnames and resolved IPs are checked against it before blocking.
// The atomic pointer provides lock-free reads on the dialer hot path;
// writes are a single Store with no read-modify-write sequence, so no
// additional mutex is needed.
var networkAllowList atomic.Pointer[NetworkAllowList]

// SetNetworkAllowList replaces the active allow-list. Pass nil to
// clear it. The previous list remains in use by any in-flight dial
// that already snapshot the pointer.
func SetNetworkAllowList(al *NetworkAllowList) {
	networkAllowList.Store(al)
}

// ErrPrivateNetworkBlocked is returned when a connection to a private network
// address is blocked.
var ErrPrivateNetworkBlocked = errors.New("connections to private network addresses are blocked")

// alwaysBlockedCIDRs are blocked unconditionally, even when
// --allow_private_network_integrations is set. No legitimate integration
// should ever target these addresses.
var alwaysBlockedCIDRs = parseCIDRs([]string{
	"0.0.0.0/8",      // "this" network (RFC 1122); 0.0.0.0 itself routes to loopback
	"127.0.0.0/8",    // loopback
	"169.254.0.0/16", // link-local (includes cloud IMDS at 169.254.169.254)
	// Covers the unspecified address (::), IPv6 loopback (::1) and the
	// deprecated IPv4-compatible form (::a.b.c.d, e.g. ::127.0.0.1).
	"::/96",
	"fe80::/10", // IPv6 link-local
})

// privateNetworkCIDRs are blocked when private network blocking is enabled.
// Customers with on-prem integrations (e.g. EJBCA, Jira, SCEP servers on
// private networks) can disable this with --allow_private_network_integrations.
var privateNetworkCIDRs = parseCIDRs([]string{
	"10.0.0.0/8",      // RFC 1918 private
	"100.64.0.0/10",   // shared address space (RFC 6598)
	"172.16.0.0/12",   // RFC 1918 private
	"192.0.0.0/24",    // IETF protocol assignments
	"192.168.0.0/16",  // RFC 1918 private
	"198.18.0.0/15",   // benchmarking (RFC 2544)
	"198.51.100.0/24", // TEST-NET-2 (documentation)
	"203.0.113.0/24",  // TEST-NET-3 (documentation)
	"224.0.0.0/4",     // multicast
	"240.0.0.0/4",     // reserved
	"fc00::/7",        // IPv6 unique local
	"ff00::/8",        // IPv6 multicast
})

// parseCIDRs converts CIDR strings (e.g. "10.0.0.0/8") into net.IPNet objects
// for IP range matching. Panics on malformed input since the lists are hardcoded
// constants -- this runs once at package init, before the server starts.
func parseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("fleethttp: bad CIDR " + cidr)
		}
		nets = append(nets, ipNet)
	}
	return nets
}

// ipInCIDRs returns true if the given IP falls within any of the provided CIDR ranges.
func ipInCIDRs(ip net.IP, cidrs []*net.IPNet) bool {
	for _, cidr := range cidrs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// nat64WellKnownPrefix is the RFC 6052 prefix used to reach IPv4 hosts from an
// IPv6-only network. Addresses under it carry an IPv4 address in their low 32
// bits, so the embedded address is what has to be checked -- the prefix itself
// carries public IPv4 traffic too and can't simply be listed as internal.
//
// RFC 6052 also allows a network-specific prefix, which is drawn from the
// operator's own address space and so can't be recognized without being
// configured: an address under one is indistinguishable from any other address
// in that space, and treating every address whose low 32 bits look internal as
// a translation would reject unrelated public addresses.
var nat64WellKnownPrefix = parseCIDRs([]string{"64:ff9b::/96"})

// embeddedIPv4 returns the IPv4 address carried by a NAT64 address, or nil if
// the address doesn't carry one.
func embeddedIPv4(ip net.IP) net.IP {
	if !ipInCIDRs(ip, nat64WellKnownPrefix) {
		return nil
	}
	// ipInCIDRs only matches a 16-byte address against an IPv6 range.
	ip16 := ip.To16()
	return net.IPv4(ip16[12], ip16[13], ip16[14], ip16[15])
}

// privateNetworkBlockingDialContext returns a DialContext function that blocks
// connections to private/reserved IP addresses. It resolves DNS first, then
// checks the resolved IP before connecting -- this catches DNS rebinding.
func privateNetworkBlockingDialContext(dialer *net.Dialer) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		mode := NetworkBlockingMode(networkBlockingMode.Load())
		if mode == BlockingDisabled || mode == BlockingBypassAll {
			return dialer.DialContext(ctx, network, addr)
		}

		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		// Snapshot the allow-list pointer so a concurrent
		// SetNetworkAllowList does not change behaviour mid-evaluation.
		allowList := networkAllowList.Load()

		portNum, _ := strconv.Atoi(port)

		// If the hostname itself is allow-listed, skip all IP-level
		// blocking — both alwaysBlockedCIDRs and privateNetworkCIDRs.
		// This is intentionally an OR with the per-IP check below:
		// hostname match alone is sufficient, because the allow-list
		// is server configuration (not user input) and legitimate
		// deployments need to reach loopback sidecars or on-prem hosts
		// whose IPs the admin may not know or want to track.

		// NOTE(fuhry@2026-10-02): possible future improvement: support
		// an AND semantic (host must match allowed pattern AND resolve
		// to an allowed IP) via a boolean opt-in setting.
		hostAllowed := allowList.MatchesHost(host, portNum)

		// Resolve host to IPs. When host is already an IP literal,
		// skip the DNS lookup — passing an IP to LookupIPAddr either
		// triggers a needless PTR query or fails, depending on the
		// resolver implementation.
		var ips []net.IPAddr
		if ip := net.ParseIP(host); ip != nil {
			ips = []net.IPAddr{{IP: ip}}
		} else {
			ips, err = net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
		}

		var lastErr error

		// Connect using the already-resolved IPs to prevent DNS rebinding
		// (a second DNS lookup could return a different, malicious IP).
		// Try each address in order so that multi-homed hosts and
		// dual-stack (A + AAAA) records work: if one address is
		// unreachable, the next is tried until the context expires.
		for _, ip := range ips {
			// Perform IP allow-list checks only if the hostname isn't allow-listed.
			// This is a deliberate usability choice: requiring both the DNS name and the
			// resolved IP to be allow-listed is not intuitive, and could lead to unexpected
			// breakage in some circumstances: for example, an internally-hosted IdP metadata
			// endpoint moves to a different IP, or IPv6 is rolled out on top of an existing
			// IPv4 network.
			if !hostAllowed {
				if err := checkIPAllowed(ip, portNum, mode, allowList); err != nil {
					lastErr = fmt.Errorf("%w: %s resolves to %s", err, host, ip.IP)
					slog.DebugContext(ctx, "disallowed IP", "err", lastErr)
					continue
				}
			}
			var conn net.Conn
			conn, lastErr = dialer.DialContext(ctx, network, net.JoinHostPort(ip.IP.String(), port))
			if lastErr == nil {
				return conn, nil
			}
			if ctx.Err() != nil {
				break
			}
		}
		return nil, lastErr
	}
}

func checkIPAllowed(ip net.IPAddr, portNum int, mode NetworkBlockingMode, allowList *NetworkAllowList) error {
	for _, check := range []net.IP{ip.IP, embeddedIPv4(ip.IP)} {
		if check == nil {
			continue
		}
		// Allow-listed IPs bypass blocking.
		if allowList.MatchesIP(check, portNum) {
			continue
		}
		// Tier 1: always blocked (loopback, cloud IMDS). Cannot be
		// overridden with --server_allow_private_network_integrations.
		if ipInCIDRs(check, alwaysBlockedCIDRs) {
			return ErrPrivateNetworkBlocked
		}
		// Tier 2: private networks. Only blocked in BlockingFull mode.
		if mode == BlockingFull && ipInCIDRs(check, privateNetworkCIDRs) {
			return ErrPrivateNetworkBlocked
		}
	}

	return nil
}

// DefaultTimeout is the request timeout applied by NewClient when the caller does not provide WithTimeout or WithNoTimeout.
const DefaultTimeout = 60 * time.Second

type clientOpts struct {
	timeout     time.Duration
	tlsConf     *tls.Config
	noFollow    bool
	cookieJar   http.CookieJar
	maxRespSize int64

	responseHeaderTimeout time.Duration
}

// ClientOpt is the type for the client-specific options.
type ClientOpt func(o *clientOpts)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) ClientOpt {
	return func(o *clientOpts) {
		o.timeout = t
	}
}

// WithNoTimeout removes the DefaultTimeout, leaving the HTTP client without a timeout. Callers that stream large responses or
// rely on a per-request context deadline need this; everything else should keep the default.
func WithNoTimeout() ClientOpt {
	return func(o *clientOpts) {
		o.timeout = 0
	}
}

// WithResponseHeaderTimeout bounds how long the client waits for response
// headers after sending the request. Without it, the client waits indefinitely.
func WithResponseHeaderTimeout(t time.Duration) ClientOpt {
	return func(o *clientOpts) {
		o.responseHeaderTimeout = t
	}
}

// WithTLSClientConfig provides the TLS configuration to use for the HTTP
// client's transport.
func WithTLSClientConfig(conf *tls.Config) ClientOpt {
	return func(o *clientOpts) {
		o.tlsConf = conf.Clone()
	}
}

// WithFollowRedir configures the HTTP client to follow redirections or not,
// based on the follow value.
func WithFollowRedir(follow bool) ClientOpt {
	return func(o *clientOpts) {
		o.noFollow = !follow
	}
}

// WithCookieJar configures the HTTP client to use the provided
// cookie jar to manage cookies between requests.
func WithCookieJar(jar http.CookieJar) ClientOpt {
	return func(o *clientOpts) {
		o.cookieJar = jar
	}
}

// WithMaxResponseSize caps the size of response bodies the client will read.
// Zero or less disables the cap.
func WithMaxResponseSize(maxSizeBytes int64) ClientOpt {
	return func(o *clientOpts) {
		o.maxRespSize = maxSizeBytes
	}
}

// defaultBaseTransport falls back to http.DefaultTransport when a test has
// replaced it with a non-*http.Transport, so mock chains are preserved.
func defaultBaseTransport() http.RoundTripper {
	if _, ok := http.DefaultTransport.(*http.Transport); ok {
		return NewTransport()
	}
	return http.DefaultTransport
}

// NewClient returns an HTTP client configured according to the provided
// options.
func NewClient(opts ...ClientOpt) *http.Client {
	co := clientOpts{timeout: DefaultTimeout}
	for _, opt := range opts {
		opt(&co)
	}

	//nolint:gocritic
	cli := &http.Client{
		Timeout: co.timeout,
	}
	if co.noFollow {
		cli.CheckRedirect = noFollowRedirect
	}
	// Always create a custom transport (even without TLS config) so that
	// every client gets the private network blocking DialContext from
	// NewTransport. Without this, nil would fall back to Go's default
	// transport which has no IP blocking.
	var baseTransport http.RoundTripper
	if co.tlsConf != nil {
		baseTransport = NewTransport(WithTLSConfig(co.tlsConf))
	} else {
		baseTransport = defaultBaseTransport()
	}
	if tr, ok := baseTransport.(*http.Transport); ok {
		tr.ResponseHeaderTimeout = co.responseHeaderTimeout
	}
	if co.maxRespSize > 0 {
		baseTransport = newSizeLimitTransport(baseTransport, co.maxRespSize)
	}
	cli.Transport = otelhttp.NewTransport(baseTransport)
	if co.cookieJar != nil {
		cli.Jar = co.cookieJar
	}
	return cli
}

type transportOpts struct {
	tlsConf *tls.Config
}

// TransportOpt is the type for transport-specific options.
type TransportOpt func(o *transportOpts)

// WithTLSConfig sets the TLS configuration of the transport.
func WithTLSConfig(conf *tls.Config) TransportOpt {
	return func(o *transportOpts) {
		o.tlsConf = conf.Clone()
	}
}

// NewTransport creates an http transport (a type that implements
// http.RoundTripper) with the provided optional options. The transport is
// derived from Go's http.DefaultTransport and only overrides the specific
// parts it needs to, so that it keeps its sane defaults for the rest (such as
// timeouts and proxy support).
func NewTransport(opts ...TransportOpt) *http.Transport {
	var to transportOpts
	for _, opt := range opts {
		opt(&to)
	}

	// Start from DefaultTransport to inherit its sane defaults. Guard the type
	// assertion in case a test replaces DefaultTransport with a non-*Transport.
	dt, ok := http.DefaultTransport.(*http.Transport)
	if !ok || dt == nil {
		dt = &http.Transport{ForceAttemptHTTP2: true} //nolint:gocritic // we are inside fleethttp itself
	}
	tr := dt.Clone()
	if to.tlsConf != nil {
		tr.TLSClientConfig = to.tlsConf
	}
	tr.DialContext = privateNetworkBlockingDialContext(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	})
	return tr
}

func noFollowRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// githubAPIHost is the only host that receives the GitHub token. Tests override it.
var githubAPIHost = "api.github.com"

// NewGithubClient returns an HTTP client customized for accessing Github.
//
// The token is read from NETWORK_TEST_GITHUB_TOKEN (network tests) or, if that
// is empty, FLEET_VULNERABILITIES_GITHUB_TOKEN.
//
// - If no token is set, then this is equivalent to call `NewClient(WithNoTimeout())`.
// - If a token is set, then the client sends it as a bearer token, but only on
// HTTPS requests to the GitHub API host.
//
// Ambient variables such as GITHUB_TOKEN or GH_TOKEN are deliberately ignored so
// that a Fleet server never authenticates to GitHub unless explicitly configured to.
func NewGithubClient() *http.Client {
	cli := NewClient(WithNoTimeout())
	githubToken := os.Getenv("NETWORK_TEST_GITHUB_TOKEN")
	if githubToken == "" {
		// Internal only, not a supported Fleet server setting. The generate-cve
		// workflow in fleetdm/vulnerabilities sets it to the job's GITHUB_TOKEN:
		// that runner's IP is shared with other Actions jobs, so the anonymous
		// limit of 60 requests an hour can run out before the CVE generator
		// finds the latest release.
		githubToken = os.Getenv("FLEET_VULNERABILITIES_GITHUB_TOKEN")
	}
	if githubToken != "" {
		cli.Transport = &githubTokenTransport{token: githubToken, base: cli.Transport}
	}
	return cli
}

// githubTokenTransport authenticates HTTPS requests to the GitHub API only. The same
// client also downloads from configurable mirror URLs and follows redirects to
// asset hosts, none of which should see the token. Deciding per request (rather
// than setting the header once) also covers redirects, because the client
// re-sends each hop through the transport.
type githubTokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *githubTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != "https" || !strings.EqualFold(req.URL.Host, githubAPIHost) {
		return t.base.RoundTrip(req)
	}
	// A RoundTripper must not modify the caller's request.
	req = req.Clone(req.Context())
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(req)
}

// HostnamesMatch is an utility function to parse two strings as
// URLs and find if their hostnames match.
func HostnamesMatch(a, b string) (bool, error) {
	ap, err := url.Parse(a)
	if err != nil {
		return false, fmt.Errorf("parsing URL %s: %w", a, err)
	}

	bp, err := url.Parse(b)
	if err != nil {
		return false, fmt.Errorf("parsing URL %s: %w", b, err)
	}

	return ap.Hostname() == bp.Hostname(), nil
}

type SizeLimitTransport struct {
	maxSizeBytes int64
	base         http.RoundTripper
}

var ErrMaxSizeExceeded = errors.New("response body exceeds max size")

// NewSizeLimitTransport wraps the default base transport. Prefer
// NewClient(WithMaxResponseSize(n)), which keeps the whole client chain.
func NewSizeLimitTransport(maxSizeBytes int64) *SizeLimitTransport {
	return newSizeLimitTransport(defaultBaseTransport(), maxSizeBytes)
}

func newSizeLimitTransport(base http.RoundTripper, maxSizeBytes int64) *SizeLimitTransport {
	return &SizeLimitTransport{
		maxSizeBytes: maxSizeBytes,
		base:         base,
	}
}

func (t *SizeLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = defaultBaseTransport()
	}
	resp, err := base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if contentLen := resp.ContentLength; contentLen > t.maxSizeBytes {
		resp.Body.Close()
		return nil, ErrMaxSizeExceeded
	}

	// if no Content-Length header, limit reading the body
	if resp.ContentLength < 0 {
		resp.Body = http.MaxBytesReader(nil, resp.Body, t.maxSizeBytes)
	}

	return resp, nil
}
