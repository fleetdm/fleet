package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
)

// reconcileAppleDeclarationsBatchSize bounds how many hosts the batched
// DDM reconciliation cron processes per tick. Kept independent of the
// profile batch size so the two passes can be tuned separately.
var reconcileAppleDeclarationsBatchSize = 5000

// ReconcileAppleDeclarationsBatched is the cursor-based DDM equivalent of
// ReconcileAppleProfilesBatched. It pulls one bounded host window per
// tick, evaluates desired declaration state per host in memory using the
// SAME label/team/platform dispatcher and handlers as the profile
// reconciler (apple_mdm.EntityAppliesToHost), diffs against current
// host_mdm_apple_declarations rows, writes the diffed state, and kicks a
// DeclarativeManagement command on changed hosts so they fetch the new
// declarations.
//
// Sharing the apple_mdm.EntityAppliesToHost dispatcher is the whole point:
// profile and declaration label-membership semantics cannot drift because
// there's only one implementation.
func ReconcileAppleDeclarationsBatched(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
) (err error) {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return fmt.Errorf("reading app config: %w", err)
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil
	}

	cursor, err := ds.GetMDMAppleDeclarationReconcileCursor(ctx)
	if err != nil {
		logger.WarnContext(ctx, "failed to read apple MDM declaration reconcile cursor; starting from beginning", "err", err)
		cursor = ""
	}

	hosts, allDecls, hostLabels, currentByHost, err := ds.GetAppleDeclarationReconcileSnapshot(ctx, cursor, reconcileAppleDeclarationsBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "loading apple declaration reconcile snapshot")
	}
	logger.DebugContext(ctx, "ddm batched reconcile: loaded snapshot",
		"cursor", cursor, "hosts_in_batch", len(hosts), "declaration_count", len(allDecls))

	if len(hosts) == 0 {
		if cursor != "" {
			logger.DebugContext(ctx, "apple MDM declaration reconcile pass complete; resetting cursor", "cursor", cursor)
			if cerr := ds.SetMDMAppleDeclarationReconcileCursor(ctx, ""); cerr != nil {
				logger.WarnContext(ctx, "failed to reset apple MDM declaration reconcile cursor", "err", cerr)
			}
		}
		return nil
	}

	var nextCursor string
	if len(hosts) >= reconcileAppleDeclarationsBatchSize {
		nextCursor = hosts[len(hosts)-1].UUID
	}

	defer func() {
		switch {
		case err != nil:
			logger.WarnContext(ctx, "ddm batched reconcile: tick errored; cursor not advanced",
				"cursor", cursor, "next_cursor", nextCursor, "err", err)
		case cursor != nextCursor:
			if cerr := ds.SetMDMAppleDeclarationReconcileCursor(ctx, nextCursor); cerr != nil {
				logger.WarnContext(ctx, "failed to advance apple MDM declaration reconcile cursor", "err", cerr)
			} else {
				logger.DebugContext(ctx, "ddm batched reconcile: cursor advanced",
					"cursor", cursor, "next_cursor", nextCursor)
			}
		default:
			logger.DebugContext(ctx, "ddm batched reconcile: tick complete, cursor unchanged",
				"cursor", cursor)
		}
	}()

	declsWithBrokenLabel := make(map[string]struct{})
	declsByTeam := make(map[uint][]*fleet.AppleDeclarationForReconcile, 4)
	for _, d := range allDecls {
		declsByTeam[d.TeamID] = append(declsByTeam[d.TeamID], d)

		if d.HasBrokenLabel() {
			declsWithBrokenLabel[d.DeclarationUUID] = struct{}{}
		}
	}

	changedDeviceHostUUIDs, changedUserHostUUIDs, declRowsToWrite := apple_mdm.ComputeDeclarationDeltas(
		hosts, hostLabels, currentByHost, declsByTeam, declsWithBrokenLabel,
	)

	logger.DebugContext(ctx, "ddm batched reconcile: computed deltas",
		"changed_device_hosts", len(changedDeviceHostUUIDs),
		"changed_user_hosts", len(changedUserHostUUIDs),
		"host_decl_rows_to_write", len(declRowsToWrite))

	// NOTE: we intentionally do NOT early-return when there are no declaration
	// deltas. Hosts that requested a resync (MDMAppleHostDeclarationsGetAndClearResync
	// below) must still be poked even when nothing changed this tick — that's the
	// whole point of the resync flag, and leaving it unhandled would strand the
	// flag set forever. All the steps below no-op cheaply on empty inputs.

	// Memoized resolver: the user-channel enrollment ID for a host, or "" if the
	// host has no user channel yet. Shared between the delta and resync passes so
	// each host is looked up at most once.
	userEnrollmentByHost := make(map[string]string)
	getUserEnrollmentID := func(hostUUID string) (string, error) {
		if id, ok := userEnrollmentByHost[hostUUID]; ok {
			return id, nil
		}
		id := ""
		ue, err := ds.GetNanoMDMUserEnrollment(ctx, hostUUID)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting user enrollment for host")
		}
		if ue != nil {
			id = ue.ID
		}
		userEnrollmentByHost[hostUUID] = id
		return id, nil
	}

	// Decide user-channel delivery for hosts with user-scoped changes: deliver
	// now if the user channel exists, hold within the grace window, or fail with
	// a user-facing detail (iOS/iPadOS have no user channel; macOS past the grace
	// window with no user channel is a hard failure). This mutates the pending
	// user-scoped install rows in declRowsToWrite before they are written, and
	// returns any user-scoped removes that can't be delivered (no user channel)
	// so they can be deleted rather than left pending forever.
	userEnrollmentIDsToSend, failedUserDecls, userRemovesToDelete, err := resolveUserChannelDeliveries(
		ctx, logger, hosts, changedUserHostUUIDs, declRowsToWrite, getUserEnrollmentID,
	)
	if err != nil {
		return err
	}

	// Undeliverable user-scoped removes are deleted, not written as pending.
	writeRows := declRowsToWrite
	if len(userRemovesToDelete) > 0 {
		skip := make(map[*fleet.MDMAppleHostDeclaration]struct{}, len(userRemovesToDelete))
		for _, r := range userRemovesToDelete {
			skip[r] = struct{}{}
		}
		writeRows = make([]*fleet.MDMAppleHostDeclaration, 0, len(declRowsToWrite))
		for _, r := range declRowsToWrite {
			if _, ok := skip[r]; !ok {
				writeRows = append(writeRows, r)
			}
		}
	}

	if err := ds.BulkUpsertMDMAppleHostDeclarations(ctx, writeRows); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert host mdm apple declarations")
	}

	if err := ds.BulkDeleteMDMAppleHostDeclarations(ctx, userRemovesToDelete); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting undeliverable user-scoped declaration removals")
	}

	// The bulk upsert writes status but not detail, so persist the user-facing
	// detail for user-scoped declarations we failed above.
	for _, f := range failedUserDecls {
		if err := ds.SetHostMDMAppleDeclarationStatus(ctx, f.hostUUID, f.declarationUUID, &fleet.MDMDeliveryFailed, f.detail, nil); err != nil {
			return ctxerr.Wrap(ctx, err, "setting failed user-scoped declaration detail")
		}
	}

	// Find any hosts that requested a resync, partitioned by channel. This is
	// used to cover special cases where we're not 100% certain of the
	// declarations on the device.
	deviceResyncHosts, userResyncHosts, err := ds.MDMAppleHostDeclarationsGetAndClearResync(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting and clearing resync hosts")
	}

	// Device channel: the enrollment ID is the host UUID.
	deviceSend := dedupeStrings(append(changedDeviceHostUUIDs, deviceResyncHosts...))

	// User channel: resync hosts also need their user enrollment resolved (and
	// are skipped if the channel doesn't exist).
	for _, hostUUID := range userResyncHosts {
		userEnrollmentID, err := getUserEnrollmentID(hostUUID)
		if err != nil {
			return err
		}
		if userEnrollmentID != "" {
			userEnrollmentIDsToSend = append(userEnrollmentIDsToSend, userEnrollmentID)
		}
	}
	userSend := dedupeStrings(userEnrollmentIDsToSend)

	// TODO: Consider a similar approach to profiles where if failed to send the command for the host, reset the status so we resend it again.
	// now it will just end up in a state where it never retries to send the DeclarativeManagement command.
	if len(deviceSend) > 0 {
		if err := commander.DeclarativeManagement(ctx, deviceSend, uuid.NewString()); err != nil {
			return ctxerr.Wrap(ctx, err, "issuing DeclarativeManagement command (device channel)")
		}
		logger.InfoContext(ctx, "ddm batched reconcile: sent DeclarativeManagement command",
			"channel", "device", "host_count", len(deviceSend))
	}
	if len(userSend) > 0 {
		if err := commander.DeclarativeManagement(ctx, userSend, uuid.NewString()); err != nil {
			return ctxerr.Wrap(ctx, err, "issuing DeclarativeManagement command (user channel)")
		}
		logger.InfoContext(ctx, "ddm batched reconcile: sent DeclarativeManagement command",
			"channel", "user", "enrollment_count", len(userSend))
	}

	return nil
}

