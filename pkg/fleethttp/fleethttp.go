// Package fleethttp provides uniform creation and configuration of HTTP
// related types used throughout Fleet.
package fleethttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/oauth2"
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
	// BlockingBypassAll performs no filtering at all. Used in dev mode
	// where integrations are tested against localhost.
	BlockingBypassAll
)

// networkBlockingMode holds the current blocking mode. Default is
// BlockingDisabled so tests, CLI tools, and non-serve callers are unaffected.
var networkBlockingMode atomic.Int32

// SetNetworkBlockingMode sets the blocking mode. Called by fleet serve at startup.
func SetNetworkBlockingMode(mode NetworkBlockingMode) {
	networkBlockingMode.Store(int32(mode))
}

// ErrPrivateNetworkBlocked is returned when a connection to a private network
// address is blocked.
var ErrPrivateNetworkBlocked = errors.New("connections to private network addresses are blocked")

// alwaysBlockedCIDRs are blocked unconditionally, even when
// --allow_private_network_integrations is set. No legitimate integration
// should ever target these addresses.
var alwaysBlockedCIDRs = parseCIDRs([]string{
	"127.0.0.0/8",    // loopback
	"169.254.0.0/16", // link-local (includes cloud IMDS at 169.254.169.254)
	"::1/128",        // IPv6 loopback
	"fe80::/10",      // IPv6 link-local
})

// privateNetworkCIDRs are blocked when private network blocking is enabled.
// Customers with on-prem integrations (e.g. EJBCA, Jira, SCEP servers on
// private networks) can disable this with --allow_private_network_integrations.
var privateNetworkCIDRs = parseCIDRs([]string{
	"0.0.0.0/8",       // "this" network (RFC 1122)
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

		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}

		for _, ip := range ips {
			// Tier 1: always blocked (loopback, cloud IMDS). Cannot be
			// overridden with --server_allow_private_network_integrations.
			if ipInCIDRs(ip.IP, alwaysBlockedCIDRs) {
				return nil, fmt.Errorf("%w: %s resolves to %s", ErrPrivateNetworkBlocked, host, ip.IP)
			}
			// Tier 2: private networks. Only blocked in BlockingFull mode.
			if mode == BlockingFull && ipInCIDRs(ip.IP, privateNetworkCIDRs) {
				return nil, fmt.Errorf("%w: %s resolves to %s", ErrPrivateNetworkBlocked, host, ip.IP)
			}
		}

		// Connect using the already-resolved IP to prevent DNS rebinding
		// (a second DNS lookup could return a different, malicious IP).
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
}

type clientOpts struct {
	timeout   time.Duration
	tlsConf   *tls.Config
	noFollow  bool
	cookieJar http.CookieJar
}

// ClientOpt is the type for the client-specific options.
type ClientOpt func(o *clientOpts)

// WithTimeout sets the timeout to use for the HTTP client.
func WithTimeout(t time.Duration) ClientOpt {
	return func(o *clientOpts) {
		o.timeout = t
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

// NewClient returns an HTTP client configured according to the provided
// options.
func NewClient(opts ...ClientOpt) *http.Client {
	var co clientOpts
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
	} else if _, ok := http.DefaultTransport.(*http.Transport); ok {
		baseTransport = NewTransport()
	} else {
		// http.DefaultTransport is not a *http.Transport (e.g. test mock).
		// Use it directly to preserve the mock chain.
		baseTransport = http.DefaultTransport
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

// NewGithubClient returns an HTTP client customized for accessing Github.
//
// - If the NETWORK_TEST_GITHUB_TOKEN variable is empty, then this is equivalent to
// call `NewClient()`.
// - If the NETWORK_TEST_GITHUB_TOKEN variable is set, then the client will use the
// token for authentication (as OAuth2 static token).
func NewGithubClient() *http.Client {
	if githubToken := os.Getenv("NETWORK_TEST_GITHUB_TOKEN"); githubToken != "" {
		cli := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: githubToken,
			},
		))
		cli.Transport = otelhttp.NewTransport(cli.Transport)
		return cli
	}
	return NewClient()
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
}

var ErrMaxSizeExceeded = errors.New("response body exceeds max size")

func NewSizeLimitTransport(maxSizeBytes int64) *SizeLimitTransport {
	return &SizeLimitTransport{
		maxSizeBytes: maxSizeBytes,
	}
}

func (t *SizeLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
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
