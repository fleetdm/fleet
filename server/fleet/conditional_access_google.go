package fleet

import "time"

// HostGoogleCloudIdentityClientState holds the last-known Google Cloud Identity
// ClientState that Fleet has PATCHed for a (host, signed-in Workspace
// deviceUser, partner_suffix) tuple. Cardinality is per-deviceUser, not
// per-host — a single host can have multiple Workspace identities signed in
// (verified empirically on a developer machine with seven concurrent
// accounts), and Fleet emits one ClientState per (host, email, suffix)
// triple.
//
// Resolution flow:
//   - Fleet's osquery layer detects that EV is installed on a host and
//     emits a workspace_email row per signed-in Workspace identity (via the
//     endpoint_verification_accounts table).
//   - The sync layer queries Cloud Identity by host.hardware_serial to
//     find the canonical Device, then enumerates its DeviceUsers and
//     matches by email to fill in DeviceUserResource.
//   - Subsequent syncs short-circuit on the cached DeviceUserResource.
type HostGoogleCloudIdentityClientState struct {
	ID uint `db:"id"`
	// HostID is the host's ID.
	HostID uint `db:"host_id"`
	// WorkspaceEmail is the Google Workspace email address that's signed in
	// on the device. Together with HostID it uniquely identifies which
	// deviceUser Fleet is targeting.
	WorkspaceEmail string `db:"workspace_email"`
	// PartnerSuffix is the suffix portion of the ClientState resource name.
	// Combined with the customer ID it forms `{customerID-without-C}-{suffix}`.
	PartnerSuffix string `db:"partner_suffix"`

	// DeviceUserResource is the canonical Cloud Identity resource name
	// `devices/{deviceId}/deviceUsers/{deviceUserId}`. NULL until the
	// resolution layer's first FindDeviceBySerial + ListDeviceUsers
	// succeeds.
	DeviceUserResource *string `db:"device_user_resource"`

	// LastCompliant is the last `complianceState` value Fleet wrote.
	LastCompliant *bool `db:"last_compliant"`
	// LastManaged is the last `managed` value Fleet wrote.
	LastManaged *bool `db:"last_managed"`
	// LastScoreReason is the last `scoreReason` string Fleet wrote.
	LastScoreReason *string `db:"last_score_reason"`
	// LastEtag is the etag Cloud Identity returned on the last successful
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
