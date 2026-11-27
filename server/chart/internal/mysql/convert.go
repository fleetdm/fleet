package mysql

import (
	"cmp"
	"container/heap"
	"context"
	"slices"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdConvertDays is how far before changes_from the conversion starts. Replay reads changes from less than 31 days
// back, so earlier changes are never read.
const scdConvertDays = 32

// scdConvertStepHours is how many hours of history one conversion step covers. var (not const) so tests can
// change it.
var scdConvertStepHours = 24

// ConvertSCDHistory converts one step of the host_scd_data history collected before the change storage started
// (host_scd_change_state.changes_from) into changes, and reports whether more remains. The conversion covers
// [changes_from - scdConvertDays, changes_from); the collector owns everything after it. When the last step is
// done, covered_from moves to the start of the conversion.
//
// Each entity's host set at an instant is the union of its rows with valid_from <= instant < valid_to, which is
// what the snapshot read counts at a bucket end after that instant. A change is written at every instant where
// that set changes. A step starts from the rows live just before its start, so it does not depend on an earlier
// step, and replaces what an earlier attempt of the same step wrote.
//
// While another writer of the dataset holds the change storage lock, it returns api.ErrChangeStorageBusy.
func (ds *Datastore) ConvertSCDHistory(ctx context.Context, dataset string) (bool, error) {
	var more bool
	err := ds.withSCDChangeLock(ctx, dataset, false, func() error {
		var err error
		more, err = ds.convertSCDHistoryStep(ctx, dataset)
		return err
	})
	return more, err
}

func (ds *Datastore) convertSCDHistoryStep(ctx context.Context, dataset string) (bool, error) {
	// A scrub that ran just before the step is not on the replicas yet.
	ctx = ctxdb.RequirePrimary(ctx, true)

	var states []struct {
		ChangesFrom time.Time  `db:"changes_from"`
		Cursor      *time.Time `db:"convert_cursor"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &states,
		`SELECT changes_from, convert_cursor FROM host_scd_change_state WHERE dataset = ?`, dataset); err != nil {
		return false, ctxerr.Wrap(ctx, err, "read chart change state")
	}
	if len(states) == 0 {
		return false, nil
	}
	changesFrom := states[0].ChangesFrom.UTC()
	start := changesFrom.AddDate(0, 0, -scdConvertDays)
	stepStart := start
	if states[0].Cursor != nil {
		stepStart = states[0].Cursor.UTC()
	}
	if !stepStart.Before(changesFrom) {
		return false, nil
	}

	w := &convertWalk{live: map[string][]liveSCDRow{}}
	if err := ds.loadLiveSCDRows(ctx, dataset, stepStart, w); err != nil {
		return false, err
	}
	from := stepStart
	if len(w.live) == 0 {
		// Nothing is live, so nothing changes until the next row starts.
		var next *time.Time
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &next,
			`SELECT MIN(valid_from) FROM host_scd_data
			 WHERE dataset = ? AND valid_from >= ? AND valid_from < ? AND valid_to > valid_from`,
			dataset, stepStart, changesFrom); err != nil {
			return false, ctxerr.Wrap(ctx, err, "find next chart history row")
		}
		from = changesFrom
		if next != nil {
			from = next.UTC()
		}
	}
	to := from.Add(time.Duration(scdConvertStepHours) * time.Hour)
	if to.After(changesFrom) {
		to = changesFrom
	}

	if err := ds.deleteInBatches(ctx,
		`DELETE FROM host_scd_changes WHERE dataset = ? AND changed_at >= ? AND changed_at < ? ORDER BY changed_at, seq LIMIT ?`,
		scdChangeDeleteBatch, dataset, stepStart, to); err != nil {
		return false, ctxerr.Wrap(ctx, err, "clear converted chart changes")
	}
	if from.Before(to) {
		if err := ds.convertSCDRows(ctx, dataset, from, to, w); err != nil {
			return false, err
		}
	}

	done := !to.Before(changesFrom)
	stmt := `UPDATE host_scd_change_state SET convert_cursor = ? WHERE dataset = ? AND changes_from = ?`
	args := []any{to, dataset, changesFrom}
	if done {
		stmt = `UPDATE host_scd_change_state SET convert_cursor = ?, covered_from = ? WHERE dataset = ? AND changes_from = ?`
		args = []any{to, start, dataset, changesFrom}
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return false, ctxerr.Wrap(ctx, err, "save chart conversion progress")
	}
	if done {
		ds.logger.InfoContext(ctx, "chart history converted to changes", "dataset", dataset, "covered_from", start)
	}
	return !done, nil
}

// loadLiveSCDRows adds the rows live just before at to w.
func (ds *Datastore) loadLiveSCDRows(ctx context.Context, dataset string, at time.Time, w *convertWalk) error {
	// Without the hint MySQL picks uniq_entity_bucket, which cannot test valid_to, so it reads every row that started
	// before at, bitmap included: 744k pages against 12k on 720 hours of 10k-host history.
	return ds.forEachConvertRow(ctx,
		`SELECT entity_id, host_bitmap, encoding_type, valid_from, valid_to FROM host_scd_data FORCE INDEX (idx_dataset_range)
		 WHERE dataset = ? AND valid_from < ? AND valid_to >= ? AND valid_to > valid_from`,
		[]any{dataset, at, at}, func(r scdRow, hosts *roaring.Bitmap) error {
			w.add(r.EntityID, hosts, r.ValidTo.UTC())
			return nil
		})
}

// convertSCDRows writes the changes in [from, to), starting from the rows in w.
func (ds *Datastore) convertSCDRows(ctx context.Context, dataset string, from, to time.Time, w *convertWalk) error {
	keys, err := ds.loadEntityKeys(ctx, dataset)
	if err != nil {
		return err
	}
	// The entities changing at the current instant, with their sets before it.
	var at time.Time
	before := map[string]*roaring.Bitmap{}
	touch := func(entityID string) {
		if _, ok := before[entityID]; !ok {
			before[entityID] = w.state(entityID)
		}
	}
	// apply ends the rows that end at or before upTo, oldest first, writing each instant's changes.
	apply := func(upTo time.Time) error {
		for w.ends.Len() > 0 && !w.ends[0].at.After(upTo) {
			if !w.ends[0].at.Equal(at) {
				if err := ds.writeConvertedChanges(ctx, dataset, at, before, w, keys); err != nil {
					return err
				}
				at = w.ends[0].at
			}
			end := heap.Pop(&w.ends).(scdRowEnd)
			touch(end.entityID)
			w.removeEnded(end.entityID, end.at)
		}
		return nil
	}

	err = ds.forEachConvertRow(ctx,
		`SELECT entity_id, host_bitmap, encoding_type, valid_from, valid_to FROM host_scd_data FORCE INDEX (idx_dataset_range)
		 WHERE dataset = ? AND valid_from >= ? AND valid_from < ? AND valid_to > valid_from
		 ORDER BY valid_from`,
		[]any{dataset, from, to}, func(r scdRow, hosts *roaring.Bitmap) error {
			start := r.ValidFrom.UTC()
			// Ends before this row's start are earlier instants; an end at the same instant joins its changes.
			if err := apply(start.Add(-1)); err != nil {
				return err
			}
			if !start.Equal(at) {
				if err := ds.writeConvertedChanges(ctx, dataset, at, before, w, keys); err != nil {
					return err
				}
				at = start
			}
			if err := apply(start); err != nil {
				return err
			}
			touch(r.EntityID)
			if end := r.ValidTo.UTC(); end.Before(to) {
				w.add(r.EntityID, hosts, end)
			} else {
				w.add(r.EntityID, hosts, time.Time{})
			}
			return nil
		})
	if err != nil {
		return err
	}
	if err := apply(to.Add(-1)); err != nil {
		return err
	}
	return ds.writeConvertedChanges(ctx, dataset, at, before, w, keys)
}

// writeConvertedChanges writes the changes at at of the entities in before, then empties before.
func (ds *Datastore) writeConvertedChanges(
	ctx context.Context, dataset string, at time.Time, before map[string]*roaring.Bitmap, w *convertWalk, keys map[string]uint32,
) error {
	if len(before) == 0 {
		return nil
	}
	type encoded struct {
		entityID       string
		added, removed []byte
	}
	var changes []encoded
	var missing []string
	for entityID, prev := range before {
		next := w.state(entityID)
		if prev == nil {
			prev = roaring.New()
		}
		if next == nil {
			next = roaring.New()
		}
		added, removed := roaring.AndNot(next, prev), roaring.AndNot(prev, next)
		if added.IsEmpty() && removed.IsEmpty() {
			continue
		}
		changes = append(changes, encoded{entityID: entityID, added: bitmapBytes(added), removed: bitmapBytes(removed)})
		if _, ok := keys[entityID]; !ok {
			missing = append(missing, entityID)
		}
	}
	clear(before)
	if len(changes) == 0 {
		return nil
	}
	if err := ds.ensureEntityKeys(ctx, dataset, missing, keys); err != nil {
		return err
	}
	slices.SortFunc(changes, func(a, b encoded) int { return cmp.Compare(keys[a.entityID], keys[b.entityID]) })

	var seq uint32
	var payload []byte
	var entities, size int
	flush := func() error {
		if entities == 0 {
			return nil
		}
		if _, err := ds.writer(ctx).ExecContext(ctx,
			`INSERT INTO host_scd_changes (dataset, changed_at, seq, changes, encoding_type) VALUES (?, ?, ?, ?, ?)`,
			dataset, at, seq, payload, changesEncodingV1); err != nil {
			return ctxerr.Wrap(ctx, err, "insert converted chart changes")
		}
		seq++
		payload, entities, size = payload[:0], 0, 0
		return nil
	}
	for _, c := range changes {
		s := encodedChangeSize(c.added, c.removed)
		if entities > 0 && (entities >= scdChangeBatchEntities || size+s > scdChangeRowBytes) {
			if err := flush(); err != nil {
				return err
			}
		}
		payload = appendChange(payload, keys[c.entityID], c.added, c.removed)
		entities++
		size += s
	}
	return flush()
}

// forEachConvertRow streams the host_scd_data rows of query and calls fn with each row and its decoded hosts.
func (ds *Datastore) forEachConvertRow(ctx context.Context, query string, args []any, fn func(scdRow, *roaring.Bitmap) error) error {
	rows, err := ds.reader(ctx).QueryxContext(ctx, query, args...) //nolint:sqlclosecheck // closed below
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read chart history to convert")
	}
	defer rows.Close()
	var r scdRow
	for rows.Next() {
		if err := rows.StructScan(&r); err != nil {
			return ctxerr.Wrap(ctx, err, "scan chart history row")
		}
		hosts, err := chart.DecodeBitmap(chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType})
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "decode bitmap for entity %q", r.EntityID)
		}
		if err := fn(r, hosts); err != nil {
			return err
		}
	}
	return ctxerr.Wrap(ctx, rows.Err(), "iterate chart history rows")
}

// convertWalk holds the host_scd_data rows live at the conversion's current instant.
type convertWalk struct {
	live map[string][]liveSCDRow
	// ends holds the end of every live row that ends within the step.
	ends scdRowEnds
}

type liveSCDRow struct {
	hosts *roaring.Bitmap
	// validTo is zero for a row that ends after the step.
	validTo time.Time
}

func (w *convertWalk) add(entityID string, hosts *roaring.Bitmap, validTo time.Time) {
	w.live[entityID] = append(w.live[entityID], liveSCDRow{hosts: hosts, validTo: validTo})
	if !validTo.IsZero() {
		heap.Push(&w.ends, scdRowEnd{at: validTo, entityID: entityID})
	}
}

func (w *convertWalk) removeEnded(entityID string, at time.Time) {
	rows := slices.DeleteFunc(w.live[entityID], func(r liveSCDRow) bool { return r.validTo.Equal(at) })
	if len(rows) == 0 {
		delete(w.live, entityID)
		return
	}
	w.live[entityID] = rows
}

// state returns the entity's host set: the union of its live rows, or nil when none is live. The result must not
// be modified.
func (w *convertWalk) state(entityID string) *roaring.Bitmap {
	rows := w.live[entityID]
	switch len(rows) {
	case 0:
		return nil
	case 1:
		return rows[0].hosts
	}
	sets := make([]*roaring.Bitmap, len(rows))
	for i, r := range rows {
		sets[i] = r.hosts
	}
	return roaring.FastOr(sets...)
}

type scdRowEnd struct {
	at       time.Time
	entityID string
}

// scdRowEnds is a min-heap of row ends by time.
type scdRowEnds []scdRowEnd

func (h scdRowEnds) Len() int           { return len(h) }
func (h scdRowEnds) Less(i, j int) bool { return h[i].at.Before(h[j].at) }
func (h scdRowEnds) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *scdRowEnds) Push(x any)        { *h = append(*h, x.(scdRowEnd)) }
func (h *scdRowEnds) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
