package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
)

const (
	// androidReconcileMaxPages bounds the AMAPI device pagination loop so a malformed
	// or cycling NextPageToken can't spin it forever. AMAPI returns 100 devices/page,
	// so this permits up to ~1,000,000 devices — a safety net well above any realistic
	// enterprise size, not a functional limit.
	androidReconcileMaxPages = 10000
	// androidReconcilePageLogInterval controls how often pagination progress is logged
	// so a slow or stuck reconcile can be diagnosed.
	androidReconcilePageLogInterval = 100
)

// ReconcileAndroidDevices polls AMAPI for devices that Fleet still considers enrolled
// and flips them to unenrolled if Google reports them missing (404).
// This complements (does not replace) Pub/Sub DELETED handling.
func ReconcileAndroidDevices(ctx context.Context, ds fleet.Datastore, logger *slog.Logger, licenseKey string, newActivityFn fleet.NewActivityFunc) error {
	appConfig, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get app config")
	}
	if !appConfig.MDM.AndroidEnabledAndConfigured {
		return nil
	}

	// Ensure an enterprise exists, otherwise nothing to do, and keep its ID to build device resource names.
	enterprise, err := ds.GetEnterprise(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get android enterprise")
	}

	client := newAMAPIClient(ctx, logger, licenseKey)

	// Best-effort set authentication secret for proxy client usage (no-op for Google client).
	if assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidFleetServerSecret}, nil); err == nil {
		if asset, ok := assets[fleet.MDMAssetAndroidFleetServerSecret]; ok && len(asset.Value) > 0 {
			_ = client.SetAuthenticationSecret(string(asset.Value))
		}
	}

	devices, err := ds.ListAndroidEnrolledDevicesForReconcile(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list enrolled android devices for reconcile")
	}
	if len(devices) == 0 {
		return nil
	}

	// Make a list of all devices in Google
	deviceNameMap, err := listAllAndroidDeviceNames(ctx, client, logger, enterprise.Name())
	if err != nil {
		return err
	}

	checked := 0
	unenrolled := 0
	for _, dev := range devices {
		if dev == nil || dev.DeviceID == "" {
			continue
		}
		checked++
		deviceName := fmt.Sprintf("%s/devices/%s", enterprise.Name(), dev.DeviceID)
		_, ok := deviceNameMap[deviceName]
		switch {
		case ok:
			// Device exists, no-op.
			continue
		case !ok:
			// BYO unenroll wipes only the work profile; clear host_mdm_actions before flipping host_mdm.enrolled so the post-ack "Wiped"
			// badge clears.
			if cerr := clearAndroidBYOWipeRef(ctx, ds, dev.HostID); cerr != nil {
				logger.ErrorContext(ctx, "failed to clear android byo wipe-ref during reconcile", "host_id", dev.HostID, "err", cerr)
				ctxerr.Handle(ctx, cerr)
				continue
			}

			if _, derr := ds.SetAndroidHostUnenrolled(ctx, dev.HostID); derr != nil {
				logger.ErrorContext(ctx, "failed to mark android host unenrolled during reconcile", "host_id", dev.HostID, "err", derr)
				continue
			}
			// Emit system activity to mirror Pub/Sub DELETED handling.
			var displayName, serial string
			if hosts, herr := ds.ListHostsLiteByIDs(ctx, []uint{dev.HostID}); herr == nil && len(hosts) == 1 && hosts[0] != nil {
				displayName = hosts[0].DisplayName()
				serial = hosts[0].HardwareSerial
			}
			if aerr := newActivityFn(ctx, nil, fleet.ActivityTypeMDMUnenrolled{
				HostID:           dev.HostID,
				HostSerial:       serial,
				HostDisplayName:  displayName,
				InstalledFromDEP: false,
				Platform:         "android",
			}); aerr != nil {
				logger.DebugContext(ctx, "failed to create mdm_unenrolled activity during android reconcile", "host_id", dev.HostID, "err", aerr)
			}
			unenrolled++
			logger.DebugContext(ctx, "android device missing in Google; marked unenrolled", "host_id", dev.HostID, "device", deviceName)
		}
	}

	logger.DebugContext(ctx, "android reconcile complete", "checked", checked, "unenrolled", unenrolled)
	return nil
}

// listAllAndroidDeviceNames pages through AMAPI and returns the set of device resource names
// Google reports for the enterprise. The pagination loop is bounded by androidReconcileMaxPages
// so a malformed or cycling NextPageToken can't spin it forever; hitting the bound returns an
// error rather than a partial set, because a partial set would make present devices look missing
// and wrongly flip them to unenrolled.
func listAllAndroidDeviceNames(ctx context.Context, client androidmgmt.Client, logger *slog.Logger, enterpriseName string) (map[string]struct{}, error) {
	deviceNameMap := make(map[string]struct{})
	pageToken := ""
	for page := 1; ; page++ {
		// We use the partial call here, to avoid getting all data for a device when we only need a subset (name).
		// should help with request speeds, and also cost for website in terms of network egress.
		resp, err := client.EnterprisesDevicesListPartial(ctx, enterpriseName, pageToken)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "listing android devices from AMAPI")
		}
		for _, dev := range resp.Devices {
			deviceNameMap[dev.Name] = struct{}{}
		}
		if resp.NextPageToken == "" {
			return deviceNameMap, nil
		}
		if page >= androidReconcileMaxPages {
			logger.ErrorContext(ctx, "android reconcile pagination exceeded max pages; aborting to avoid unbounded loop",
				"max_pages", androidReconcileMaxPages, "devices_seen", len(deviceNameMap))
			return nil, ctxerr.Errorf(ctx, "android reconcile pagination exceeded max pages (%d)", androidReconcileMaxPages)
		}
		if page%androidReconcilePageLogInterval == 0 {
			logger.InfoContext(ctx, "android reconcile pagination progress", "pages", page, "devices_seen", len(deviceNameMap))
		}
		pageToken = resp.NextPageToken
	}
}
