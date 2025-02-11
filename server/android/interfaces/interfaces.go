package interfaces

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type FleetDatastore interface {
	AppConfig(ctx context.Context) (*fleet.AppConfig, error)
}
