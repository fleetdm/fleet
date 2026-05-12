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
func (u *UptimeDataset) DefaultResolutionHours() int        { return 3 }
func (u *UptimeDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategyAccumulate }
func (u *UptimeDataset) DefaultVisualization() string       { return "checkerboard" }

func (u *UptimeDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
	hostIDs, err := store.FindRecentlySeenHostIDs(ctx, now.Add(-uptimeRecentlySeenWindow), disabledFleetIDs)
	if err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return nil
	}
	bucketStart := now.UTC().Truncate(time.Hour)
	return store.RecordBucketData(ctx, u.Name(), bucketStart, time.Hour, u.SampleStrategy(),
		// The empty string key means "all entities" since uptime isn't tracked per host.
		// The value is a bitmap of host IDs that were active in this bucket.
		map[string][]byte{"": HostIDsToBlob(hostIDs)})
}

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct{}

func (c *CVEDataset) Name() string                       { return "cve" }
func (c *CVEDataset) DefaultResolutionHours() int        { return 3 }
func (c *CVEDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategySnapshot }
func (c *CVEDataset) DefaultVisualization() string       { return "line" }

func (c *CVEDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
	// TODO(iteration-2): once roaring-bitmap encoding lands, drop the
	// TrackedCriticalCVEs scoping and pass nil to AffectedHostIDsByCVE so
	// every CVE is collected again. See the iteration-2 TODO in
	// server/chart/internal/mysql/charts.go.
	tracked, err := store.TrackedCriticalCVEs(ctx)
	if err != nil {
		return err
	}

	hostIDsByCVE, err := store.AffectedHostIDsByCVE(ctx, disabledFleetIDs, tracked)
	if err != nil {
		return err
	}
	bitmaps := make(map[string][]byte, len(hostIDsByCVE))
	for cve, hostIDs := range hostIDsByCVE {
		bitmaps[cve] = HostIDsToBlob(hostIDs)
	}
	bucketStart := now.UTC().Truncate(time.Hour)
	// Always call RecordBucketData, even when bitmaps is empty: snapshot
	// semantics use an empty input to close any open rows for entities no
	// longer in the tracked set (recordSnapshot's "absent entities" branch).
	return store.RecordBucketData(ctx, c.Name(), bucketStart, time.Hour, c.SampleStrategy(), bitmaps)
}
