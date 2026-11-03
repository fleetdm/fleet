package fleet

import (
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGoogleWorkspaceIntegrations(t *testing.T) {
	validKey := func() GoogleCalendarApiKey {
		return GoogleCalendarApiKey{Values: map[string]string{
			GoogleCalendarEmail:      "svc@project.iam.gserviceaccount.com",
			GoogleCalendarPrivateKey: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
		}}
	}

	cases := []struct {
		name      string
		intgs     []*GoogleWorkspaceIntegration
		wantField string // empty means no error expected
	}{
		{
			name: "valid",
			intgs: []*GoogleWorkspaceIntegration{{
				Domain:                "example.com",
				ImpersonatedUserEmail: "admin@example.com",
				ApiKey:                validKey(),
			}},
		},
		{
			name:  "empty list is valid",
			intgs: nil,
		},
		{
			name: "more than one integration",
			intgs: []*GoogleWorkspaceIntegration{
				{Domain: "a.com", ImpersonatedUserEmail: "admin@a.com", ApiKey: validKey()},
				{Domain: "b.com", ImpersonatedUserEmail: "admin@b.com", ApiKey: validKey()},
			},
			wantField: "integrations.google_workspace",
		},
		{
			name: "missing client_email",
			intgs: []*GoogleWorkspaceIntegration{{
				Domain:                "example.com",
				ImpersonatedUserEmail: "admin@example.com",
				ApiKey:                GoogleCalendarApiKey{Values: map[string]string{GoogleCalendarPrivateKey: "key"}},
			}},
			wantField: "integrations.google_workspace.api_key_json.client_email",
		},
		{
			name: "missing private_key",
			intgs: []*GoogleWorkspaceIntegration{{
				Domain:                "example.com",
				ImpersonatedUserEmail: "admin@example.com",
				ApiKey:                GoogleCalendarApiKey{Values: map[string]string{GoogleCalendarEmail: "svc@x.com"}},
			}},
			wantField: "integrations.google_workspace.api_key_json.private_key",
		},
		{
			name: "blank domain",
			intgs: []*GoogleWorkspaceIntegration{{
				Domain:                "   ",
				ImpersonatedUserEmail: "admin@example.com",
				ApiKey:                validKey(),
			}},
			wantField: "integrations.google_workspace.domain",
		},
		{
			name: "missing impersonated_user_email",
			intgs: []*GoogleWorkspaceIntegration{{
				Domain: "example.com",
				ApiKey: validKey(),
			}},
			wantField: "integrations.google_workspace.impersonated_user_email",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			invalid := &InvalidArgumentError{}
			ValidateGoogleWorkspaceIntegrations(c.intgs, invalid)
			if c.wantField == "" {
				assert.False(t, invalid.HasErrors(), "expected no validation errors, got: %v", invalid)
				return
			}
			require.True(t, invalid.HasErrors(), "expected a validation error for field %q", c.wantField)
			var found bool
			for _, e := range invalid.Errors {
				if e.name == c.wantField {
					found = true
					break
				}
			}
			assert.True(t, found, "expected error on field %q, got %v", c.wantField, invalid.Errors)
		})
	}
}

func TestGoogleWorkspaceObfuscateAndClone(t *testing.T) {
	ac := &AppConfig{}
	ac.Integrations.GoogleWorkspace = []*GoogleWorkspaceIntegration{{
		Domain:                "example.com",
		ImpersonatedUserEmail: "admin@example.com",
		ApiKey: GoogleCalendarApiKey{Values: map[string]string{
			GoogleCalendarEmail:      "svc@x.com",
			GoogleCalendarPrivateKey: "secret",
		}},
	}}

	// Clone must deep-copy the ApiKey values (mutating the clone must not affect the original).
	cloned, err := ac.Clone()
	require.NoError(t, err)
	clonedAC := cloned.(*AppConfig)
	require.Len(t, clonedAC.Integrations.GoogleWorkspace, 1)
	clonedAC.Integrations.GoogleWorkspace[0].ApiKey.Values[GoogleCalendarPrivateKey] = "mutated"
	assert.Equal(t, "secret", ac.Integrations.GoogleWorkspace[0].ApiKey.Values[GoogleCalendarPrivateKey])

	// Obfuscate masks the service account key.
	ac.Obfuscate()
	b, err := ac.Integrations.GoogleWorkspace[0].ApiKey.MarshalJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `"`+MaskedPassword+`"`, string(b))
}

