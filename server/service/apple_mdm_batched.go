package service

import (
	"context"
	"encoding/pem"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
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

// ReconcileAppleProfilesBatched is the batched Apple MDM profile
// reconciler cron entry point. It pulls one bounded host window per
// tick (cursor in Redis), then delegates the compute + execute pipeline
// to the shared apple_mdm package so the same desired-state logic runs
// for the cron, the per-host enrollment path, and the DDM reconciler.
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
		return ctxerr.New(ctx, "failed to decode PEM block from SCEP certificate")
	}
	if err := ensureFleetProfiles(ctx, ds, logger, block.Bytes); err != nil {
		logger.ErrorContext(ctx, "unable to ensure fleetd configuration profiles are in place", "details", err)
	}

	cursor, err := ds.GetMDMAppleReconcileCursor(ctx)
	if err != nil {
		logger.WarnContext(ctx, "failed to read apple MDM reconcile cursor; starting from beginning", "err", err)
		cursor = ""
	}

	hosts, err := ds.ListAppleMDMHostsForReconcileBatch(ctx, cursor, reconcileAppleProfilesBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple MDM hosts for reconcile batch")
	}
	logger.DebugContext(ctx, "batched reconcile: listed hosts",
		"cursor", cursor, "hosts_in_batch", len(hosts))

	if len(hosts) == 0 {
		if cursor != "" {
			logger.DebugContext(ctx, "apple MDM reconcile pass complete; resetting cursor", "cursor", cursor)
			if cerr := ds.SetMDMAppleReconcileCursor(ctx, ""); cerr != nil {
				logger.WarnContext(ctx, "failed to reset apple MDM reconcile cursor", "err", cerr)
			}
		}
		return nil
	}

	var nextCursor string
	if len(hosts) >= reconcileAppleProfilesBatchSize {
		nextCursor = hosts[len(hosts)-1].UUID
	}

	defer func() {
		switch {
		case err != nil:
			logger.WarnContext(ctx, "batched reconcile: tick errored; cursor not advanced",
				"cursor", cursor, "next_cursor", nextCursor, "err", err)
		case cursor != nextCursor:
			if cerr := ds.SetMDMAppleReconcileCursor(ctx, nextCursor); cerr != nil {
				logger.WarnContext(ctx, "failed to advance apple MDM reconcile cursor", "err", cerr)
			} else {
				logger.DebugContext(ctx, "batched reconcile: cursor advanced",
					"cursor", cursor, "next_cursor", nextCursor)
			}
		default:
			logger.DebugContext(ctx, "batched reconcile: tick complete, cursor unchanged",
				"cursor", cursor)
		}
	}()

	if cursor != "" || nextCursor != "" {
		logger.DebugContext(ctx, "apple MDM reconcile tick using cursor",
			"cursor", cursor, "next_cursor", nextCursor,
			"batch_size", reconcileAppleProfilesBatchSize,
			"hosts_in_batch", len(hosts),
		)
	}

	allProfiles, err := ds.ListAppleProfilesForReconcile(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple profiles for reconcile")
	}
	logger.DebugContext(ctx, "batched reconcile: loaded profiles",
		"profile_count", len(allProfiles))

	profilesWithBrokenLabel := make(map[string]struct{})
	profilesByTeam := make(map[uint][]*fleet.AppleProfileForReconcile, 4)
	labelIDSet := make(map[uint]struct{})
	for _, p := range allProfiles {
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

	currentByHost, err := ds.BulkGetHostMDMAppleProfilesByUUIDs(ctx, hostUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "bulk get host mdm apple profiles")
	}

	toInstall, toRemove := apple_mdm.ComputeReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel)
	toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

	logger.DebugContext(ctx, "batched reconcile: computed deltas",
		"to_install", len(toInstall), "to_remove", len(toRemove),
		"host_ids", len(hostIDs), "label_ids", len(labelIDs))

	if len(toInstall) == 0 && len(toRemove) == 0 {
		return nil
	}

	_, err = apple_mdm.ExecuteReconcileBatch(
		ctx, ds, commander, redisKeyValue, logger,
		appConfig, certProfilesLimit, toInstall, toRemove,
	)
	return err
}
