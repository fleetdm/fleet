package chart

import (
	"context"
	"errors"
	"time"
)

// UptimeDataset implements Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string             { return "uptime" }
func (u *UptimeDataset) StorageType() StorageType { return StorageTypeBlob }

func (u *UptimeDataset) Collect(ctx context.Context, store DatasetStore, now time.Time) error {
	return store.CollectUptimeChartData(ctx, now)
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *UptimeDataset) SupportedFilters() []FilterDef {
	return nil
}

func (u *UptimeDataset) DefaultVisualization() string {
	return "checkerboard"
}

func (u *UptimeDataset) HasEntityDimension() bool {
	return false
}

// CVEDataset implements Dataset for host CVE tracking.
type CVEDataset struct{}

func (u *CVEDataset) Name() string             { return "cve" }
func (u *CVEDataset) StorageType() StorageType { return StorageTypeBlob }

func (u *CVEDataset) Collect(ctx context.Context, store DatasetStore, now time.Time) error {
	return errors.New("CVE dataset collection not implemented yet")
}

func (u *CVEDataset) ResolveFilters(_ context.Context, _ DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

func (u *CVEDataset) SupportedFilters() []FilterDef {
	return nil
}

func (u *CVEDataset) DefaultVisualization() string {
	return "line"
}

func (u *CVEDataset) HasEntityDimension() bool {
	return false
}
