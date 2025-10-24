package software_ingestion

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
)

// ModernDirectIngestSoftware is a drop-in replacement for the original directIngestSoftware function
// It uses the new SoftwareIngestionService while maintaining the same interface
func ModernDirectIngestSoftware(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	rows []map[string]string,
) error {
	// Create the software ingestion service with an adapter
	datastoreAdapter := NewDatastoreAdapter(ds)
	service := NewService(datastoreAdapter, logger)

	// Use the new service to ingest the software
	return service.IngestOsquerySoftware(ctx, host.ID, host, rows)
}

// Integration example showing how to use this in the serve command
// This would replace the current service creation in cmd/fleet/serve.go

/*
// In cmd/fleet/serve.go, instead of directly using the monolithic service:

func createSoftwareIngestionService(ds fleet.Datastore, logger log.Logger) *software_ingestion.SoftwareIngestionService {
	datastoreAdapter := software_ingestion.NewDatastoreAdapter(ds)
	return software_ingestion.NewService(datastoreAdapter, logger)
}

// Then in the main service creation:
svc := service.NewService(
	// ... other dependencies
	softwareIngestionService, // inject the software ingestion service
	// ... other dependencies
)

// The main service would then delegate software ingestion calls to this service:
func (svc *Service) IngestHostSoftware(ctx context.Context, host *fleet.Host, softwareRows []map[string]string) error {
	return svc.softwareIngestionService.IngestOsquerySoftware(ctx, host.ID, host, softwareRows)
}
*/

// Eventually, the osquery_utils package would be updated to use this service:
/*
// In server/service/osquery_utils/queries.go, replace directIngestSoftware with:

var softwareMacOS = DetailQuery{
	Query: `...`, // existing query
	Platforms: []string{"darwin"},
	DirectIngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
		return software_ingestion.ModernDirectIngestSoftware(ctx, logger, host, ds, rows)
	},
}
*/