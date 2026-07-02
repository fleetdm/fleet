package microsoft_mdm

import (
	"bytes"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/reconcile"
)

// ComputeWindowsReconcileDeltas evaluates the desired profile state for each host in the input set using the SHARED dispatcher
// (server/mdm/reconcile), then diffs against current host_mdm_windows_profiles rows to produce install and remove sets.
//
// profilesByTeam groups every loaded profile by its team_id; profilesWithBrokenLabels holds the UUIDs of profiles carrying at
// least one deleted label (kept out of removal).
func ComputeWindowsReconcileDeltas(
	hosts []*fleet.WindowsHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMWindowsProfilePayload,
	profilesByTeam map[uint][]*fleet.WindowsProfileForReconcile,
	profilesWithBrokenLabels map[string]struct{},
) (toInstall, toRemove []*fleet.MDMWindowsProfilePayload) {
	for _, host := range hosts {
		teamProfiles := profilesByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.WindowsProfileForReconcile, len(teamProfiles))

		labelsForHost := hostLabels[host.HostID]

		for _, p := range teamProfiles {
			// Determine if this profile should be on this host
			if !reconcile.EntityAppliesToHost(p, host.EffectiveTeamID(), host.LabelUpdatedAt, labelsForHost) {
				continue
			}
			desired[p.ProfileUUID] = p
		}

		current := currentByHost[host.UUID]
		currentByProfile := make(map[string]*fleet.MDMWindowsProfilePayload, len(current))
		for _, c := range current {
			currentByProfile[c.ProfileUUID] = c
		}

		// Install set
		for profUUID, p := range desired {
			c, present := currentByProfile[profUUID]
			needsInstall := false
			// previousInstalledChecksum is set only when the install is triggered by a content change (a modify): it is the version
			// the host currently has, which the cron uses to look up the LocURIs the edit removed so it can <Delete> them.
			var previousInstalledChecksum []byte
			switch {
			case !present:
				// profile in desired (A) but not in current (B).
				needsInstall = true
			case !bytes.Equal(c.Checksum, p.Checksum):
				// profile content changed (hmwp.checksum != ds.checksum).
				needsInstall = true
				previousInstalledChecksum = c.Checksum
			case p.SecretsUpdatedAt != nil && c.SecretsUpdatedAt != nil && c.SecretsUpdatedAt.Before(*p.SecretsUpdatedAt):
				// secret variables updated. Matches
				// IFNULL(hmwp.secrets_updated_at < ds.secrets_updated_at, FALSE):
				// only fires when BOTH timestamps are present and current is older.
				needsInstall = true
			case c.OperationType == fleet.MDMOperationTypeInstall && c.Status == nil:
				// install was never sent (NULL status); re-push.
				needsInstall = true
			case c.OperationType == fleet.MDMOperationTypeRemove && !isTerminalRemoveStatus(c.Status):
				// currently marked for removal but not an in-flight or completed
				// removal — flip back to install. Matches
				// operation_type = remove AND COALESCE(status,'') NOT IN ('verifying','verified').
				needsInstall = true
			}
			if !needsInstall {
				continue
			}

			toInstall = append(toInstall, &fleet.MDMWindowsProfilePayload{
				ProfileUUID:               p.ProfileUUID,
				ProfileName:               p.ProfileName,
				HostUUID:                  host.UUID,
				Checksum:                  p.Checksum,
				SecretsUpdatedAt:          p.SecretsUpdatedAt,
				PreviousInstalledChecksum: previousInstalledChecksum,
			})
		}

		// Remove set
		for profUUID, c := range currentByProfile {
			if _, stillDesired := desired[profUUID]; stillDesired {
				continue
			}
			// Skip rows already processing a remove
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			// Keep (don't remove) profiles with a broken label
			if _, broken := profilesWithBrokenLabels[profUUID]; broken {
				continue
			}

			toRemove = append(toRemove, &fleet.MDMWindowsProfilePayload{
				ProfileUUID:   c.ProfileUUID,
				ProfileName:   c.ProfileName,
				HostUUID:      host.UUID,
				OperationType: c.OperationType,
				Detail:        c.Detail,
				Status:        c.Status,
				CommandUUID:   c.CommandUUID,
			})
		}
	}
	return toInstall, toRemove
}

// DesiredWindowsProfileUUIDsByHost returns, for each host UUID, the live profile UUIDs that apply to it (its desired state), using the
// same team+label applicability rules as ComputeWindowsReconcileDeltas. The reconciler uses this to protect LocURIs that a remove target
// shares with a profile still desired on the same host: a <Delete> must not revert a setting another applicable profile still enforces.
// Applicability is evaluated per host, so a label-scoped profile only protects the hosts it actually applies to.
func DesiredWindowsProfileUUIDsByHost(
	hosts []*fleet.WindowsHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	profilesByTeam map[uint][]*fleet.WindowsProfileForReconcile,
) map[string][]string {
	out := make(map[string][]string, len(hosts))
	for _, host := range hosts {
		teamProfiles := profilesByTeam[host.EffectiveTeamID()]
		labelsForHost := hostLabels[host.HostID]
		var desired []string
		for _, p := range teamProfiles {
			if !reconcile.EntityAppliesToHost(p, host.EffectiveTeamID(), host.LabelUpdatedAt, labelsForHost) {
				continue
			}
			desired = append(desired, p.ProfileUUID)
		}
		if len(desired) > 0 {
			out[host.UUID] = desired
		}
	}
	return out
}

// isTerminalRemoveStatus reports whether a remove row's status is one that the install query treats as "leave alone"
// (verifying/verified). A NULL status, or any other status (e.g. pending, failed), means the remove can be flipped back to
// install.
func isTerminalRemoveStatus(status *fleet.MDMDeliveryStatus) bool {
	if status == nil {
		return false
	}
	return *status == fleet.MDMDeliveryVerifying || *status == fleet.MDMDeliveryVerified
}
