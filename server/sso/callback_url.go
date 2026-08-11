package sso

import (
	"net/url"
	"strings"
)

// CallbackURL builds a SAML ACS callback URL by appending callbackPath to base
// (e.g. the parsed server_url). When urlPrefix is configured, it is inserted
// before callbackPath only if base's path does not already include it, so the
// configured subpath appears exactly once whether or not the base URL was
// configured with the prefix. This keeps existing deployments working regardless
// of which convention they used for server_url.
//
// base is not mutated; a new URL is returned.
func CallbackURL(base *url.URL, urlPrefix, callbackPath string) *url.URL {
	prefix := strings.TrimSuffix(urlPrefix, "/")
	// JoinPath returns a new URL rather than mutating the receiver, so base is left
	// untouched and callers can still use it (e.g. as the expected SAML audience).
	result := base
	if prefix != "" && !strings.HasSuffix(strings.TrimSuffix(base.Path, "/"), prefix) {
		result = result.JoinPath(prefix)
	}
	return result.JoinPath(callbackPath)
}
