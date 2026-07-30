package fleet

import (
	"testing"

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
