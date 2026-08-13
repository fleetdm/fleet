package cron

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/microsoft/msgraph"
	"github.com/fleetdm/fleet/v4/server/service/schedule"
)

// microsoftAutopilotSyncInterval mirrors the Apple DEP sync cadence.
const microsoftAutopilotSyncInterval = 5 * time.Minute

// NewMicrosoftAutopilotSchedule registers the periodic Windows Autopilot device sync. The job no-ops when no Microsoft
// Graph credential is configured.
func NewMicrosoftAutopilotSchedule(
	ctx context.Context,
	instanceID string,
	ds fleet.Datastore,
	factory msgraph.ClientFactory,
	logger *slog.Logger,
) (*schedule.Schedule, error) {
	name := string(fleet.CronMicrosoftAutopilotSync)
	logger = logger.With("cron", name)
	s := schedule.New(
		ctx, name, instanceID, microsoftAutopilotSyncInterval, ds, ds,
		schedule.WithLogger(logger),
		schedule.WithJob("microsoft_autopilot_sync", func(ctx context.Context) error {
			return cronMicrosoftAutopilotSync(ctx, ds, factory, logger)
		}),
	)
	return s, nil
}

// cronMicrosoftAutopilotSync syncs every configured tenant, isolating failures so one bad credential neither stops the
// others nor disturbs its own tenant's existing hosts.
func cronMicrosoftAutopilotSync(ctx context.Context, ds fleet.Datastore, factory msgraph.ClientFactory, logger *slog.Logger) error {
	creds, err := ds.ListMicrosoftGraphCredentials(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list microsoft graph credentials")
	}
	if len(creds) == 0 {
		return nil
	}

	var flagChanged bool
	for _, cred := range creds {
		changed, err := syncMicrosoftAutopilotTenant(ctx, ds, factory, cred, logger)
		if changed {
			flagChanged = true
		}
		if err != nil {
			// Logged, not returned: the next tenant still gets its turn.
			logger.ErrorContext(ctx, "microsoft autopilot sync failed for tenant", "tenant_id", cred.TenantID, "err", err)
		}
	}

	// Update the app-wide stored aggregate, so it is only as fresh as its last recomputation.
	if flagChanged {
		// RequirePrimary to read the flag we just wrote.
		if err := ds.UpdateMicrosoftGraphCredentialInvalidAggregate(ctxdb.RequirePrimary(ctx, true)); err != nil {
			return ctxerr.Wrap(ctx, err, "refresh microsoft graph credential invalid aggregate")
		}
	}
	return nil
}

// syncMicrosoftAutopilotTenant runs one tenant's sync and records its outcome. It reports whether credential_invalid changed.
func syncMicrosoftAutopilotTenant(
	ctx context.Context,
	ds fleet.Datastore,
	factory msgraph.ClientFactory,
	cred *fleet.MicrosoftGraphCredential,
	logger *slog.Logger,
) (flagChanged bool, err error) {
	syncErr := reconcileMicrosoftAutopilotTenant(ctx, ds, factory, cred, logger)

	// Only an explicit credential rejection sets the flag. Throttling and server errors must never raise a credential
	// alarm, or a Microsoft outage would flag every Fleet deployment at once.
	invalid := isMicrosoftGraphCredentialRejected(syncErr)
	// Set on rejection, clear on success, and leave a transient failure alone.
	if invalid || syncErr == nil {
		changed, setErr := ds.SetMicrosoftGraphCredentialInvalid(ctx, cred.TenantID, invalid)
		if setErr != nil {
			logger.ErrorContext(ctx, "set microsoft graph credential invalid flag", "tenant_id", cred.TenantID, "err", setErr)
		}
		flagChanged = changed
	}

	var syncErrMsg *string
	if syncErr != nil {
		msg := syncErr.Error()
		syncErrMsg = &msg
	}
	if recErr := ds.RecordMicrosoftGraphSyncResult(ctx, cred.TenantID, syncErrMsg); recErr != nil {
		logger.ErrorContext(ctx, "record microsoft graph sync result", "tenant_id", cred.TenantID, "err", recErr)
	}

	return flagChanged, syncErr
}

// isMicrosoftGraphCredentialRejected reports whether the failure means the credential itself needs an admin's
// attention, as opposed to Microsoft being temporarily unavailable.
func isMicrosoftGraphCredentialRejected(err error) bool {
	if err == nil {
		return false
	}
	graphErr, ok := errors.AsType[*msgraph.Error](err)
	if !ok {
		return false
	}
	return graphErr.IsAuthError() || graphErr.IsPermissionError()
}

