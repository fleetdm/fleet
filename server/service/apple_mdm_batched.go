package service

import (
	"context"
	"encoding/pem"
	"fmt"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
)

// reconcileAppleProfilesBatchSize is the scan window: how many enrolled Apple hosts the reconciler reads per snapshot.
// Snapshot reads are cheap (indexed, no set-difference), so within a single tick the drain loop pages through many windows
// until a budget is hit.
//
// var rather than const so tests can override it.
var reconcileAppleProfilesBatchSize = 2000

// reconcileAppleProfilesDeliveryCap bounds how many distinct hosts the cron schedules for install/remove per tick. It governs
// the bulk case: once this many hosts have been delivered work, the tick stops even if scan budget remains, advancing the
// cursor only to the last delivered host so the remainder resumes next tick. This is what smooths writer pressure — a bulk
// change is spread across ~ceil(hosts/cap) ticks. Set <= 0 to disable the cap (drain the whole fleet, bounded only by the
// scan budget).
//
// var rather than const so tests can override it.
var reconcileAppleProfilesDeliveryCap = 2000

// reconcileAppleProfilesScanBudget is the wall-clock budget for a single tick's drain loop. It governs the sparse/idle case: a
// no-work pass over the whole fleet completes within one tick, collapsing single-change latency from ceil(hosts/batch) x
// interval to roughly the actual work time.
//
// Shorter than the Windows equivalent's 24s because the Apple schedule runs three jobs sequentially per 30s tick (profiles,
// declarations, device names) — see newAppleMDMProfileManagerSchedule. Taking 24s here would starve the other two.
//
// var rather than const so tests can override it.
var reconcileAppleProfilesScanBudget = 12 * time.Second

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
	useOneTimeEnrollSecrets bool,
) (err error) {
	// Require primary here for reconciling apple profiles, to avoid read-write races and stale opt-in installs
	ctx = ctxdb.RequirePrimary(ctx, true)
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
	if err := ensureFleetProfiles(ctx, ds, logger, block.Bytes, useOneTimeEnrollSecrets); err != nil {
		logger.ErrorContext(ctx, "unable to ensure fleetd configuration profiles are in place", "details", err)
	}

	// Read the cursor; on error, treat as start-of-pass and continue. A stale or missing cursor is harmless because the
	// in-memory diff installs only what actually differs from the current state.
	entryCursor, cerr := ds.GetMDMAppleReconcileCursor(ctx)
	if cerr != nil {
		logger.WarnContext(ctx, "failed to read apple MDM reconcile cursor; starting from beginning", "err", cerr)
		entryCursor = ""
	}

	cursor := entryCursor
	// commitCursor is the cursor value to persist at tick end. It advances only past windows that were fully delivered; the
	// deferred write fires only when err == nil, so an error leaves the cursor untouched and the next tick re-scans from the
	// same point. Re-scanning is cheap and idempotent since delivered work is now pending, so it no longer computes as work.
	commitCursor := entryCursor

	defer func() {
		switch {
		case err != nil:
			logger.WarnContext(ctx, "batched reconcile: tick errored; cursor not advanced",
				"cursor", entryCursor, "err", err)
		case commitCursor != entryCursor:
			if serr := ds.SetMDMAppleReconcileCursor(ctx, commitCursor); serr != nil {
				logger.WarnContext(ctx, "failed to advance apple MDM reconcile cursor", "err", serr)
			} else {
				logger.DebugContext(ctx, "batched reconcile: cursor advanced",
					"cursor", entryCursor, "next_cursor", commitCursor)
			}
		default:
			logger.DebugContext(ctx, "batched reconcile: tick complete, cursor unchanged", "cursor", entryCursor)
		}
	}()

	// One CA budget for the whole tick, shared across every window the loop drains. A per-window limit would multiply the
	// issuance rate against the customer's CA by the number of windows, which is not what
	// mdm.certificate_profiles_limit promises. 0 means unlimited, which is a nil budget.
	var caInstallBudget *int
	if certProfilesLimit > 0 {
		caInstallBudget = new(certProfilesLimit)
	}

	deadline := time.Now().Add(reconcileAppleProfilesScanBudget)
	deliveredHosts := 0

	for {
		hosts, allProfiles, hostLabels, currentByHost, pageFull, serr := ds.GetAppleProfileReconcileSnapshot(ctx, cursor, reconcileAppleProfilesBatchSize)
		if serr != nil {
			err = ctxerr.Wrap(ctx, serr, "loading apple profile reconcile snapshot")
			return err
		}
		logger.DebugContext(ctx, "batched reconcile: loaded snapshot",
			"cursor", cursor, "hosts_in_batch", len(hosts), "profile_count", len(allProfiles))

		if len(hosts) == 0 {
			// Reached the end of the host space (or empty fleet): reset the cursor so the next pass restarts from the beginning.
			commitCursor = ""
			return nil
		}

		profilesWithBrokenLabel := make(map[string]struct{})
		profilesByTeam := make(map[uint][]*fleet.AppleProfileForReconcile, 4)
		for _, p := range allProfiles {
			profilesByTeam[p.TeamID] = append(profilesByTeam[p.TeamID], p)
			if p.HasBrokenLabel() {
				profilesWithBrokenLabel[p.ProfileUUID] = struct{}{}
			}
		}

		hostUUIDs := make([]string, 0, len(hosts))
		for _, h := range hosts {
			hostUUIDs = append(hostUUIDs, h.UUID)
		}
		optInsByHost, oerr := ds.BulkGetHostMDMProfileOptIns(ctx, hostUUIDs)
		if oerr != nil {
			err = oerr
			return err
		}

		toInstall, toRemove, optInChanges := apple_mdm.ComputeReconcileDeltas(hosts, hostLabels, currentByHost, profilesByTeam, profilesWithBrokenLabel, optInsByHost)
		toInstall = fleet.FilterMacOSOnlyProfilesFromIOSIPadOS(toInstall)

		logger.DebugContext(ctx, "batched reconcile: computed deltas",
			"to_install", len(toInstall), "to_remove", len(toRemove))

		// Opt-in changes are applied for the whole window, even hosts the delivery cap defers: they are idempotent and the
		// deferred hosts recompute the same install/remove next tick.
		if len(optInChanges.Add) > 0 || len(optInChanges.Purge) > 0 {
			if aerr := ds.ApplyHostMDMProfileOptInChanges(ctx, optInChanges); aerr != nil {
				err = aerr
				return err
			}
		}

		// Apply the per-tick delivery cap at host granularity. Hosts come back ascending by uuid, so capping keeps a contiguous
		// prefix of the work-hosts and the cursor can resume at the last delivered host.
		workHosts := appleHostsWithWork(hosts, toInstall, toRemove)
		// Advance past the whole window by default. Deciding end-of-space from len(hosts) is wrong: duplicate-UUID host rows
		// are collapsed after the SQL LIMIT, so a full page can dedupe to fewer than batchSize hosts — treating that as the end
		// of the host universe wraps the cursor early and permanently starves every host later in the UUID ordering. That is
		// what pageFull is for.
		advanceTo := hosts[len(hosts)-1].UUID

		partial := false
		if reconcileAppleProfilesDeliveryCap > 0 {
			// Invariant: deliveredHosts < cap here. We return below as soon as it reaches the cap. So remaining >= 1.
			remaining := reconcileAppleProfilesDeliveryCap - deliveredHosts
			if len(workHosts) > remaining {
				allowed := make(map[string]struct{}, remaining)
				for _, h := range workHosts[:remaining] {
					allowed[h] = struct{}{}
				}
				toInstall = filterApplePayloadsByHost(toInstall, allowed)
				toRemove = filterApplePayloadsByHost(toRemove, allowed)
				advanceTo = workHosts[remaining-1] // resume after the last delivered host
				workHosts = workHosts[:remaining]
				partial = true
			}
		}

		if len(toInstall) > 0 || len(toRemove) > 0 {
			if _, eerr := apple_mdm.ExecuteReconcileBatch(
				ctx, ds, commander, redisKeyValue, logger,
				appConfig, caInstallBudget, toInstall, toRemove,
			); eerr != nil {
				err = eerr
				return err
			}
		}
		deliveredHosts += len(workHosts)

		// Advance only after a successful execute.
		commitCursor = advanceTo
		cursor = advanceTo

		switch {
		case partial:
			// Delivery cap hit mid-window; the un-delivered remainder resumes next tick from cursor = advanceTo.
			return nil
		case !pageFull:
			// Short page => end of the host space; reset for the next pass.
			commitCursor = ""
			return nil
		case reconcileAppleProfilesDeliveryCap > 0 && deliveredHosts >= reconcileAppleProfilesDeliveryCap:
			// Delivery cap reached exactly at a window boundary.
			return nil
		case time.Now().After(deadline):
			// Scan budget exhausted; resume next tick from cursor = advanceTo.
			return nil
		}
		// Otherwise keep draining the next window within this tick.
	}
}

