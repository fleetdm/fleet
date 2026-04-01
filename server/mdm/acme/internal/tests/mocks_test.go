package tests

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
	"github.com/jmoiron/sqlx"
)

// Mock implementations for dependencies outside the bounded context

// mockDataProviders combines all provider interfaces for testing.
type mockDataProviders struct {
	appCfg *fleet.AppConfig
	assets map[fleet.MDMAssetName]fleet.MDMConfigAsset
	signer acme.CSRSigner
}

func newMockDataProviders(appCfg *fleet.AppConfig, signer acme.CSRSigner, assets map[fleet.MDMAssetName]fleet.MDMConfigAsset) *mockDataProviders {
	return &mockDataProviders{appCfg: appCfg, signer: signer, assets: assets}
}

func (m *mockDataProviders) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return m.appCfg, nil
}

func (m *mockDataProviders) GetAllMDMConfigAssetsByName(ctx context.Context, assetNames []fleet.MDMAssetName,
	queryerContext sqlx.QueryerContext) (map[fleet.MDMAssetName]fleet.MDMConfigAsset, error) {
	return m.assets, nil
}

func (m *mockDataProviders) CSRSigner(ctx context.Context) (acme.CSRSigner, error) {
	return m.signer, nil
}
