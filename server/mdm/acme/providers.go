package acme

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// DataProviders combines all external dependency interfaces for the ACME
// bounded context.
type DataProviders interface {
	AppConfig(ctx context.Context) (*fleet.AppConfig, error)
}
