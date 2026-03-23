package tests

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// Mock implementations for dependencies outside the bounded context

// mockDataProviders combines all provider interfaces for testing.
type mockDataProviders struct {
	appCfg *fleet.AppConfig
}

func newMockDataProviders(appCfg *fleet.AppConfig) *mockDataProviders {
	return &mockDataProviders{appCfg: appCfg}
}

func (m *mockDataProviders) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	return m.appCfg, nil
}
