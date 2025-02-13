package job

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/option"
)

var (
	// Required env vars
	androidServiceCredentials = os.Getenv("FLEET_ANDROID_SERVICE_CREDENTIALS")
	androidProjectID          = os.Getenv("FLEET_ANDROID_PROJECT_ID")
)

func ReconcileDevices(ctx context.Context, ds fleet.Datastore, androidDS android.Datastore, logger kitlog.Logger) error {
	if androidServiceCredentials == "" || androidProjectID == "" {
		return errors.New("FLEET_ANDROID_SERVICE_CREDENTIALS and FLEET_ANDROID_PROJECT_ID must be set")
	}

	mgmt, err := androidmanagement.NewService(ctx, option.WithCredentialsJSON([]byte(androidServiceCredentials)))
	if err != nil {
		return ctxerr.Wrap(ctx, err, "creating android management service")
	}

	enterprises, err := androidDS.ListEnterprises(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing enterprises")
	}

	for _, enterprise := range enterprises {
		if !enterprise.IsValid() {
			continue
		}
		// Note: we can optimize this by using Fields to retrieve partial data https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
		// But actually this is not scalable for 100,000s devices, so we need to use PubSub.
		devices, err := mgmt.Enterprises.Devices.List(enterprise.Name()).Do()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing devices with Google API")
		}

		for _, device := range devices.Devices {
			// Get the deviceId from the name: enterprises/{enterpriseId}/devices/{deviceId}
			nameParts := strings.Split(device.Name, "/")
			if len(nameParts) != 4 {
				return ctxerr.Errorf(ctx, "invalid Android device name: %s", device.Name)
			}
			deviceID := nameParts[3]

			host, err := androidDS.GetHost(ctx, enterprise.ID, deviceID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting host")
			}
			if host != nil {
				// TODO: Update host if needed
				continue
			}

			// TODO: Do EnrollHost and androidDS.AddHost inside a transaction so we don't add duplicate hosts
			fleetHost, err := ds.EnrollHost(ctx, true, device.HardwareInfo.SerialNumber, device.HardwareInfo.SerialNumber,
				device.HardwareInfo.SerialNumber, "", nil, 0)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "enrolling host")
			}
			err = androidDS.AddHost(ctx, &android.Host{
				FleetEnterpriseID: enterprise.ID,
				DeviceID:          deviceID,
				HostID:            fleetHost.ID,
			})
			if err != nil {
				return ctxerr.Wrap(ctx, err, "adding Android host")
			}

			fleetHost.DiskEncryptionEnabled = &device.DeviceSettings.IsEncrypted
			fleetHost.Platform = "ubuntu"
			fleetHost.HardwareVendor = device.HardwareInfo.Manufacturer
			fleetHost.HardwareModel = device.HardwareInfo.Model
			fleetHost.OSVersion = "Android " + device.SoftwareInfo.AndroidVersion
			lastEnrolledAt, err := time.Parse(time.RFC3339, device.EnrollmentTime)
			switch {
			case err != nil:
				level.Error(logger).Log("msg", "parsing Android device last enrolled at", "err", err, "deviceId", deviceID)
			default:
				fleetHost.LastEnrolledAt = lastEnrolledAt
			}
			detailUpdatedAt, err := time.Parse(time.RFC3339, device.LastStatusReportTime)
			switch {
			case err != nil:
				level.Error(logger).Log("msg", "parsing Android device detail updated at", "err", err, "deviceId", deviceID)
			default:
				fleetHost.DetailUpdatedAt = detailUpdatedAt
			}
			err = ds.UpdateHost(ctx, fleetHost)
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("updating host with deviceId %s", deviceID))
			}

			err = ds.UpdateHostOperatingSystem(ctx, fleetHost.ID, fleet.OperatingSystem{
				Name:          "Android",
				Version:       device.SoftwareInfo.AndroidVersion,
				Platform:      "android",
				KernelVersion: device.SoftwareInfo.DeviceKernelVersion,
			})
			if err != nil {
				return ctxerr.Wrap(ctx, err, fmt.Sprintf("updating host operating system with deviceId %s", deviceID))
			}

		}

	}

	return nil
}
