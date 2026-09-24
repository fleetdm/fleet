package fleet

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// entraGUIDRegex matches an Azure/Entra GUID in 8-4-4-4-12 form, case-insensitively. Entra emits IDs in lower case but
// admins paste them either way, so the match is deliberately case-insensitive.
var entraGUIDRegex = regexp.MustCompile("^[A-Fa-f0-9]{8}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}-[A-Fa-f0-9]{12}$")

// IsValidEntraGUID reports whether s is a well-formed Microsoft Entra identifier: a tenant ID, an application (client)
// ID, or any other Entra object ID. It lives here rather than in a service package because both the core app config
// validation and the premium Graph credential validation need it.
func IsValidEntraGUID(s string) bool {
	return entraGUIDRegex.MatchString(s)
}

// MicrosoftGraphCredentialMetadata is a stored credential without its client secret. This is what the read endpoint
// serializes, so it must never gain a secret field.
type MicrosoftGraphCredentialMetadata struct {
	TenantID string `json:"tenant_id" db:"tenant_id"`
	ClientID string `json:"client_id" db:"client_id"`

	// CredentialInvalid is set by the sync when the credential fails to authenticate or is denied, and cleared on the
	// next successful sync.
	CredentialInvalid bool `json:"credential_invalid" db:"credential_invalid"`
	// LastSyncedAt and LastSyncError report the outcome of the most recent sync for this tenant.
	LastSyncedAt  *time.Time `json:"last_synced_at" db:"last_synced_at"`
	LastSyncError *string    `json:"last_sync_error" db:"last_sync_error"`
}

// MicrosoftGraphCredential is the Entra app-registration credential Fleet authenticates with when calling Microsoft
// Graph: the metadata plus the secret.
type MicrosoftGraphCredential struct {
	MicrosoftGraphCredentialMetadata

	// ClientSecret is write-only. Omitting it on a write means "keep the stored secret".
	ClientSecret string `json:"client_secret,omitempty" db:"-"`
}

func (c MicrosoftGraphCredential) AuthzType() string {
	return "microsoft_graph_credential"
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
	// HardwareModel and HardwareVendor seed the host row when a pending host is created and are deliberately not
	// columns on host_autopilot_devices.
	HardwareModel  string `db:"-" json:"-"`
	HardwareVendor string `db:"-" json:"-"`
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