// failedUserDeclaration records a user-scoped declaration that couldn't be
// delivered so its user-facing detail can be persisted after the bulk upsert.
type failedUserDeclaration struct {
	hostUUID        string
	declarationUUID string
	detail          string
}

// resolveUserChannelDeliveries decides, per host with user-scoped declaration
// changes, whether to deliver on the user channel now, hold until the user
// channel materializes (within the grace window), or fail.
//
// For hosts whose user channel exists, the enrollment ID is returned to send a
// DeclarativeManagement command to.
//
// For hosts with no user channel it mutates the pending user-scoped INSTALL
// rows in declRowsToWrite in place: held rows get a nil status so the next
// reconcile tick retries once the user channel exists; failed rows get a failed
// status (their detail is returned to be persisted separately). User-scoped
// REMOVE rows for such hosts can't be delivered to a channel that doesn't
// exist, so they are returned in toDelete to be hard-deleted rather than left
// as permanent "pending" rows (mirrors how the profile reconciler cleans up
// undeliverable user-scoped profiles).
func resolveUserChannelDeliveries(
	ctx context.Context,
	logger *slog.Logger,
	hosts []*fleet.AppleHostReconcileInfo,
	changedUserHostUUIDs []string,
	declRowsToWrite []*fleet.MDMAppleHostDeclaration,
	getUserEnrollmentID func(hostUUID string) (string, error),
) (enrollmentIDsToSend []string, failed []failedUserDeclaration, toDelete []*fleet.MDMAppleHostDeclaration, err error) {
	if len(changedUserHostUUIDs) == 0 {
		return nil, nil, nil, nil
	}

	hostsByUUID := make(map[string]*fleet.AppleHostReconcileInfo, len(hosts))
	for _, h := range hosts {
		hostsByUUID[h.UUID] = h
	}

	userInstallRowsByHost := make(map[string][]*fleet.MDMAppleHostDeclaration)
	userRemoveRowsByHost := make(map[string][]*fleet.MDMAppleHostDeclaration)
	for _, row := range declRowsToWrite {
		if row.Scope != fleet.PayloadScopeUser {
			continue
		}
		switch row.OperationType {
		case fleet.MDMOperationTypeInstall:
			userInstallRowsByHost[row.HostUUID] = append(userInstallRowsByHost[row.HostUUID], row)
		case fleet.MDMOperationTypeRemove:
			userRemoveRowsByHost[row.HostUUID] = append(userRemoveRowsByHost[row.HostUUID], row)
		}
	}

	for _, hostUUID := range changedUserHostUUIDs {
		userEnrollmentID, gerr := getUserEnrollmentID(hostUUID)
		if gerr != nil {
			return nil, nil, nil, gerr
		}
		if userEnrollmentID != "" {
			enrollmentIDsToSend = append(enrollmentIDsToSend, userEnrollmentID)
			continue
		}

		// No user channel: a removal can't be delivered, so drop the row instead
		// of leaving a permanent pending tombstone.
		toDelete = append(toDelete, userRemoveRowsByHost[hostUUID]...)

		installRows := userInstallRowsByHost[hostUUID]
		if len(installRows) == 0 {
			// Nothing to install on this channel (e.g. the host is only here
			// because a scope flip poked the old user channel to drop a
			// declaration); no hold/fail decision to make.
			continue
		}
		host := hostsByUUID[hostUUID]

		switch {
		case host != nil && fleet.IsAppleMobilePlatform(host.Platform):
			for _, row := range installRows {
				row.Status = &fleet.MDMDeliveryFailed
				failed = append(failed, failedUserDeclaration{
					hostUUID: hostUUID, declarationUUID: row.DeclarationUUID,
					detail: "This setting couldn't be enforced because the user channel isn't available on iOS and iPadOS hosts.",
				})
			}

		case host != nil && host.DeviceEnrolledAt != nil &&
			time.Since(*host.DeviceEnrolledAt) < apple_mdm.HoursToWaitForUserEnrollmentAfterDeviceEnrollment*time.Hour:
			// Within the grace window: hold. Leaving a nil status makes the next
			// tick re-detect and retry once the user channel materializes.
			for _, row := range installRows {
				row.Status = nil
			}
			logger.DebugContext(ctx, "ddm batched reconcile: holding user-scoped declarations pending user channel",
				"host_uuid", hostUUID, "declaration_count", len(installRows))

		default:
			for _, row := range installRows {
				row.Status = &fleet.MDMDeliveryFailed
				failed = append(failed, failedUserDeclaration{
					hostUUID: hostUUID, declarationUUID: row.DeclarationUUID,
					detail: "This setting couldn't be enforced because the user channel doesn't exist for this host. Currently, Fleet creates the user channel for hosts that automatically enroll.",
				})
			}
			logger.WarnContext(ctx, "ddm batched reconcile: no user channel after grace window, failing user-scoped declarations",
				"host_uuid", hostUUID, "declaration_count", len(installRows))
		}
	}

	return enrollmentIDsToSend, failed, toDelete, nil
}

// dedupeStrings returns the input with duplicates removed, preserving order.
func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return in
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
