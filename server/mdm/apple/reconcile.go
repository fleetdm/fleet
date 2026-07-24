package apple_mdm

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/reconcile"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/google/uuid"
)

// HoursToWaitForUserEnrollmentAfterDeviceEnrollment is how long the
// reconciler waits for a user-channel enrollment to materialise before
// failing user-scoped profile delivery on that host. Mirrors the
// constant the legacy service-package reconciler uses.
const HoursToWaitForUserEnrollmentAfterDeviceEnrollment = 2

// EntityAppliesToHost is the SHARED top-level dispatcher for Apple MDM
// label-gated entities (profiles and declarations). It applies the Apple
// platform gate, then delegates the team + include/exclude label gates to
// the platform-neutral dispatcher in server/mdm/reconcile.
//
// entityOnHost reports whether the entity currently has an install-operation
// row on the host (any status); the shared dispatcher uses it to preserve the
// host's current state when a dynamic label's membership is still unknown.
//
// Both the batched profile and declaration reconcilers — and the per-host
// enrollment path — route through this function. The shared package is
// the single source of truth for "does this label-gated MDM entity apply
// to this host" so label / team semantics cannot drift across paths or
// platforms.
func EntityAppliesToHost(
	e fleet.AppleLabeledEntity,
	host *fleet.AppleHostReconcileInfo,
	hostLabels map[uint]struct{},
	entityOnHost bool,
) bool {
	if !IsEligiblePlatform(host.Platform) {
		return false
	}
	return reconcile.EntityAppliesToHost(e, host.EffectiveTeamID(), host.LabelUpdatedAt, hostLabels, entityOnHost)
}

// IsEligiblePlatform reports whether the host's platform is one of the
// Apple platforms this reconciler can manage.
func IsEligiblePlatform(platform string) bool {
	return platform == "darwin" || platform == "ios" || platform == "ipados"
}

// ComputeReconcileDeltas evaluates desired profile state for each host in
// the input set using the SHARED dispatcher, then diffs against current
// host_mdm_apple_profiles rows to produce install and remove sets.
func ComputeReconcileDeltas(
	hosts []*fleet.AppleHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMAppleProfilePayload,
	profilesByTeam map[uint][]*fleet.AppleProfileForReconcile,
	profilesWithBrokenLabels map[string]struct{},
) (toInstall, toRemove []*fleet.MDMAppleProfilePayload) {
	for _, host := range hosts {
		teamProfiles := profilesByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.AppleProfileForReconcile, len(teamProfiles))

		labelsForHost := hostLabels[host.HostID]

		current := currentByHost[host.UUID]
		currentByProfile := make(map[string]*fleet.MDMAppleProfilePayload, len(current))
		for _, c := range current {
			currentByProfile[c.ProfileUUID] = c
		}

		for _, p := range teamProfiles {
			onHost := false
			if c, ok := currentByProfile[p.ProfileUUID]; ok {
				onHost = c.OperationType == fleet.MDMOperationTypeInstall
			}
			if !EntityAppliesToHost(p, host, labelsForHost, onHost) {
				continue
			}
			desired[p.ProfileUUID] = p
		}

		for profUUID, p := range desired {
			c, present := currentByProfile[profUUID]
			needsInstall := false
			switch {
			case !present:
				needsInstall = true
			case !bytes.Equal(c.Checksum, p.Checksum):
				needsInstall = true
			case p.SecretsUpdatedAt != nil && (c.SecretsUpdatedAt == nil || c.SecretsUpdatedAt.Before(*p.SecretsUpdatedAt)):
				needsInstall = true
			case c.OperationType == "" || c.OperationType == fleet.MDMOperationTypeRemove:
				needsInstall = true
			case c.OperationType == fleet.MDMOperationTypeInstall && c.Status == nil:
				needsInstall = true
			}
			if !needsInstall {
				continue
			}

			toInstall = append(toInstall, &fleet.MDMAppleProfilePayload{
				ProfileUUID:       p.ProfileUUID,
				ProfileIdentifier: p.ProfileIdentifier,
				ProfileName:       p.ProfileName,
				HostUUID:          host.UUID,
				HostPlatform:      host.Platform,
				Checksum:          p.Checksum,
				SecretsUpdatedAt:  p.SecretsUpdatedAt,
				Scope:             p.Scope,
				DeviceEnrolledAt:  host.DeviceEnrolledAt,
			})
		}

		for profUUID, c := range currentByProfile {
			if _, stillDesired := desired[profUUID]; stillDesired {
				continue
			}
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			if IsBrokenProfile(profUUID, profilesWithBrokenLabels) {
				continue
			}

			toRemove = append(toRemove, &fleet.MDMAppleProfilePayload{
				ProfileUUID:       c.ProfileUUID,
				ProfileIdentifier: c.ProfileIdentifier,
				ProfileName:       c.ProfileName,
				HostUUID:          host.UUID,
				HostPlatform:      host.Platform,
				Checksum:          c.Checksum,
				SecretsUpdatedAt:  c.SecretsUpdatedAt,
				Status:            c.Status,
				OperationType:     c.OperationType,
				Detail:            c.Detail,
				CommandUUID:       c.CommandUUID,
				IgnoreError:       c.IgnoreError,
				Scope:             c.Scope,
				DeviceEnrolledAt:  host.DeviceEnrolledAt,
			})
		}
	}
	return toInstall, toRemove
}

