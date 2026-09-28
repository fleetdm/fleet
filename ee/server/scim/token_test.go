package scim

import "testing"

func TestIdentityProviderNameFromPath(t *testing.T) {
	cases := map[string]string{
		"/api/latest/fleet/identity_providers/Entra/scim_token": "Entra",
		"/api/v1/fleet/identity_providers/Okta/scim_token/":     "Okta",
		"/api/latest/fleet/scim/Users":                          "",
	}
	for path, want := range cases {
		if got := identityProviderNameFromPath(path); got != want {
			t.Errorf("identityProviderNameFromPath(%q) = %q, want %q", path, got, want)
		}
	}
}
