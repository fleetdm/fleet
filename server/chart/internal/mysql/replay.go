package mysql

import (
	"context"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// ReplaySCDChanges answers a snapshot chart request from the change storage. It returns what GetSCDData returns
// with api.SampleStrategySnapshot for the same history: per bucket, the number of hosts in filterMask that at least
// one matching entity holds at the bucket's end, with the same labels and the same meaning of nil and empty
// filterMask and entityIDs. Buckets that end before the change storage started (host_scd_change_state.covered_from)
// hold the state it started from, not the history.
//
// It starts from the current sets of the matching entities, counting for each host how many of them hold it, and
// walks the changes newest first, undoing each. A bucket is recorded once every change at or after its end is
// undone. Memory is the counters plus one change row: the counters are sized by filterMask, and rows are streamed.
func (ds *Datastore) ReplaySCDChanges(
	ctx context.Context,
	dataset string,
	startDate, endDate time.Time,
	bucketSize time.Duration,
	filterMask *roaring.Bitmap,
	entityIDs []string,
) ([]api.DataPoint, error) {
	startDate = startDate.UTC()
	endDate = endDate.UTC()

	numBuckets := int(endDate.Sub(startDate) / bucketSize)
	if numBuckets <= 0 {
		return nil, nil
	}
	// Bucket i is [windowStart + i*bucketSize, windowStart + (i+1)*bucketSize), as in GetSCDData.
	windowStart := startDate.Add(bucketSize)
	results := make([]api.DataPoint, numBuckets)
	for i := range results {
		results[i].Timestamp = windowStart.Add(time.Duration(i) * bucketSize)
	}
	if (entityIDs != nil && len(entityIDs) == 0) || (filterMask != nil && filterMask.IsEmpty()) {
		return results, nil
	}

	db := ds.replica
	if ctxdb.IsPrimaryRequired(ctx) {
		db = ds.primary
	}
	// One snapshot for every read: a collection that commits between reading the current sets and reading the
	// changes would leave a change whose effect is missing from the sets already read.
	err := platform_mysql.WithReadOnlyTxx(ctx, db, func(tx platform_mysql.DBReadTx) error {
		keys, err := ds.replayEntityKeys(ctx, tx, dataset, entityIDs)
		if err != nil {
			return err
		}
		if keys != nil && len(keys) == 0 {
			return nil
		}
		counts := newReplayCounts(filterMask)
		if err := ds.replayCurrentSets(ctx, tx, dataset, keys, counts); err != nil {
			return err
		}

		next := numBuckets - 1
		bucketEnd := func(i int) time.Time { return windowStart.Add(time.Duration(i+1) * bucketSize) }
		// record fills every bucket whose end is after at, the time of the next change to undo.
		record := func(at time.Time) {
			for next >= 0 && at.Before(bucketEnd(next)) {
				results[next].Value = counts.nonzero
				next--
			}
		}

		// Changes before the first bucket's end count for every bucket, so they are not read.
		rows, err := tx.QueryxContext(ctx, //nolint:sqlclosecheck // closed below
			`SELECT changed_at, changes, encoding_type FROM host_scd_changes
			 WHERE dataset = ? AND changed_at >= ?
			 ORDER BY changed_at DESC, seq DESC`,
			dataset, bucketEnd(0))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "read chart changes")
		}
		defer rows.Close()
		var r struct {
			ChangedAt    time.Time `db:"changed_at"`
			Changes      []byte    `db:"changes"`
			EncodingType uint8     `db:"encoding_type"`
		}
		for rows.Next() {
			if err := rows.StructScan(&r); err != nil {
				return ctxerr.Wrap(ctx, err, "scan chart changes")
			}
			if r.EncodingType != changesEncodingV1 {
				return ctxerr.Errorf(ctx, "unknown chart changes encoding %d at %s", r.EncodingType, r.ChangedAt)
			}
			record(r.ChangedAt)
			err := walkChanges(r.Changes, func(key uint32, added, removed []byte) error {
				if keys != nil {
					if _, ok := keys[key]; !ok {
						return nil
					}
				}
				if err := counts.applyBlob(added, -1); err != nil {
					return err
				}
				return counts.applyBlob(removed, 1)
			})
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "decode chart changes at %s", r.ChangedAt)
			}
		}
		if err := rows.Err(); err != nil {
			return ctxerr.Wrap(ctx, err, "iterate chart changes")
		}
		record(time.Time{})
		return nil
	}, ds.logger)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// replayEntityKeys resolves entityIDs to the dataset's entity keys. It returns nil when entityIDs is nil (every
