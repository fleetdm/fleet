// Package token enables setting and reading
// authentication token contexts
package token

import (
	"context"
	"net/http"
	"strings"
)

type key int

const tokenKey key = 0

// FromHTTPRequest extracts an Authorization
// from an HTTP header if present.
func FromHTTPRequest(r *http.Request) string {
	headers := r.Header.Get("Authorization")
	headerParts := strings.Split(headers, " ")
	if len(headerParts) != 2 || strings.ToUpper(headerParts[0]) != "BEARER" {
		return ""
	}
	return headerParts[1]
}

// NewContext returns a new context carrying the Authorization Bearer token.
func NewContext(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenKey, token)
}

// FromContext extracts the Authorization Bearer token if present.
func FromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenKey).(string)
	return token, ok
}
