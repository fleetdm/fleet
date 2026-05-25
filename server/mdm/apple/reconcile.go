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
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/google/uuid"
)

// HoursToWaitForUserEnrollmentAfterDeviceEnrollment is how long the
// reconciler waits for a user-channel enrollment to materialise before
// failing user-scoped profile delivery on that host. Mirrors the
// constant the legacy service-package reconciler uses.
const HoursToWaitForUserEnrollmentAfterDeviceEnrollment = 2

// EntityAppliesToHost is the SHARED top-level dispatcher for Apple MDM
// label-gated entities (profiles and declarations). It applies team and
// platform gates, then composes the include + exclude label handlers
// carried by the entity.
//
// Both the batched profile and declaration reconcilers — and the per-host
// enrollment path — route through this function. The handlers themselves
// operate on []AppleProfileLabelRef and are entity-agnostic. This is the
// single source of truth for "does this Apple MDM entity apply to this
// host" so label / team / platform semantics cannot drift across paths.
func EntityAppliesToHost(
	e fleet.AppleLabeledEntity,
	host *fleet.AppleHostReconcileInfo,
	hostLabels map[uint]struct{},
) bool {
	if e.GetTeamID() != host.EffectiveTeamID() {
		return false
	}
	if !IsEligiblePlatform(host.Platform) {
		return false
	}

	if e.GetIncludeMode() != fleet.AppleProfileIncludeNone {
		var ok bool
		switch e.GetIncludeMode() {
		case fleet.AppleProfileIncludeAll:
			ok = HandlerIncludeAll(e.GetIncludeLabels(), hostLabels)
		case fleet.AppleProfileIncludeAny:
			ok = HandlerIncludeAny(e.GetIncludeLabels(), hostLabels)
		default:
			return false
		}
		if !ok {
			return false
		}
	}

	if exc := e.GetExcludeLabels(); len(exc) > 0 {
		if !HandlerExcludeAny(exc, host, hostLabels) {
			return false
		}
	}

	return true
}

// IsEligiblePlatform reports whether the host's platform is one of the
// Apple platforms this reconciler can manage.
func IsEligiblePlatform(platform string) bool {
	return platform == "darwin" || platform == "ios" || platform == "ipados"
}

// HandlerIncludeAll: host must be a member of every (non-broken) include
// label. A broken label disqualifies the entity, mirroring the legacy SQL
// where include-* with a broken label produces no desired-state row.
func HandlerIncludeAll(labels []fleet.AppleProfileLabelRef, hostLabels map[uint]struct{}) bool {
	if len(labels) == 0 {
		return false
	}
	for _, l := range labels {
		if l.LabelID == nil {
			return false
		}
		if _, ok := hostLabels[*l.LabelID]; !ok {
			return false
		}
	}
	return true
}

// HandlerIncludeAny: host must be a member of at least one include label.
// Broken labels can't match (host can't be a member of a deleted label)
// so they're silently skipped.
func HandlerIncludeAny(labels []fleet.AppleProfileLabelRef, hostLabels map[uint]struct{}) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			continue
		}
		if _, ok := hostLabels[*l.LabelID]; ok {
			return true
		}
	}
	return false
}

// HandlerExcludeAny: entity passes the exclude gate when the host is NOT
// a member of any referenced label, with two safety rules:
//
//   - Any broken exclude label disqualifies the entity entirely (we can't
//     prove the exclusion).
//   - Dynamic labels created after the host's last label scan are treated
//     as "results not yet reported" — also disqualify, so we don't
//     install a profile that the not-yet-scanned label would exclude.
//     Manual labels (membership_type=1) skip this timing check.
func HandlerExcludeAny(
	labels []fleet.AppleProfileLabelRef,
	host *fleet.AppleHostReconcileInfo,
	hostLabels map[uint]struct{},
) bool {
	for _, l := range labels {
		if l.LabelID == nil {
			return false
		}
		if l.LabelMembershipType != 1 && !l.CreatedAt.IsZero() && host.LabelUpdatedAt.Before(l.CreatedAt) {
			return false
		}
		if _, isMember := hostLabels[*l.LabelID]; isMember {
			return false
		}
	}
	return true
}