// reconcileMicrosoftAutopilotTenant pulls one tenant's Autopilot registry and reconciles it into pending hosts.
func reconcileMicrosoftAutopilotTenant(
	ctx context.Context,
	ds fleet.Datastore,
	factory msgraph.ClientFactory,
	cred *fleet.MicrosoftGraphCredential,
	logger *slog.Logger,
) error {
	client, err := factory(cred)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create microsoft graph client")
	}

	// The client errors rather than returning a partial list when its pagination cursor stops advancing, precisely so
	// the sync is never handed a truncated list to delete against.
	devices, err := client.ListWindowsAutopilotDevices(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list windows autopilot devices")
	}

	incoming, skipped := autopilotDevicesToIngest(devices, cred.TenantID)
	if skipped > 0 {
		logger.InfoContext(ctx, "skipped autopilot devices missing a usable device id or serial",
			"tenant_id", cred.TenantID, "skipped", skipped)
	}

	stored, err := ds.ListHostAutopilotDevices(ctx, cred.TenantID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list stored autopilot devices")
	}

	changed, removedHostIDs := diffAutopilotDevices(incoming, stored)

	if len(changed) > 0 {
		if err := ds.IngestWindowsAutopilotDevices(ctx, changed); err != nil {
			return ctxerr.Wrap(ctx, err, "ingest windows autopilot devices")
		}
	}

	// An empty device list at this point is treated as authoritative, including when it empties the tenant.
	if len(removedHostIDs) > 0 {
		if err := ds.RemoveWindowsAutopilotHosts(ctx, removedHostIDs); err != nil {
			return ctxerr.Wrap(ctx, err, "remove windows autopilot hosts")
		}
		logger.InfoContext(ctx, "removed hosts for devices no longer in autopilot",
			"tenant_id", cred.TenantID, "removed", len(removedHostIDs))
	}
	return nil
}

// autopilotDevicesToIngest converts Graph devices into storable records, dropping the ones that can never become a
// usable pending host and reporting how many were skipped.
func autopilotDevicesToIngest(devices []msgraph.WindowsAutopilotDevice, tenantID string) (out []*fleet.HostAutopilotDevice, skipped int) {
	out = make([]*fleet.HostAutopilotDevice, 0, len(devices))
	for _, dev := range devices {
		// Both fields are load-bearing. The Autopilot device ID is what every downstream lookup resolves on, and the
		// serial is the only identity a pending host has until the device boots.
		if dev.ID == "" || dev.SerialNumber == "" || fleet.IsPlaceholderHardwareSerial(dev.SerialNumber) {
			skipped++
			continue
		}
		out = append(out, &fleet.HostAutopilotDevice{
			AutopilotDeviceID: dev.ID,
			EntraDeviceID:     dev.EntraDeviceID,
			GroupTag:          dev.GroupTag,
			HardwareSerial:    dev.SerialNumber,
			TenantID:          tenantID,
		})
	}
	return out, skipped
}

// diffAutopilotDevices compares the tenant's Autopilot registry against what Fleet already stores, returning the
// records that need writing and the host IDs whose devices have left.
func diffAutopilotDevices(incoming []*fleet.HostAutopilotDevice, stored []*fleet.HostAutopilotDevice) (changed []*fleet.HostAutopilotDevice, removedHostIDs []uint) {
	storedByDeviceID := make(map[string]*fleet.HostAutopilotDevice, len(stored))
	for _, dev := range stored {
		storedByDeviceID[dev.AutopilotDeviceID] = dev
	}

	incomingDeviceIDs := make(map[string]struct{}, len(incoming))
	for _, dev := range incoming {
		incomingDeviceIDs[dev.AutopilotDeviceID] = struct{}{}

		existing, ok := storedByDeviceID[dev.AutopilotDeviceID]
		if !ok {
			changed = append(changed, dev)
			continue
		}
		if existing.GroupTag != dev.GroupTag ||
			existing.EntraDeviceID != dev.EntraDeviceID ||
			existing.HardwareSerial != dev.HardwareSerial {
			changed = append(changed, dev)
		}
	}

	for deviceID, dev := range storedByDeviceID {
		if _, ok := incomingDeviceIDs[deviceID]; !ok {
			removedHostIDs = append(removedHostIDs, dev.HostID)
		}
	}
	return changed, removedHostIDs
}
