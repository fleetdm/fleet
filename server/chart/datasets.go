package chart

import (
	"context"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart/api"
)

// UptimeDataset implements api.Dataset for host uptime tracking.
type UptimeDataset struct{}

func (u *UptimeDataset) Name() string                       { return "uptime" }
func (u *UptimeDataset) DefaultResolutionHours() int        { return 3 }
func (u *UptimeDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategyAccumulate }
func (u *UptimeDataset) DefaultVisualization() string       { return "checkerboard" }

func (u *UptimeDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
	hostIDs, err := store.FindOnlineHostIDs(ctx, now, disabledFleetIDs)
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
		map[string]*roaring.Bitmap{"": NewBitmap(hostIDs)})
}

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct{}

func (c *CVEDataset) Name() string                       { return "cve" }
func (c *CVEDataset) DefaultResolutionHours() int        { return 3 }
func (c *CVEDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategySnapshot }
func (c *CVEDataset) DefaultVisualization() string       { return "line" }

func (c *CVEDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
	// Only track the CVEs that the chart API currently returns.
	// TODO: implement bitmap compression so we can track all CVEs.
	tracked, err := store.TrackedCriticalCVEs(ctx)
	if err != nil {
		return err
	}

	hostIDsByCVE, err := store.AffectedHostIDsByCVE(ctx, disabledFleetIDs, tracked)
	if err != nil {
		return err
	}
	bitmaps := make(map[string]*roaring.Bitmap, len(hostIDsByCVE))
	for cve, hostIDs := range hostIDsByCVE {
		bitmaps[cve] = NewBitmap(hostIDs)
	}
	bucketStart := now.UTC().Truncate(time.Hour)
	// Always call RecordBucketData, even when bitmaps is empty: snapshot
	// semantics use an empty input to close any open rows for entities no
	// longer in the tracked set (recordSnapshot's "absent entities" branch).
	return store.RecordBucketData(ctx, c.Name(), bucketStart, time.Hour, c.SampleStrategy(), bitmaps)
}