// appleHostsWithWork returns the host UUIDs that have at least one install or remove in this window, in the order hosts are
// given (ascending by uuid). The drain loop uses this both to count delivered hosts against the cap and to pick the contiguous
// prefix to deliver when the cap is reached mid-window.
func appleHostsWithWork(hosts []*fleet.AppleHostReconcileInfo, toInstall, toRemove []*fleet.MDMAppleProfilePayload) []string {
	work := make(map[string]struct{})
	for _, p := range toInstall {
		work[p.HostUUID] = struct{}{}
	}
	for _, p := range toRemove {
		work[p.HostUUID] = struct{}{}
	}
	ordered := make([]string, 0, len(work))
	for _, h := range hosts {
		if _, ok := work[h.UUID]; ok {
			ordered = append(ordered, h.UUID)
		}
	}
	return ordered
}

// filterApplePayloadsByHost returns only the payloads whose HostUUID is in the allowed set, preserving order. Used to trim a
// window's deltas to the hosts that fit under the per-tick delivery cap.
func filterApplePayloadsByHost(payloads []*fleet.MDMAppleProfilePayload, allowed map[string]struct{}) []*fleet.MDMAppleProfilePayload {
	out := make([]*fleet.MDMAppleProfilePayload, 0, len(payloads))
	for _, p := range payloads {
		if _, ok := allowed[p.HostUUID]; ok {
			out = append(out, p)
		}
	}
	return out
}