// ComputeReconcileDeltas evaluates desired profile state for each host in
// the input set using the SHARED dispatcher, then diffs against current
// host_mdm_apple_profiles rows to produce install and remove sets.
func ComputeReconcileDeltas(
	hosts []*fleet.AppleHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMAppleProfilePayload,
	profilesByTeam map[uint][]*fleet.AppleProfileForReconcile,
) (toInstall, toRemove []*fleet.MDMAppleProfilePayload) {
	for _, host := range hosts {
		teamProfiles := profilesByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.AppleProfileForReconcile, len(teamProfiles))

		labelsForHost := hostLabels[host.HostID]

		for _, p := range teamProfiles {
			if !EntityAppliesToHost(p, host, labelsForHost) {
				continue
			}
			desired[p.ProfileUUID] = p
		}

		current := currentByHost[host.UUID]
		currentByProfile := make(map[string]*fleet.MDMAppleProfilePayload, len(current))
		for _, c := range current {
			currentByProfile[c.ProfileUUID] = c
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
			if IsBrokenProfile(profUUID, profilesByTeam) {
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
func IsBrokenProfile(profileUUID string, profilesByTeam map[uint][]*fleet.AppleProfileForReconcile) bool {
	for _, ps := range profilesByTeam {
		for _, p := range ps {
			if p.ProfileUUID != profileUUID {
				continue
			}
			return p.HasBrokenLabel()
		}
	}
	return false
}

// IsBrokenDeclaration is the DDM equivalent of IsBrokenProfile.
func IsBrokenDeclaration(declUUID string, declsByTeam map[uint][]*fleet.AppleDeclarationForReconcile) bool {
	for _, ds := range declsByTeam {
		for _, d := range ds {
			if d.DeclarationUUID != declUUID {
				continue
			}
			return d.HasBrokenLabel()
		}
	}
	return false
}

// ComputeDeclarationDeltas is the DDM equivalent of ComputeReconcileDeltas.
// Uses the SAME shared dispatcher (EntityAppliesToHost) so profile and
// declaration label semantics cannot drift.
func ComputeDeclarationDeltas(
	hosts []*fleet.AppleHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMAppleHostDeclaration,
	declsByTeam map[uint][]*fleet.AppleDeclarationForReconcile,
) (changedHostUUIDs []string, declRowsToWrite []*fleet.MDMAppleHostDeclaration) {
	pendingStatus := fleet.MDMDeliveryPending
	changedSet := make(map[string]struct{})

	for _, host := range hosts {
		teamDecls := declsByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.AppleDeclarationForReconcile, len(teamDecls))

		labelsForHost := hostLabels[host.HostID]

		for _, d := range teamDecls {
			if !EntityAppliesToHost(d, host, labelsForHost) {
				continue
			}
			desired[d.DeclarationUUID] = d
		}

		current := currentByHost[host.UUID]
		currentByDecl := make(map[string]*fleet.MDMAppleHostDeclaration, len(current))
		for _, c := range current {
			currentByDecl[c.DeclarationUUID] = c
		}

		for declUUID, d := range desired {
			c, present := currentByDecl[declUUID]
			needsInstall := false
			switch {
			case !present:
				needsInstall = true
			case !bytes.Equal([]byte(c.Token), d.Token):
				needsInstall = true
			case d.SecretsUpdatedAt != nil && (c.SecretsUpdatedAt == nil || c.SecretsUpdatedAt.Before(*d.SecretsUpdatedAt)):
				needsInstall = true
			case c.OperationType == "" || c.OperationType == fleet.MDMOperationTypeRemove:
				needsInstall = true
			case c.OperationType == fleet.MDMOperationTypeInstall && c.Status == nil:
				needsInstall = true
			}
			if !needsInstall {
				continue
			}

			declRowsToWrite = append(declRowsToWrite, &fleet.MDMAppleHostDeclaration{
				HostUUID:         host.UUID,
				DeclarationUUID:  d.DeclarationUUID,
				Name:             d.DeclarationName,
				Identifier:       d.DeclarationIdentifier,
				Status:           &pendingStatus,
				OperationType:    fleet.MDMOperationTypeInstall,
				Token:            string(d.Token),
				SecretsUpdatedAt: d.SecretsUpdatedAt,
			})
			changedSet[host.UUID] = struct{}{}
		}

		for declUUID, c := range currentByDecl {
			if _, stillDesired := desired[declUUID]; stillDesired {
				continue
			}
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			if IsBrokenDeclaration(declUUID, declsByTeam) {
				continue
			}

			declRowsToWrite = append(declRowsToWrite, &fleet.MDMAppleHostDeclaration{
				HostUUID:         host.UUID,
				DeclarationUUID:  c.DeclarationUUID,
				Name:             c.Name,
				Identifier:       c.Identifier,
				Status:           &pendingStatus,
				OperationType:    fleet.MDMOperationTypeRemove,
				Token:            c.Token,
				SecretsUpdatedAt: c.SecretsUpdatedAt,
			})
			changedSet[host.UUID] = struct{}{}
		}
	}

	changedHostUUIDs = make([]string, 0, len(changedSet))
	for u := range changedSet {
		changedHostUUIDs = append(changedHostUUIDs, u)
	}
	return changedHostUUIDs, declRowsToWrite
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
			if pp.Status != &fleet.MDMDeliveryFailed && bytes.Equal(pp.Checksum, p.Checksum) {
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

	logger.InfoContext(ctx, "batched reconcile: before bulk upsert",
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

// ReconcileProfilesForHost is the per-host orchestrator. Both the
// apple_mdm worker (post-enrollment) and any other caller that wants to
// reconcile a single host call this directly — no dependency injection,
// no import cycle. The shared compute/handlers/dispatcher above guarantee
// the same desired-state semantics as the batched cron reconciler.
func ReconcileProfilesForHost(
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

	allProfiles, err := ds.ListAppleProfilesForReconcile(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing apple profiles for reconcile")
	}

	profilesByTeam := make(map[uint][]*fleet.AppleProfileForReconcile, 4)
	labelIDSet := make(map[uint]struct{})
	for _, p := range allProfiles {
		profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)
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
		[]*fleet.AppleHostReconcileInfo{host}, hostLabels, currentByHost, profilesByTeam,
	)
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

	if len(toInstall) == 0 && len(toRemove) == 0 {
		return nil, nil
	}

	// nil redisKeyValue → skip the "host being set up" check. By
	// construction this host IS being set up.
	return ExecuteReconcileBatch(
		ctx, ds, commander, nil, logger,
		appConfig, certProfilesLimit, toInstall, toRemove,
	)
}
