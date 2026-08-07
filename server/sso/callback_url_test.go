package sso

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCallbackURL(t *testing.T) {
	const callbackPath = "/api/v1/fleet/sso/callback"

	testCases := []struct {
		name      string
		baseURL   string
		urlPrefix string
		want      string
	}{
		{
			name:      "no prefix configured",
			baseURL:   "https://fleet.example.com",
			urlPrefix: "",
			want:      "https://fleet.example.com/api/v1/fleet/sso/callback",
		},
		{
			name:      "root prefix is treated as no prefix",
			baseURL:   "https://fleet.example.com",
			urlPrefix: "/",
			want:      "https://fleet.example.com/api/v1/fleet/sso/callback",
		},
		{
			name:      "prefix set and base url already includes it",
			baseURL:   "https://fleet.example.com/apps/fleet",
			urlPrefix: "/apps/fleet",
			want:      "https://fleet.example.com/apps/fleet/api/v1/fleet/sso/callback",
		},
		{
			name:      "prefix set and base url omits it",
			baseURL:   "https://fleet.example.com",
			urlPrefix: "/apps/fleet",
			want:      "https://fleet.example.com/apps/fleet/api/v1/fleet/sso/callback",
		},
		{
			name:      "base url includes prefix with trailing slash",
			baseURL:   "https://fleet.example.com/apps/fleet/",
			urlPrefix: "/apps/fleet",
			want:      "https://fleet.example.com/apps/fleet/api/v1/fleet/sso/callback",
		},
		{
			name:      "prefix configured with trailing slash",
			baseURL:   "https://fleet.example.com/apps/fleet",
			urlPrefix: "/apps/fleet/",
			want:      "https://fleet.example.com/apps/fleet/api/v1/fleet/sso/callback",
		},
		{
			name:      "proxy mounts fleet under an additional outer segment",
			baseURL:   "https://fleet.example.com/gateway/apps/fleet",
			urlPrefix: "/apps/fleet",
			want:      "https://fleet.example.com/gateway/apps/fleet/api/v1/fleet/sso/callback",
		},
		{
			name:      "outer segment that is not a full path segment still gets the prefix",
			baseURL:   "https://fleet.example.com/myapps/fleet",
			urlPrefix: "/apps/fleet",
			want:      "https://fleet.example.com/myapps/fleet/apps/fleet/api/v1/fleet/sso/callback",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			base, err := url.Parse(tc.baseURL)
			require.NoError(t, err)
			got := CallbackURL(base, tc.urlPrefix, callbackPath)
			require.Equal(t, tc.want, got.String())
			// The base URL must not be mutated, so callers can still use it for
			// other purposes (e.g. the expected SAML audience).
			require.Equal(t, tc.baseURL, base.String())
		})
	}
}
