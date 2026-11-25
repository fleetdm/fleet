package chart

import (
	"context"
	"errors"
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

// cveConvertBudget bounds the time one collection spends converting the history collected before the change
// storage started. var (not const) so tests can change it.
var cveConvertBudget = 5 * time.Minute

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct{}

func (c *CVEDataset) Name() string                       { return api.MetricCVE }
func (c *CVEDataset) DefaultResolutionHours() int        { return 3 }
func (c *CVEDataset) SampleStrategy() api.SampleStrategy { return api.SampleStrategySnapshot }
func (c *CVEDataset) DefaultVisualization() string       { return "line" }

func (c *CVEDataset) Collect(ctx context.Context, store api.DatasetStore, now time.Time, disabledFleetIDs []uint) error {
	// Collect CVEs at all severities on the curated set of tracked software and
	// OS vulnerabilities. Display-time narrowing (critical-only this round,
	// plus user filters) happens at read time via ResolveCVEChartEntities.
	tracked, err := store.CollectibleCVEs(ctx)
	if err != nil {
		return err
	}

	// The store sets bits while streaming the vulnerability joins, so peak
	// memory here is one bitmap per CVE — never the raw (CVE, host) pairs.
	bitmaps, err := store.AffectedHostIDsByCVE(ctx, disabledFleetIDs, tracked)
	if err != nil {
		return err
	}
	bucketStart := now.UTC().Truncate(time.Hour)
	// Always call RecordBucketData, even when bitmaps is empty: snapshot
	// semantics use an empty input to close any open rows for entities no
	// longer in the tracked set (recordSnapshot's "absent entities" branch).
	// It goes first so a failure in the change storage cannot affect the rows
	// the chart reads today.
	if err := store.RecordBucketData(ctx, c.Name(), bucketStart, time.Hour, c.SampleStrategy(), bitmaps); err != nil {
		return err
	}
	// A busy change storage means a scrub is running; the next collection
	// records this hour's changes.
	if err := store.RecordSCDChanges(ctx, c.Name(), bucketStart, bitmaps); err != nil {
		if errors.Is(err, api.ErrChangeStorageBusy) {
			return nil
		}
		return err
	}

	started := time.Now()
	for {
		more, err := store.ConvertSCDHistory(ctx, c.Name())
		if errors.Is(err, api.ErrChangeStorageBusy) {
			return nil
		}
		if err != nil || !more || time.Since(started) >= cveConvertBudget {
			return err
		}
	}
}
