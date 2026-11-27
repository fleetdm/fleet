package mysql

import (
	"cmp"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/RoaringBitmap/roaring"
	"github.com/fleetdm/fleet/v4/server/chart"
	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// changesEncodingV1 is the format of host_scd_changes.changes: for each entity, its uvarint key, then the
// uvarint length and roaring bytes of the hosts added, then the same for the hosts removed.
const changesEncodingV1 uint8 = 1

// Batch limits for the change storage. vars (not consts) so tests can shrink them.
var (
	// scdChangeRowBytes caps the encoded changes in one host_scd_changes row. Reads hold one row at a time, so this
	// bounds their memory. One entity's change is never split, so a row holding a single large change, or a row the
	// fleet scrub rewrote, can exceed it.
	scdChangeRowBytes = 1 << 20
	// scdCurrentWriteBytes caps the host sets written to host_scd_current in one transaction.
	scdCurrentWriteBytes = 4 << 20
	// scdChangeBatchEntities caps how many entities one statement or change row covers, which keeps statements under
	// MySQL's placeholder limit when host sets are small.
	scdChangeBatchEntities = 1000
	// scdChangeDeleteBatch caps how many host_scd_changes rows one DELETE removes. Rows hold up to scdChangeRowBytes.
	scdChangeDeleteBatch = 50
	// scdCurrentDeleteBatch caps how many host_scd_current rows one DELETE removes when a dataset's sets are cleared.
	scdCurrentDeleteBatch = 5000
	// scdEntityKeyBatch caps how many entities one host_scd_entities insert covers.
	scdEntityKeyBatch = 500
	// scdChangeScrubPage is how many host_scd_changes rows the fleet scrub reads at a time.
	scdChangeScrubPage = 16
)

// scdChangeLockPollSeconds is how long one GET_LOCK call waits when a writer waits for the change storage lock, so
// the wait notices a cancelled context.
const scdChangeLockPollSeconds = 5

// scdChangeLockName is the SQL expression for a dataset's change storage lock name. Named locks are global to the
// MySQL server, so the name includes the database; GET_LOCK rejects names longer than 64 characters.
const scdChangeLockName = `LEFT(CONCAT('chart_changes:', ?, ':', IFNULL(DATABASE(), '')), 64)`

// withSCDChangeLock runs fn while holding the dataset's change storage lock, which serializes the writers of the
// change storage: collection, conversion and the scrubs. Each writes from what it read, so one running in the
// middle of another would write from a stale read. When wait is false and another session holds the lock, it
// returns api.ErrChangeStorageBusy without running fn.
//
// The lock is a MySQL named lock held by the session of a connection kept for the whole call, so it is released
// if Fleet stops.
func (ds *Datastore) withSCDChangeLock(ctx context.Context, dataset string, wait bool, fn func() error) error {
	conn, err := ds.primary.Connx(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get connection for chart change lock")
	}
	defer func() { _ = conn.Close() }()

	timeout := 0
	if wait {
		timeout = scdChangeLockPollSeconds
	}
	for {
		var got sql.NullInt64
		if err := conn.GetContext(ctx, &got, `SELECT GET_LOCK(`+scdChangeLockName+`, ?)`, dataset, timeout); err != nil {
			return ctxerr.Wrap(ctx, err, "acquire chart change lock")
		}
		if !got.Valid {
			return ctxerr.New(ctx, "acquire chart change lock: GET_LOCK returned NULL")
		}
		if got.Int64 == 1 {
			break
		}
		if !wait {
			return api.ErrChangeStorageBusy
		}
	}
	defer func() {
		// A cancelled ctx must not keep the lock: release on a context that outlives it, and if that fails, close
		// the session instead of returning it to the pool.
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(releaseCtx, `DO RELEASE_LOCK(`+scdChangeLockName+`)`, dataset); err != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
	}()
	return fn()
}

// entityChange is one entity's change within a host_scd_changes row.
type entityChange struct {
	key            uint32
	added, removed *roaring.Bitmap
}

func bitmapBytes(rb *roaring.Bitmap) []byte {
	if rb == nil || rb.IsEmpty() {
		return nil
	}
	return chart.BitmapToBlob(rb).Bytes
}

func encodedChangeSize(added, removed []byte) int {
	return 3*binary.MaxVarintLen32 + len(added) + len(removed)
}

func appendChange(buf []byte, key uint32, added, removed []byte) []byte {
	buf = binary.AppendUvarint(buf, uint64(key))
	buf = binary.AppendUvarint(buf, uint64(len(added)))
	buf = append(buf, added...)
	buf = binary.AppendUvarint(buf, uint64(len(removed)))
	return append(buf, removed...)
}

// walkChanges calls fn with each entity's key and the encoded hosts added and removed, which alias b.
func walkChanges(b []byte, fn func(key uint32, added, removed []byte) error) error {
	next := func() (uint64, error) {
		v, n := binary.Uvarint(b)
		if n <= 0 {
			return 0, errors.New("truncated chart changes")
		}
		b = b[n:]
		return v, nil
	}
	bitmap := func() ([]byte, error) {
		n, err := next()
		if err != nil {
			return nil, err
		}
		if n > uint64(len(b)) {
			return nil, errors.New("truncated chart changes")
		}
		out := b[:n]
		b = b[n:]
		return out, nil
	}
	for len(b) > 0 {
		key, err := next()
		if err != nil {
			return err
		}
		if key > uint64(^uint32(0)) {
			return errors.New("chart change entity key out of range")
		}
		added, err := bitmap()
		if err != nil {
			return err
		}
		removed, err := bitmap()
		if err != nil {
			return err
		}
		if err := fn(uint32(key), added, removed); err != nil {
			return err
		}
	}
	return nil
}

func decodeChanges(b []byte) ([]entityChange, error) {
	var out []entityChange
	err := walkChanges(b, func(key uint32, addedBytes, removedBytes []byte) error {
		added, err := chart.DecodeBitmap(chart.Blob{Bytes: addedBytes, Encoding: chart.EncodingRoaring})
		if err != nil {
			return err
		}
		removed, err := chart.DecodeBitmap(chart.Blob{Bytes: removedBytes, Encoding: chart.EncodingRoaring})
		if err != nil {
			return err
		}
		out = append(out, entityChange{key: key, added: added, removed: removed})
		return nil
	})
	return out, err
}

func encodeChanges(chs []entityChange) []byte {
	var buf []byte
	for _, c := range chs {
		buf = appendChange(buf, c.key, bitmapBytes(c.added), bitmapBytes(c.removed))
	}
	return buf
}

// RecordSCDChanges records an hour's host set per entity in the change storage: the entity's current set in
// host_scd_current, and what changed since the previous call in host_scd_changes at that hour. An entity missing
// from entityBitmaps, or with an empty set, has no hosts.
//
// The first call for a dataset starts from the host_scd_data rows live just before hour, so its changes are true
// differences from the stored history, and records that hour in host_scd_change_state.
//
// Every write appends: a second call for the same hour adds rows that hold the changes since the first. Each
// transaction writes one host_scd_changes row together with the current sets of the entities it covers, so a
// failure part way leaves the two consistent and the next call records what is left.
//
// While another writer of the dataset holds the change storage lock, it returns api.ErrChangeStorageBusy and
// writes nothing; the next call records the changes since the last one written.
func (ds *Datastore) RecordSCDChanges(ctx context.Context, dataset string, hour time.Time, entityBitmaps map[string]*roaring.Bitmap) error {
	return ds.withSCDChangeLock(ctx, dataset, false, func() error {
		return ds.recordSCDChanges(ctx, dataset, hour, entityBitmaps)
	})
}

func (ds *Datastore) recordSCDChanges(ctx context.Context, dataset string, hour time.Time, entityBitmaps map[string]*roaring.Bitmap) error {
	hour = hour.UTC()
	// Every read here decides what to write next, so a stale replica would write wrong differences.
	ctx = ctxdb.RequirePrimary(ctx, true)

	keys, err := ds.loadEntityKeys(ctx, dataset)
	if err != nil {
		return err
	}
	var started bool
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &started,
		`SELECT COUNT(*) > 0 FROM host_scd_change_state WHERE dataset = ?`, dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "read chart change state")
	}
	var current map[uint32]*roaring.Bitmap
	if started {
		current, err = ds.loadCurrentSets(ctx, dataset)
	} else {
		current, err = ds.startChanges(ctx, dataset, hour, keys)
	}
	if err != nil {
		return err
	}

	var missing []string
	for entityID, rb := range entityBitmaps {
		if _, ok := keys[entityID]; !ok && rb != nil && !rb.IsEmpty() {
			missing = append(missing, entityID)
		}
	}
	if err := ds.ensureEntityKeys(ctx, dataset, missing, keys); err != nil {
		return err
	}
	next := make(map[uint32]*roaring.Bitmap, len(entityBitmaps))
	for entityID, rb := range entityBitmaps {
		if rb != nil && !rb.IsEmpty() {
			next[keys[entityID]] = rb
		}
	}

	// The current sets are serialized one batch at a time, when written, so only the changes are held serialized.
	type pending struct {
		key            uint32
		added, removed []byte
		next           *roaring.Bitmap // nil: the entity has no hosts now
	}
	var changes []pending
	diff := func(key uint32) {
		cur, nxt := current[key], next[key]
		if cur == nil {
			cur = roaring.New()
		}
		after := nxt
		if after == nil {
			after = roaring.New()
		}
		added, removed := roaring.AndNot(after, cur), roaring.AndNot(cur, after)
		if added.IsEmpty() && removed.IsEmpty() {
			return
		}
		changes = append(changes, pending{key: key, added: bitmapBytes(added), removed: bitmapBytes(removed), next: nxt})
	}
	for key := range current {
		diff(key)
	}
	for key := range next {
		if _, ok := current[key]; !ok {
			diff(key)
		}
	}
	if len(changes) == 0 {
		return nil
	}
	slices.SortFunc(changes, func(a, b pending) int { return cmp.Compare(a.key, b.key) })
	if ds.testHookSCDChangesBeforeWrite != nil {
		ds.testHookSCDChangesBeforeWrite()
	}

	var seq uint32
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &seq,
		`SELECT COALESCE(MAX(seq) + 1, 0) FROM host_scd_changes WHERE dataset = ? AND changed_at = ?`, dataset, hour); err != nil {
		return ctxerr.Wrap(ctx, err, "read next chart change seq")
	}

	write := func(batch []pending) error {
		var payload []byte
		sets := make([]currentSet, len(batch))
		for i, c := range batch {
			payload = appendChange(payload, c.key, c.added, c.removed)
			sets[i] = currentSet{key: c.key, hosts: c.next}
		}
		err := platform_mysql.WithRetryTxx(ctx, ds.writer(ctx), func(tx sqlx.ExtContext) error {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO host_scd_changes (dataset, changed_at, seq, changes, encoding_type) VALUES (?, ?, ?, ?, ?)`,
				dataset, hour, seq, payload, changesEncodingV1); err != nil {
				return ctxerr.Wrap(ctx, err, "insert chart changes")
			}
			return writeCurrentSets(ctx, tx, dataset, sets)
		}, ds.logger)
		if err != nil {
			return err
		}
		seq++
		return nil
	}

	var batch []pending
	var payloadBytes, currentBytes int
	for _, c := range changes {
		size := encodedChangeSize(c.added, c.removed)
		var nextSize int
		if c.next != nil {
			nextSize = int(c.next.GetSerializedSizeInBytes()) //nolint:gosec // G115: a host set's size fits in an int
		}
		if len(batch) > 0 && (len(batch) >= scdChangeBatchEntities || payloadBytes+size > scdChangeRowBytes ||
			currentBytes+nextSize > scdCurrentWriteBytes) {
			if err := write(batch); err != nil {
				return err
			}
			batch, payloadBytes, currentBytes = batch[:0], 0, 0
		}
		batch = append(batch, c)
		payloadBytes += size
		currentBytes += nextSize
	}
	return write(batch)
}

// currentSet is one entity's row in host_scd_current. A nil or empty set means the entity has no hosts, so its
// row is deleted.
type currentSet struct {
	key   uint32
	hosts *roaring.Bitmap
}

// writeCurrentSets upserts the non-empty sets and deletes the rows of the empty ones, in statements capped by
// scdChangeBatchEntities and scdCurrentWriteBytes.
func writeCurrentSets(ctx context.Context, exec sqlx.ExecerContext, dataset string, sets []currentSet) error {
	var placeholders []string
	var args []any
	var size int
	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		// Concatenating hardcoded "(?, ?, ?, ?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_current (dataset, entity_key, host_bitmap, encoding_type) VALUES ` + //nolint:gosec // G202
			strings.Join(placeholders, ", ") +
			` ON DUPLICATE KEY UPDATE host_bitmap = VALUES(host_bitmap), encoding_type = VALUES(encoding_type)`
		if _, err := exec.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert chart current sets")
		}
		placeholders, args, size = placeholders[:0], args[:0], 0
		return nil
	}
	var deleteKeys []uint32
	for _, s := range sets {
		if s.hosts == nil || s.hosts.IsEmpty() {
			deleteKeys = append(deleteKeys, s.key)
			continue
		}
		b := bitmapBytes(s.hosts)
		if len(placeholders) >= scdChangeBatchEntities || size+len(b) > scdCurrentWriteBytes {
			if err := flush(); err != nil {
				return err
			}
		}
		placeholders = append(placeholders, "(?, ?, ?, ?)")
		args = append(args, dataset, s.key, b, chart.EncodingRoaring)
		size += len(b)
	}
	if err := flush(); err != nil {
		return err
	}
	for i := 0; i < len(deleteKeys); i += scdChangeBatchEntities {
		stmt, dargs, err := sqlx.In(`DELETE FROM host_scd_current WHERE dataset = ? AND entity_key IN (?)`,
			dataset, deleteKeys[i:min(i+scdChangeBatchEntities, len(deleteKeys))])
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand chart current delete args")
		}
		if _, err := exec.ExecContext(ctx, stmt, dargs...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete chart current sets")
		}
	}
	return nil
}

