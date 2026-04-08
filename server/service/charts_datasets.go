package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// UptimeDataset implements fleet.ChartDataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string { return "uptime" }

func (u *UptimeDataset) Collect(ctx context.Context, ds fleet.Datastore, now time.Time) error {
	return ds.CollectUptimeChartData(ctx, now)
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ fleet.Datastore, _ map[string]string) ([]uint, error) {
	return nil, nil
}

func (u *UptimeDataset) SupportedFilters() []fleet.ChartFilterDef {
	return nil
}

func (u *UptimeDataset) DefaultVisualization() string {
	return "line"
}

func (u *UptimeDataset) HasEntityDimension() bool {
	return false
}
