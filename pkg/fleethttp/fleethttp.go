// Package fleethttp provides uniform creation and configuration of HTTP
// related types.
package fleethttp

import (
	"crypto/tls"
	"net/http"
	"time"
)

type clientOpts struct {
	timeout  time.Duration
	tlsConf  *tls.Config
	noFollow bool
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
// parts it needs to, so that it keeps its sane defaults for the rest.
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
