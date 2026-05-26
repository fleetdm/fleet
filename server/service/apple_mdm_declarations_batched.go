package service

import (
	"context"
	"fmt"
	"log/slog"

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

	hosts, err := ds.ListAppleMDMHostsForReconcileBatch(ctx, cursor, reconcileAppleDeclarationsBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple MDM hosts for ddm reconcile batch")
	}
	logger.DebugContext(ctx, "ddm batched reconcile: listed hosts",
		"cursor", cursor, "hosts_in_batch", len(hosts))

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

	allDecls, err := ds.ListAppleDeclarationsForReconcile(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple declarations for reconcile")
	}
	logger.DebugContext(ctx, "ddm batched reconcile: loaded declarations",
		"declaration_count", len(allDecls))

	declsWithBrokenLabel := make(map[string]struct{})
	declsByTeam := make(map[uint][]*fleet.AppleDeclarationForReconcile, 4)
	labelIDSet := make(map[uint]struct{})
	for _, d := range allDecls {
		declsByTeam[d.TeamID] = append(declsByTeam[d.TeamID], d)

		if d.HasBrokenLabel() {
			declsWithBrokenLabel[d.DeclarationUUID] = struct{}{}
		}

		for _, lr := range d.IncludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
		for _, lr := range d.ExcludeLabels {
			if lr.LabelID != nil {
				labelIDSet[*lr.LabelID] = struct{}{}
			}
		}
	}

	hostIDs := make([]uint, 0, len(hosts))
	hostUUIDs := make([]string, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.HostID)
		hostUUIDs = append(hostUUIDs, h.UUID)
	}
	labelIDs := make([]uint, 0, len(labelIDSet))
	for id := range labelIDSet {
		labelIDs = append(labelIDs, id)
	}

	hostLabels, err := ds.BulkGetHostLabelMemberships(ctx, hostIDs, labelIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host label memberships")
	}

	currentByHost, err := ds.BulkGetHostMDMAppleDeclarationsByUUIDs(ctx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host mdm apple declarations")
	}

	changedHostUUIDs, declRowsToWrite := apple_mdm.ComputeDeclarationDeltas(
		hosts, hostLabels, currentByHost, declsByTeam, declsWithBrokenLabel,
	)

	logger.DebugContext(ctx, "ddm batched reconcile: computed deltas",
		"changed_hosts", len(changedHostUUIDs), "host_decl_rows_to_write", len(declRowsToWrite))

	if len(declRowsToWrite) == 0 {
		return nil
	}

	if err := ds.BulkUpsertMDMAppleHostDeclarations(ctx, declRowsToWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert host mdm apple declarations")
	}

	// Find any hosts that requested a resync. This is used to cover special cases where we're not
	// 100% certain of the declarations on the device.
	// This should be a simple and often no-op, so we are good to still call this each cron run.
	resyncHosts, err := ds.MDMAppleHostDeclarationsGetAndClearResync(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting and clearing resync hosts")
	}
	if len(resyncHosts) > 0 {
		changedHostUUIDs = append(changedHostUUIDs, resyncHosts...)
		// Deduplicate changedHosts
		uniqueHosts := make(map[string]struct{})
		deduplicatedHosts := make([]string, 0, len(changedHostUUIDs))
		for _, id := range changedHostUUIDs {
			if _, exists := uniqueHosts[id]; !exists {
				uniqueHosts[id] = struct{}{}
				deduplicatedHosts = append(deduplicatedHosts, id)
			}
		}
		changedHostUUIDs = deduplicatedHosts
	}

	// TODO: Consider a similar approach to profiles where if failed to send the command for the host, reset the status so we resent it again.
	// now it will just end up in a state where it never retries to send the DeclarativeManagement command.
	if len(changedHostUUIDs) > 0 {
		if err := commander.DeclarativeManagement(ctx, changedHostUUIDs, uuid.NewString()); err != nil {
			return ctxerr.Wrap(ctx, err, "issuing DeclarativeManagement command")
		}
		logger.InfoContext(ctx, "ddm batched reconcile: sent DeclarativeManagement command",
			"host_count", len(changedHostUUIDs))
	}

	return nil
}
