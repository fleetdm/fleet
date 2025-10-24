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

// ModernIngestMDMSoftware handles software ingestion from MDM sources (iOS, iPadOS, macOS apps)
// This would replace the inline software handling in InstalledApplicationListResultsHandler
func ModernIngestMDMSoftware(
	ctx context.Context,
	logger log.Logger,
	host *fleet.Host,
	ds fleet.Datastore,
	software []fleet.Software,
) error {
	// Create the software ingestion service with an adapter
	datastoreAdapter := NewDatastoreAdapter(ds)
	service := NewService(datastoreAdapter, logger)

	// Use the new service to ingest MDM software
	return service.IngestMDMSoftware(ctx, host.ID, host, software)
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

// Eventually, the MDM and osquery packages would be updated to use this service:

/*
// 1. In server/service/osquery_utils/queries.go, replace directIngestSoftware with:

var softwareMacOS = DetailQuery{
	Query: `...`, // existing query
	Platforms: []string{"darwin"},
	DirectIngestFunc: func(ctx context.Context, logger log.Logger, host *fleet.Host, ds fleet.Datastore, rows []map[string]string) error {
		return software_ingestion.ModernDirectIngestSoftware(ctx, logger, host, ds, rows)
	},
}

// 2. In server/service/apple_mdm.go, update NewInstalledApplicationListResultsHandler:

func NewInstalledApplicationListResultsHandler(...) fleet.MDMCommandResultsHandler {
	return func(ctx context.Context, commandResults fleet.MDMCommandResults) error {
		installedAppResult, ok := commandResults.(InstalledApplicationListResult)
		if !ok {
			return ctxerr.New(ctx, "unexpected results type")
		}

		// Get host information
		host, err := ds.Host(ctx, installedAppResult.HostUUID())
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting host for MDM software ingestion")
		}

		// Use the new software ingestion service for iOS/iPadOS apps
		installedApps := installedAppResult.AvailableApps()
		if err := software_ingestion.ModernIngestMDMSoftware(ctx, logger, host, ds, installedApps); err != nil {
			return ctxerr.Wrap(ctx, err, "ingesting MDM software")
		}

		// Continue with existing VPP verification logic...
		// ... rest of the handler
	}
}

// 3. In cmd/fleet/serve.go, inject the software ingestion service:

softwareIngestionService := software_ingestion.NewService(
	software_ingestion.NewDatastoreAdapter(ds),
	logger,
)

// Pass it to services that need it or use it directly in handlers
*/