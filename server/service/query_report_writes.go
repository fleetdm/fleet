package service

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"slices"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const (
	queryReportSkipReadBusy           = "read_busy"
	queryReportSkipWriteBusy          = "write_busy"
	queryReportSkipUnchanged          = "unchanged"
	queryReportSkipLastFetchedDropped = "last_fetched_dropped"
)

// queryReportWritesSkipped counts report results not written to the database, by reason:
// "read_busy" when the server is at osquery.max_concurrent_query_report_reads, "write_busy" when
// the deployment is at osquery.max_concurrent_query_report_writes, "unchanged" when the host's
// stored rows already match, and "last_fetched_dropped" when they match but the pending
// last_fetched updates are full, so last_fetched isn't refreshed.
var queryReportWritesSkipped = mustNewQueryReportWritesSkippedCounter()

func mustNewQueryReportWritesSkippedCounter() metric.Int64Counter {
	c, err := otel.Meter("fleet").Int64Counter(
		"fleet.query_reports.result_writes_skipped",
		metric.WithDescription("Count of report results not written to the database, by reason"),
		metric.WithUnit("{result}"),
	)
	if err != nil {
		panic(err)
	}
	return c
}

func queryReportSkipReason(reason string) metric.AddOption {
	return metric.WithAttributes(attribute.String("reason", reason))
}