// IsBrokenProfile returns true if any label assignment on the profile
// references a deleted label. Used to skip removal of broken profiles
// (matches legacy behavior).
func IsBrokenProfile(profileUUID string, profilesWithBrokenLabel map[string]struct{}) bool {
	_, broken := profilesWithBrokenLabel[profileUUID]
	return broken
}

// IsBrokenDeclaration is the DDM equivalent of IsBrokenProfile.
func IsBrokenDeclaration(declUUID string, declsWithBrokenLabel map[string]struct{}) bool {
	_, broken := declsWithBrokenLabel[declUUID]
	return broken
}

// scopeOrDefaultDDM normalizes an empty scope to System (device channel) so
// pre-scope rows and callers that leave scope unset behave as device-scoped.
func scopeOrDefaultDDM(s fleet.PayloadScope) fleet.PayloadScope {
	if s == "" {
		return fleet.PayloadScopeSystem
	}
	return s
}

// ComputeDeclarationDeltas is the DDM equivalent of ComputeReconcileDeltas.
// Uses the SAME shared dispatcher (EntityAppliesToHost) so profile and
// declaration label semantics cannot drift.
//
// Returns the host declaration rows to write plus the hosts whose declaration
// set changed, partitioned by channel. A DDM DeclarativeManagement command only
// needs to reach the channel(s) that actually changed, so device-scoped and
// user-scoped changes are tracked separately: a host that only changed on one
// channel is poked on that channel alone. A scope flip (a declaration whose
// PayloadScope changed) counts as a change on BOTH channels — the new channel
// installs it and the old channel drops it because its scoped declaration set
// no longer includes it.
func ComputeDeclarationDeltas(
	hosts []*fleet.AppleHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMAppleHostDeclaration,
	declsByTeam map[uint][]*fleet.AppleDeclarationForReconcile,
	declsWithBrokenLabel map[string]struct{},
) (changedDeviceHostUUIDs, changedUserHostUUIDs []string, declRowsToWrite []*fleet.MDMAppleHostDeclaration) {
	pendingStatus := fleet.MDMDeliveryPending
	deviceChanged := make(map[string]struct{})
	userChanged := make(map[string]struct{})

	markChanged := func(hostUUID string, scope fleet.PayloadScope) {
		if scopeOrDefaultDDM(scope) == fleet.PayloadScopeUser {
			userChanged[hostUUID] = struct{}{}
		} else {
			deviceChanged[hostUUID] = struct{}{}
		}
	}

	for _, host := range hosts {
		teamDecls := declsByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.AppleDeclarationForReconcile, len(teamDecls))

		labelsForHost := hostLabels[host.HostID]

		current := currentByHost[host.UUID]
		currentByDecl := make(map[string]*fleet.MDMAppleHostDeclaration, len(current))
		for _, c := range current {
			currentByDecl[c.DeclarationUUID] = c
		}

		for _, d := range teamDecls {
			onHost := false
			if c, ok := currentByDecl[d.DeclarationUUID]; ok {
				onHost = c.OperationType == fleet.MDMOperationTypeInstall
			}
			if !EntityAppliesToHost(d, host, labelsForHost, onHost) {
				continue
			}
			desired[d.DeclarationUUID] = d
		}

		for declUUID, d := range desired {
			c, present := currentByDecl[declUUID]
			desiredScope := scopeOrDefaultDDM(d.Scope)
			needsInstall := false
			switch {
			case !present:
				needsInstall = true
			case scopeOrDefaultDDM(c.Scope) != desiredScope:
				// scope flip: re-deliver on the new channel; handled below by also
				// marking the previous channel changed so it drops the declaration.
				needsInstall = true
			case !bytes.Equal([]byte(c.Token), d.Token):
				needsInstall = true
			case d.SecretsUpdatedAt != nil && (c.SecretsUpdatedAt == nil || c.SecretsUpdatedAt.Before(*d.SecretsUpdatedAt)):
				needsInstall = true
			case d.AssetsUpdatedAt != nil && (c.AssetsUpdatedAt == nil || c.AssetsUpdatedAt.Before(*d.AssetsUpdatedAt)):
				// A referenced asset was edited (its uploaded_at moved forward)
				// since we last delivered this declaration to the host. Re-deliver
				// so the per-host effective token changes and the host re-fetches,
				// even though the declaration's own content/token is unchanged.
				needsInstall = true
			case c.OperationType == "" || c.OperationType == fleet.MDMOperationTypeRemove:
				needsInstall = true
			case c.OperationType == fleet.MDMOperationTypeInstall && c.Status == nil:
				needsInstall = true
			}
			if !needsInstall {
				continue
			}

			row := &fleet.MDMAppleHostDeclaration{
				HostUUID:         host.UUID,
				DeclarationUUID:  d.DeclarationUUID,
				Name:             d.DeclarationName,
				Identifier:       d.DeclarationIdentifier,
				Status:           &pendingStatus,
				OperationType:    fleet.MDMOperationTypeInstall,
				Token:            string(d.Token),
				SecretsUpdatedAt: d.SecretsUpdatedAt,
				Scope:            desiredScope,
			}

			if d.HasFleetVariables {
				now := time.Now().UTC()
				row.VariablesUpdatedAt = &now
			}
			// Stamp the referenced assets' latest uploaded_at so the per-host
			// effective token folds it in (see fleet.EffectiveDDMToken) and stays
			// idempotent: on the next reconcile c.AssetsUpdatedAt equals this value
			// and no needless re-delivery is triggered.
			if d.AssetsUpdatedAt != nil {
				row.AssetsUpdatedAt = d.AssetsUpdatedAt
			}
			declRowsToWrite = append(declRowsToWrite, row)
			markChanged(host.UUID, desiredScope)
			if present {
				if prev := scopeOrDefaultDDM(c.Scope); prev != desiredScope {
					markChanged(host.UUID, prev)
				}
			}
		}

		for declUUID, c := range currentByDecl {
			if _, stillDesired := desired[declUUID]; stillDesired {
				continue
			}
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			if IsBrokenDeclaration(declUUID, declsWithBrokenLabel) {
				continue
			}

			removeScope := scopeOrDefaultDDM(c.Scope)
			declRowsToWrite = append(declRowsToWrite, &fleet.MDMAppleHostDeclaration{
				HostUUID:         host.UUID,
				DeclarationUUID:  c.DeclarationUUID,
				Name:             c.Name,
				Identifier:       c.Identifier,
				Status:           &pendingStatus,
				OperationType:    fleet.MDMOperationTypeRemove,
				Token:            c.Token,
				SecretsUpdatedAt: c.SecretsUpdatedAt,
				Scope:            removeScope,
			})
			markChanged(host.UUID, removeScope)
		}
	}

	changedDeviceHostUUIDs = make([]string, 0, len(deviceChanged))
	for u := range deviceChanged {
		changedDeviceHostUUIDs = append(changedDeviceHostUUIDs, u)
	}
	changedUserHostUUIDs = make([]string, 0, len(userChanged))
	for u := range userChanged {
		changedUserHostUUIDs = append(changedUserHostUUIDs, u)
	}
	return changedDeviceHostUUIDs, changedUserHostUUIDs, declRowsToWrite
}

