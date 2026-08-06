package fleet

import (
	"strings"
	"time"
)

// MicrosoftGraphCredential is the Entra app-registration credential Fleet authenticates with when calling Microsoft
// Graph. The three fields are the inputs to the OAuth2 client-credentials grant: the token endpoint is tenant-specific
// (Microsoft rejects /common and /organizations for this flow), and there is no app-only token without a client ID.
//
// This is not the same thing as MDM.WindowsEntraTenantIDs / MDM.WindowsEntraClientIDs. Those are inbound allowlists
// checked against the claims of an enrollment JWT a device presents; this credential is outbound, and is what Fleet
// authenticates as. The two need not overlap, and under least privilege the Graph client ID should not appear in the
// enrollment allowlist.
//
// ClientSecret never round-trips in the AppConfig JSON: it is stored encrypted in the mdm_microsoft_graph_credentials
// table, and API responses carry MaskedPassword in its place.
type MicrosoftGraphCredential struct {
	TenantID     string `json:"tenant_id" db:"tenant_id"`
	ClientID     string `json:"client_id" db:"client_id"`
	ClientSecret string `json:"client_secret" db:"-"`

	// CredentialInvalid is set by the sync when the credential fails to authenticate or is denied, and cleared on the
	// next successful sync. It drives the app-wide banner, as ABMToken.TokenInvalid does.
	CredentialInvalid bool `json:"credential_invalid" db:"credential_invalid"`
	// LastSyncedAt and LastSyncError report the outcome of the most recent sync for this tenant.
	LastSyncedAt  *time.Time `json:"last_synced_at" db:"last_synced_at"`
	LastSyncError *string    `json:"last_sync_error" db:"last_sync_error"`
}

// Configured reports whether the credential carries everything needed to mint a token. ClientSecret is included
// because a credential is only ever assembled with it present: on read from the datastore it is decrypted, and on
// write it is either supplied or resolved from the stored value.
func (c MicrosoftGraphCredential) Configured() bool {
	return c.TenantID != "" && c.ClientID != "" && c.ClientSecret != ""
}

// Clone implements Cloner so the credential list can be cached safely. The pointer fields are deep-copied, so a caller
// mutating what it read cannot corrupt the cached copy.
//
// WARNING: adding a pointer, slice, or map field to this struct means updating this method. tools/cloner-check guards
// against forgetting.
func (c *MicrosoftGraphCredential) Clone() (Cloner, error) {
	if c == nil {
		return (*MicrosoftGraphCredential)(nil), nil
	}
	clone := *c
	if c.LastSyncedAt != nil {
		syncedAt := *c.LastSyncedAt
		clone.LastSyncedAt = &syncedAt
	}
	if c.LastSyncError != nil {
		syncErr := *c.LastSyncError
		clone.LastSyncError = &syncErr
	}
	return &clone, nil
}

// Equal reports whether two credentials describe the same app registration with the same secret. Used to decide
// whether a config write actually changed anything, so that re-applying an identical GitOps config emits no activity.
func (c MicrosoftGraphCredential) Equal(other MicrosoftGraphCredential) bool {
	return strings.EqualFold(c.TenantID, other.TenantID) &&
		strings.EqualFold(c.ClientID, other.ClientID) &&
		c.ClientSecret == other.ClientSecret
}

// WindowsAutopilotDevice is a Windows Autopilot device identity as returned by Microsoft Graph from
// /deviceManagement/windowsAutopilotDeviceIdentities.
//
// ID is the Graph resource id of the Autopilot registration. It is the identity Fleet reconciles on: it is unique and
// stable for the life of the registration, whereas SerialNumber is neither (Graph paginates this collection on serial,
// and placeholder serials such as "Default string" ship on real hardware).
type WindowsAutopilotDevice struct {
	ID              string `json:"id"`
	SerialNumber    string `json:"serialNumber"`
	GroupTag        string `json:"groupTag"`
	AzureADDeviceID string `json:"azureActiveDirectoryDeviceId"`
}

// HostAutopilotDevice is the Windows-Autopilot-only metadata Fleet stores for a host, keyed by host ID so the group tag
// survives the pending -> enrolled transition.
type HostAutopilotDevice struct {
	HostID            uint   `db:"host_id" json:"host_id"`
	AutopilotDeviceID string `db:"autopilot_device_id" json:"autopilot_device_id"`
	AzureADDeviceID   string `db:"azure_ad_device_id" json:"azure_ad_device_id"`
	GroupTag          string `db:"group_tag" json:"group_tag"`
	HardwareSerial    string `db:"hardware_serial" json:"hardware_serial"`
	TenantID          string `db:"tenant_id" json:"tenant_id"`
}
