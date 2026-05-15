package service

import (
	"bytes"
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/google/uuid"
)

// reconcileAppleProfilesBatchSize bounds how many distinct hosts the
// batched Apple MDM reconciliation cron processes per tick. The cron uses
// a host_uuid cursor (persisted in Redis via the mysqlredis wrapper) to
// page through the host universe in batches, smoothing the writer pressure
// that the legacy unbounded reconciliation generates during bulk events
// (team transfers, profile changes).
//
// var (not const) so tests can override it.
var reconcileAppleProfilesBatchSize = 5000

// ReconcileAppleProfilesBatched is an alternative implementation of
// ReconcileAppleProfiles that:
//
//   - Pulls a bounded batch of Apple-MDM-enrolled host UUIDs each tick
//     (via a host_uuid cursor persisted in Redis), instead of computing
//     desired-vs-current state across the entire host population in one
//     giant SQL UNION.
//   - Loads label memberships, team membership, and the profile catalog
//     for the batch and evaluates per-host desired state in memory using
//     small, single-responsibility handlers (one per label mode).
//   - Computes the install/remove delta against the host's current
//     host_mdm_apple_profiles rows in memory, then feeds the deltas into
//     the existing ProcessAndEnqueueProfiles path so all of the variable
//     substitution, CA throttle, user-enrollment, and "host being set up"
//     handling stays identical.
//
// The function is intended to be A/B-tested against ReconcileAppleProfiles
// via FLEET_MDM_APPLE_BATCHED_RECONCILER=true. When the cursor reaches the
// end of the universe it is reset so the next tick starts from the
// beginning. The returned error is non-nil only for fatal-to-this-tick
// failures; per-host issues are logged and skipped.
func ReconcileAppleProfilesBatched(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	redisKeyValue fleet.AdvancedKeyValueStore,
	logger *slog.Logger,
	certProfilesLimit int,
) (err error) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil
	}

	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetCACert,
	}, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting Apple SCEP")
	}
	block, _ := pem.Decode(assets[fleet.MDMAssetCACert].Value)
	if block == nil || block.Type != "CERTIFICATE" {
		return ctxerr.Wrap(ctx, errors.New("failed to decode PEM block from SCEP certificate"), "")
	}
	if err := ensureFleetProfiles(ctx, ds, logger, block.Bytes); err != nil {
		logger.ErrorContext(ctx, "unable to ensure fleetd configuration profiles are in place", "details", err)
	}

	// Read cursor (treat any error as a fresh start).
	cursor, err := ds.GetMDMAppleReconcileCursor(ctx)
	if err != nil {
		logger.WarnContext(ctx, "failed to read apple MDM reconcile cursor; starting from beginning", "err", err)
		cursor = ""
	}

	hosts, err := ds.ListAppleMDMHostsForReconcileBatch(ctx, cursor, reconcileAppleProfilesBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple MDM hosts for reconcile batch")
	}

	if len(hosts) == 0 {
		// End of pass or empty universe — reset cursor if it was advanced.
		if cursor != "" {
			logger.InfoContext(ctx, "apple MDM reconcile pass complete; resetting cursor", "cursor", cursor)
			if cerr := ds.SetMDMAppleReconcileCursor(ctx, ""); cerr != nil {
				logger.WarnContext(ctx, "failed to reset apple MDM reconcile cursor", "err", cerr)
			}
		}
		return nil
	}

	// Compute next cursor before doing the work so we can advance on
	// success. If we got fewer than the batch size, the universe fits in
	// this tick and the next tick should restart at "".
	var nextCursor string
	if len(hosts) >= reconcileAppleProfilesBatchSize {
		nextCursor = hosts[len(hosts)-1].UUID
	}

	// Defer cursor advance on success only; on error we leave the cursor
	// untouched so the next tick retries the same window.
	defer func() {
		if err == nil && cursor != nextCursor {
			if cerr := ds.SetMDMAppleReconcileCursor(ctx, nextCursor); cerr != nil {
				logger.WarnContext(ctx, "failed to advance apple MDM reconcile cursor", "err", cerr)
			}
		}
	}()

	if cursor != "" || nextCursor != "" {
		logger.InfoContext(ctx, "apple MDM reconcile tick using cursor",
			"cursor", cursor,
			"next_cursor", nextCursor,
			"batch_size", reconcileAppleProfilesBatchSize,
			"hosts_in_batch", len(hosts),
		)
	}

	// Load all Apple profiles + their labels (one query per tick). Profile
	// counts are small relative to host counts, so we don't bother
	// paginating these.
	allProfiles, err := ds.ListAppleProfilesForReconcile(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple profiles for reconcile")
	}

	// Group profiles by team_id (0 = global). Each host evaluates only
	// global + its team's profiles.
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

	// Collect host IDs and UUIDs for the batch.
	hostIDs := make([]uint, 0, len(hosts))
	hostUUIDs := make([]string, 0, len(hosts))
	hostsByUUID := make(map[string]*fleet.AppleHostReconcileInfo, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.HostID)
		hostUUIDs = append(hostUUIDs, h.UUID)
		hostsByUUID[h.UUID] = h
	}

	// Collect label IDs we actually care about.
	labelIDs := make([]uint, 0, len(labelIDSet))
	for id := range labelIDSet {
		labelIDs = append(labelIDs, id)
	}

	// Load host-label memberships and current host_mdm_apple_profiles rows
	// for the batch.
	hostLabels, err := ds.BulkGetHostLabelMemberships(ctx, hostIDs, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host label memberships")
	}

	currentByHost, err := ds.BulkGetHostMDMAppleProfilesByUUIDs(ctx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host mdm apple profiles")
	}

	// Compute desired state per host and diff against current.
	toInstall, toRemove := computeAppleReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam)

	// Filter out macOS-only profiles from iOS/iPadOS hosts (matches legacy).
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

	if len(toInstall) == 0 && len(toRemove) == 0 {
		// Nothing to do this tick. Still advance the cursor.
		return nil
	}

	// From here on, everything mirrors the legacy ReconcileAppleProfiles
	// post-listing logic so all downstream behaviour (CA throttle, user-
	// enrollment fallback, host-being-set-up skip, retry on failed enqueue)
	// is identical.
	return executeAppleReconcileBatch(
		ctx, ds, commander, redisKeyValue, logger,
		appConfig, certProfilesLimit, toInstall, toRemove,
	)
}

