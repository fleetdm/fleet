package fleet

import "time"

// HostGoogleCloudIdentityClientState holds the last-known Google Cloud Identity
// ClientState that Fleet has PATCHed for a (host, signed-in Workspace deviceUser)
// pair. Cardinality is per-deviceUser, not per-host — a single host can have
// multiple Workspace identities signed into Endpoint Verification, and Fleet
// emits one ClientState per (host, deviceUser, partner_suffix) tuple.
type HostGoogleCloudIdentityClientState struct {
	ID uint `db:"id"`
	// HostID is the host's ID.
	HostID uint `db:"host_id"`
	// RawResourceID is the device_resource_id Fleet read from Endpoint
	// Verification's local accounts.json. Used to resolve to the canonical
	// deviceUser resource name on first sync.
	RawResourceID string `db:"raw_resource_id"`
	// DeviceUserResource is the canonical Cloud Identity resource name for the
	// deviceUser this row tracks: "devices/{deviceId}/deviceUsers/{deviceUserId}".
	// NULL until the resolution layer's first lookup call succeeds.
	DeviceUserResource *string `db:"device_user_resource"`
	// WorkspaceEmail is the Google Workspace email address that's signed in on
	// the device for this deviceUser.
	WorkspaceEmail string `db:"workspace_email"`
	// PartnerSuffix is the suffix portion of the ClientState resource name.
	// Combined with the customer ID it forms `{customerID-without-C}-{suffix}`.
	// Defaults to the server-config default; teams may override.
	PartnerSuffix string `db:"partner_suffix"`

	// LastCompliant is the last `complianceState` value Fleet wrote.
	// COMPLIANT → true, NON_COMPLIANT → false, never written → nil.
	LastCompliant *bool `db:"last_compliant"`
	// LastManaged is the last `managed` value Fleet wrote.
	// MANAGED → true, UNMANAGED → false, never written → nil.
	LastManaged *bool `db:"last_managed"`
	// LastScoreReason is the last `scoreReason` string Fleet wrote.
	LastScoreReason *string `db:"last_score_reason"`
	// LastEtag is the etag returned by Cloud Identity on the last successful
	// PATCH, used for optimistic concurrency on subsequent writes.
	LastEtag *string `db:"last_etag"`
	// LastSyncedAt is when Fleet last wrote a ClientState for this row.
	LastSyncedAt *time.Time `db:"last_synced_at"`

	UpdateCreateTimestamps
}

// AuthzType implements authz.AuthzTyper.
func (h *HostGoogleCloudIdentityClientState) AuthzType() string {
	return "host_google_cloud_identity_clientstate"
}
