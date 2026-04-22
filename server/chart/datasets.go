package chart

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/api"
)

// uptimeRecentlySeenWindow must match the cron schedule cadence so each sample
// reflects activity since the last run.
const uptimeRecentlySeenWindow = 10 * time.Minute

// UptimeDataset implements api.Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string                       { return "uptime" }
func (u *UptimeDataset) BucketSize() time.Duration          { return time.Hour }
func (u *UptimeDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategyAccumulate }
func (u *UptimeDataset) DefaultVisualization() string       { return "checkerboard" }
func (u *UptimeDataset) HasEntityDimension() bool           { return false }
func (u *UptimeDataset) SupportedFilters() []api.FilterDef  { return nil }

func (u *UptimeDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time) error {
	hostIDs, err := store.FindRecentlySeenHostIDs(ctx, uptimeRecentlySeenWindow)
	if err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return nil
	}
	bucketStart := now.UTC().Truncate(u.BucketSize())
	return store.RecordBucketData(ctx, u.Name(), bucketStart, u.BucketSize(), u.SampleStrategy(),
		map[string][]byte{"": HostIDsToBlob(hostIDs)})
}

func (u *UptimeDataset) ResolveFilters(_ context.Context, _ api.DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct{}

func (c *CVEDataset) Name() string                       { return "cve" }
func (c *CVEDataset) BucketSize() time.Duration          { return 24 * time.Hour }
func (c *CVEDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategySnapshot }
func (c *CVEDataset) DefaultVisualization() string       { return "line" }
func (c *CVEDataset) HasEntityDimension() bool           { return true }
func (c *CVEDataset) SupportedFilters() []api.FilterDef  { return nil }

// Collect is a stub — CVE data is recorded externally (by the charts-collect
// tool) via the RecordBucketData path, not via this in-process collector.
func (c *CVEDataset) Collect(_ context.Context, _ api.DatasetStore, _ time.Time) error {
	return nil
}

func (c *CVEDataset) ResolveFilters(_ context.Context, _ api.DatasetStore, _ map[string]string) ([]string, error) {
	return nil, nil
}