// ExecuteReconcileBatch runs the post-listing reconcile pipeline against
// the in-memory toInstall / toRemove sets produced by ComputeReconcileDeltas.
//
// Returns the list of successfully enqueued command UUIDs. Callers that
// don't care (the cron) can discard them. The per-host enrollment path
// uses them so the worker can wait on the commands it queued.
//
// Pass redisKeyValue == nil to skip the "host being set up" Redis check —
// the per-host enrollment path passes nil since by construction the host
// IS being set up and we explicitly want to install the profiles right now.
func ExecuteReconcileBatch(
	ctx context.Context,
	ds fleet.Datastore,
	commander *MDMAppleCommander,
	redisKeyValue fleet.AdvancedKeyValueStore,
	logger *slog.Logger,
	appConfig *fleet.AppConfig,
	certProfilesLimit int,
	toInstall, toRemove []*fleet.MDMAppleProfilePayload,
) ([]string, error) {
	userEnrollmentMap := make(map[string]string)
	userEnrollmentsToHostUUIDsMap := make(map[string]string)

	getHostUserEnrollmentID := func(hostUUID string) (string, error) {
		userEnrollmentID, ok := userEnrollmentMap[hostUUID]
		if !ok {
			userNanoEnrollment, err := ds.GetNanoMDMUserEnrollment(ctx, hostUUID)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "getting user enrollment for host")
			}
			if userNanoEnrollment != nil {
				userEnrollmentID = userNanoEnrollment.ID
			}
			userEnrollmentMap[hostUUID] = userEnrollmentID
			if userEnrollmentID != "" {
				userEnrollmentsToHostUUIDsMap[userEnrollmentID] = hostUUID
			}
		}
		return userEnrollmentID, nil
	}

	isAwaitingUserEnrollment := func(prof *fleet.MDMAppleProfilePayload) (bool, error) {
		if prof.Scope != fleet.PayloadScopeUser {
			return false, nil
		}
		userEnrollmentID, err := getHostUserEnrollmentID(prof.HostUUID)
		if userEnrollmentID != "" || err != nil {
			return false, err
		}
		if prof.DeviceEnrolledAt != nil && time.Since(*prof.DeviceEnrolledAt) < HoursToWaitForUserEnrollmentAfterDeviceEnrollment*time.Hour {
			return true, nil
		}
		return false, nil
	}

	hostProfiles := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(toInstall)+len(toRemove))

	profileIntersection := NewProfileBimap()
	profileIntersection.IntersectByIdentifierAndHostUUID(toInstall, toRemove)

	hostProfilesToCleanup := []*fleet.MDMAppleProfilePayload{}
	hostProfilesToInstallMap := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(toInstall))

	var caProfileUUIDs map[string]struct{}
	var prefetchedContents map[string]mobileconfig.Mobileconfig
	if certProfilesLimit > 0 {
		uniqueUUIDs := make(map[string]struct{}, len(toInstall))
		for _, p := range toInstall {
			uniqueUUIDs[p.ProfileUUID] = struct{}{}
		}
		uuids := make([]string, 0, len(uniqueUUIDs))
		for u := range uniqueUUIDs {
			uuids = append(uuids, u)
		}
		var err error
		prefetchedContents, err = ds.GetMDMAppleProfilesContents(ctx, uuids)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting profile contents for CA classification")
		}
		caProfileUUIDs = make(map[string]struct{}, len(prefetchedContents))
		for pUUID, content := range prefetchedContents {
			fleetVars := variables.Find(string(content))
			if fleet.HasCAVariables(fleetVars) {
				caProfileUUIDs[pUUID] = struct{}{}
			}
		}
	}

	var caInstallCount int
	throttledHostsByProfile := make(map[string][]string)
	installTargets, removeTargets := make(map[string]*fleet.CmdTarget), make(map[string]*fleet.CmdTarget)

	for _, p := range toInstall {
		if pp, ok := profileIntersection.GetMatchingProfileInCurrentState(p); ok && pp != nil {
			if (pp.Status != nil && *pp.Status != fleet.MDMDeliveryFailed) && bytes.Equal(pp.Checksum, p.Checksum) {
				hp := &fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileUUID:       p.ProfileUUID,
					HostUUID:          p.HostUUID,
					ProfileIdentifier: p.ProfileIdentifier,
					ProfileName:       p.ProfileName,
					Checksum:          p.Checksum,
					SecretsUpdatedAt:  p.SecretsUpdatedAt,
					OperationType:     pp.OperationType,
					Status:            pp.Status,
					CommandUUID:       pp.CommandUUID,
					Detail:            pp.Detail,
					Scope:             pp.Scope,
				}
				hostProfiles = append(hostProfiles, hp)
				hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: p.HostUUID, ProfileUUID: p.ProfileUUID}] = hp
				continue
			}
		}

		wait, err := isAwaitingUserEnrollment(p)
		if err != nil {
			return nil, err
		}
		if wait {
			hp := &fleet.MDMAppleBulkUpsertHostProfilePayload{
				ProfileUUID:       p.ProfileUUID,
				HostUUID:          p.HostUUID,
				ProfileIdentifier: p.ProfileIdentifier,
				ProfileName:       p.ProfileName,
				Checksum:          p.Checksum,
				SecretsUpdatedAt:  p.SecretsUpdatedAt,
				OperationType:     fleet.MDMOperationTypeInstall,
				Status:            nil,
				Scope:             p.Scope,
			}
			hostProfiles = append(hostProfiles, hp)
			hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: p.HostUUID, ProfileUUID: p.ProfileUUID}] = hp
			continue
		}

		recentlyEnrolled := p.DeviceEnrolledAt != nil && time.Since(*p.DeviceEnrolledAt) < 1*time.Hour
		_, isCA := caProfileUUIDs[p.ProfileUUID]
		isThrottledCA := certProfilesLimit > 0 && isCA && !recentlyEnrolled
		if isThrottledCA && caInstallCount >= certProfilesLimit {
			throttledHostsByProfile[p.ProfileUUID] = append(throttledHostsByProfile[p.ProfileUUID], p.HostUUID)
			continue
		}

		target := installTargets[p.ProfileUUID]
		if target == nil {
			target = &fleet.CmdTarget{
				CmdUUID:           uuid.New().String(),
				ProfileIdentifier: p.ProfileIdentifier,
				ProfileName:       p.ProfileName,
			}
			installTargets[p.ProfileUUID] = target
		}

		if p.Scope == fleet.PayloadScopeUser {
			userEnrollmentID, err := getHostUserEnrollmentID(p.HostUUID)
			if err != nil {
				return nil, err
			}
			if userEnrollmentID == "" {
				var errorDetail string
				if fleet.IsAppleMobilePlatform(p.HostPlatform) {
					errorDetail = "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts."
				} else {
					errorDetail = "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll."
					logger.WarnContext(ctx, "host does not have a user enrollment, failing profile installation",
						"host_uuid", p.HostUUID, "profile_uuid", p.ProfileUUID, "profile_identifier", p.ProfileIdentifier)
				}

				hp := &fleet.MDMAppleBulkUpsertHostProfilePayload{
					ProfileUUID:       p.ProfileUUID,
					HostUUID:          p.HostUUID,
					OperationType:     fleet.MDMOperationTypeInstall,
					Status:            &fleet.MDMDeliveryFailed,
					Detail:            errorDetail,
					CommandUUID:       "",
					ProfileIdentifier: p.ProfileIdentifier,
					ProfileName:       p.ProfileName,
					Checksum:          p.Checksum,
					SecretsUpdatedAt:  p.SecretsUpdatedAt,
					Scope:             p.Scope,
				}
				hostProfiles = append(hostProfiles, hp)
				continue
			}
			target.EnrollmentIDs = append(target.EnrollmentIDs, userEnrollmentID)
		} else {
			target.EnrollmentIDs = append(target.EnrollmentIDs, p.HostUUID)
		}

		if isThrottledCA {
			caInstallCount++
		}

		hp := &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMOperationTypeInstall,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       target.CmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			Checksum:          p.Checksum,
			SecretsUpdatedAt:  p.SecretsUpdatedAt,
			Scope:             p.Scope,
		}
		hostProfiles = append(hostProfiles, hp)
		hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: p.HostUUID, ProfileUUID: p.ProfileUUID}] = hp
	}

	const throttleLogBatchSize = 1000
	for profileUUID, hostUUIDs := range throttledHostsByProfile {
		for i := 0; i < len(hostUUIDs); i += throttleLogBatchSize {
			end := min(i+throttleLogBatchSize, len(hostUUIDs))
			logger.InfoContext(ctx, "throttled CA certificate profile installation",
				"profile.uuid", profileUUID,
				"mdm.target.host.uuids", hostUUIDs[i:end],
				"mdm.certificate.profiles.limit", certProfilesLimit,
				"batch", fmt.Sprintf("%d-%d/%d", i+1, end, len(hostUUIDs)),
			)
		}
	}

	for _, p := range toRemove {
		if _, ok := profileIntersection.GetMatchingProfileInDesiredState(p); ok {
			hostProfilesToCleanup = append(hostProfilesToCleanup, p)
			continue
		}

		if p.FailedInstallOnHost() {
			hostProfilesToCleanup = append(hostProfilesToCleanup, p)
			continue
		}
		if p.PendingInstallOnHost() {
			hostProfilesToCleanup = append(hostProfilesToCleanup, p)
			p.IgnoreError = true
		}

		target := removeTargets[p.ProfileUUID]
		if target == nil {
			target = &fleet.CmdTarget{
				CmdUUID:           uuid.New().String(),
				ProfileIdentifier: p.ProfileIdentifier,
				ProfileName:       p.ProfileName,
			}
			removeTargets[p.ProfileUUID] = target
		}

		if p.Scope == fleet.PayloadScopeUser {
			userEnrollmentID, err := getHostUserEnrollmentID(p.HostUUID)
			if err != nil {
				return nil, err
			}
			if userEnrollmentID == "" {
				logger.WarnContext(ctx, "host does not have a user enrollment, cannot remove user scoped profile",
					"host_uuid", p.HostUUID, "profile_uuid", p.ProfileUUID, "profile_identifier", p.ProfileIdentifier)
				hostProfilesToCleanup = append(hostProfilesToCleanup, p)
				continue
			}
			target.EnrollmentIDs = append(target.EnrollmentIDs, userEnrollmentID)
		} else {
			target.EnrollmentIDs = append(target.EnrollmentIDs, p.HostUUID)
		}

		hostProfiles = append(hostProfiles, &fleet.MDMAppleBulkUpsertHostProfilePayload{
			ProfileUUID:       p.ProfileUUID,
			HostUUID:          p.HostUUID,
			OperationType:     fleet.MDMOperationTypeRemove,
			Status:            &fleet.MDMDeliveryPending,
			CommandUUID:       target.CmdUUID,
			ProfileIdentifier: p.ProfileIdentifier,
			ProfileName:       p.ProfileName,
			Checksum:          p.Checksum,
			SecretsUpdatedAt:  p.SecretsUpdatedAt,
			IgnoreError:       p.IgnoreError,
			Scope:             p.Scope,
		})
	}

	const isBeingSetupBatchSize = 1000
	for i := 0; redisKeyValue != nil && i < len(hostProfiles); i += isBeingSetupBatchSize {
		end := min(i+isBeingSetupBatchSize, len(hostProfiles))
		batch := hostProfiles[i:end]
		keyedHostUUIDs := make([]string, len(batch))
		hostUUIDToHostProfiles := make(map[string][]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(batch))
		for j, hp := range batch {
			keyedHostUUIDs[j] = fleet.MDMProfileProcessingKeyPrefix + ":" + hp.HostUUID
			hostUUIDToHostProfiles[hp.HostUUID] = append(hostUUIDToHostProfiles[hp.HostUUID], hp)
		}

		setupHostUUIDs, err := redisKeyValue.MGet(ctx, keyedHostUUIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "filtering hosts being set up")
		}
		for keyedHostUUID, exists := range setupHostUUIDs {
			if exists != nil {
				hostUUID := strings.TrimPrefix(keyedHostUUID, fleet.MDMProfileProcessingKeyPrefix+":")
				logger.DebugContext(ctx, "skipping profile reconciliation for host being set up", "host_uuid", hostUUID)
				hps, ok := hostUUIDToHostProfiles[hostUUID]
				if !ok {
					logger.DebugContext(ctx, "expected host uuid to be present but was not, do not skip profile reconciliation", "host_uuid", hostUUID)
					continue
				}
				for _, hp := range hps {
					hp.Status = nil
					hp.CommandUUID = ""
					hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: hp.HostUUID, ProfileUUID: hp.ProfileUUID}] = hp

					// Also remove this host from installTargets to prevent sending MDM commands for this host.
					// Note: user-scoped profiles use user enrollment IDs (not host UUIDs) in EnrollmentIDs, so
					// the removal below is a no-op for those profiles, which is acceptable, since they are not enqueued via the worker.
					if hp.OperationType == fleet.MDMOperationTypeInstall {
						if target, ok := installTargets[hp.ProfileUUID]; ok {
							var newEnrollmentIDs []string
							for _, id := range target.EnrollmentIDs {
								if id != hp.HostUUID {
									newEnrollmentIDs = append(newEnrollmentIDs, id)
								}
							}
							if len(newEnrollmentIDs) == 0 {
								delete(installTargets, hp.ProfileUUID)
							} else {
								target.EnrollmentIDs = newEnrollmentIDs
							}
						}
					}
				}
			}
		}
	}

	commandUUIDToHostIDsCleanupMap := make(map[string][]string)
	for _, hp := range hostProfilesToCleanup {
		if hp.CommandUUID != "" {
			commandUUIDToHostIDsCleanupMap[hp.CommandUUID] = append(commandUUIDToHostIDsCleanupMap[hp.CommandUUID], hp.HostUUID)
		}
	}
	if len(commandUUIDToHostIDsCleanupMap) > 0 {
		if err := commander.BulkDeleteHostUserCommandsWithoutResults(ctx, commandUUIDToHostIDsCleanupMap); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "deleting nano commands without results")
		}
	}
	if err := ds.BulkDeleteMDMAppleHostsConfigProfiles(ctx, hostProfilesToCleanup); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "deleting profiles that didn't change")
	}

	// Defense in depth: the batch host query dedupes hosts by UUID at the
	// source, but any caller could still hand us a target whose EnrollmentIDs
	// contain the same ID twice (e.g. duplicate host rows sharing a UUID).
	// That would make the per-command INSERT into nano_enrollment_queue collide
	// on its (id, command_uuid) primary key and fail the whole enqueue, so
	// collapse duplicates before we build commands.
	var duplicateEnrollmentIDs int
	for _, target := range installTargets {
		var removed int
		target.EnrollmentIDs, removed = dedupeEnrollmentIDs(target.EnrollmentIDs)
		duplicateEnrollmentIDs += removed
	}
	for _, target := range removeTargets {
		var removed int
		target.EnrollmentIDs, removed = dedupeEnrollmentIDs(target.EnrollmentIDs)
		duplicateEnrollmentIDs += removed
	}
	if duplicateEnrollmentIDs > 0 {
		logger.WarnContext(ctx, "batched reconcile: removed duplicate enrollment IDs from command targets; likely duplicate host rows sharing a UUID",
			"removed", duplicateEnrollmentIDs)
	}

	logger.DebugContext(ctx, "batched reconcile: before bulk upsert",
		"host_profiles", len(hostProfiles),
		"install_targets", len(installTargets),
		"remove_targets", len(removeTargets),
		"cleanup", len(hostProfilesToCleanup))

	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	enqueueResult, err := ProcessAndEnqueueProfiles(
		ctx,
		ds,
		logger,
		appConfig,
		commander,
		installTargets,
		removeTargets,
		hostProfilesToInstallMap,
		userEnrollmentsToHostUUIDsMap,
		prefetchedContents,
	)
	if err != nil {
		logger.ErrorContext(ctx, "batched reconcile: ProcessAndEnqueueProfiles returned error", "err", err)
		for _, hp := range hostProfiles {
			if hp.Status != nil && *hp.Status == fleet.MDMDeliveryPending {
				hp.Status = nil
				hp.CommandUUID = ""
			}
		}
		if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reverting host profiles after failed enqueue")
		}
		return nil, ctxerr.Wrap(ctx, err, "processing and enqueuing profiles")
	}

	if enqueueResult != nil {
		logger.InfoContext(ctx, "batched reconcile: enqueue complete",
			"succeeded_cmds", len(enqueueResult.SucceededCmdUUIDs),
			"failed_cmds", len(enqueueResult.FailedCmdUUIDs))
		for cmdUUID, ferr := range enqueueResult.FailedCmdUUIDs {
			logger.WarnContext(ctx, "batched reconcile: failed command UUID",
				"cmd_uuid", cmdUUID, "err", ferr)
		}
	}

	hostProfsByCmdUUID := make(map[string][]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(hostProfiles))
	for _, hp := range hostProfiles {
		if hp.CommandUUID != "" {
			hostProfsByCmdUUID[hp.CommandUUID] = append(hostProfsByCmdUUID[hp.CommandUUID], hp)
		}
	}

	var failed []*fleet.MDMAppleBulkUpsertHostProfilePayload
	for cmdUUID := range enqueueResult.FailedCmdUUIDs {
		for _, hp := range hostProfsByCmdUUID[cmdUUID] {
			hp.CommandUUID = ""
			hp.Status = nil
			failed = append(failed, hp)
		}
	}

	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, failed); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "reverting status of failed profiles")
	}

	return enqueueResult.SucceededCmdUUIDs, nil
}