// startChanges sets up the change storage for a dataset: it fills host_scd_current with the host_scd_data state
// just before hour, then records hour as the start of the changes. It is safe to repeat after a failure: it
// starts by clearing what an earlier attempt, or a global scrub interrupted after deleting the state row, left.
func (ds *Datastore) startChanges(ctx context.Context, dataset string, hour time.Time, keys map[string]uint32) (map[uint32]*roaring.Bitmap, error) {
	if err := ds.deleteInBatches(ctx, `DELETE FROM host_scd_changes WHERE dataset = ? LIMIT ?`, scdChangeDeleteBatch, dataset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "clear chart changes")
	}
	if err := ds.deleteInBatches(ctx, `DELETE FROM host_scd_current WHERE dataset = ? LIMIT ?`, scdCurrentDeleteBatch, dataset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "clear chart current sets")
	}

	// The rows a snapshot read counts at hour: valid_from < hour <= valid_to.
	seeded := map[string]*roaring.Bitmap{}
	rows, err := ds.reader(ctx).QueryxContext(ctx, //nolint:sqlclosecheck // closed below
		`SELECT entity_id, host_bitmap, encoding_type FROM host_scd_data WHERE dataset = ? AND valid_from < ? AND valid_to >= ?`,
		dataset, hour, hour)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read chart history for change storage")
	}
	defer rows.Close()
	for rows.Next() {
		var r struct {
			EntityID     string `db:"entity_id"`
			HostBitmap   []byte `db:"host_bitmap"`
			EncodingType uint8  `db:"encoding_type"`
		}
		if err := rows.StructScan(&r); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scan chart history row")
		}
		rb, err := chart.DecodeBitmap(chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType})
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decode bitmap for entity %q", r.EntityID)
		}
		if prev, ok := seeded[r.EntityID]; ok {
			prev.Or(rb)
		} else {
			seeded[r.EntityID] = rb
		}
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterate chart history rows")
	}
	rows.Close()

	var missing []string
	for entityID, rb := range seeded {
		if _, ok := keys[entityID]; !ok && !rb.IsEmpty() {
			missing = append(missing, entityID)
		}
	}
	if err := ds.ensureEntityKeys(ctx, dataset, missing, keys); err != nil {
		return nil, err
	}

	current := make(map[uint32]*roaring.Bitmap, len(seeded))
	sets := make([]currentSet, 0, len(seeded))
	for entityID, rb := range seeded {
		if rb.IsEmpty() {
			continue
		}
		key := keys[entityID]
		current[key] = rb
		sets = append(sets, currentSet{key: key, hosts: rb})
	}
	if err := writeCurrentSets(ctx, ds.writer(ctx), dataset, sets); err != nil {
		return nil, err
	}

	if _, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT INTO host_scd_change_state (dataset, changes_from, covered_from) VALUES (?, ?, ?)`,
		dataset, hour, hour); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert chart change state")
	}
	return current, nil
}

func (ds *Datastore) loadEntityKeys(ctx context.Context, dataset string) (map[string]uint32, error) {
	var rows []struct {
		ID       uint32 `db:"id"`
		EntityID string `db:"entity_id"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT id, entity_id FROM host_scd_entities WHERE dataset = ?`, dataset); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load chart entity keys")
	}
	keys := make(map[string]uint32, len(rows))
	for _, r := range rows {
		keys[r.EntityID] = r.ID
	}
	return keys, nil
}

// ensureEntityKeys assigns keys to entityIDs and adds them to keys.
func (ds *Datastore) ensureEntityKeys(ctx context.Context, dataset string, entityIDs []string, keys map[string]uint32) error {
	for i := 0; i < len(entityIDs); i += scdEntityKeyBatch {
		chunk := entityIDs[i:min(i+scdEntityKeyBatch, len(entityIDs))]
		placeholders := make([]string, len(chunk))
		args := make([]any, 0, 2*len(chunk))
		for j, id := range chunk {
			placeholders[j] = "(?, ?)"
			args = append(args, dataset, id)
		}
		// Concatenating hardcoded "(?, ?)" placeholder strings, not user input.
		stmt := `INSERT INTO host_scd_entities (dataset, entity_id) VALUES ` + strings.Join(placeholders, ", ") + //nolint:gosec // G202
			` ON DUPLICATE KEY UPDATE id = id`
		if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "insert chart entity keys")
		}
		query, qargs, err := sqlx.In(`SELECT id, entity_id FROM host_scd_entities WHERE dataset = ? AND entity_id IN (?)`, dataset, chunk)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand chart entity key args")
		}
		var rows []struct {
			ID       uint32 `db:"id"`
			EntityID string `db:"entity_id"`
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, ds.rebind(query), qargs...); err != nil {
			return ctxerr.Wrap(ctx, err, "read chart entity keys")
		}
		for _, r := range rows {
			keys[r.EntityID] = r.ID
		}
		// The unique key compares entity IDs without case, so another spelling of a stored ID finds that ID's row
		// instead of getting a key of its own.
		for _, id := range chunk {
			if _, ok := keys[id]; !ok {
				return ctxerr.Errorf(ctx, "chart entity %q matches a stored entity spelled differently", id)
			}
		}
	}
	return nil
}

func (ds *Datastore) loadCurrentSets(ctx context.Context, dataset string) (map[uint32]*roaring.Bitmap, error) {
	rows, err := ds.reader(ctx).QueryxContext(ctx, //nolint:sqlclosecheck // closed below
		`SELECT entity_key, host_bitmap, encoding_type FROM host_scd_current WHERE dataset = ?`, dataset)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read chart current sets")
	}
	defer rows.Close()
	current := map[uint32]*roaring.Bitmap{}
	for rows.Next() {
		var r struct {
			EntityKey    uint32 `db:"entity_key"`
			HostBitmap   []byte `db:"host_bitmap"`
			EncodingType uint8  `db:"encoding_type"`
		}
		if err := rows.StructScan(&r); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "scan chart current set")
		}
		rb, err := chart.DecodeBitmap(chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType})
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "decode current set for entity key %d", r.EntityKey)
		}
		current[r.EntityKey] = rb
	}
	if err := rows.Err(); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "iterate chart current sets")
	}
	return current, nil
}

// deleteInBatches runs stmt, which must end in "LIMIT ?", with args and limit until it deletes fewer than limit
// rows.
func (ds *Datastore) deleteInBatches(ctx context.Context, stmt string, limit int, args ...any) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		res, err := ds.writer(ctx).ExecContext(ctx, stmt, append(args, limit)...)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n < int64(limit) {
			return nil
		}
	}
}

// deleteSCDChangesForDataset removes a dataset's change storage, waiting for the change storage lock. The state row
// goes first: a collection that runs after a failure part way then starts over from host_scd_data instead of
// diffing against what is left.
func (ds *Datastore) deleteSCDChangesForDataset(ctx context.Context, dataset string, batchSize int) error {
	return ds.withSCDChangeLock(ctx, dataset, true, func() error {
		return ds.deleteSCDChangesForDatasetLocked(ctx, dataset, batchSize)
	})
}

func (ds *Datastore) deleteSCDChangesForDatasetLocked(ctx context.Context, dataset string, batchSize int) error {
	if _, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_scd_change_state WHERE dataset = ?`, dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "delete chart change state for dataset")
	}
	if err := ds.deleteInBatches(ctx, `DELETE FROM host_scd_changes WHERE dataset = ? LIMIT ?`, scdChangeDeleteBatch, dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "delete chart changes for dataset")
	}
	if err := ds.deleteInBatches(ctx, `DELETE FROM host_scd_current WHERE dataset = ? LIMIT ?`, batchSize, dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "delete chart current sets for dataset")
	}
	if err := ds.deleteInBatches(ctx, `DELETE FROM host_scd_entities WHERE dataset = ? LIMIT ?`, batchSize, dataset); err != nil {
		return ctxerr.Wrap(ctx, err, "delete chart entity keys for dataset")
	}
	return nil
}

