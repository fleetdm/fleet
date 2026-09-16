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
	return URLWithPrefix(base, urlPrefix, callbackPath)
}

// URLWithPrefix joins path onto base, inserting urlPrefix first when base does
// not already carry it. server_url may or may not include the configured
// url_prefix, so anything building an absolute Fleet URL has to go through here
// or it emits the prefix zero times or twice.
func URLWithPrefix(base *url.URL, urlPrefix, path string) *url.URL {
	prefix := strings.TrimSuffix(urlPrefix, "/")
	// JoinPath returns a new URL rather than mutating the receiver, so base is left
	// untouched and callers can still use it (e.g. as the expected SAML audience).
	result := base
	if prefix != "" && !strings.HasSuffix(strings.TrimSuffix(base.Path, "/"), prefix) {
		result = result.JoinPath(prefix)
	}
	return result.JoinPath(path)
}