// tryAcquireSlot takes a slot of sem without waiting and reports whether it got one. A nil sem
// has no limit.
func tryAcquireSlot(sem chan struct{}) bool {
	if sem == nil {
		return true
	}
	select {
	case sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// releaseSlot releases a slot taken with tryAcquireSlot.
func releaseSlot(sem chan struct{}) {
	if sem != nil {
		<-sem
	}
}

const (
	// queryReportWriteTimeout bounds a request's report writes, so they end before their write
	// slot's lease expires and another server can take it.
	queryReportWriteTimeout   = 30 * time.Second
	queryReportWriteSlotLease = 2 * queryReportWriteTimeout
	// queryReportWriteFallbackLimit is the per-server write limit used when the shared slots in
	// Redis can't be reached.
	queryReportWriteFallbackLimit = 20
)

// acquireQueryReportWriteSlot takes one of the osquery.max_concurrent_query_report_writes slots
// shared by all Fleet servers. If they can't be reached it falls back to a per-server limit, so
// a Redis outage neither blocks nor unbounds writes. ok is false if no slot is free; otherwise
// release must be called once the writes are done.
func (svc *Service) acquireQueryReportWriteSlot(ctx context.Context) (release func(), ok bool) {
	if svc.queryReportWriteLimit <= 0 {
		return func() {}, true
	}
	if svc.liveQueryStore != nil {
		token := uuid.NewString()
		acquired, err := svc.liveQueryStore.AcquireQueryReportWriteSlot(token, svc.queryReportWriteLimit, queryReportWriteSlotLease)
		if err == nil {
			if !acquired {
				return nil, false
			}
			return func() {
				if err := svc.liveQueryStore.ReleaseQueryReportWriteSlot(token); err != nil {
					// The lease frees the slot anyway.
					svc.logger.DebugContext(ctx, "release query report write slot", "err", err)
				}
			}, true
		}
		svc.logger.ErrorContext(ctx, "acquire query report write slot, using per-server limit", "err", err)
	}
	if !tryAcquireSlot(svc.queryReportWriteFallbackSem) {
		return nil, false
	}
	return func() { releaseSlot(svc.queryReportWriteFallbackSem) }, true
}

// readStoredQueryResultRows reads a host's stored rows for the given queries from the replica.
// busy is true if the server is at osquery.max_concurrent_query_report_reads, in which case
// nothing is read. On a read error the rows are returned empty, so every result is written.
func (svc *Service) readStoredQueryResultRows(ctx context.Context, hostID uint, queryIDs []uint) (rows map[uint][]*fleet.StoredQueryResultRow, busy bool) {
	if !tryAcquireSlot(svc.queryReportReadSem) {
		return nil, true
	}
	defer releaseSlot(svc.queryReportReadSem)

	rows, err := svc.ds.QueryResultRowsForHostByQuery(ctx, hostID, queryIDs)
	if err != nil {
		svc.logger.ErrorContext(ctx, "read stored query results for host", "err", err, "host_id", hostID)
		return nil, false
	}
	return rows, false
}

// queryResultRowsUnchanged reports whether rows hold the same data as stored, in any order.
func queryResultRowsUnchanged(rows []*fleet.ScheduledQueryResultRow, stored []*fleet.StoredQueryResultRow) bool {
	if len(stored) == 0 || len(rows) != len(stored) {
		return false
	}
	newData := make([]*json.RawMessage, 0, len(rows))
	for _, r := range rows {
		newData = append(newData, r.Data)
	}
	storedData := make([]*json.RawMessage, 0, len(stored))
	for _, s := range stored {
		storedData = append(storedData, s.Data)
	}
	newCanonical, ok := canonicalResultRows(newData)
	if !ok {
		return false
	}
	storedCanonical, ok := canonicalResultRows(storedData)
	if !ok {
		return false
	}
	return slices.Equal(newCanonical, storedCanonical)
}

// queryResultsLastFetchedRefreshAge is how old stored rows' last_fetched must be before an
// unchanged result refreshes it: updating last_fetched rewrites both secondary indexes of every
// row and logs full row images to the binlog, so doing it on every run costs the writer nearly as
// much as rewriting the rows.
const queryResultsLastFetchedRefreshAge = time.Hour

// queryResultRowsLastFetchedStale reports whether any of the stored rows' last_fetched is older
// than queryResultsLastFetchedRefreshAge at fetchedAt.
func queryResultRowsLastFetchedStale(stored []*fleet.StoredQueryResultRow, fetchedAt time.Time) bool {
	for _, row := range stored {
		if fetchedAt.Sub(row.LastFetched) >= queryResultsLastFetchedRefreshAge {
			return true
		}
	}
	return false
}

// canonicalResultRows returns each row's data re-encoded and sorted, so rows compare equal
// regardless of row order or of how MySQL's JSON type reorders keys and re-escapes strings.
// Null data is kept as an empty string.
func canonicalResultRows(rows []*json.RawMessage) ([]string, bool) {
	out := make([]string, 0, len(rows))
	for _, data := range rows {
		if data == nil {
			out = append(out, "")
			continue
		}
		dec := json.NewDecoder(bytes.NewReader(*data))
		// Keep numbers as their literals, so large integers don't lose precision and compare equal.
		dec.UseNumber()
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, false
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		out = append(out, string(b))
	}
	slices.Sort(out)
	return out, true
}

// queryResultsLastFetchedBatchSize is the number of query_results rows updated per statement.
const queryResultsLastFetchedBatchSize = 1000

// UpdateQueryResultsLastFetched updates last_fetched of the rows recorded by
// RecordQueryResultsLastFetched for results that arrived unchanged. Run by the
// query_results_cleanup cron.
func UpdateQueryResultsLastFetched(ctx context.Context, ds fleet.Datastore, lq fleet.LiveQueryStore) error {
	fetched, err := lq.LoadQueryResultsLastFetched()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load query results last fetched")
	}

	// Fetch times are truncated to the minute so most rows share one and fill whole batches. IDs
	// are sorted so concurrent transactions lock rows in the same order.
	idsByFetchedAt := make(map[time.Time][]uint)
	for id, ts := range fetched {
		minute := ts.Truncate(time.Minute)
		idsByFetchedAt[minute] = append(idsByFetchedAt[minute], id)
	}
	for _, ts := range slices.SortedFunc(maps.Keys(idsByFetchedAt), time.Time.Compare) {
		ids := idsByFetchedAt[ts]
		slices.Sort(ids)
		for batch := range slices.Chunk(ids, queryResultsLastFetchedBatchSize) {
			if err := ds.UpdateQueryResultsLastFetched(ctx, batch, ts); err != nil {
				// The processing set is kept, so the next run retries these rows.
				return ctxerr.Wrap(ctx, err, "update query results last fetched")
			}
		}
	}

	return ctxerr.Wrap(ctx, lq.ClearProcessedQueryResultsLastFetched(), "clear processed query results last fetched")
}
