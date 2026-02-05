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
	headers := r.Header.Get("Authorization")
	headerParts, ok := splitToken(headers)
	if !ok {
		return ""
	}
	// If the Authorization header is present and properly formatted, return the token.
	if strings.ToUpper(headerParts[0]) == "BEARER" {
		if headerParts[1] == "" {
			// Empty "BEARER" value in header, return empty token
			return ""
		}
		return Token(headerParts[1])
	}
	// If the Authorization header is not present, try to extract the token from the form data.
	if err := r.ParseForm(); err != nil {
		return ""
	}
	return Token(r.FormValue("token"))
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

func splitToken(token string) ([]string, bool) {
	parts := make([]string, 2)
	tokenType, remain, found := strings.Cut(token, tokenDelimiter)
	if !found {
		return nil, false
	}
	parts[0] = tokenType
	// Ensure the token value is the last part of the string and there are no more
	// delimiters. This avoids an issue where malicious input could contain additional delimiters
	// causing unecessary overhead parsing tokens.
	tokenVal, _, unexpectedDelimeterFound := strings.Cut(remain, tokenDelimiter)
	if unexpectedDelimeterFound {
		return nil, false
	}
	parts[1] = tokenVal
	return parts, true
}
