package service

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
	"time"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryResultRowsUnchanged(t *testing.T) {
	data := func(d string) *json.RawMessage {
		if d == "" {
			return nil
		}
		return new(json.RawMessage(d))
	}
	incoming := func(ds ...string) []*fleet.ScheduledQueryResultRow {
		var rows []*fleet.ScheduledQueryResultRow
		for _, d := range ds {
			rows = append(rows, &fleet.ScheduledQueryResultRow{Data: data(d)})
		}
		return rows
	}
	stored := func(ds ...string) []*fleet.StoredQueryResultRow {
		var rows []*fleet.StoredQueryResultRow
		for i, d := range ds {
			rows = append(rows, &fleet.StoredQueryResultRow{ID: uint(i + 1), Data: data(d)}) //nolint:gosec // dismiss G115
		}
		return rows
	}

	cases := []struct {
		name   string
		rows   []*fleet.ScheduledQueryResultRow
		stored []*fleet.StoredQueryResultRow
		want   bool
	}{
		{"nothing stored", incoming(`{"a":"1"}`), nil, false},
		{"same data", incoming(`{"a":"1","b":"2"}`), stored(`{"a": "1", "b": "2"}`), true},
		{"keys reordered and strings re-escaped by MySQL", incoming(`{"b":"caf\u00e9","a":"1"}`), stored(`{"a": "1", "b": "café"}`), true},
		{"rows in a different order", incoming(`{"a":"1"}`, `{"a":"2"}`), stored(`{"a": "2"}`, `{"a": "1"}`), true},
		{"large numbers keep precision", incoming(`{"a":9007199254740993}`), stored(`{"a": 9007199254740992}`), false},
		{"null data on both sides", incoming(""), stored(""), true},
		{"null data replaced by data", incoming(`{"a":"1"}`), stored(""), false},
		{"value changed", incoming(`{"a":"2"}`), stored(`{"a": "1"}`), false},
		{"row added", incoming(`{"a":"1"}`, `{"a":"2"}`), stored(`{"a": "1"}`), false},
		{"duplicate rows count", incoming(`{"a":"1"}`, `{"a":"1"}`), stored(`{"a": "1"}`, `{"a": "2"}`), false},
		{"invalid incoming JSON", incoming(`{"a":`), stored(`{"a": "1"}`), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, queryResultRowsUnchanged(c.rows, c.stored))
		})
	}
}

func TestQueryResultRowsLastFetchedStale(t *testing.T) {
	now := time.Now()
	stored := func(ages ...time.Duration) []*fleet.StoredQueryResultRow {
		rows := make([]*fleet.StoredQueryResultRow, 0, len(ages))
		for _, age := range ages {
			rows = append(rows, &fleet.StoredQueryResultRow{LastFetched: now.Add(-age)})
		}
		return rows
	}
	assert.False(t, queryResultRowsLastFetchedStale(stored(time.Minute, queryResultsLastFetchedRefreshAge-time.Minute), now))
	assert.True(t, queryResultRowsLastFetchedStale(stored(queryResultsLastFetchedRefreshAge), now))
	assert.True(t, queryResultRowsLastFetchedStale(stored(time.Minute, queryResultsLastFetchedRefreshAge+time.Minute), now))
}

