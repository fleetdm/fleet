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

// 3. In cmd/fleet/serve.go, inject the software ingestion service with load management:

func createSoftwareIngestionService(ds fleet.Datastore, config configpkg.FleetConfig, logger log.Logger) SoftwareIngestionService {
	// Create base service
	datastoreAdapter := software_ingestion.NewDatastoreAdapter(ds)
	baseService := software_ingestion.NewService(datastoreAdapter, logger)

	// Configure load management
	loadConfig := software_ingestion.LoadManagementConfig{
		MaxRequestsPerSecond:   float64(config.Osquery.MaxRequestsPerSecond), // Reuse existing config
		BurstSize:             100,
		FailureThreshold:      5,
		RecoveryTimeout:       30 * time.Second,
		BatchSize:             10,
		BatchTimeout:          100 * time.Millisecond,
		MaxBatchDelay:         1 * time.Second,
		MaxConcurrentHosts:    20,
		DatabaseTimeout:       10 * time.Second,
		EnableAsyncProcessing: config.Osquery.EnableAsyncHostProcessing, // Reuse existing config
		AsyncQueueSize:        1000,
	}

	// Wrap with load management
	loadManagedService := software_ingestion.NewLoadManagedService(baseService, loadConfig, logger)

	// Optionally wrap with async processing for high-load environments
	if loadConfig.EnableAsyncProcessing {
		return software_ingestion.NewAsyncProcessor(loadManagedService, loadConfig, logger)
	}

	return loadManagedService
}

// Usage in serve.go:
softwareIngestionService := createSoftwareIngestionService(ds, config, logger)

// Start periodic tracking reports
softwareIngestionService.StartTrackingReports(ctx)

// Add monitoring endpoints
healthHandler := software_ingestion.NewHealthHandler(softwareIngestionService, logger)
rootMux.Handle("/api/v1/fleet/software_ingestion/health", healthHandler)

monitoringHandler := software_ingestion.NewMonitoringHandler(softwareIngestionService.GetTracker(), logger)
rootMux.Handle("/api/v1/fleet/software_ingestion/tracking/", monitoringHandler)

// For Grafana/Prometheus integration
rootMux.Handle("/api/v1/fleet/software_ingestion/metrics", software_ingestion.NewPrometheusHandler(softwareIngestionService))
*/

// Example monitoring queries:

/*
// Check overall ingestion health
curl http://localhost:8080/api/v1/fleet/software_ingestion/tracking/summary

// Find hosts that haven't updated in > 1.5 hours
curl http://localhost:8080/api/v1/fleet/software_ingestion/tracking/stale_hosts

// Check specific host status
curl "http://localhost:8080/api/v1/fleet/software_ingestion/tracking/host_status?host_id=123"

// Get active alerts
curl http://localhost:8080/api/v1/fleet/software_ingestion/tracking/alerts

// Expected response for healthy state:
{
  "total_hosts": 1000,
  "healthy_hosts": 950,
  "stale_hosts": 30,
  "over_active_hosts": 20,
  "active_hosts": 980,
  "health_percentage": 95.0,
  "average_ingestion_rate": 1.2,
  "max_ingestion_rate": 2.1,
  "timestamp": "2024-01-15T10:30:00Z"
}

// For alerting integration (e.g., with Slack, PagerDuty):
if stale_hosts > 100 OR health_percentage < 90 {
  send_alert("Software ingestion health degraded")
}
*/