func TestIntegrationsIsGoogleWorkspaceConfigured(t *testing.T) {
	apiKey := GoogleCalendarApiKey{Values: map[string]string{
		GoogleCalendarEmail:      "svc@project.iam.gserviceaccount.com",
		GoogleCalendarPrivateKey: "-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n",
	}}

	cases := []struct {
		name     string
		intg     Integrations
		expected bool
	}{
		{
			name:     "no integration",
			intg:     Integrations{},
			expected: false,
		},
		{
			name: "missing domain",
			intg: Integrations{GoogleWorkspace: []*GoogleWorkspaceIntegration{
				{ImpersonatedUserEmail: "admin@example.com", ApiKey: apiKey},
			}},
			expected: false,
		},
		{
			name: "empty api key",
			intg: Integrations{GoogleWorkspace: []*GoogleWorkspaceIntegration{
				{Domain: "example.com", ImpersonatedUserEmail: "admin@example.com"},
			}},
			expected: false,
		},
		{
			name: "fully configured",
			intg: Integrations{GoogleWorkspace: []*GoogleWorkspaceIntegration{
				{Domain: "example.com", ImpersonatedUserEmail: "admin@example.com", ApiKey: apiKey},
			}},
			expected: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.expected, c.intg.IsGoogleWorkspaceConfigured())
		})
	}
}

func TestCheckCertIdPIntrospection(t *testing.T) {
	const listedURL = "https://company.okta.com/oauth2/v1/introspect"
	ptr := func(s string) *string { return &s }

	urlsOnly := Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{listedURL})}
	both := Integrations{
		CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{listedURL}),
		CertificatesIdPClientIDs:         optjson.SetSlice([]string{"listed-client"}),
	}

	// Closed by default: nothing is allowlisted, so no endpoint may be contacted, but credentials
	// are not demanded either.
	t.Run("unconfigured refuses supplied credentials and allows none", func(t *testing.T) {
		require.NoError(t, Integrations{}.CheckCertIdPIntrospection(nil, nil))
		err := Integrations{}.CheckCertIdPIntrospection(ptr(listedURL), ptr("any-client"))
		require.ErrorContains(t, err, "IdP introspection endpoint is not permitted")
		var pe *PermissionError
		require.ErrorAs(t, err, &pe)
	})

	// Configuring URLs makes credentials mandatory, not merely constrained.
	t.Run("configured urls make credentials mandatory", func(t *testing.T) {
		for _, intgs := range []Integrations{urlsOnly, both} {
			err := intgs.CheckCertIdPIntrospection(nil, nil)
			require.ErrorContains(t, err, "IdP verification is required")
			// A missing credential is the caller's mistake, not a forbidden one.
			var bre *BadRequestError
			require.ErrorAs(t, err, &bre)
		}
	})

	// Client IDs alone never arm the feature; validation rejects that configuration anyway.
	t.Run("client ids alone permit nothing and demand nothing", func(t *testing.T) {
		clientsOnly := Integrations{CertificatesIdPClientIDs: optjson.SetSlice([]string{"listed-client"})}
		require.NoError(t, clientsOnly.CheckCertIdPIntrospection(nil, nil))
		require.ErrorContains(t, clientsOnly.CheckCertIdPIntrospection(ptr(listedURL), ptr("listed-client")), "endpoint is not permitted")
	})

	t.Run("a listed pair is allowed", func(t *testing.T) {
		require.NoError(t, both.CheckCertIdPIntrospection(ptr(listedURL), ptr("listed-client")))
	})

	// The two are reported separately so an admin can tell which list to fix.
	t.Run("an unlisted url is forbidden", func(t *testing.T) {
		err := both.CheckCertIdPIntrospection(ptr("https://evil.example.com/introspect"), ptr("listed-client"))
		require.ErrorContains(t, err, "IdP introspection endpoint is not permitted")
	})

	t.Run("an unlisted client id is forbidden", func(t *testing.T) {
		err := both.CheckCertIdPIntrospection(ptr(listedURL), ptr("attacker-client"))
		require.ErrorContains(t, err, "IdP client ID is not permitted")
	})

	// URLs are stored as given and matched exactly, so a variant spelling is a different endpoint.
	t.Run("url matching is exact", func(t *testing.T) {
		require.Error(t, both.CheckCertIdPIntrospection(ptr(listedURL+"/"), ptr("listed-client")))
		require.Error(t, both.CheckCertIdPIntrospection(ptr(strings.ToUpper(listedURL)), ptr("listed-client")))
	})

	// The client ID list is an optional extra constraint; with only URLs, any client ID passes.
	t.Run("client ids are optional", func(t *testing.T) {
		require.NoError(t, urlsOnly.CheckCertIdPIntrospection(ptr(listedURL), ptr("any-client")))
		require.Error(t, urlsOnly.CheckCertIdPIntrospection(ptr("https://evil.example.com/x"), ptr("any-client")))
	})
}

