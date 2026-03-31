package acme

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// DataProviders combines all external dependency interfaces for the ACME
// bounded context.
type DataProviders interface {
	AppConfig(ctx context.Context) (*fleet.AppConfig, error)
	GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName,
		queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error)
}