func TestSaveResultLogsToQueryReportsSkipsUnchanged(t *testing.T) {
	ds := new(mock.Store)
	lq := makeLiveQueryStore(t, 0)
	svc, ctx := newTestService(t, ds, nil, lq)
	serv := ((svc.(validationMiddleware)).Service).(*Service)
	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 42})

	results := []*fleet.ScheduledQueryResult{
		{
			QueryName: "pack/Global/Same",
			Snapshot:  []*json.RawMessage{new(json.RawMessage(`{"hour":"20"}`))},
			UnixTime:  1484078931,
		},
		{
			QueryName: "pack/Global/Changed",
			Snapshot:  []*json.RawMessage{new(json.RawMessage(`{"hour":"21"}`))},
			UnixTime:  1484078931,
		},
	}
	queries := map[string]*fleet.Query{
		"pack/Global/Same":    {ID: 1, Logging: fleet.LoggingSnapshot},
		"pack/Global/Changed": {ID: 2, Logging: fleet.LoggingSnapshot},
	}

	storedLastFetched := time.Now().Add(-queryResultsLastFetchedRefreshAge - time.Hour)
	stored := func(queryID, rowID uint) []*fleet.StoredQueryResultRow {
		return []*fleet.StoredQueryResultRow{{
			ID:          rowID,
			QueryID:     queryID,
			HostID:      42,
			Data:        new(json.RawMessage(`{"hour": "20"}`)),
			LastFetched: storedLastFetched,
		}}
	}
	ds.QueryResultRowsForHostByQueryFunc = func(ctx context.Context, hostID uint, queryIDs []uint) (map[uint][]*fleet.StoredQueryResultRow, error) {
		require.Equal(t, uint(42), hostID)
		require.ElementsMatch(t, []uint{1, 2}, queryIDs)
		return map[uint][]*fleet.StoredQueryResultRow{1: stored(1, 100), 2: stored(2, 200)}, nil
	}
	var written []uint
	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows, currentCount int) (fleet.QueryReportWriteResult, error) {
		written = append(written, rows[0].QueryID)
		return fleet.QueryReportWriteResult{}, nil
	}
	var recorded []uint
	recordErr := error(nil)
	lq.RecordQueryResultsLastFetchedOverride = func(rowIDs []uint, fetchedAt time.Time) error {
		recorded = append(recorded, rowIDs...)
		return recordErr
	}
	reset := func() {
		written, recorded, recordErr = nil, nil, nil
		storedLastFetched = time.Now().Add(-queryResultsLastFetchedRefreshAge - time.Hour)
	}

	t.Run("unchanged rows are recorded instead of written", func(t *testing.T) {
		reset()
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.Equal(t, []uint{2}, written)
		require.Equal(t, []uint{100}, recorded)
	})

	t.Run("unchanged rows with a recent last_fetched are neither recorded nor written", func(t *testing.T) {
		reset()
		storedLastFetched = time.Now().Add(-queryResultsLastFetchedRefreshAge / 2)
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.Equal(t, []uint{2}, written)
		require.Empty(t, recorded)
	})

	t.Run("rows are written if they can't be recorded", func(t *testing.T) {
		reset()
		recordErr = errors.New("redis down")
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.ElementsMatch(t, []uint{1, 2}, written)
	})

	t.Run("rows aren't written while too many are pending", func(t *testing.T) {
		reset()
		recordErr = fleet.ErrQueryResultsLastFetchedFull
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.Equal(t, []uint{2}, written)
	})

	t.Run("stored rows can't be read", func(t *testing.T) {
		reset()
		ds.QueryResultRowsForHostByQueryFunc = func(ctx context.Context, hostID uint, queryIDs []uint) (map[uint][]*fleet.StoredQueryResultRow, error) {
			return nil, context.DeadlineExceeded
		}
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.ElementsMatch(t, []uint{1, 2}, written)
		require.Empty(t, recorded)
	})

	t.Run("without a live query store stored rows aren't read", func(t *testing.T) {
		reset()
		ds.QueryResultRowsForHostByQueryFuncInvoked = false
		serv.liveQueryStore = nil
		t.Cleanup(func() { serv.liveQueryStore = lq })
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		require.ElementsMatch(t, []uint{1, 2}, written)
		require.False(t, ds.QueryResultRowsForHostByQueryFuncInvoked)
	})
}

