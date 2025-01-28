// Pacakge proxy provides a reverse proxy for talking to Apple DEP APIs
// based on the standard Go reverse proxy.
package proxy

import (
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
)

// New creates new NanoDEP ReverseProxy. It dispatches requests using transport
// which should be a NanoDEP RoundTripper transport (which handles
// authentication and session management). DEP name configurations are retrieved
// using store and logger is used for logging.
func New(transport http.RoundTripper, store client.ConfigRetriever, logger log.Logger) *httputil.ReverseProxy {
	return &httputil.ReverseProxy{
		Transport:    transport,
		Director:     newDirector(store, logger.With("function", "director")),
		ErrorHandler: newErrorHandler(logger.With("msg", "proxy error")),
	}
}

// newErrorHandler creates a new function for ReverseProxy.ErrorHandler.
func newErrorHandler(logger log.Logger) func(http.ResponseWriter, *http.Request, error) {
	return func(rw http.ResponseWriter, req *http.Request, err error) {
		// use the same error as the standrad reverse proxy
		rw.WriteHeader(http.StatusBadGateway)

		logger := ctxlog.Logger(req.Context(), logger)

		var depErr *client.AuthError
		if errors.As(err, &depErr) {
			logger.Info(
				"err", "DEP auth error",
				"status", depErr.Status,
				"body", string(depErr.Body),
			)
			// write the same body content to try and give some clue of what
			// happened to the proxy user
			_, _ = rw.Write(depErr.Body)
			return
		}

		logger.Info("err", err)
	}
}

// newDirector creates a new httputil.ReverseProxy director which dynamically
// resolves the destination server based on the config. The config name is
// retrieved from the request context using client.GetName. We also implement
// a parsed URL cache (which means the proxy may not be aware of underlying
// config changes).
func newDirector(store client.ConfigRetriever, logger log.Logger) func(*http.Request) {
	urlCache := make(map[string]*url.URL)
	urlCacheMu := sync.RWMutex{}
	store = client.NewDefaultConfigRetreiver(store)
	return func(req *http.Request) {
		name := client.GetName(req.Context())
		if name == "" {
			ctxlog.Logger(req.Context(), logger).Info("err", "missing name")
			// this will probably lead to a very broken proxy.
			// but we can't really do anything about it here.
			return
		}

		// attempt to read the URL from urlCache, or retreive it from store
		urlCacheMu.RLock()
		url := urlCache[name]
		urlCacheMu.RUnlock()
		if url == nil {
			logger := ctxlog.Logger(req.Context(), logger).With("name", name)
			config, err := store.RetrieveConfig(req.Context(), name)
			if err != nil {
				logger.Info("msg", "retrieve config", "err", err)
			}
			url, err = url.Parse(config.BaseURL)
			if err != nil {
				logger.Info("msg", "parse", "err", err)
				// this will probably lead to a very broken proxy.
				// but we can't really do anything about it here.
				return
			}
			urlCacheMu.Lock()
			urlCache[name] = url
			urlCacheMu.Unlock()
		}

		// perform our actual request modifications (i.e. swapping in the
		// correct DEP URL components based on the context)
		req.URL.Scheme = url.Scheme
		req.URL.Host = url.Host
		req.Host = url.Host
	}
}

// newCopiedRequest makes a copy of r with a new copy of r.URL and returns it.
func newCopiedRequest(r *http.Request) *http.Request {
	r2 := new(http.Request)
	*r2 = *r
	r2.URL = new(url.URL)
	*r2.URL = *r.URL
	return r2
}

// ProxyDEPNameHandler tries to extract the DEP name from the request URL path
// and replaces it with the just the endpoint and embeds the name as a context
// value.
//
// For example if the request URL path is "hello/world/" then "hello" is the
// DEP name and is set in the request context and "/world/" is then set in the
// HTTP request passed onto p.
//
// Note the very beginning of the URL path is used as the DEP name. This
// necessitates stripping the URL prefix before using this handler. Note also
// that DEP names with a "/" or "%2F" are likely to cause issues as we naively
// search and cut by "/" in the path.
func ProxyDEPNameHandler(p *httputil.ReverseProxy, logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r2 := newCopiedRequest(r)

		name, endpoint, found := CutIncl(r.URL.Path, "/")
		if found {
			r2.URL.Path = endpoint
		}

		logger := ctxlog.Logger(r.Context(), logger)

		if name == "" {
			logger.Info("msg", "extracting DEP name", "err", "name not found in path")
			http.NotFound(w, r)
			return
		}

		// try to perform the same extraction on the RawPath as we did for Path
		if r.URL.RawPath != "" {
			if _, endpoint, found = CutIncl(r.URL.RawPath, "/"); found {
				r2.URL.RawPath = endpoint
			}
		}

		logger.Debug("msg", "proxy serve", "name", name, "endpoint", endpoint)

		p.ServeHTTP(w, r2.WithContext(client.WithName(r2.Context(), name)))
	}
}

// CutIncl is like strings.Cut but keeps sep in after.
func CutIncl(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i:], true
	}
	return s, "", false
}
