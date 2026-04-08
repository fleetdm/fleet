package main

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// bearerAuthMiddleware rejects requests whose Authorization header does not
// match "Bearer <token>", returning 401 Unauthorized. The comparison uses
// crypto/subtle.ConstantTimeCompare to prevent timing side-channel attacks.
func bearerAuthMiddleware(token string, next http.Handler) http.Handler {
	expected := []byte("Bearer " + token)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare(got, expected) != 1 {
			ip := clientIP(r)
			logrus.WithField("ip", ip).Warn("authentication failed: invalid bearer token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the client IP address from the request, preferring
// X-Forwarded-For or X-Real-IP when present (for deployments behind a reverse
// proxy), and stripping the port from RemoteAddr as a fallback.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For may contain a comma-separated chain; the first
		// entry is the original client.
		first, _, _ := strings.Cut(xff, ",")
		if ip := strings.TrimSpace(first); ip != "" {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
