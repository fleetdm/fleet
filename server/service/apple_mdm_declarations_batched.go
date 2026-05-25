package service

import (
	"bytes"
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
// reconciler (appleEntityAppliesToHost), diffs against current
// host_mdm_apple_declarations rows, writes the diffed state, and kicks a
// DeclarativeManagement command on changed hosts so they fetch the new
// declarations.
//
// Sharing the appleEntityAppliesToHost dispatcher is the whole point:
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
	logger.InfoContext(ctx, "ddm batched reconcile: listed hosts",
		"cursor", cursor, "hosts_in_batch", len(hosts))

	if len(hosts) == 0 {
		if cursor != "" {
			logger.InfoContext(ctx, "apple MDM declaration reconcile pass complete; resetting cursor", "cursor", cursor)
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
				logger.InfoContext(ctx, "ddm batched reconcile: cursor advanced",
					"cursor", cursor, "next_cursor", nextCursor)
			}
		default:
			logger.InfoContext(ctx, "ddm batched reconcile: tick complete, cursor unchanged",
				"cursor", cursor)
		}
	}()

	allDecls, err := ds.ListAppleDeclarationsForReconcile(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple declarations for reconcile")
	}
	logger.InfoContext(ctx, "ddm batched reconcile: loaded declarations",
		"declaration_count", len(allDecls))

	declsByTeam := make(map[uint][]*fleet.AppleDeclarationForReconcile, 4)
	labelIDSet := make(map[uint]struct{})
	for _, d := range allDecls {
		declsByTeam[d.TeamID] = append(declsByTeam[d.TeamID], d)
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

	changedHostUUIDs, declRowsToWrite := computeAppleDeclarationDeltas(
		hosts, hostLabels, currentByHost, declsByTeam,
	)

	logger.InfoContext(ctx, "ddm batched reconcile: computed deltas",
		"changed_hosts", len(changedHostUUIDs), "host_decl_rows_to_write", len(declRowsToWrite))

	if len(declRowsToWrite) == 0 {
		return nil
	}

	// Persist the pending host_mdm_apple_declarations rows.
	if err := bulkUpsertHostMDMAppleDeclarations(ctx, ds, declRowsToWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk upsert host mdm apple declarations")
	}

	if len(changedHostUUIDs) > 0 {
		if err := commander.DeclarativeManagement(ctx, changedHostUUIDs, uuid.NewString()); err != nil {
			return ctxerr.Wrap(ctx, err, "issuing DeclarativeManagement command")
		}
		logger.InfoContext(ctx, "ddm batched reconcile: sent DeclarativeManagement command",
			"host_count", len(changedHostUUIDs))
	}

	return nil
}

// computeAppleDeclarationDeltas is the DDM equivalent of
// computeAppleReconcileDeltas. It uses the SHARED
// appleEntityAppliesToHost dispatcher to decide which declarations apply
// to each host — same code path the profile reconciler runs against —
// then diffs against the current host_mdm_apple_declarations rows.
//
// Returns:
//   - changedHostUUIDs: the set of host UUIDs that have at least one
//     install or remove diff, so the caller can target a single
//     DeclarativeManagement command per host.
//   - declRowsToWrite: the upsert payloads for host_mdm_apple_declarations
//     setting pending status for the diff'd rows.
func computeAppleDeclarationDeltas(
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
			if !appleEntityAppliesToHost(d, host, labelsForHost) {
				continue
			}
			desired[d.DeclarationUUID] = d
		}

		current := currentByHost[host.UUID]
		currentByDecl := make(map[string]*fleet.MDMAppleHostDeclaration, len(current))
		for _, c := range current {
			currentByDecl[c.DeclarationUUID] = c
		}

		// INSTALL diffs: desired declarations that aren't in current state,
		// have a token mismatch, or are flagged for re-install.
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

		// REMOVE diffs: current rows whose declaration is no longer desired,
		// excluding remove-with-non-NULL-status (already in flight/done) and
		// broken-label declarations (legacy behavior: never auto-remove).
		for declUUID, c := range currentByDecl {
			if _, stillDesired := desired[declUUID]; stillDesired {
				continue
			}
			if c.OperationType == fleet.MDMOperationTypeRemove && c.Status != nil {
				continue
			}
			if isBrokenAppleDeclaration(declUUID, declsByTeam) {
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

// bulkUpsertHostMDMAppleDeclarations writes the diff'd
// host_mdm_apple_declarations rows. Kept as a tiny helper close to the
// reconciler so the upsert SQL stays beside the code that produces the
// payloads it expects.
func bulkUpsertHostMDMAppleDeclarations(
	ctx context.Context,
	ds fleet.Datastore,
	rows []*fleet.MDMAppleHostDeclaration,
) error {
	// Reuse the existing datastore helper if it exists; otherwise fall
	// back to writing through MDMAppleStoreDDMStatusReport-style logic.
	// For now we route through the existing MDMAppleBatchSetHostDeclarationState
	// pathway via a thin wrapper. This is deliberately conservative —
	// the goal of the batched reconciler is the *desired state*
	// computation, not the upsert mechanics, which already exist.
	//
	// Implementation note: to stay symmetric with the profile path and
	// avoid duplicating SQL, we use the existing
	// MDMAppleStoreDDMStatusReport... wait, no — that handles incoming
	// device reports. We need the upsert path used by
	// mdmAppleBatchSetPendingHostDeclarationsDB.
	//
	// Until a clean public helper exists, expose a thin datastore method.
	return ds.BulkUpsertMDMAppleHostDeclarations(ctx, rows)
}
