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

// aggregateBackfillBatchesPerTick rations how much history one tick rebuilds,
// keeping the tick's cost flat however many series are catching up. Series
// still get their current row every tick. One at a time also means they finish
// in order, so the band the chart opens on stops being slow soonest.
const aggregateBackfillBatchesPerTick = 1

// CVEDataset implements api.Dataset for host CVE tracking.
type CVEDataset struct {
	// PreaggregateFilters lists the filters whose per-bucket unions are stored
	// at collection time, so a request carrying one reads a single series
	// instead of re-unioning thousands of per-CVE bitmaps. Empty disables it.
	PreaggregateFilters []api.CVEFilter
}

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
	if bitmaps == nil {
		// The store may return nil; addAggregates writes into the map.
		bitmaps = make(map[string]*roaring.Bitmap)
	}
	// Aggregates join the same map: snapshot semantics close every entity
	// missing from the input, so a separate call would close the other's rows.
	if err := c.addAggregates(ctx, store, bitmaps, now); err != nil {
		return err
	}

	bucketStart := now.UTC().Truncate(time.Hour)
	// Always call RecordBucketData, even when bitmaps is empty: snapshot
	// semantics use an empty input to close any open rows for entities no
	// longer in the tracked set (recordSnapshot's "absent entities" branch).
	return store.RecordBucketData(ctx, c.Name(), bucketStart, time.Hour, c.SampleStrategy(), bitmaps)
}

// addAggregates adds one entity per precomputed filter, holding the union of
// hosts affected by every CVE it selects. Errors abort the tick: a partial map
// would close series the previous tick opened, leaving gaps nothing repairs.
func (c *CVEDataset) addAggregates(
	ctx context.Context,
	store api.DatasetStore,
	bitmaps map[string]*roaring.Bitmap,
	now time.Time,
) error {
	backfillBudget := aggregateBackfillBatchesPerTick
	horizon := now.AddDate(0, 0, -api.RetentionDays)
	for _, filter := range c.PreaggregateFilters {
		entityID := filter.AggregateEntityID()
		if _, done := bitmaps[entityID]; done {
			// Another filter in this batch canonicalizes to the same CVE set.
			continue
		}

		cves, err := store.ResolveCVEChartEntities(ctx, filter)
		if err != nil {
			return err
		}

		// Never nil: a filter matching nothing stores a zero-valued row rather
		// than vanishing from the snapshot and closing its own series.
		union := roaring.New()
		for _, cve := range cves {
			if rb, ok := bitmaps[cve]; ok {
				union.Or(rb)
			}
		}
		bitmaps[entityID] = union

		// Before the write below, which would otherwise leave a new series
		// looking established with no history behind it. A finished series
		// declines and costs no budget, so others aren't starved.
		if backfillBudget > 0 {
			wrote, err := store.BackfillAggregateEntity(ctx, c.Name(), entityID, cves, now, horizon)
			if err != nil {
				return err
			}
			if wrote {
				backfillBudget--
			}
		}
	}
	return nil
}
