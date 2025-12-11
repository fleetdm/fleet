package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// ReconcileAndroidDevices polls AMAPI for devices that Fleet still considers enrolled
// and flips them to unenrolled if Google reports them missing (404).
// This complements (does not replace) Pub/Sub DELETED handling.
func ReconcileAndroidDevices(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger, licenseKey string) error {
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
	deviceNameMap := make(map[string]struct{})
	pageToken := ""
	for {
		// We use the partial call here, to avoid getting all data for a device when we only need a subset (name).
		// should help with request speeds, and also cost for website in terms of network egress.
		resp, err := client.EnterprisesDevicesListPartial(ctx, enterprise.Name(), pageToken)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing android devices from AMAPI")
		}
		for _, dev := range resp.Devices {
			deviceNameMap[dev.Name] = struct{}{}
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
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
			if _, derr := ds.SetAndroidHostUnenrolled(ctx, dev.HostID); derr != nil {
				level.Error(logger).Log("msg", "failed to mark android host unenrolled during reconcile", "host_id", dev.HostID, "err", derr)
				continue
			}
			// Emit system activity to mirror Pub/Sub DELETED handling.
			var displayName, serial string
			if hosts, herr := ds.ListHostsLiteByIDs(ctx, []uint{dev.HostID}); herr == nil && len(hosts) == 1 && hosts[0] != nil {
				displayName = hosts[0].DisplayName()
				serial = hosts[0].HardwareSerial
			}
			if aerr := ds.NewActivity(ctx, nil, fleet.ActivityTypeMDMUnenrolled{
				HostSerial:       serial,
				HostDisplayName:  displayName,
				InstalledFromDEP: false,
				Platform:         "android",
			}, nil, time.Now()); aerr != nil {
				level.Debug(logger).Log("msg", "failed to create mdm_unenrolled activity during android reconcile", "host_id", dev.HostID, "err", aerr)
			}
			unenrolled++
			level.Debug(logger).Log("msg", "android device missing in Google; marked unenrolled", "host_id", dev.HostID, "device", deviceName)
		}
	}

	level.Debug(logger).Log("msg", "android reconcile complete", "checked", checked, "unenrolled", unenrolled)
	return nil
}