func TestValidateCertIdPIntrospectionAllowlists(t *testing.T) {
	t.Run("trims in place and stores URLs as given", func(t *testing.T) {
		intgs := Integrations{
			CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"  https://Company.Okta.com:443/oauth2/v1/introspect/?x=1  "}),
			CertificatesIdPClientIDs:         optjson.SetSlice([]string{"  client  "}),
		}
		invalid := &InvalidArgumentError{}
		ValidateCertIdPIntrospectionAllowlists(&intgs, invalid)
		require.False(t, invalid.HasErrors())
		require.Equal(t, []string{"https://Company.Okta.com:443/oauth2/v1/introspect/?x=1"}, intgs.CertificatesIdPIntrospectionURLs.Value)
		require.Equal(t, []string{"client"}, intgs.CertificatesIdPClientIDs.Value)
	})

	t.Run("nothing configured is valid", func(t *testing.T) {
		invalid := &InvalidArgumentError{}
		ValidateCertIdPIntrospectionAllowlists(&Integrations{}, invalid)
		require.False(t, invalid.HasErrors())
	})

	for name, tc := range map[string]struct {
		intgs Integrations
		msg   string
	}{
		"empty url": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{" "})},
			"url cannot be empty",
		},
		"relative url": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"/oauth2/v1/introspect"})},
			"must be an absolute https URL",
		},
		"unparseable url": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"not a url"})},
			"must be an absolute https URL",
		},
		"non-https url": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"http://company.okta.com/introspect"})},
			"must be an absolute https URL",
		},
		"url with embedded credentials": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"https://user:secret@company.okta.com/introspect"})},
			"must be an absolute https URL",
		},
		"duplicate url": {
			Integrations{CertificatesIdPIntrospectionURLs: optjson.SetSlice([]string{"https://company.okta.com/introspect", " https://company.okta.com/introspect"})},
			"duplicate url",
		},
		"empty client id": {
			Integrations{CertificatesIdPClientIDs: optjson.SetSlice([]string{" "})},
			"client ID cannot be empty",
		},
		"duplicate client id": {
			Integrations{CertificatesIdPClientIDs: optjson.SetSlice([]string{"client", "client "})},
			"duplicate client ID",
		},
		"client ids without urls": {
			Integrations{CertificatesIdPClientIDs: optjson.SetSlice([]string{"client"})},
			"requires integrations.certificates_idp_introspection_urls",
		},
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			invalid := &InvalidArgumentError{}
			ValidateCertIdPIntrospectionAllowlists(&tc.intgs, invalid)
			require.True(t, invalid.HasErrors())
			require.Contains(t, invalid.Error(), tc.msg)
		})
	}
}
