// Package token enables setting and reading
// authentication token contexts
package token

import (
	"context"
	"net/http"
	"strings"
)

type key int

const (
	tokenKey       key = 0
	tokenDelimiter     = " "
)

// Token is the concrete type that represents Fleet session tokens
type Token string

// FromHTTPRequest extracts an Authorization
// from an HTTP request if present.
func FromHTTPRequest(r *http.Request) Token {
	authHeader := r.Header.Get("Authorization")
	headerCouple, ok := parseHeaderLimited(authHeader)
	if ok && strings.ToUpper(headerCouple[0]) == "BEARER" {
		// If the Authorization header is present and properly formatted, return the token.
		// Preserve case-insensitivity for "Bearer" prefix while case-sensitivity for the token value
		return Token(headerCouple[1])
	}
	return ""
}

// NewContext returns a new context carrying the Authorization Bearer token.
func NewContext(ctx context.Context, token Token) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, tokenKey, token)
}

// FromContext extracts the Authorization Bearer token if present.
func FromContext(ctx context.Context) (Token, bool) {
	token, ok := ctx.Value(tokenKey).(Token)
	return token, ok
}

// parseHeaderLimited splits an authHeader into exactly two parts or nil, and returns an ok boolean indicating whether the passed in authHeader did have exactly 2 parts. If it did not, nil and false are returned.
func parseHeaderLimited(authHeader string) ([]string, bool) {
	parts := make([]string, 2)

	pre, remain, foundDelimeter := strings.Cut(authHeader, tokenDelimiter)
	if !foundDelimeter {
		return nil, false
	}
	parts[0] = pre
	// Ensure the token value is the last part of the string and there are no more
	// delimiters. This avoids an issue where malicious input could contain additional delimiters
	// causing unecessary overhead parsing tokens.
	post, _, unexpectedDelimeterFound := strings.Cut(remain, tokenDelimiter)
	if unexpectedDelimeterFound {
		// more than 2 parts
		return nil, false
	}
	parts[1] = post
	return parts, true
}