// entity), and an empty set when none of them has a key.
func (ds *Datastore) replayEntityKeys(ctx context.Context, tx platform_mysql.DBReadTx, dataset string, entityIDs []string) (map[uint32]struct{}, error) {
	if entityIDs == nil {
		return nil, nil
	}
	var rows []struct {
		ID       uint32 `db:"id"`
		EntityID string `db:"entity_id"`
	}
	keys := make(map[uint32]struct{}, len(entityIDs))
	if len(entityIDs) <= scdEntityInListMax {
		query, args, err := sqlx.In(`SELECT id, entity_id FROM host_scd_entities WHERE dataset = ? AND entity_id IN (?)`, dataset, entityIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "expand chart entity key args")
		}
		if err := sqlx.SelectContext(ctx, tx, &rows, tx.Rebind(query), args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "read chart entity keys")
		}
		for _, r := range rows {
			keys[r.ID] = struct{}{}
		}
		return keys, nil
	}

	if err := sqlx.SelectContext(ctx, tx, &rows, `SELECT id, entity_id FROM host_scd_entities WHERE dataset = ?`, dataset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read chart entity keys")
	}
	allow := make(map[string]struct{}, len(entityIDs))
	for _, id := range entityIDs {
		allow[id] = struct{}{}
	}
	for _, r := range rows {
		if _, ok := allow[r.EntityID]; ok {
			keys[r.ID] = struct{}{}
		}
	}
	return keys, nil
}

// replayCurrentSets adds the current sets of the entities in keys (every entity when nil) to counts.
func (ds *Datastore) replayCurrentSets(ctx context.Context, tx platform_mysql.DBReadTx, dataset string, keys map[uint32]struct{}, counts *replayCounts) error {
	query := `SELECT entity_key, host_bitmap, encoding_type FROM host_scd_current WHERE dataset = ?`
	args := []any{dataset}
	filterInGo := len(keys) > scdEntityInListMax
	if keys != nil && !filterInGo {
		list := make([]uint32, 0, len(keys))
		for k := range keys {
			list = append(list, k)
		}
		var err error
		query, args, err = sqlx.In(query+` AND entity_key IN (?)`, dataset, list)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand chart current set args")
		}
		query = tx.Rebind(query)
	}
	rows, err := tx.QueryxContext(ctx, query, args...) //nolint:sqlclosecheck // closed below
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read chart current sets")
	}
	defer rows.Close()
	var r struct {
		EntityKey    uint32 `db:"entity_key"`
		HostBitmap   []byte `db:"host_bitmap"`
		EncodingType uint8  `db:"encoding_type"`
	}
	for rows.Next() {
		if err := rows.StructScan(&r); err != nil {
			return ctxerr.Wrap(ctx, err, "scan chart current set")
		}
		if filterInGo {
			if _, ok := keys[r.EntityKey]; !ok {
				continue
			}
		}
		rb, err := chart.DecodeBitmap(chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType})
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "decode current set for entity key %d", r.EntityKey)
		}
		counts.apply(rb, 1)
	}
	if err := rows.Err(); err != nil {
		return ctxerr.Wrap(ctx, err, "iterate chart current sets")
	}
	return nil
}

// replayCounts holds, per host, how many matching entities hold it, and how many hosts have a count above zero.
type replayCounts struct {
	// member has a bit per host in the mask; nil when every host counts. Checking it per host costs less than
	// intersecting every change with the mask, which allocates a bitmap per change.
	member  []uint64
	counts  []int32
	nonzero int
}

func newReplayCounts(mask *roaring.Bitmap) *replayCounts {
	c := &replayCounts{}
	if mask != nil {
		n := 0
		if !mask.IsEmpty() {
			n = int(mask.Maximum()) + 1
		}
		c.counts = make([]int32, n)
		c.member = make([]uint64, (n+63)/64)
		mask.Iterate(func(h uint32) bool {
			c.member[h/64] |= 1 << (h % 64)
			return true
		})
	}
	return c
}

func (c *replayCounts) applyBlob(b []byte, delta int32) error {
	if len(b) == 0 {
		return nil
	}
	rb, err := chart.DecodeBitmap(chart.Blob{Bytes: b, Encoding: chart.EncodingRoaring})
	if err != nil {
		return err
	}
	c.apply(rb, delta)
	return nil
}

func (c *replayCounts) apply(rb *roaring.Bitmap, delta int32) {
	if rb.IsEmpty() {
		return
	}
	// Without a mask the counters grow to the largest host seen. With one, they already cover every host that counts.
	if top := int(rb.Maximum()); c.member == nil && top >= len(c.counts) {
		c.counts = append(c.counts, make([]int32, top+1-len(c.counts))...)
	}
	rb.Iterate(func(h uint32) bool {
		if c.member != nil && (int(h) >= len(c.counts) || c.member[h/64]&(1<<(h%64)) == 0) {
			return true
		}
		before := c.counts[h]
		after := before + delta
		c.counts[h] = after
		switch {
		case before <= 0 && after > 0:
			c.nonzero++
		case before > 0 && after <= 0:
			c.nonzero--
		}
		return true
	})
}
