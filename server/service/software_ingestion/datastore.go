package software_ingestion

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// SoftwareDatastore defines the interface for software ingestion database operations
// that the SoftwareIngestionService needs.
//
// This interface should be implemented by the MySQL datastore and can be mocked for testing.
// Note: Vulnerability operations are handled by a separate VulnerabilitiesService.
type SoftwareDatastore interface {
	// Software and Host Software Management
	UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error)
	UpdateHostSoftwareInstalledPaths(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error

	// Cleanup and Maintenance
	CleanupOrphanSoftware(ctx context.Context) error
}

// DatastoreAdapter adapts the full fleet.Datastore to only expose software-related methods
// This ensures the SoftwareIngestionService only depends on what it actually needs
type DatastoreAdapter struct {
	ds fleet.Datastore
}

// NewDatastoreAdapter creates an adapter that implements SoftwareDatastore
func NewDatastoreAdapter(ds fleet.Datastore) SoftwareDatastore {
	return &DatastoreAdapter{ds: ds}
}

func (a *DatastoreAdapter) UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
	return a.ds.UpdateHostSoftware(ctx, hostID, software)
}

func (a *DatastoreAdapter) UpdateHostSoftwareInstalledPaths(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error {
	return a.ds.UpdateHostSoftwareInstalledPaths(ctx, hostID, reported, mutationResults)
}

func (a *DatastoreAdapter) CleanupOrphanSoftware(ctx context.Context) error {
	// This would be a new method to clean up software entries that are no longer referenced
	// Implementation would be added to the MySQL datastore
	return nil // placeholder
}