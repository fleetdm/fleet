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

// reconcileAppleDeclarationsBatchSize is the scan window: how many enrolled Apple hosts the DDM reconciler reads per snapshot.
// Kept independent of the profile batch size so the two passes can be tuned separately.
//
// var rather than const so tests can override it.
var reconcileAppleDeclarationsBatchSize = 2000

// reconcileAppleDeclarationsDeliveryCap bounds how many distinct hosts the DDM cron schedules work for per tick, smoothing
// writer pressure during bulk events the same way the profile reconciler's cap does. Set <= 0 to disable.
//
// var rather than const so tests can override it.
var reconcileAppleDeclarationsDeliveryCap = 2000

// reconcileAppleDeclarationsScanBudget is the wall-clock budget for a single tick's drain loop, so a no-work pass over the
// whole fleet finishes inside one tick instead of taking ceil(hosts/batch) ticks.
//
// Smaller than the profile reconciler's budget because both run in the same 30s schedule, along with device names — see
// newAppleMDMProfileManagerSchedule.
//
// var rather than const so tests can override it.
var reconcileAppleDeclarationsScanBudget = 8 * time.Second

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

	// Read the cursor; on error, treat as start-of-pass and continue. A stale or missing cursor is harmless because the
	// in-memory diff writes only what actually differs from the current state.
	entryCursor, cerr := ds.GetMDMAppleDeclarationReconcileCursor(ctx)
	if cerr != nil {
		logger.WarnContext(ctx, "failed to read apple MDM declaration reconcile cursor; starting from beginning", "err", cerr)
		entryCursor = ""
	}

	cursor := entryCursor
	// commitCursor is the cursor value to persist at tick end. It advances only past windows that were fully delivered; the
	// deferred write fires only when err == nil, so an error leaves the cursor untouched and the next tick re-scans from the
	// same point. Re-scanning is cheap and idempotent since delivered work is now pending.
	commitCursor := entryCursor

	defer func() {
		switch {
		case err != nil:
			logger.WarnContext(ctx, "ddm batched reconcile: tick errored; cursor not advanced",
				"cursor", entryCursor, "err", err)
		case commitCursor != entryCursor:
			if serr := ds.SetMDMAppleDeclarationReconcileCursor(ctx, commitCursor); serr != nil {
				logger.WarnContext(ctx, "failed to advance apple MDM declaration reconcile cursor", "err", serr)
			} else {
				logger.DebugContext(ctx, "ddm batched reconcile: cursor advanced",
					"cursor", entryCursor, "next_cursor", commitCursor)
			}
		default:
			logger.DebugContext(ctx, "ddm batched reconcile: tick complete, cursor unchanged", "cursor", entryCursor)
		}
	}()

	// Memoized resolver: the user-channel enrollment ID for a host, or "" if the
	// host has no user channel yet. Tick-scoped (not per window) so the resync pass
	// below reuses the lookups the drain loop already paid for.
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

	// Accumulated across every window drained this tick, so a bulk change sends one DeclarativeManagement command per channel
	// per tick rather than one per window.
	var deviceSendAccum, userSendAccum []string

	deadline := time.Now().Add(reconcileAppleDeclarationsScanBudget)
	deliveredHosts := 0

	for drain := true; drain; {
		hosts, allDecls, hostLabels, currentByHost, pageFull, serr := ds.GetAppleDeclarationReconcileSnapshot(ctx, cursor, reconcileAppleDeclarationsBatchSize)
		if serr != nil {
			err = ctxerr.Wrap(ctx, serr, "loading apple declaration reconcile snapshot")
			return err
		}
		logger.DebugContext(ctx, "ddm batched reconcile: loaded snapshot",
			"cursor", cursor, "hosts_in_batch", len(hosts), "declaration_count", len(allDecls))

		if len(hosts) == 0 {
			// Reached the end of the host space (or empty fleet): reset the cursor so the next pass restarts from the beginning.
			commitCursor = ""
			break
		}

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

		// NOTE: unlike the old single-window version, an empty-delta window may fall through cheaply here. The resync flag is
		// serviced once per tick after the loop, so it can no longer be stranded by an early exit from one window.

		// Apply the per-tick delivery cap at host granularity, before any user-channel resolution, so we don't pay for work we
		// would only discard. Hosts come back ascending by uuid, so capping keeps a contiguous prefix of the work-hosts and the
		// cursor can resume at the last delivered host.
		workHosts := declarationHostsWithWork(hosts, changedDeviceHostUUIDs, changedUserHostUUIDs)
		// Advance past the whole window by default; pageFull (not len(hosts)) decides end-of-space, because duplicate-UUID host
		// rows are collapsed after the SQL LIMIT and a full page can dedupe to fewer than batchSize hosts.
		advanceTo := hosts[len(hosts)-1].UUID

		partial := false
		if reconcileAppleDeclarationsDeliveryCap > 0 {
			// Invariant: deliveredHosts < cap here. We stop below as soon as it reaches the cap. So remaining >= 1.
			remaining := reconcileAppleDeclarationsDeliveryCap - deliveredHosts
			if len(workHosts) > remaining {
				allowed := make(map[string]struct{}, remaining)
				for _, h := range workHosts[:remaining] {
					allowed[h] = struct{}{}
				}
				changedDeviceHostUUIDs = filterStringsBySet(changedDeviceHostUUIDs, allowed)
				changedUserHostUUIDs = filterStringsBySet(changedUserHostUUIDs, allowed)
				declRowsToWrite = filterDeclarationRowsByHost(declRowsToWrite, allowed)
				advanceTo = workHosts[remaining-1] // resume after the last delivered host
				workHosts = workHosts[:remaining]
				partial = true
			}
		}

		// Decide user-channel delivery for hosts with user-scoped changes: deliver
		// now if the user channel exists, hold within the grace window, or fail with
		// a user-facing detail (iOS/iPadOS have no user channel; macOS past the grace
		// window with no user channel is a hard failure). This mutates the pending
		// user-scoped install rows in declRowsToWrite before they are written, and
		// returns any user-scoped removes that can't be delivered (no user channel)
		// so they can be deleted rather than left pending forever.
		userEnrollmentIDsToSend, failedUserDecls, userRemovesToDelete, uerr := resolveUserChannelDeliveries(
			ctx, logger, hosts, changedUserHostUUIDs, declRowsToWrite, getUserEnrollmentID,
		)
		if uerr != nil {
			err = uerr
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

		if werr := ds.BulkUpsertMDMAppleHostDeclarations(ctx, writeRows); werr != nil {
			err = ctxerr.Wrap(ctx, werr, "bulk upsert host mdm apple declarations")
			return err
		}

		if derr := ds.BulkDeleteMDMAppleHostDeclarations(ctx, userRemovesToDelete); derr != nil {
			err = ctxerr.Wrap(ctx, derr, "deleting undeliverable user-scoped declaration removals")
			return err
		}

		// The bulk upsert writes status but not detail, so persist the user-facing
		// detail for user-scoped declarations we failed above.
		for _, f := range failedUserDecls {
			if serr := ds.SetHostMDMAppleDeclarationStatus(ctx, f.hostUUID, f.declarationUUID, &fleet.MDMDeliveryFailed, f.detail, nil); serr != nil {
				err = ctxerr.Wrap(ctx, serr, "setting failed user-scoped declaration detail")
				return err
			}
		}

		deviceSendAccum = append(deviceSendAccum, changedDeviceHostUUIDs...)
		userSendAccum = append(userSendAccum, userEnrollmentIDsToSend...)
		deliveredHosts += len(workHosts)

		// Advance only after the window's writes succeeded.
		commitCursor = advanceTo
		cursor = advanceTo

		switch {
		case partial:
			// Delivery cap hit mid-window; the un-delivered remainder resumes next tick from cursor = advanceTo.
			drain = false
		case !pageFull:
			// Short page => end of the host space; reset for the next pass.
			commitCursor = ""
			drain = false
		case reconcileAppleDeclarationsDeliveryCap > 0 && deliveredHosts >= reconcileAppleDeclarationsDeliveryCap:
			// Delivery cap reached exactly at a window boundary.
			drain = false
		case time.Now().After(deadline):
			// Scan budget exhausted; resume next tick from cursor = advanceTo.
			drain = false
		}
		// Otherwise keep draining the next window within this tick.
	}

	// Find any hosts that requested a resync, partitioned by channel. This is
	// used to cover special cases where we're not 100% certain of the
	// declarations on the device.
	//
	// Deliberately outside the drain loop: this is a fleet-wide, destructive read-and-clear, not scoped to the current host
	// window. Inside the loop the first window would claim and clear the flags for the whole fleet and every later window
	// would re-query for nothing.
	deviceResyncHosts, userResyncHosts, rerr := ds.MDMAppleHostDeclarationsGetAndClearResync(ctx)
	if rerr != nil {
		err = ctxerr.Wrap(ctx, rerr, "getting and clearing resync hosts")
		return err
	}

	// Device channel: the enrollment ID is the host UUID.
	deviceSend := dedupeStrings(append(deviceSendAccum, deviceResyncHosts...))

	// User channel: resync hosts also need their user enrollment resolved (and
	// are skipped if the channel doesn't exist).
	for _, hostUUID := range userResyncHosts {
		userEnrollmentID, uerr := getUserEnrollmentID(hostUUID)
		if uerr != nil {
			err = uerr
			return err
		}
		if userEnrollmentID != "" {
			userSendAccum = append(userSendAccum, userEnrollmentID)
		}
	}
	userSend := dedupeStrings(userSendAccum)

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

// declarationHostsWithWork returns the host UUIDs that have at least one device- or user-channel declaration change in this
// window, in the order hosts are given (ascending by uuid). The drain loop uses this both to count delivered hosts against the
// cap and to pick the contiguous prefix to deliver when the cap is reached mid-window.
func declarationHostsWithWork(hosts []*fleet.AppleHostReconcileInfo, changedDeviceHostUUIDs, changedUserHostUUIDs []string) []string {
	work := make(map[string]struct{}, len(changedDeviceHostUUIDs)+len(changedUserHostUUIDs))
	for _, u := range changedDeviceHostUUIDs {
		work[u] = struct{}{}
	}
	for _, u := range changedUserHostUUIDs {
		work[u] = struct{}{}
	}
	ordered := make([]string, 0, len(work))
	for _, h := range hosts {
		if _, ok := work[h.UUID]; ok {
			ordered = append(ordered, h.UUID)
		}
	}
	return ordered
}

// filterStringsBySet returns only the entries present in allowed, preserving order.
func filterStringsBySet(in []string, allowed map[string]struct{}) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := allowed[s]; ok {
			out = append(out, s)
		}
	}
	return out
}

// filterDeclarationRowsByHost returns only the rows whose HostUUID is in the allowed set, preserving order. Filtering by host
// (rather than by row) keeps every row for a delivered host together, so a host is never written half-reconciled.
func filterDeclarationRowsByHost(rows []*fleet.MDMAppleHostDeclaration, allowed map[string]struct{}) []*fleet.MDMAppleHostDeclaration {
	out := make([]*fleet.MDMAppleHostDeclaration, 0, len(rows))
	for _, r := range rows {
		if _, ok := allowed[r.HostUUID]; ok {
			out = append(out, r)
		}
	}
	return out
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
