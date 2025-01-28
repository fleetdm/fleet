package job

import (
	"context"
	"errors"
	"os"

	"github.com/fleetdm/fleet/v4/server/android"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
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
		// Note: we can optimize this by using Fields to retrieve partial data https://developers.google.com/gdata/docs/2.0/basics#PartialResponse
		devices, err := mgmt.Enterprises.Devices.List(enterprise.Name()).Do()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing devices with Google API")
		}

		for _, device := range devices.Devices {
			logger.Log("msg", "device", "device", device)
		}

		// For each device, check whether it is in Fleet. If not, add it
	}

	return nil
}
