// Package authproxy is a simple reverse proxy for Apple MDM clients.
package authproxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/micromdm/nanolib/log"
	"github.com/micromdm/nanolib/log/ctxlog"
)

// HeaderFunc takes an HTTP request and returns a string value.
// Ostensibly to be set in a header on the proxy target.
type HeaderFunc func(context.Context) string
type config struct {
	logger      log.Logger
	fwdSig      bool
	headerFuncs map[string]HeaderFunc
}
type Option func(*config)

// WithLogger sets a logger for error reporting.
func WithLogger(logger log.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithHeaderFunc configures fn to be called and added as an HTTP header to the proxy target request.
func WithHeaderFunc(header string, fn HeaderFunc) Option {
	return func(c *config) {
		c.headerFuncs[header] = fn
	}
}

// WithForwardMDMSignature forwards the MDM-Signature header onto the proxy destination.
// This option is off by default because the header adds about two kilobytes to the request.
func WithForwardMDMSignature() Option {
	return func(c *config) {
		c.fwdSig = true
	}
}

// New creates a new NanoMDM enrollment authenticating reverse proxy.
// This reverse proxy is mostly the standard httputil proxy. It depends
// on middleware HTTP handlers to enforce authentication and set the
// context value for the enrollment ID.
func New(dest string, opts ...Option) (*httputil.ReverseProxy, error) {
	config := &config{
		logger:      log.NopLogger,
		headerFuncs: make(map[string]HeaderFunc),
	}
	for _, opt := range opts {
		opt(config)
	}
	target, err := url.Parse(dest)
	if err != nil {
		return nil, err
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		ctxlog.Logger(r.Context(), config.logger).Info("err", err)
		// use the same error as the standrad reverse proxy
		w.WriteHeader(http.StatusBadGateway)
	}
	dir := proxy.Director
	proxy.Director = func(req *http.Request) {
		dir(req)
		req.Host = target.Host
		if !config.fwdSig {
			// save the effort of forwarding this huge header
			req.Header.Del("Mdm-Signature")
		}
		// set any headers we want to forward.
		for k, fn := range config.headerFuncs {
			if k == "" || fn == nil {
				continue
			}
			if v := fn(req.Context()); v != "" {
				req.Header.Set(k, v)
			}
		}
	}
	return proxy, nil
}
