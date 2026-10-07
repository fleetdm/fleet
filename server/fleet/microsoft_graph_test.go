package fleet

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMicrosoftGraphCredentialConfigured checks that tenant, client, and secret are required.
func TestMicrosoftGraphCredentialConfigured(t *testing.T) {
	for _, tc := range []struct {
		name string
		cred MicrosoftGraphCredential
		want bool
	}{
		{"all set", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "t", ClientID: "c"}, ClientSecret: "s"}, true},
		{"missing tenant", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{ClientID: "c"}, ClientSecret: "s"}, false},
		{"missing client", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "t"}, ClientSecret: "s"}, false},
		{"missing secret", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "t", ClientID: "c"}}, false},
		{"empty", MicrosoftGraphCredential{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.cred.Configured())
		})
	}
}

// TestMicrosoftGraphCredentialEqual checks case-insensitive IDs and case-sensitive secrets.
func TestMicrosoftGraphCredentialEqual(t *testing.T) {
	base := MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-a", ClientID: "client-a"}, ClientSecret: "secret"}

	for _, tc := range []struct {
		name  string
		other MicrosoftGraphCredential
		want  bool
	}{
		{"identical", base, true},
		// Entra emits tenant and client IDs lower-cased but admins paste them either way, so identity is
		// case-insensitive. The secret is not.
		{"ids differ only by case", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "TENANT-A", ClientID: "CLIENT-A"}, ClientSecret: "secret"}, true},
		{"different secret", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-a", ClientID: "client-a"}, ClientSecret: "other"}, false},
		{"different tenant", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-b", ClientID: "client-a"}, ClientSecret: "secret"}, false},
		{"different client", MicrosoftGraphCredential{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-a", ClientID: "client-b"}, ClientSecret: "secret"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, base.Equal(tc.other))
		})
	}
}

// TestParseMicrosoftGraphCredentials checks declarative GitOps decoding, including clearing absent credentials and rejecting malformed values.
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
			want: []MicrosoftGraphCredential{{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-a", ClientID: "client-a"}, ClientSecret: "secret-a"}},
		},
		{
			name: "server-computed status in the payload is accepted and ignored on the way in",
			raw: []any{map[string]any{
				"tenant_id": "tenant-a", "client_id": "client-a", "client_secret": "secret-a",
				"credential_invalid": true,
			}},
			// It decodes onto the struct, but nothing downstream reads it: the datastore writes only the three input
			// columns. Round-tripping a generated file must not fail.
			want: []MicrosoftGraphCredential{{MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant-a", ClientID: "client-a", CredentialInvalid: true}, ClientSecret: "secret-a"}},
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

// TestMicrosoftGraphCredentialCloudEqual checks cloud identity and omitted-global equivalence.
func TestMicrosoftGraphCredentialCloudEqual(t *testing.T) {
	base := MicrosoftGraphCredential{
		MicrosoftGraphCredentialMetadata: MicrosoftGraphCredentialMetadata{TenantID: "tenant", ClientID: "client"},
		ClientSecret:                     "secret",
	}
	explicitGlobal := base
	explicitGlobal.Cloud = MicrosoftGraphCloudGlobal
	assert.True(t, base.Equal(explicitGlobal))
	assert.True(t, explicitGlobal.Equal(base))
	for _, cloud := range []MicrosoftGraphCloud{MicrosoftGraphCloudGCCHigh, MicrosoftGraphCloudDoD, MicrosoftGraphCloudChina} {
		other := base
		other.Cloud = cloud
		assert.False(t, base.Equal(other))
		assert.False(t, other.Equal(base))
		assert.True(t, other.Equal(other))
	}
}

// TestParseMicrosoftGraphCredentialsCloud checks that GitOps parsing preserves the selected cloud.
func TestParseMicrosoftGraphCredentialsCloud(t *testing.T) {
	creds, err := ParseMicrosoftGraphCredentials([]any{map[string]any{
		"tenant_id": "tenant", "client_id": "client", "client_secret": "secret", "cloud": "gcc_high",
	}})
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, MicrosoftGraphCloudGCCHigh, creds[0].Cloud)
}
