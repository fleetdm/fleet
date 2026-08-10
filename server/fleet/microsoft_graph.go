package fleet

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// MicrosoftGraphCredential is the Entra app-registration credential Fleet authenticates with when calling Microsoft Graph.
type MicrosoftGraphCredential struct {
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	ClientID     string `json:"client_id" db:"client_id"`
	ClientSecret string `json:"client_secret" db:"-"`

	// CredentialInvalid is set by the sync when the credential fails to authenticate or is denied, and cleared on the
	// next successful sync.
	CredentialInvalid bool `json:"credential_invalid" db:"credential_invalid"`
	// LastSyncedAt and LastSyncError report the outcome of the most recent sync for this tenant.
	LastSyncedAt  *time.Time `json:"last_synced_at" db:"last_synced_at"`
	LastSyncError *string    `json:"last_sync_error" db:"last_sync_error"`
}

// Configured reports whether the credential carries everything needed to mint a token.
func (c MicrosoftGraphCredential) Configured() bool {
	return c.TenantID != "" && c.ClientID != "" && c.ClientSecret != ""
}

// Equal reports whether two credentials describe the same app registration with the same secret.
func (c MicrosoftGraphCredential) Equal(other MicrosoftGraphCredential) bool {
	return strings.EqualFold(c.TenantID, other.TenantID) &&
		strings.EqualFold(c.ClientID, other.ClientID) &&
		c.ClientSecret == other.ClientSecret
}

// HostAutopilotDevice is the Windows-Autopilot-only metadata Fleet stores for a host, keyed by host ID so the group tag
// survives the pending -> enrolled transition.
type HostAutopilotDevice struct {
	HostID            uint   `db:"host_id" json:"host_id"`
	AutopilotDeviceID string `db:"autopilot_device_id" json:"autopilot_device_id"`
	EntraDeviceID     string `db:"entra_device_id" json:"entra_device_id"`
	GroupTag          string `db:"group_tag" json:"group_tag"`
	HardwareSerial    string `db:"hardware_serial" json:"hardware_serial"`
	TenantID          string `db:"tenant_id" json:"tenant_id"`
}

// ParseMicrosoftGraphCredentials decodes the raw GitOps `org_settings.microsoft_graph_credentials` value into a typed list.
func ParseMicrosoftGraphCredentials(raw any) ([]MicrosoftGraphCredential, error) {
	if raw == nil {
		return []MicrosoftGraphCredential{}, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshal microsoft graph credentials: %w", err)
	}
	var creds []MicrosoftGraphCredential
	if err := json.Unmarshal(encoded, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal microsoft graph credentials: %w", err)
	}
	if creds == nil {
		creds = []MicrosoftGraphCredential{}
	}
	return creds, nil
}
