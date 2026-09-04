package apple_mdm

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

const (
	// apnsSweepSilence is how long an enrollment must have been silent to be
	// re-armed with a fresh stored push.
	apnsSweepSilence = 24 * time.Hour
	// apnsSweepMinBatch floors pages at one enrollment per tick;
	// apnsSweepMaxBatch bounds per-tick work on very large fleets (a lap
	// then takes longer than a day, which the pass-start warning surfaces).
	apnsSweepMinBatch = 1
	apnsSweepMaxBatch = 2000
)

// apnsNotifier is the subset of MDMAppleCommander the sweep needs; tests
// substitute a fake.
type apnsNotifier interface {
	SendNotifications(ctx context.Context, ids []string) error
}

// SweepAPNsPushes runs one tick of the APNs sweep: it walks one page of
// enabled enrollments behind a persisted cursor and pushes every enrollment
// that has been silent for more than a day, so every offline device always
// has a stored push waiting at APNs (see the apns-expiration header set by
// the push provider). Deliberately not scoped to enrollments with pending
// commands: any such filter recreates one of the legacy pusher's delivery
// gaps, and a spurious nudge costs one empty Idle check-in.
//
// The batch size is computed at pass start so one lap completes in roughly a
// day at the given tick interval, and rides along with the cursor. Fleets
// smaller than a day's worth of ticks lap faster than a day at one
// enrollment per tick; the resulting extra pushes only reach devices that
// are still silent and offline, where re-arming a fresh stored push is
// harmless (APNs coalesces them) and extends the expiration hedge.
func SweepAPNsPushes(ctx context.Context, ds fleet.Datastore, commander *MDMAppleCommander,
	logger *slog.Logger, interval time.Duration,
) error {
	return sweepAPNsPushes(ctx, ds, commander, logger, interval)
}

func sweepAPNsPushes(ctx context.Context, ds fleet.Datastore, notifier apnsNotifier,
	logger *slog.Logger, interval time.Duration,
) error {
	if interval <= 0 {
		// the cron constructor guards this too; a local clamp keeps the
		// batch-size division from panicking for any future caller
		interval = time.Minute
	}
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieving app config")
	}
	if !appCfg.MDM.EnabledAndConfigured {
		return nil
	}

	state, err := ds.GetMDMAppleAPNsSweepState(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get apns sweep state")
	}
	if state != nil && state.BatchSize <= 0 {
		// same self-heal posture as undecodable state: a non-positive batch
		// size would error on LIMIT every tick, before the state is ever
		// rewritten, wedging the cron on the poisoned value.
		state = nil
	}
	if state == nil { // pass start: size batches so one lap takes ~a day
		count, err := ds.CountEnabledNanoEnrollments(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "count enabled nano enrollments")
		}
		if count == 0 {
			return nil
		}
		ticksPerDay := max(int(24*time.Hour/interval), 1)
		batch := (count + ticksPerDay - 1) / ticksPerDay
		if batch > apnsSweepMaxBatch {
			logger.WarnContext(ctx, "APNs sweep batch size clamped, a full pass will take longer than a day",
				"computed_batch_size", batch, "max_batch_size", apnsSweepMaxBatch, "enrollments", count)
			batch = apnsSweepMaxBatch
		}
		batch = max(batch, apnsSweepMinBatch)
		state = &fleet.MDMAppleAPNsSweepState{BatchSize: batch}
	}

	eligible, next, pageFull, err := ds.ListNanoEnrollmentIDsForAPNsSweep(ctx, state.Cursor, state.BatchSize, apnsSweepSilence)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list nano enrollment ids for apns sweep")
	}

	if len(eligible) > 0 {
		if err := notifier.SendNotifications(ctx, eligible); err != nil {
			var delivery *APNSDeliveryError
			if !errors.As(err, &delivery) {
				// provider or push-cert failure: leave the cursor unadvanced
				// so this page is retried next tick.
				return ctxerr.Wrap(ctx, err, "apns sweep send notifications")
			}
			// Per-enrollment rejections don't stop the lap. The sweep takes
			// no action on Unregistered (its ids span both channels while
			// MDM turn-off is host-keyed); the pool self-corrects because
			// eligibility requires host_mdm.enrolled = 1.
			for id, pushErr := range delivery.errorsByUUID {
				logger.InfoContext(ctx, "APNs sweep push rejected",
					"enrollment_id", id, "reason", APNSReason(pushErr), "err", pushErr)
			}
		}
	}

	if pageFull {
		state.Cursor = next
		return ctxerr.Wrap(ctx, ds.SetMDMAppleAPNsSweepState(ctx, state), "advance apns sweep state")
	}
	return ctxerr.Wrap(ctx, ds.SetMDMAppleAPNsSweepState(ctx, nil), "reset apns sweep state")
}
