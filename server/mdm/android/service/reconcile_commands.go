package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
)

const (
	// androidCommandReconcileMinAge is how long a command must sit in the pending status before we poll
	// AMAPI for it. Pub/Sub delivers within seconds in normal operation, so anything younger than this is
	// still expected to resolve on its own and polling it would only burn AMAPI quota.
	androidCommandReconcileMinAge = 24 * time.Hour

	// androidCommandReconcileNotFoundGrace is how long we keep waiting on a command whose Operation AMAPI
	// no longer knows about before declaring it failed. AMAPI drops Operation resources it has finished
	// with, and it also 404s for a device that was deleted, so a NotFound on its own does not tell us the
	// command is dead -- AMAPI may still be holding it (e.g. a WIPE waiting for the device to come back
	// online). Once the row is older than GCP Pub/Sub's maximum retention no notification can arrive
	// anymore, so at that point the row can only be stuck and marking it failed is what unsticks the host.
	androidCommandReconcileNotFoundGrace = 7 * 24 * time.Hour

	// androidCommandReconcileBatchSize bounds how many commands (and therefore AMAPI calls) a single run
	// makes. Combined with the rate limit below this caps a run at ~10 minutes of polling. Rows that don't
	// fit are picked up by the next run: they are ordered oldest-first, so the most stuck ones go first.
	androidCommandReconcileBatchSize = 500

	// androidCommandReconcileCallsPerMinute is the AMAPI request rate the reconciler paces itself to, to
	// stay well under the per-project request budget shared with the rest of Fleet's AMAPI traffic.
	androidCommandReconcileCallsPerMinute = 50

	// googleStatusCodeNotFound is google.rpc.Code NOT_FOUND, recorded on rows we fail because AMAPI no
	// longer has the Operation.
	googleStatusCodeNotFound = 5
)

// ReconcileAndroidCommands recovers Android MDM commands whose Pub/Sub COMMAND notification never
// arrived (Fleet's push endpoint down longer than GCP's retention, a subscription misconfiguration, a
// Google Cloud incident). Without this, such a command sits in mdm_android_commands.status='pending'
// forever, the host reads as perpetually pending lock/wipe/clear-passcode, and the admin cannot
// re-issue it. AMAPI's operations.get is the authoritative source for a command's outcome and, unlike
// Apple's and Windows' equivalents, needs neither the device to come back online nor AMAPI to re-send
// anything.
func ReconcileAndroidCommands(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, licenseKey string, newActivityFn fleet.NewActivityFunc) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil
	}

	client := newAMAPIClient(ctx, logger, licenseKey)

	// Best-effort set authentication secret for proxy client usage (no-op for Google client).
	if assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidFleetServerSecret}, nil); err == nil {
		if asset, ok := assets[fleet.MDMAssetAndroidFleetServerSecret]; ok && len(asset.Value) > 0 {
			_ = client.SetAuthenticationSecret(string(asset.Value))
		}
	}

	return reconcileAndroidCommands(ctx, ds, client, logger, newActivityFn, time.Now().UTC(),
		time.Minute/androidCommandReconcileCallsPerMinute)
}

// reconcileAndroidCommands is the testable core of ReconcileAndroidCommands. now anchors both the
// pending-age cutoff and the NotFound grace period; callInterval is the delay between AMAPI calls.
func reconcileAndroidCommands(ctx context.Context, ds fleet.Datastore, client androidmgmt.Client, logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc, now time.Time, callInterval time.Duration,
) error {
	cmds, err := ds.ListPendingMDMAndroidCommands(ctx, now.Add(-androidCommandReconcileMinAge), androidCommandReconcileBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list pending android commands for reconcile")
	}
	if len(cmds) == 0 {
		return nil
	}

	ticker := time.NewTicker(callInterval)
	defer ticker.Stop()

	var resolved, stillRunning int
	for i, cmd := range cmds {
		// Pace ourselves between AMAPI calls, but don't pay the delay before the first one.
		if i > 0 {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return ctxerr.Wrap(ctx, ctx.Err(), "android command reconcile interrupted")
			}
		}

		op, err := client.EnterprisesDevicesOperationsGet(ctx, cmd.OperationName)
		switch {
		case androidmgmt.IsTooManyRequestsError(err):
			// Out of AMAPI quota. Stop the run rather than hammering a rate-limited API; the remaining rows
			// stay pending and the next run resumes with them (oldest first).
			logger.WarnContext(ctx, "android command reconcile hit AMAPI quota, stopping run",
				"command_uuid", cmd.CommandUUID, "resolved", resolved, "remaining", len(cmds)-i)
			return ctxerr.Wrap(ctx, err, "android command reconcile exceeded AMAPI quota")

		case androidmgmt.IsNotFoundError(err):
			age := now.Sub(cmd.CreatedAt)
			if age < androidCommandReconcileNotFoundGrace {
				logger.DebugContext(ctx, "android command operation not found in AMAPI, still within grace period",
					"command_uuid", cmd.CommandUUID, "operation_name", cmd.OperationName, "age", age)
				stillRunning++
				continue
			}
			errCode := googleStatusCode(googleStatusCodeNotFound)
			errMsg := "Fleet did not receive a result for this command and Google no longer has a record of it."
			if err := setAndroidCommandTerminalState(ctx, ds, newActivityFn, cmd,
				string(android.MDMAndroidCommandStatusError), &errCode, &errMsg); err != nil {
				logger.ErrorContext(ctx, "failed to fail android command with unknown operation",
					"command_uuid", cmd.CommandUUID, "err", err)
				ctxerr.Handle(ctx, err)
				continue
			}
			resolved++
			logger.InfoContext(ctx, "android command operation unknown to AMAPI past grace period, marked error",
				"command_uuid", cmd.CommandUUID, "operation_name", cmd.OperationName, "age", age)

		case err != nil:
			// Transient AMAPI/network failure for this one command. Keep going: the rest of the batch is
			// independent, and this row is retried on the next run.
			logger.ErrorContext(ctx, "failed to get android command operation from AMAPI",
				"command_uuid", cmd.CommandUUID, "operation_name", cmd.OperationName, "err", err)
			ctxerr.Handle(ctx, err)

		case !op.Done:
			// Still queued at AMAPI (e.g. the device has not come online yet). Leave it pending.
			stillRunning++

		default:
			status, errCode, errMsg := androidOperationTerminalState(op)
			if err := setAndroidCommandTerminalState(ctx, ds, newActivityFn, cmd, status, errCode, errMsg); err != nil {
				logger.ErrorContext(ctx, "failed to apply reconciled android command status",
					"command_uuid", cmd.CommandUUID, "status", status, "err", err)
				ctxerr.Handle(ctx, err)
				continue
			}
			resolved++
			logger.InfoContext(ctx, "android command reconciled from AMAPI",
				"command_uuid", cmd.CommandUUID, "command_type", cmd.CommandType, "new_status", status)
		}
	}

	logger.DebugContext(ctx, "android command reconcile complete",
		"checked", len(cmds), "resolved", resolved, "still_running", stillRunning)
	return nil
}
