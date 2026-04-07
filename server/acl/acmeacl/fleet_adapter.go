// Package acmeacl provides the anti-corruption layer between the ACME
// bounded context and legacy Fleet code.
//
// This package is the ONLY place that imports both ACME types and fleet types.
// It translates between them, allowing the ACME context to remain decoupled
// from legacy code.
package acmeacl

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/acme"
)

// FleetDatastoreAdapter adapts fleet.Datastore to the narrow
// acme.DataProviders interface that the ACME bounded context requires.
type FleetDatastoreAdapter struct {
	ds     fleet.Datastore
	signer acme.CSRSigner
}

// NewFleetDatastoreAdapter creates a new adapter for the Fleet datastore.
func NewFleetDatastoreAdapter(ds fleet.Datastore, signer acme.CSRSigner) *FleetDatastoreAdapter {
	return &FleetDatastoreAdapter{ds: ds, signer: signer}
}

// Ensure FleetDatastoreAdapter implements acme.DataProviders
var _ acme.DataProviders = (*FleetDatastoreAdapter)(nil)

func (a *FleetDatastoreAdapter) ServerURL(ctx context.Context) (string, error) {
	appCfg, err := a.ds.AppConfig(ctx)
	if err != nil {
		return "", err
	}
	return appCfg.MDMUrl(), nil
}

func (a *FleetDatastoreAdapter) GetCACertificatePEM(ctx context.Context) ([]byte, error) {
	assets, err := a.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetCACert}, nil)
	if err != nil {
		return nil, err
	}
	return assets[fleet.MDMAssetCACert].Value, nil
}

func (a *FleetDatastoreAdapter) CSRSigner(_ context.Context) (acme.CSRSigner, error) {
	return a.signer, nil
}

func (a *FleetDatastoreAdapter) IsDEPEnrolled(ctx context.Context, serial string) (bool, error) {
	assignments, err := a.ds.GetHostDEPAssignmentsBySerial(ctx, serial)
	if err != nil {
		return false, err
	}
	return len(assignments) > 0, nil
}
