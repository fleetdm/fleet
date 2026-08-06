package fleet

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
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

func TestAppConfigMicrosoftGraphCredentialsObfuscate(t *testing.T) {
	ac := &AppConfig{}
	ac.MDM.MicrosoftGraphCredentials = optjson.SetSlice([]MicrosoftGraphCredential{
		{TenantID: "tenant-a", ClientID: "client-a", ClientSecret: "super-secret"},
	})

	ac.Obfuscate()

	require.Len(t, ac.MDM.MicrosoftGraphCredentials.Value, 1)
	assert.Equal(t, MaskedPassword, ac.MDM.MicrosoftGraphCredentials.Value[0].ClientSecret)
	// The non-secret fields still round-trip so the UI can show which app registration is configured.
	assert.Equal(t, "tenant-a", ac.MDM.MicrosoftGraphCredentials.Value[0].TenantID)
	assert.Equal(t, "client-a", ac.MDM.MicrosoftGraphCredentials.Value[0].ClientID)
}

func TestAppConfigMicrosoftGraphCredentialsClone(t *testing.T) {
	syncedAt := time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC)
	syncErrMsg := "boom"
	ac := &AppConfig{}
	ac.MDM.MicrosoftGraphCredentials = optjson.SetSlice([]MicrosoftGraphCredential{
		{
			TenantID:      "tenant-a",
			ClientID:      "client-a",
			LastSyncedAt:  &syncedAt,
			LastSyncError: &syncErrMsg,
		},
	})

	clone := ac.Copy()
	require.NotNil(t, clone)
	require.Len(t, clone.MDM.MicrosoftGraphCredentials.Value, 1)

	// Mutating the clone must not reach back into the original: the slice, and both pointer fields, are copied.
	clone.MDM.MicrosoftGraphCredentials.Value[0].TenantID = "mutated"
	*clone.MDM.MicrosoftGraphCredentials.Value[0].LastSyncedAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	*clone.MDM.MicrosoftGraphCredentials.Value[0].LastSyncError = "mutated"

	assert.Equal(t, "tenant-a", ac.MDM.MicrosoftGraphCredentials.Value[0].TenantID)
	assert.Equal(t, syncedAt, *ac.MDM.MicrosoftGraphCredentials.Value[0].LastSyncedAt)
	assert.Equal(t, "boom", *ac.MDM.MicrosoftGraphCredentials.Value[0].LastSyncError)
}