func TestUpdateQueryResultsLastFetched(t *testing.T) {
	ctx := t.Context()
	ds := new(mock.Store)
	lq := makeLiveQueryStore(t, 0)

	minute1 := time.Unix(1_700_000_000, 0).UTC().Truncate(time.Minute)
	minute2 := minute1.Add(time.Minute)
	fetched := map[uint]time.Time{5: minute1.Add(5 * time.Second), 3: minute1.Add(40 * time.Second), 1: minute1, 9: minute2.Add(59 * time.Second)}
	for id := uint(10); id < 10+queryResultsLastFetchedBatchSize; id++ {
		fetched[id] = minute2.Add(time.Duration(id%60) * time.Second)
	}
	lq.LoadQueryResultsLastFetchedOverride = func() (map[uint]time.Time, error) { return fetched, nil }
	var cleared bool
	lq.ClearProcessedQueryResultsLastFetchedOverride = func() error {
		cleared = true
		return nil
	}

	type call struct {
		ids []uint
		ts  time.Time
	}
	var calls []call
	updateErr := error(nil)
	ds.UpdateQueryResultsLastFetchedFunc = func(ctx context.Context, ids []uint, lastFetched time.Time) error {
		calls = append(calls, call{ids: slices.Clone(ids), ts: lastFetched})
		return updateErr
	}

	require.NoError(t, UpdateQueryResultsLastFetched(ctx, ds, lq))
	require.True(t, cleared)
	// Grouped by fetch minute in order, IDs sorted, and split into batches.
	require.Len(t, calls, 3)
	require.Equal(t, call{ids: []uint{1, 3, 5}, ts: minute1}, calls[0])
	require.Equal(t, minute2, calls[1].ts)
	require.Len(t, calls[1].ids, queryResultsLastFetchedBatchSize)
	require.Equal(t, uint(9), calls[1].ids[0])
	require.Equal(t, call{ids: []uint{10 + queryResultsLastFetchedBatchSize - 1}, ts: minute2}, calls[2])

	// On failure the processing set is kept for the next run to retry.
	calls, cleared = nil, false
	updateErr = errors.New("db down")
	require.Error(t, UpdateQueryResultsLastFetched(ctx, ds, lq))
	require.False(t, cleared)
}

func TestSaveResultLogsToQueryReportsSkipsWhenBusy(t *testing.T) {
	ds := new(mock.Store)
	lq := makeLiveQueryStore(t, 0)
	svc, ctx := newTestService(t, ds, nil, lq)
	serv := ((svc.(validationMiddleware)).Service).(*Service)
	ctx = hostctx.NewContext(ctx, &fleet.Host{ID: 42})
	serv.queryReportReadSem = make(chan struct{}, 1)
	serv.queryReportWriteLimit = 1
	writeSlotsHeld := 0
	lq.AcquireQueryReportWriteSlotOverride = func(token string, limit int, lease time.Duration) (bool, error) {
		if writeSlotsHeld >= limit {
			return false, nil
		}
		writeSlotsHeld++
		return true, nil
	}
	lq.ReleaseQueryReportWriteSlotOverride = func(token string) error {
		writeSlotsHeld--
		return nil
	}

	results := []*fleet.ScheduledQueryResult{
		{
			QueryName: "pack/Global/Same",
			Snapshot:  []*json.RawMessage{new(json.RawMessage(`{"hour":"20"}`))},
			UnixTime:  1484078931,
		},
		{
			QueryName: "pack/Global/Changed",
			Snapshot:  []*json.RawMessage{new(json.RawMessage(`{"hour":"21"}`))},
			UnixTime:  1484078931,
		},
	}
	queries := map[string]*fleet.Query{
		"pack/Global/Same":    {ID: 1, Logging: fleet.LoggingSnapshot},
		"pack/Global/Changed": {ID: 2, Logging: fleet.LoggingSnapshot},
	}
	ds.QueryResultRowsForHostByQueryFunc = func(ctx context.Context, hostID uint, queryIDs []uint) (map[uint][]*fleet.StoredQueryResultRow, error) {
		return map[uint][]*fleet.StoredQueryResultRow{
			1: {{ID: 100, QueryID: 1, HostID: 42, Data: new(json.RawMessage(`{"hour": "20"}`))}},
		}, nil
	}
	var written []uint
	ds.OverwriteQueryResultRowsFunc = func(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows, currentCount int) (fleet.QueryReportWriteResult, error) {
		written = append(written, rows[0].QueryID)
		return fleet.QueryReportWriteResult{}, nil
	}
	var recorded []uint
	lq.RecordQueryResultsLastFetchedOverride = func(rowIDs []uint, fetchedAt time.Time) error {
		recorded = append(recorded, rowIDs...)
		return nil
	}
	var countsRead bool
	lq.GetQueryResultsCountsOverride = func(queryIDs []uint) (map[uint]int, error) {
		countsRead = true
		return map[uint]int{1: 0, 2: 0}, nil
	}
	reset := func() {
		written, recorded, countsRead = nil, nil, false
		ds.QueryResultRowsForHostByQueryFuncInvoked = false
	}

	t.Run("no read slot", func(t *testing.T) {
		reset()
		serv.queryReportReadSem <- struct{}{}
		t.Cleanup(func() { <-serv.queryReportReadSem })

		// Nothing can be compared, so the results are dropped without touching Redis or the database.
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		assert.False(t, ds.QueryResultRowsForHostByQueryFuncInvoked)
		assert.Empty(t, recorded)
		assert.Empty(t, written)
		assert.False(t, countsRead)
	})

	t.Run("no write slot", func(t *testing.T) {
		reset()
		writeSlotsHeld = 1
		t.Cleanup(func() { writeSlotsHeld = 0 })

		// Unchanged results are still recorded; only the changed one is dropped.
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		assert.True(t, ds.QueryResultRowsForHostByQueryFuncInvoked)
		assert.Equal(t, []uint{100}, recorded)
		assert.Empty(t, written)
		assert.False(t, countsRead)
		assert.Empty(t, serv.queryReportReadSem)
	})

	t.Run("slots free", func(t *testing.T) {
		reset()
		serv.saveResultLogsToQueryReports(ctx, results, queries, fleet.DefaultMaxQueryReportRows)
		assert.Equal(t, []uint{100}, recorded)
		assert.Equal(t, []uint{2}, written)
		assert.True(t, countsRead)
		assert.Empty(t, serv.queryReportReadSem)
		assert.Zero(t, writeSlotsHeld)
	})

	t.Run("nothing changed takes no write slot", func(t *testing.T) {
		reset()
		writeSlotsHeld = 1
		t.Cleanup(func() { writeSlotsHeld = 0 })

		serv.saveResultLogsToQueryReports(ctx, results[:1], queries, fleet.DefaultMaxQueryReportRows)
		assert.Equal(t, []uint{100}, recorded)
		assert.Empty(t, written)
		assert.False(t, countsRead)
	})
}

