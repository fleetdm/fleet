package tests

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
)

// Mock implementations for dependencies outside the bounded context

// mockDataProviders combines all provider interfaces for testing.
type mockDataProviders struct {
	appCfg *fleet.AppConfig
	signer acme.CSRSigner
}

func newMockDataProviders(appCfg *fleet.AppConfig, signer acme.CSRSigner) *mockDataProviders {
	return &mockDataProviders{appCfg: appCfg, signer: signer}
}

func (m *mockDataProviders) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return m.appCfg, nil
}

func (m *mockDataProviders) CSRSigner(ctx context.Context) (acme.CSRSigner, error) {
	return m.signer, nil
}
