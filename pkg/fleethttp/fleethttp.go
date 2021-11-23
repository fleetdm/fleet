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

type ClientOpt func(o *clientOpts)

func WithTimeout(t time.Duration) ClientOpt {
	return func(o *clientOpts) {
		o.timeout = t
	}
}

func WithTLSConfig(conf *tls.Config) ClientOpt {
	return func(o *clientOpts) {
		o.tlsConf = conf.Clone()
	}
}

func WithFollowRedir(follow bool) ClientOpt {
	return func(o *clientOpts) {
		o.noFollow = !follow
	}
}

func NewClient(opts ...ClientOpt) *http.Client {
	var co clientOpts
	for _, opt := range opts {
		opt(&co)
	}

	cli := &http.Client{
		Timeout: co.timeout,
	}
	if co.noFollow {
		cli.CheckRedirect = noFollowRedirect
	}
	if co.tlsConf != nil {
		// make sure to start from DefaultTransport to inherit its sane defaults
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = co.tlsConf
		cli.Transport = tr
	}
	return cli
}

func noFollowRedirect(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}