// computeAppleReconcileDeltas evaluates desired state for each host in the
// batch using the in-code label handlers, then diffs against the host's
// current host_mdm_apple_profiles rows to produce install and remove sets.
//
// Package-private so tests can exercise the in-memory logic without a
// database round-trip.
func computeAppleReconcileDeltas(
	hosts []*fleet.AppleHostReconcileInfo,
	hostLabels map[uint]map[uint]struct{},
	currentByHost map[string][]*fleet.MDMAppleProfilePayload,
	profilesByTeam map[uint][]*fleet.AppleProfileForReconcile,
) (toInstall, toRemove []*fleet.MDMAppleProfilePayload) {
	for _, host := range hosts {
		// Build the host's desired profile set by running each applicable
		// profile through the appropriate label handler.
		teamProfiles := profilesByTeam[host.EffectiveTeamID()]
		desired := make(map[string]*fleet.AppleProfileForReconcile, len(teamProfiles))

		labelsForHost := hostLabels[host.HostID] // may be nil

		for _, p := range teamProfiles {
			if !appleProfileAppliesToHost(p, host, labelsForHost) {
				continue
			}
			desired[p.ProfileUUID] = p
		}

		current := currentByHost[host.UUID] // []*MDMAppleProfilePayload, may be nil
		currentByProfile := make(map[string]*fleet.MDMAppleProfilePayload, len(current))
		for _, c := range current {
			currentByProfile[c.ProfileUUID] = c
		}

		// INSTALL set: desired profiles where (no current row) OR
		// (checksum differs) OR (secrets_updated_at advanced) OR
		// (current operation_type is remove OR NULL) OR
		// (current operation_type is install with NULL status).
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

		// REMOVE set: current rows whose profile is no longer in desired,
		// excluding rows whose current state is already a remove in a
		// terminal/pending state, and excluding rows for broken
		// label-based profiles (those linger by design).
		for profUUID, c := range currentByProfile {
			if _, stillDesired := desired[profUUID]; stillDesired {
				continue
			}

			// Match legacy "except remove operations in a terminal state
			// or already pending" — skip if operation is remove with a
			// non-NULL status.
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}

			// Skip broken label-based profiles (preserve legacy behavior:
			// broken profiles are never removed automatically).
			if isBrokenAppleProfile(profUUID, profilesByTeam) {
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

// appleProfileAppliesToHost is the top-level dispatcher. It applies the
// team and platform gates, then composes whichever include + exclude
// label handlers the profile carries. A profile may carry both an
// include set (in one consistent mode) and an exclude set; both gates
// must pass for the profile to apply.
func appleProfileAppliesToHost(
	p *fleet.AppleProfileForReconcile,
	host *fleet.AppleHostReconcileInfo,
	hostLabels map[uint]struct{},
) bool {
	// Team gate (already filtered upstream by profilesByTeam, but
	// double-check so handlers stay composable in isolation).
	if p.TeamID != host.EffectiveTeamID() {
		return false
	}

	// Platform gate. macOS-only profiles on iOS/iPadOS hosts are filtered
	// later by FilterMacOSOnlyProfilesFromIOSIPadOS.
	if !isAppleProfileEligiblePlatform(host.Platform) {
		return false
	}

	// Include gate: only run if the profile has include labels.
	if p.IncludeMode != fleet.AppleProfileIncludeNone {
		var ok bool
		switch p.IncludeMode {
		case fleet.AppleProfileIncludeAll:
			ok = appleProfileHandlerIncludeAll(p.IncludeLabels, hostLabels)
		case fleet.AppleProfileIncludeAny:
			ok = appleProfileHandlerIncludeAny(p.IncludeLabels, hostLabels)
		default:
			return false
		}
		if !ok {
			return false
		}
	}

	// Exclude gate: only run if the profile has exclude labels.
	if len(p.ExcludeLabels) > 0 {
		if !appleProfileHandlerExcludeAny(p.ExcludeLabels, host, hostLabels) {
			return false
		}
	}

	return true
}

func isAppleProfileEligiblePlatform(platform string) bool {
	return platform == "darwin" || platform == "ios" || platform == "ipados"
}

// appleProfileHandlerIncludeAll: host must be a member of every (non-
// broken) include label. A broken label disqualifies the profile, mirroring
// the legacy SQL where include-* with a broken label produces no desired-
// state row.
func appleProfileHandlerIncludeAll(labels []fleet.AppleProfileLabelRef, hostLabels map[uint]struct{}) bool {
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

// appleProfileHandlerIncludeAny: host must be a member of at least one
// include label. Broken labels can't match (host can't be a member of a
// deleted label) so they're silently skipped.
func appleProfileHandlerIncludeAny(labels []fleet.AppleProfileLabelRef, hostLabels map[uint]struct{}) bool {
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

// appleProfileHandlerExcludeAny: profile passes the exclude gate when the
// host is NOT a member of any referenced label, with two safety rules:
//
//   - Any broken exclude label disqualifies the profile entirely (we can't
//     prove the exclusion).
//   - Dynamic labels created after the host's last label scan are treated
//     as "results not yet reported" — also disqualify, so we don't
//     install a profile that the not-yet-scanned label would exclude.
//     Manual labels (membership_type=1) skip this timing check.
//
// Returns true when called with an empty slice — the dispatcher won't do
// that, but the handler's contract should still be "no exclusions = pass".
func appleProfileHandlerExcludeAny(
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

// isBrokenAppleProfile returns true if any label assignment on the profile
// references a deleted label. Used to skip removal of broken profiles
// (matches legacy behavior).
func isBrokenAppleProfile(profileUUID string, profilesByTeam map[uint][]*fleet.AppleProfileForReconcile) bool {
	for _, ps := range profilesByTeam {
		for _, p := range ps {
			if p.ProfileUUID != profileUUID {
				continue
			}
			return p.HasBrokenLabel()
		}
	}
	// Profile not found in catalog at all — the row in host_mdm_apple_profiles
	// is for a profile that has been deleted from the team or globally. It
	// is safe to remove (it is NOT a "broken label" case).
	return false
}

// executeAppleReconcileBatch runs the legacy post-listing logic against the
// in-memory toInstall/toRemove sets produced by the batched listing path.
// It is intentionally a near-clone of the corresponding section of
// ReconcileAppleProfiles so that semantic parity is easy to audit.
func executeAppleReconcileBatch(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	redisKeyValue fleet.AdvancedKeyValueStore,
	logger *slog.Logger,
	appConfig *fleet.AppConfig,
	certProfilesLimit int,
	toInstall, toRemove []*fleet.MDMAppleProfilePayload,
) error {
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
		if prof.DeviceEnrolledAt != nil && time.Since(*prof.DeviceEnrolledAt) < hoursToWaitForUserEnrollmentAfterDeviceEnrollment*time.Hour {
			return true, nil
		}
		return false, nil
	}

	hostProfiles := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(toInstall)+len(toRemove))

	profileIntersection := apple_mdm.NewProfileBimap()
	profileIntersection.IntersectByIdentifierAndHostUUID(toInstall, toRemove)

	hostProfilesToCleanup := []*fleet.MDMAppleProfilePayload{}
	hostProfilesToInstallMap := make(map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, len(toInstall))

	// Pre-fetch contents for CA classification when CA throttling is on.
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
			return ctxerr.Wrap(ctx, err, "getting profile contents for CA classification")
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
		if pp, ok := profileIntersection.GetMatchingProfileInCurrentState(p); ok {
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
			return err
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
				return err
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
				return err
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

	// Skip hosts currently being set up (Redis MGet).
	const isBeingSetupBatchSize = 1000
	for i := 0; i < len(hostProfiles); i += isBeingSetupBatchSize {
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
			return ctxerr.Wrap(ctx, err, "filtering hosts being set up")
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
			return ctxerr.Wrap(ctx, err, "deleting nano commands without results")
		}
	}
	if err := ds.BulkDeleteMDMAppleHostsConfigProfiles(ctx, hostProfilesToCleanup); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting profiles that didn't change")
	}

	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
		return ctxerr.Wrap(ctx, err, "updating host profiles")
	}

	enqueueResult, err := apple_mdm.ProcessAndEnqueueProfiles(
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
		for _, hp := range hostProfiles {
			if hp.Status != nil && *hp.Status == fleet.MDMDeliveryPending {
				hp.Status = nil
				hp.CommandUUID = ""
			}
		}
		if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, hostProfiles); err != nil {
			return ctxerr.Wrap(ctx, err, "reverting host profiles after failed enqueue")
		}
		return ctxerr.Wrap(ctx, err, "processing and enqueuing profiles")
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
		return ctxerr.Wrap(ctx, err, "reverting status of failed profiles")
	}

	return nil
}
