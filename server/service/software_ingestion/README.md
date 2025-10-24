# Software Ingestion Service

A modular service for ingesting software data from multiple sources (osquery, iOS MDM, etc.) into Fleet's database.

## Overview

This service demonstrates the **modular monolith** pattern by:
- **Single Responsibility**: Only handles software ingestion writes
- **Clear Boundaries**: Owns specific database tables (`software`, `host_software`, `host_software_installed_paths`)
- **Dependency Injection**: Uses minimal interface instead of full datastore
- **Testable**: Easy to mock and unit test

## Usage

### Osquery Software Ingestion
```go
// Create service
datastoreAdapter := software_ingestion.NewDatastoreAdapter(ds)
service := software_ingestion.NewService(datastoreAdapter, logger)

// Ingest osquery software data
err := service.IngestOsquerySoftware(ctx, host.ID, host, osqueryRows)
```

### iOS/MDM Software Ingestion
```go
// Ingest iOS apps from MDM
err := service.IngestMDMSoftware(ctx, host.ID, host, iosApps)
```

### Drop-in Replacements
```go
// Replace directIngestSoftware calls
err := software_ingestion.ModernDirectIngestSoftware(ctx, logger, host, ds, rows)

// Replace inline MDM software handling
err := software_ingestion.ModernIngestMDMSoftware(ctx, logger, host, ds, software)
```

## Testing

### Generate Mocks
```bash
# From the Fleet root directory
make mock

# Or generate just this service's mocks
cd server/service/software_ingestion
go generate
```

### Run Tests
```bash
cd server/service/software_ingestion
go test ./...
```

### Example Test
```go
func TestSoftwareIngestion(t *testing.T) {
    // Use generated mock
    mockDS := &MockSoftwareDatastore{}

    // Set up expectations
    mockDS.UpdateHostSoftwareFunc = func(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error) {
        assert.Equal(t, expectedHostID, hostID)
        assert.Len(t, software, expectedCount)
        return &fleet.UpdateHostSoftwareDBResult{}, nil
    }

    // Test the service
    service := NewService(mockDS, logger)
    err := service.IngestOsquerySoftware(ctx, hostID, host, rows)

    // Verify
    require.NoError(t, err)
    assert.True(t, mockDS.UpdateHostSoftwareInvoked)
}
```

## Architecture

### Database Ownership
- `software` - Software catalog entries
- `host_software` - Host-to-software relationships
- `host_software_installed_paths` - Installation path tracking

### Service Interface
```go
type SoftwareIngestionService interface {
    IngestOsquerySoftware(ctx context.Context, hostID uint, host *fleet.Host, softwareRows []map[string]string) error
    IngestMDMSoftware(ctx context.Context, hostID uint, host *fleet.Host, software []fleet.Software) error
}
```

### Datastore Interface
```go
type SoftwareDatastore interface {
    UpdateHostSoftware(ctx context.Context, hostID uint, software []fleet.Software) (*fleet.UpdateHostSoftwareDBResult, error)
    UpdateHostSoftwareInstalledPaths(ctx context.Context, hostID uint, reported map[string]struct{}, mutationResults *fleet.UpdateHostSoftwareDBResult) error
    CleanupOrphanSoftware(ctx context.Context) error
}
```

## Integration

### Current Integration Points
1. **osquery_utils/queries.go**: Replace `directIngestSoftware` calls
2. **apple_mdm.go**: Use in `InstalledApplicationListResultsHandler`
3. **serve.go**: Inject service into main service creation

### Migration Strategy
1. **Phase 1**: Deploy alongside existing code
2. **Phase 2**: Update callers to use new service
3. **Phase 3**: Remove old implementations
4. **Phase 4**: Extract to separate module if needed

## Benefits

- ✅ **Focused Responsibility**: Only software ingestion
- ✅ **Platform Support**: osquery, iOS, iPadOS, macOS apps
- ✅ **Easy Testing**: Minimal dependencies, clear mocks
- ✅ **Maintainable**: Clear separation of concerns
- ✅ **Extensible**: Easy to add new software sources