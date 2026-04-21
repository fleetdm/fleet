package chart

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/api"
)

// UptimeDataset implements api.Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string                 { return "uptime" }
func (u *UptimeDataset) StorageType() api.StorageType { return api.StorageTypeBlob }

func (u *UptimeDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time) error {
	return store.CollectUptimeChartData(ctx, now)
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ api.DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *UptimeDataset) SupportedFilters() []api.FilterDef {
	return nil
}

func (u *UptimeDataset) DefaultVisualization() string {
	return "checkerboard"
}

func (u *UptimeDataset) HasEntityDimension() bool {
	return false
}

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct{}

func (u *CVEDataset) Name() string                 { return "cve" }
func (u *CVEDataset) StorageType() api.StorageType { return api.StorageTypeSCD }

// Collect is a stub for now — CVE SCD data is recorded externally (by the
// charts-collect script) via the SCD record path, not via this in-process collector.
func (u *CVEDataset) Collect(_ context.Context, _ api.DatasetStore, _ time.Time) error {
	return nil
}

func (u *CVEDataset) ResolveFilters(_ context.Context, _ api.DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *CVEDataset) SupportedFilters() []api.FilterDef {
	return nil
}

func (u *CVEDataset) DefaultVisualization() string {
	return "line"
}

func (u *CVEDataset) HasEntityDimension() bool {
	return true
}