// cleanupSCDChanges deletes changes recorded before cutoff. It deletes per dataset so each statement is a
// primary-key range that stops at the cutoff, and does not lock the newest rows the collector writes.
func (ds *Datastore) cleanupSCDChanges(ctx context.Context, cutoff time.Time) error {
	var datasets []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &datasets, `SELECT DISTINCT dataset FROM host_scd_changes`); err != nil {
		return ctxerr.Wrap(ctx, err, "list chart change datasets")
	}
	for _, dataset := range datasets {
		for {
			if err := ctx.Err(); err != nil {
				return ctxerr.Wrap(ctx, err, "cleanup chart changes")
			}
			res, err := ds.writer(ctx).ExecContext(ctx,
				`DELETE FROM host_scd_changes WHERE dataset = ? AND changed_at < ? ORDER BY changed_at, seq LIMIT ?`,
				dataset, cutoff, scdChangeDeleteBatch)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "cleanup chart changes")
			}
			n, err := res.RowsAffected()
			if err != nil {
				return ctxerr.Wrap(ctx, err, "cleanup chart changes rows affected")
			}
			if n < int64(scdChangeDeleteBatch) {
				break
			}
		}
	}
	return nil
}

// scrubSCDChanges clears the hosts in mask from a dataset's current sets and changes, waiting for the change
// storage lock. Sets left empty are deleted, as are changes left with no hosts and rows left with no changes.
func (ds *Datastore) scrubSCDChanges(ctx context.Context, dataset string, mask *roaring.Bitmap) error {
	return ds.withSCDChangeLock(ctx, dataset, true, func() error {
		return ds.scrubSCDChangesLocked(ctx, dataset, mask)
	})
}

