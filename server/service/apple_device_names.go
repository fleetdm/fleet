package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/google/uuid"
)

// reconcileHostDeviceNamesBatchSize bounds how many queued rename rows a
// single cron tick processes; remaining rows are picked up on subsequent
// ticks, amortizing command delivery for large teams.
//
// var (not const) so tests can override it.
var reconcileHostDeviceNamesBatchSize = 500

// ReconcileHostDeviceNames runs one pass of host-name template enforcement:
// for each host whose enforcement row is queued (status NULL), it resolves
// the host's team name template and either enqueues a Settings/DeviceName
// command or records the outcome directly (name already matching → verified;
// resolved name unusable → failed).
func ReconcileHostDeviceNames(
	ctx context.Context,
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger *slog.Logger,
) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading app config")
	}
	if !appConfig.MDM.EnabledAndConfigured {
		return nil
	}

	pending, err := ds.ListHostsPendingDeviceNameCommand(ctx, reconcileHostDeviceNamesBatchSize)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list hosts pending device name command")
	}
	if len(pending) == 0 {
		return nil
	}

	templates := make(map[uint]string) // team ID → name template
	var notify []string                // hosts with a freshly-enqueued command to push
	for _, host := range pending {
		if host.TeamID == nil {
			// Enforcement rows are only created for hosts on a team with a
			// template; the host moved to "no team" between cron runs and its
			// row is reconciled by the transfer path. Leave it queued.
			continue
		}
		tmpl, ok := templates[*host.TeamID]
		if !ok {
			mdmConfig, err := ds.TeamMDMConfig(ctx, *host.TeamID)
			if err != nil {
				if fleet.IsNotFound(err) {
					// team deleted between cron runs
					templates[*host.TeamID] = ""
					continue
				}
				return ctxerr.Wrap(ctx, err, "get team mdm config for device name")
			}
			tmpl = mdmConfig.HostNameTemplate
			templates[*host.TeamID] = tmpl
		}
		if tmpl == "" {
			// Template cleared between cron runs, or the team was deleted (cached
			// as "" above). Either way the clear/delete path removes the rows, so
			// there's nothing to enforce here.
			continue
		}

		resolved := fleet.ResolveHostNameTemplate(tmpl, &fleet.Host{
			UUID:           host.HostUUID,
			HardwareSerial: host.HardwareSerial,
			Platform:       host.Platform,
		})
		switch {
		case len(resolved) > fleet.MaxResolvedHostNameBytes:
			// The resolved name is not stored: it can exceed the column width,
			// and failed rows are never compared against reported names.
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryFailed, nil, "",
				"Resolved name exceeds 63 bytes."); err != nil {
				return ctxerr.Wrap(ctx, err, "mark device name row failed for too-long name")
			}
			logger.InfoContext(ctx, "host name template resolves past the device name limit, not sending command",
				"host_uuid", host.HostUUID, "resolved_bytes", len(resolved))
		case resolved == host.ComputerName:
			// The device already carries the resolved name; no command needed.
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryVerified, nil, resolved, ""); err != nil {
				return ctxerr.Wrap(ctx, err, "mark device name row verified for matching name")
			}
		default:
			cmdUUID := fleet.DeviceNameCommandUUIDPrefix + uuid.NewString()
			if err := commander.DeviceNameSettingWithoutNotifications(ctx, host.HostUUID, cmdUUID, resolved); err != nil {
				// The command was not persisted; leave the row queued so the
				// next cron run retries this host, and move on so one bad
				// host doesn't starve the rest of the batch.
				logger.ErrorContext(ctx, "enqueue device name command", "host_uuid", host.HostUUID, "err", err)
				continue
			}
			// The command is persisted; the device will apply it on its next
			// check-in even if the batched push below never reaches it. Collect
			// the UUID so every device in this batch is woken with a single APNs
			// push instead of one request per host.
			notify = append(notify, host.HostUUID)
			if err := ds.SetHostDeviceNameStatus(ctx, host.HostUUID, fleet.MDMDeliveryPending, &cmdUUID, resolved, ""); err != nil {
				// The command was sent but recording it failed; log and move on
				// rather than aborting the batch, consistent with the
				// enqueue-failure handling above. The row stays queued and a
				// later cron run re-sends; the device resolves to the latest
				// command, and the superseded one's result is dropped as stale.
				logger.ErrorContext(ctx, "mark device name command sent", "host_uuid", host.HostUUID, "command_uuid", cmdUUID, "err", err)
				continue
			}
		}
	}

	if len(notify) > 0 {
		// One batched push wakes every device whose command was just enqueued.
		// Same handling as the iOS/iPadOS revive cron: a per-device APNs failure
		// is tolerable (the command is already persisted, so the device applies
		// it on its next check-in) so it's logged and the run succeeds — retrying
		// would enqueue duplicates. Any other error means the push subsystem
		// itself failed, so surface it.
		if err := commander.SendNotifications(ctx, notify); err != nil {
			var apnsErr *apple_mdm.APNSDeliveryError
			if !errors.As(err, &apnsErr) {
				return ctxerr.Wrap(ctx, err, "push device name commands")
			}
			logger.InfoContext(ctx, "failed to push device name command to some hosts", "err", apnsErr.Error())
		}
	}
	return nil
}
