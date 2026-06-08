package microsoft_mdm

import (
	"bytes"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/reconcile"
)

// ComputeWindowsReconcileDeltas evaluates desired profile state for each host
// in the input set using the SHARED dispatcher (server/mdm/reconcile), then
// diffs against current host_mdm_windows_profiles rows to produce install and
// remove sets. It is the in-memory port of the legacy set-difference SQL
// (windowsProfilesToInstallQuery / windowsProfilesToRemoveQuery); the diff
// rules below mirror those WHERE clauses exactly.
//
// Windows hosts are all platform 'windows', so there is no per-host platform
// gate here (unlike Apple's EntityAppliesToHost wrapper) — the snapshot's host
// listing already filters by platform and enrollment.
//
// profilesByTeam groups every loaded profile by its team_id; profilesWithBrokenLabels
// holds the UUIDs of profiles carrying at least one deleted label (kept out of
// removal, matching the legacy NOT EXISTS ... label_id IS NULL guard).
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

		// Install set — port of windowsProfilesToInstallQuery's WHERE clause
		// (the LEFT JOIN from desired state to host_mdm_windows_profiles).
		for profUUID, p := range desired {
			c, present := currentByProfile[profUUID]
			needsInstall := false
			switch {
			case !present:
				// profile in desired (A) but not in current (B).
				needsInstall = true
			case !bytes.Equal(c.Checksum, p.Checksum):
				// profile content changed (hmwp.checksum != ds.checksum).
				needsInstall = true
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
				ProfileUUID:      p.ProfileUUID,
				ProfileName:      p.ProfileName,
				HostUUID:         host.UUID,
				Checksum:         p.Checksum,
				SecretsUpdatedAt: p.SecretsUpdatedAt,
			})
		}

		// Remove set — port of windowsProfilesToRemoveQuery's WHERE clause
		// (the RIGHT JOIN: current rows with no desired-state match).
		for profUUID, c := range currentByProfile {
			if _, stillDesired := desired[profUUID]; stillDesired {
				continue
			}
			// Skip rows already processing a remove (operation_type = remove
			// AND status IS NOT NULL).
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			// Keep (don't remove) profiles with a broken label — matches the
			// legacy NOT EXISTS ... label_id IS NULL guard.
			if _, broken := profilesWithBrokenLabels[profUUID]; broken {
				continue
			}
			// "host still enrolled" is guaranteed by the snapshot's host
			// listing, which already applies the enrollment EXISTS predicates.

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

// isTerminalRemoveStatus reports whether a remove row's status is one that the
// install query treats as "leave alone" (verifying/verified). A NULL status,
// or any other status (e.g. pending, failed), means the remove can be flipped
// back to install. Mirrors COALESCE(status,”) NOT IN ('verifying','verified').
func isTerminalRemoveStatus(status *fleet.MDMDeliveryStatus) bool {
	if status == nil {
		return false
	}
	return *status == fleet.MDMDeliveryVerifying || *status == fleet.MDMDeliveryVerified
}