func TestAcquireQueryReportWriteSlot(t *testing.T) {
	ds := new(mock.Store)
	lq := makeLiveQueryStore(t, 0)
	svc, ctx := newTestService(t, ds, nil, lq)
	serv := ((svc.(validationMiddleware)).Service).(*Service)
	serv.queryReportWriteLimit = 5
	serv.queryReportWriteFallbackSem = make(chan struct{}, 1)

	var acquiredToken, releasedToken string
	acquireErr := error(nil)
	lq.AcquireQueryReportWriteSlotOverride = func(token string, limit int, lease time.Duration) (bool, error) {
		require.Equal(t, 5, limit)
		require.Greater(t, lease, queryReportWriteTimeout)
		acquiredToken = token
		return acquireErr == nil, acquireErr
	}
	lq.ReleaseQueryReportWriteSlotOverride = func(token string) error {
		releasedToken = token
		return nil
	}

	t.Run("shared slot", func(t *testing.T) {
		release, ok := serv.acquireQueryReportWriteSlot(ctx)
		require.True(t, ok)
		require.NotEmpty(t, acquiredToken)
		release()
		require.Equal(t, acquiredToken, releasedToken)
	})

	t.Run("redis error falls back to the per-server limit", func(t *testing.T) {
		acquireErr = errors.New("redis down")
		t.Cleanup(func() { acquireErr = nil })

		release, ok := serv.acquireQueryReportWriteSlot(ctx)
		require.True(t, ok)
		_, ok = serv.acquireQueryReportWriteSlot(ctx)
		require.False(t, ok)
		release()
		require.Empty(t, serv.queryReportWriteFallbackSem)
	})

	t.Run("no limit", func(t *testing.T) {
		serv.queryReportWriteLimit = 0
		t.Cleanup(func() { serv.queryReportWriteLimit = 5 })
		acquiredToken = ""

		release, ok := serv.acquireQueryReportWriteSlot(ctx)
		require.True(t, ok)
		release()
		require.Empty(t, acquiredToken)
	})
}
