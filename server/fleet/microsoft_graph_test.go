package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMicrosoftGraphCredentialConfigured(t *testing.T) {
	for _, tc := range []struct {
		name string
		cred MicrosoftGraphCredential
		want bool
	}{
		{"all set", MicrosoftGraphCredential{TenantID: "t", ClientID: "c", ClientSecret: "s"}, true},
		{"missing tenant", MicrosoftGraphCredential{ClientID: "c", ClientSecret: "s"}, false},
		{"missing client", MicrosoftGraphCredential{TenantID: "t", ClientSecret: "s"}, false},
		{"missing secret", MicrosoftGraphCredential{TenantID: "t", ClientID: "c"}, false},
		{"empty", MicrosoftGraphCredential{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cred.Configured())
		})
	}
}

func TestMicrosoftGraphCredentialEqual(t *testing.T) {
	base := MicrosoftGraphCredential{TenantID: "tenant-a", ClientID: "client-a", ClientSecret: "secret"}

	for _, tc := range []struct {
		name  string
		other MicrosoftGraphCredential
		want  bool
	}{
		{"identical", base, true},
		// Entra emits tenant and client IDs lower-cased but admins paste them either way, so identity is
		// case-insensitive. The secret is not.
		{"ids differ only by case", MicrosoftGraphCredential{TenantID: "TENANT-A", ClientID: "CLIENT-A", ClientSecret: "secret"}, true},
		{"different secret", MicrosoftGraphCredential{TenantID: "tenant-a", ClientID: "client-a", ClientSecret: "other"}, false},
		{"different tenant", MicrosoftGraphCredential{TenantID: "tenant-b", ClientID: "client-a", ClientSecret: "secret"}, false},
		{"different client", MicrosoftGraphCredential{TenantID: "tenant-a", ClientID: "client-b", ClientSecret: "secret"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, base.Equal(tc.other))
		})
	}
}

// GitOps hands the credentials over as an untyped value decoded from YAML, and an absent key has to mean "clear them"
// rather than "leave them alone" -- GitOps is declarative, so the nil case is the one that matters most here.
func TestParseMicrosoftGraphCredentials(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     any
		want    []MicrosoftGraphCredential
		wantErr bool
	}{
		{
			name: "absent key clears credentials",
			raw:  nil,
			want: []MicrosoftGraphCredential{},
		},
		{
			name: "explicit empty list clears credentials",
			raw:  []any{},
			want: []MicrosoftGraphCredential{},
		},
		{
			name: "one credential",
			raw: []any{map[string]any{
				"tenant_id": "tenant-a", "client_id": "client-a", "client_secret": "secret-a",
			}},
			want: []MicrosoftGraphCredential{{TenantID: "tenant-a", ClientID: "client-a", ClientSecret: "secret-a"}},
		},
		{
			name: "server-computed status in the payload is accepted and ignored on the way in",
			raw: []any{map[string]any{
				"tenant_id": "tenant-a", "client_id": "client-a", "client_secret": "secret-a",
				"credential_invalid": true,
			}},
			// It decodes onto the struct, but nothing downstream reads it: the datastore writes only the three input
			// columns. Round-tripping a generated file must not fail.
			want: []MicrosoftGraphCredential{{
				TenantID: "tenant-a", ClientID: "client-a", ClientSecret: "secret-a", CredentialInvalid: true,
			}},
		},
		{
			name:    "wrong shape is rejected rather than silently dropped",
			raw:     map[string]any{"tenant_id": "tenant-a"},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMicrosoftGraphCredentials(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got, "a nil slice would be indistinguishable from \"not provided\" downstream")
			assert.Equal(t, tc.want, got)
		})
	}
}
