package tests

import (
	"context"
	"errors"

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

// Returns a valid row with `valid-serial` else no row
func (m *mockDataProviders) GetHostDEPAssignmentsBySerial(ctx context.Context, serial string) ([]*fleet.HostDEPAssignment, error) {
	// For testing, we can return a fixed response or an empty slice based on the serial number
	if serial == "valid-serial" {
		return []*fleet.HostDEPAssignment{
			{
				HostID: 1,
			},
		}, nil
	} else if serial == "error-serial" {
		return nil, errors.New("Mocked error for GetHostDEPAssignmentsBySerial")
	}
	return []*fleet.HostDEPAssignment{}, nil
}