func (ds *Datastore) scrubSCDChangesLocked(ctx context.Context, dataset string, mask *roaring.Bitmap) error {
	current, err := ds.loadCurrentSets(ctx, dataset)
	if err != nil {
		return err
	}
	var scrubbed []currentSet
	for key, rb := range current {
		if rb.Intersects(mask) {
			rb.AndNot(mask)
			scrubbed = append(scrubbed, currentSet{key: key, hosts: rb})
		}
	}
	if err := writeCurrentSets(ctx, ds.writer(ctx), dataset, scrubbed); err != nil {
		return ctxerr.Wrap(ctx, err, "scrub chart current sets")
	}

	var lastAt time.Time
	var lastSeq int64 = -1
	for {
		if err := ctx.Err(); err != nil {
			return ctxerr.Wrap(ctx, err, "scrub chart changes")
		}
		var rows []struct {
			ChangedAt time.Time `db:"changed_at"`
			Seq       uint32    `db:"seq"`
			Changes   []byte    `db:"changes"`
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
			`SELECT changed_at, seq, changes FROM host_scd_changes
			 WHERE dataset = ? AND (changed_at > ? OR (changed_at = ? AND seq > ?))
			 ORDER BY changed_at, seq LIMIT ?`,
			dataset, lastAt, lastAt, lastSeq, scdChangeScrubPage); err != nil {
			return ctxerr.Wrap(ctx, err, "read chart changes to scrub")
		}
		if len(rows) == 0 {
			return nil
		}
		for _, r := range rows {
			lastAt, lastSeq = r.ChangedAt, int64(r.Seq)
			chs, err := decodeChanges(r.Changes)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "decode chart changes at %s", r.ChangedAt)
			}
			kept := chs[:0]
			touched := false
			for _, c := range chs {
				if c.added.Intersects(mask) || c.removed.Intersects(mask) {
					touched = true
					c.added.AndNot(mask)
					c.removed.AndNot(mask)
				}
				if !c.added.IsEmpty() || !c.removed.IsEmpty() {
					kept = append(kept, c)
				}
			}
			if !touched {
				continue
			}
			if len(kept) == 0 {
				_, err = ds.writer(ctx).ExecContext(ctx,
					`DELETE FROM host_scd_changes WHERE dataset = ? AND changed_at = ? AND seq = ?`, dataset, r.ChangedAt, r.Seq)
			} else {
				_, err = ds.writer(ctx).ExecContext(ctx,
					`UPDATE host_scd_changes SET changes = ?, encoding_type = ? WHERE dataset = ? AND changed_at = ? AND seq = ?`,
					encodeChanges(kept), changesEncodingV1, dataset, r.ChangedAt, r.Seq)
			}
			if err != nil {
				return ctxerr.Wrap(ctx, err, "write scrubbed chart changes")
			}
		}
	}
}
