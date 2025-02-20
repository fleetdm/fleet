// Package fleethttp provides uniform creation and configuration of HTTP
// related types used throughout Fleet.
package fleethttp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/oauth2"
)

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
	if co.tlsConf != nil {
		cli.Transport = NewTransport(WithTLSConfig(co.tlsConf))
	}
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

	// make sure to start from DefaultTransport to inherit its sane defaults
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if to.tlsConf != nil {
		tr.TLSClientConfig = to.tlsConf
	}
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
		return oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: githubToken,
			},
		))
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
