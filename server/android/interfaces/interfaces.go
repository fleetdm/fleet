package interfaces

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet/common"
)

type FleetDatastore interface {
	CommonAppConfig(ctx context.Context) (common.AppConfig, error)
}
