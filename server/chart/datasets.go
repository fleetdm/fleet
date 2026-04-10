package chart

import (
	"context"
	"time"
)

// UptimeDataset implements Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string        { return "uptime" }
func (u *UptimeDataset) StorageType() StorageType { return StorageTypeBlob }

func (u *UptimeDataset) Collect(ctx context.Context, store DatasetStore, now time.Time) error {
	return store.CollectUptimeChartData(ctx, now)
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ DatasetStore, _ map[string]string) ([]uint, error) {
	return nil, nil
}

func (u *UptimeDataset) SupportedFilters() []FilterDef {
	return nil
}

func (u *UptimeDataset) DefaultVisualization() string {
	return "line"
}

func (u *UptimeDataset) HasEntityDimension() bool {
	return false
}