// dedupeEnrollmentIDs removes duplicate enrollment IDs, preserving first-seen
// order, and reports how many it dropped. Duplicates would collide on the
// nano_enrollment_queue (id, command_uuid) primary key when the command is
// enqueued.
func dedupeEnrollmentIDs(ids []string) ([]string, int) {
	if len(ids) < 2 {
		return ids, 0
	}
	seen := make(map[string]struct{}, len(ids))
	out := ids[:0]
	var removed int
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			removed++
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, removed
}

// ReconcileProfilesForEnrollingHost is the per-host reconciler invoked
// after enrollment (typically by the apple_mdm worker post-DEP / post-
// manual-enrollment tasks). It reuses the shared compute/handlers/
// dispatcher so it can't drift from the batched cron reconciler on
// "what should be installed."
//
// Enrollment-specific differences from the batched cron path:
//
//   - User-scoped profiles are filtered out. The user channel typically
//     doesn't exist yet at this stage, and we don't want to write
//     Status=NULL rows for them here — the cron's isAwaitingUserEnrollment
//     path handles user-scoped delivery once the user channel materialises.
//   - The "host being set up" Redis check in ExecuteReconcileBatch is
//     skipped (redisKeyValue=nil) because by construction the host IS
//     being set up and we explicitly want its device-scoped profiles
//     installed immediately.
//   - CA throttling still applies through ExecuteReconcileBatch, but
//     freshly-enrolled hosts bypass it via the existing recentlyEnrolled
//     check.
//
// Returns the list of successfully enqueued command UUIDs so the worker
// can wait on them. Returns (nil, nil) when MDM is disabled, the host
// isn't enrolled, or there is no work to do.
func ReconcileProfilesForEnrollingHost(
	ctx context.Context,
	ds fleet.Datastore,
	commander *MDMAppleCommander,
	logger *slog.Logger,
	hostUUID string,
	certProfilesLimit int,
) ([]string, error) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil, nil
	}

	host, err := ds.GetAppleMDMHostForReconcile(ctx, hostUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get apple mdm host for reconcile")
	}
	if host == nil {
		return nil, nil
	}

	// Load only profiles for this host's team (and global team_id=0) so
	// the per-host worker call doesn't scan every profile in the system.
	// In suites that accumulate profile rows across sub-tests without
	// cleanup, the unbounded "all profiles" query holds the connection
	// open longer and longer — observed as MySQL EOF / "invalid
	// connection" after many enrollments.
	teamProfiles, err := ds.ListAppleProfilesForReconcileByTeam(ctx, host.EffectiveTeamID())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing apple profiles for reconcile by team")
	}

	profilesWithBrokenLabel := make(map[string]struct{})
	profilesByTeam := make(map[uint][]*fleet.AppleProfileForReconcile, 2)
	labelIDSet := make(map[uint]struct{})
	for _, p := range teamProfiles {
		profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)

		if p.HasBrokenLabel() {
			profilesWithBrokenLabel[p.ProfileUUID] = struct{}{}
		}

		for _, lr := range p.IncludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
		for _, lr := range p.ExcludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
	}
	labelIDs := make([]uint, 0, len(labelIDSet))
	for id := range labelIDSet {
		labelIDs = append(labelIDs, id)
	}

	hostLabels, err := ds.BulkGetHostLabelMemberships(ctx, []uint{host.HostID}, labelIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk get host label memberships")
	}

	currentByHost, err := ds.BulkGetHostMDMAppleProfilesByUUIDs(ctx, []string{host.UUID})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk get host mdm apple profiles")
	}

	toInstall, toRemove := ComputeReconcileDeltas(
		[]*fleet.AppleHostReconcileInfo{host}, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel,
	)
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

	// Defer user-scoped profile delivery to the cron — see function comment.
	toInstall = fleet.FilterOutUserScopedProfiles(toInstall)

	if len(toInstall) == 0 && len(toRemove) == 0 {
		return nil, nil
	}

	return ExecuteReconcileBatch(
		ctx, ds, commander, nil, logger,
		appConfig, certProfilesLimit, toInstall, toRemove,
	)
}

