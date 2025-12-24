package service

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	kitlog "github.com/go-kit/log"
	"github.com/micromdm/plist"
)

func NewDeviceLocationResult(result *mdm.CommandResults, hostID uint) (DeviceLocationResult, error) {
	var x deviceLocationResult

	// parse results
	var deviceLocResult struct {
		Latitude  float64 `plist:"Latitude"`
		Longitude float64 `plist:"Longitude"`
	}

	if err := plist.Unmarshal(result.Raw, &deviceLocResult); err != nil {
		return nil, fmt.Errorf("device location command result: xml unmarshal: %w", err)
	}

	x.hostID = hostID
	x.latitude = deviceLocResult.Latitude
	x.longitude = deviceLocResult.Longitude

	return &x, nil

}

func NewDeviceLocationResultsHandler(
	ds fleet.Datastore,
	commander *apple_mdm.MDMAppleCommander,
	logger kitlog.Logger,
) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, commandResults fleet.MDMCommandResults) error {
		deviceLocResult, ok := commandResults.(DeviceLocationResult)
		if !ok {
			return ctxerr.New(ctx, "unexpected results type")
		}

		err := ds.InsertHostLocationData(ctx, deviceLocResult.HostID(), deviceLocResult.Latitude(), deviceLocResult.Longitude())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "device location command result: insert host location data")
		}

		return nil

	}
}
