// Package http includes handlers and utilties
package http

import (
	"bytes"
	"context"
	"crypto/subtle"
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/log/ctxlog"
)

// ReadAllAndReplaceBody reads all of r.Body and replaces it with a new byte buffer.
func ReadAllAndReplaceBody(r *http.Request) ([]byte, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return b, err
	}
	defer r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(b))
	return b, nil
}

// BasicAuthMiddleware is a simple HTTP plain authentication middleware.
func BasicAuthMiddleware(next http.Handler, username, password, realm string) http.HandlerFunc {
	uBytes := []byte(username)
	pBytes := []byte(password)
	return func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), uBytes) != 1 || subtle.ConstantTimeCompare([]byte(p), pBytes) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// VersionHandler returns a simple JSON response from a version string.
func VersionHandler(version string) http.HandlerFunc {
	bodyBytes := []byte(`{"version":"` + version + `"}`)
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(bodyBytes)
	}
}

type ctxKeyTraceID struct{}

// TraceLoggingMiddleware sets up a trace ID in the request context and
// logs HTTP requests.
func TraceLoggingMiddleware(next http.Handler, logger log.Logger, newTrace func() string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), ctxKeyTraceID{}, newTrace())
		ctx = ctxlog.AddFunc(ctx, ctxlog.SimpleStringFunc("trace_id", ctxKeyTraceID{}))

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}
		logs := []interface{}{
			"addr", host,
			"method", r.Method,
			"path", r.URL.Path,
			"agent", r.UserAgent(),
		}

		if fwdedFor := r.Header.Get("X-Forwarded-For"); fwdedFor != "" {
			logs = append(logs, "x_forwarded_for", fwdedFor)
		}

		ctxlog.Logger(ctx, logger).Info(logs...)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