// PendingProfilesForHost returns the (toInstall, toRemove) deltas for a
// single host, including user-scoped profiles.
func PendingProfilesForHost(
	ctx context.Context,
	ds fleet.Datastore,
	hostUUID string,
) (toInstall, toRemove []*fleet.MDMAppleProfilePayload, err error) {
	host, err := ds.GetAppleMDMHostForReconcile(ctx, hostUUID)
	if err != nil || host == nil {
		return nil, nil, err
	}

	profs, err := ds.ListAppleProfilesForReconcileByTeam(ctx, host.EffectiveTeamID())
	if err != nil {
		return nil, nil, err
	}

	profilesWithBrokenLabel := make(map[string]struct{})
	profilesByTeam := make(map[uint][]*fleet.AppleProfileForReconcile, 2)
	labelIDSet := make(map[uint]struct{})
	for _, p := range profs {
		profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)

		if p.HasBrokenLabel() {
			profilesWithBrokenLabel[p.ProfileUUID] = struct{}{}
		}

		for _, lr := range p.IncludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
		for _, lr := range p.ExcludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
	}
	labelIDs := make([]uint, 0, len(labelIDSet))
	for id := range labelIDSet {
		labelIDs = append(labelIDs, id)
	}

	hostLabels, err := ds.BulkGetHostLabelMemberships(ctx, []uint{host.HostID}, labelIDs)
	if err != nil {
		return nil, nil, err
	}

	currentByHost, err := ds.BulkGetHostMDMAppleProfilesByUUIDs(ctx, []string{host.UUID})
	if err != nil {
		return nil, nil, err
	}

	toInstall, toRemove = ComputeReconcileDeltas(
		[]*fleet.AppleHostReconcileInfo{host}, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel,
	)
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

	return toInstall, toRemove, nil
}
