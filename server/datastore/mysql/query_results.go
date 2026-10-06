package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host.
// It deletes existing rows for the host/query and inserts the new rows.
// If the incoming result set has more than the row limit, it bails early without storing anything.
// If replacing the host's rows would push the query's total above maxQueryReportRows, nothing is
// changed: hosts already in the report keep updating, hosts not yet in it are skipped once it's full.
// Excess rows across all hosts are cleaned up by a separate cron job.
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows, currentCount int) (res fleet.QueryReportWriteResult, err error) {
	if len(rows) == 0 {
		return res, nil
	}

	// Bail early if the incoming result set is too large (more than the row limit from a single host)
	if len(rows) > 1000 {
		return res, nil
	}

	newDataRows := 0
	for _, row := range rows {
		if row.Data != nil {
			newDataRows++
		}
	}

	// Not retried: failures here mostly come from lock contention, which retries would add to,
	// and the host sends fresh results on the report's next run.
	err = ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		// Since we assume all rows have the same queryID, take it from the first row
		queryID := rows[0].QueryID
		hostID := rows[0].HostID

		// Stale rows are still replaced, but aren't counted: the report's count no longer includes them.
		var existing []struct {
			ID      uint `db:"id"`
			HasData bool `db:"has_data"`
		}
		selectStmt := `
			SELECT qr.id, (qr.has_data = 1 AND qr.id >= q.results_valid_from_id) AS has_data
			FROM query_results qr
			JOIN queries q ON q.id = qr.query_id
			WHERE qr.query_id = ? AND qr.host_id = ?`
		if err := sqlx.SelectContext(ctx, tx, &existing, selectStmt, queryID, hostID); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting existing query results for host")
		}
		existingDataRows := 0
		existingIDs := make([]uint, 0, len(existing))
		for _, e := range existing {
			existingIDs = append(existingIDs, e.ID)
			if e.HasData {
				existingDataRows++
			}
		}
		if currentCount-existingDataRows+newDataRows > maxQueryReportRows {
			res.Rejected = true
			return nil
		}

		// Delete by primary key: a DELETE by (query_id, host_id) takes next-key locks on the
		// secondary index, which block other hosts inserting results for the same query.
		if err := deleteQueryResultsByID(ctx, tx, existingIDs); err != nil {
			return err
		}

		// Insert the new rows
		valueStrings := make([]string, 0, len(rows))
		valueArgs := make([]interface{}, 0, len(rows)*4)
		for _, row := range rows {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			valueArgs = append(valueArgs, queryID, hostID, row.LastFetched, row.Data)
		}

		//nolint:gosec // SQL query is constructed using constant strings
		insertStmt := `
		INSERT IGNORE INTO query_results (query_id, host_id, last_fetched, data) VALUES
	` + strings.Join(valueStrings, ",")

		if _, err := tx.ExecContext(ctx, insertStmt, valueArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting new rows")
		}

		res.RowsAdded = newDataRows - existingDataRows
		res.NewHost = existingDataRows == 0 && newDataRows > 0
		return nil
	})

	return res, ctxerr.Wrap(ctx, err, "overwriting query result rows")
}

// deleteQueryResultsByID deletes query_results rows by primary key.
func deleteQueryResultsByID(ctx context.Context, db sqlx.ExecerContext, ids []uint) error {
	if len(ids) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`DELETE FROM query_results WHERE id IN (?)`, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building delete query results by id")
	}
	if _, err := db.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting query results by id")
	}
	return nil
}

// deleteQueryResultsBatchSize is the number of query_results rows deleted per statement when
// clearing a query's results.
var deleteQueryResultsBatchSize = 500

// Both page through a query's rows in index order with a host_id cursor, so each page is a
// range read that never rescans rows already handled.
const (
	selectQueryResultsPageStmt = `
		SELECT id, host_id FROM query_results FORCE INDEX (idx_query_id_host_id_last_fetched)
		WHERE query_id = ? AND host_id >= ? AND id < ?
		ORDER BY host_id LIMIT ?`
	selectQueryResultsWithDataPageStmt = `
		SELECT id, host_id FROM query_results FORCE INDEX (idx_query_id_has_data_host_id_last_fetched)
		WHERE query_id = ? AND has_data = 1 AND host_id >= ? AND id < ?
		ORDER BY host_id LIMIT ?`
)

// deleteQueryResultsBeforeID deletes a query's rows with id below beforeID (only rows with data
// if onlyWithData), reading and deleting one page of batchSize rows at a time. Rows are deleted
// by primary key: a range DELETE on query_id locks every row of the report, and the gaps between
// them, while hosts keep writing results for it.
func (ds *Datastore) deleteQueryResultsBeforeID(ctx context.Context, queryID, beforeID uint, onlyWithData bool, batchSize int) error {
	selectStmt := selectQueryResultsPageStmt
	if onlyWithData {
		selectStmt = selectQueryResultsWithDataPageStmt
	}
	var fromHostID uint
	for {
		// Read from the primary: the next page must not return rows the previous one deleted.
		var page []struct {
			ID     uint `db:"id"`
			HostID uint `db:"host_id"`
		}
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &page, selectStmt, queryID, fromHostID, beforeID, batchSize); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting query_results page to delete")
		}
		if len(page) == 0 {
			return nil
		}
		ids := make([]uint, 0, len(page))
		for _, row := range page {
			ids = append(ids, row.ID)
		}
		if err := deleteQueryResultsByID(ctx, ds.writer(ctx), ids); err != nil {
			return err
		}
		if len(page) < batchSize {
			return nil
		}
		// The last host may have more rows past this page; its deleted rows won't match again.
		fromHostID = page[len(page)-1].HostID
	}
}

// queryResultHostDisplayNameExpr mirrors fleet.HostDisplayName so sorting and
// searching by host name agree with the name shown in the report.
const queryResultHostDisplayNameExpr = `COALESCE(
	NULLIF(h.computer_name, ''),
	NULLIF(h.hostname, ''),
	IF(h.hardware_model != '' AND h.hardware_serial != '', CONCAT(h.hardware_model, ' (', h.hardware_serial, ')'), '')
)`

// queryResultRowsAllowedOrderKeys are the built-in order keys for QueryResultRows,
// referencing columns of its materialized page derived table.
// Any other key is treated as a result column name and sorted through the
// sort_value column selected alongside the row, so the column name is always a
// bound parameter and never part of the SQL text. Built-in keys take precedence
// over result columns with the same name.
var queryResultRowsAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"last_fetched": "page.last_fetched",
	"host_name":    "page.host_name",
	"host_id":      "page.host_id",
	"id":           "page.id",
}

const queryResultColumnOrderKey = "page.sort_value"

// queryResultRowPage is a row of the sorted, paginated page of query result IDs.
// sort_value is only selected so ORDER BY can reference it.
type queryResultRowPage struct {
	ID        uint    `db:"id"`
	SortValue *string `db:"sort_value"`
}

// queryResultRowWithID is a query result row scanned with its ID so the page
// order can be restored.
type queryResultRowWithID struct {
	fleet.ScheduledQueryResultRow
	ID uint `db:"id"`
}

// QueryResultRows returns the query result rows for a given query.
func (ds *Datastore) QueryResultRows(ctx context.Context, queryID uint, filter fleet.TeamFilter, opts fleet.ListOptions) ([]*fleet.ScheduledQueryResultRow, int, *fleet.PaginationMetadata, error) {
	whereClause := fmt.Sprintf(`
		FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		LEFT JOIN hosts h ON (qr.host_id=h.id)
		WHERE qr.query_id = ? AND qr.has_data = 1 AND %s
	`, ds.whereFilterHostsByTeams(filter, "h"))
	whereArgs := []any{queryID}

	if match := strings.TrimSpace(opts.MatchQuery); match != "" {
		// JSON_SEARCH uses LIKE semantics but compares with the binary JSON
		// collation, so lowercase both sides to keep the search case-insensitive
		// like the host name columns are.
		pattern := likePattern(match)
		whereClause += ` AND (h.hostname LIKE ? OR h.computer_name LIKE ? OR ` + queryResultHostDisplayNameExpr + ` LIKE ? OR JSON_SEARCH(LOWER(qr.data), 'one', ?) IS NOT NULL)`
		whereArgs = append(whereArgs, pattern, pattern, pattern, strings.ToLower(pattern))
	}

	// Sorting by a result column extracts it into sort_value with the column name
	// bound as a parameter; JSON_QUOTE builds a valid path member for any name.
	sortValueExpr := "NULL"
	var sortValueArgs []any
	allowedKeys := queryResultRowsAllowedOrderKeys
	if key := opts.OrderKey; key != "" {
		if _, ok := allowedKeys[key]; !ok {
			sortValueExpr = "JSON_UNQUOTE(JSON_EXTRACT(qr.data, CONCAT('$.', JSON_QUOTE(?))))"
			sortValueArgs = []any{key}
			allowedKeys = maps.Clone(queryResultRowsAllowedOrderKeys)
			allowedKeys[key] = queryResultColumnOrderKey
		}
		// Result columns and timestamps aren't unique, so break ties deterministically.
		opts.TestSecondaryOrderKey = "id"
	}

	// Sort and paginate on IDs only, then fetch the data. MySQL's filesort
	// carries every column the query reads from qr (including qr.data read by
	// the search filter or sort_value), and a single stored result can exceed
	// the default 256 KiB sort_buffer_size ("Out of sort memory"). NO_MERGE
	// keeps the derived table materialized so the sort only sees its columns.
	pageStmt := `
		SELECT /*+ NO_MERGE(page) */ page.id, page.sort_value FROM (
			SELECT qr.id, qr.host_id, qr.last_fetched,
				` + queryResultHostDisplayNameExpr + ` AS host_name,
				` + sortValueExpr + ` AS sort_value
			` + whereClause + `
		) page
	`
	pageArgs := append(append([]any{}, sortValueArgs...), whereArgs...)
	// Unpaginated callers expect every row; the list-options helper would
	// otherwise silently cap them at DefaultPerPage.
	if opts.PerPage == 0 {
		opts.PerPage = fleet.PerPageUnlimited
	}
	pagedStmt, pagedArgs, err := appendListOptionsWithCursorToSQLSecure(pageStmt, pageArgs, &opts, allowedKeys)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "apply list options for query result rows")
	}

	dbReader := ds.reader(ctx)
	var page []queryResultRowPage
	if err := sqlx.SelectContext(ctx, dbReader, &page, pagedStmt, pagedArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "selecting query result rows")
	}
	hasNextResults := opts.IncludeMetadata && len(page) > int(opts.PerPage) //nolint:gosec // dismiss G115
	if hasNextResults {
		page = page[:opts.PerPage]
	}

	ids := make([]uint, 0, len(page))
	for _, p := range page {
		ids = append(ids, p.ID)
	}
	rowsByID := make(map[uint]*fleet.ScheduledQueryResultRow, len(ids))
	const idBatchSize = 10000
	for batch := range slices.Chunk(ids, idBatchSize) {
		stmt, args, err := sqlx.In(`
			SELECT qr.id, qr.query_id, qr.host_id, qr.last_fetched, qr.data,
				h.hostname, h.computer_name, h.hardware_model, h.hardware_serial
			FROM query_results qr
			LEFT JOIN hosts h ON (qr.host_id=h.id)
			WHERE qr.id IN (?)
		`, batch)
		if err != nil {
			return nil, 0, nil, ctxerr.Wrap(ctx, err, "building query result rows data statement")
		}
		var rows []queryResultRowWithID
		if err := sqlx.SelectContext(ctx, dbReader, &rows, dbReader.Rebind(stmt), args...); err != nil {
			return nil, 0, nil, ctxerr.Wrap(ctx, err, "selecting query result rows data")
		}
		for i := range rows {
			rowsByID[rows[i].ID] = &rows[i].ScheduledQueryResultRow
		}
	}
	results := make([]*fleet.ScheduledQueryResultRow, 0, len(ids))
	for _, id := range ids {
		// A row can be replaced by a host check-in between the two queries.
		if row, ok := rowsByID[id]; ok {
			results = append(results, row)
		}
	}

	var total int
	if err := sqlx.GetContext(ctx, dbReader, &total, "SELECT COUNT(*) "+whereClause, whereArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "counting query result rows")
	}

	var metadata *fleet.PaginationMetadata
	if opts.IncludeMetadata {
		metadata = &fleet.PaginationMetadata{
			HasPreviousResults: opts.Page > 0,
			TotalResults:       uint(total), //nolint:gosec // dismiss G115
			HasNextResults:     hasNextResults,
		}
	}

	// total is also returned on its own because callers need it when metadata is
	// not requested (per_page unset).
	return results, total, metadata, nil
}

// ResultCountForQueryAndHost counts the query report rows for a given query and host
// excluding rows with null data
func (ds *Datastore) ResultCountForQueryAndHost(ctx context.Context, queryID, hostID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `
		SELECT COUNT(*) FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		WHERE qr.query_id = ? AND qr.host_id = ? AND qr.has_data = 1`, queryID, hostID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query and host")
	}

	return count, nil
}

// ResultCountsForQueries counts the stored rows with data for each query.
func (ds *Datastore) ResultCountsForQueries(ctx context.Context, queryIDs []uint) (map[uint]int, error) {
	counts := make(map[uint]int, len(queryIDs))
	if len(queryIDs) == 0 {
		return counts, nil
	}

	stmt, args, err := sqlx.In(`
		SELECT qr.query_id, COUNT(*) AS n FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		WHERE qr.query_id IN (?) AND qr.has_data = 1
		GROUP BY qr.query_id`, queryIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query results count statement")
	}
	var rows []struct {
		QueryID uint `db:"query_id"`
		N       int  `db:"n"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "counting query results for queries")
	}
	for _, row := range rows {
		counts[row.QueryID] = row.N
	}
	return counts, nil
}

// QueryResultRowsForHost returns the query result rows for a given query and host
// including rows with null data
func (ds *Datastore) QueryResultRowsForHost(ctx context.Context, queryID, hostID uint) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := `
		SELECT qr.query_id, qr.host_id, qr.last_fetched, qr.data FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		WHERE qr.query_id = ? AND qr.host_id = ?
	`
	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows for host")
	}

	return results, nil
}

// QueryResultRowsForHostByQuery returns a host's stored rows for each of the given queries,
// including rows with null data. Rows are returned in insert order.
func (ds *Datastore) QueryResultRowsForHostByQuery(ctx context.Context, hostID uint, queryIDs []uint) (map[uint][]*fleet.StoredQueryResultRow, error) {
	if len(queryIDs) == 0 {
		return nil, nil
	}
	// Rows hidden by a results-clearing edit must not count as stored, or identical results from
	// the new version would be skipped as unchanged and then deleted with the hidden rows.
	stmt, args, err := sqlx.In(`
		SELECT qr.id, qr.query_id, qr.host_id, qr.last_fetched, qr.data FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		WHERE qr.host_id = ? AND qr.query_id IN (?)
		ORDER BY qr.id`, hostID, queryIDs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building select query result rows for host")
	}
	var rows []*fleet.StoredQueryResultRow
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows for host")
	}
	byQuery := make(map[uint][]*fleet.StoredQueryResultRow)
	for _, row := range rows {
		byQuery[row.QueryID] = append(byQuery[row.QueryID], row)
	}
	return byQuery, nil
}

// UpdateQueryResultsLastFetched updates rows by primary key: an update by (query_id, host_id)
// takes next-key locks on the secondary index, which block other hosts inserting results for
// the same query. Both secondary indexes include last_fetched, so their entries for the updated
// rows are still rewritten.
func (ds *Datastore) UpdateQueryResultsLastFetched(ctx context.Context, ids []uint, lastFetched time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	stmt, args, err := sqlx.In(`UPDATE query_results SET last_fetched = GREATEST(last_fetched, ?) WHERE id IN (?)`, lastFetched, ids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building update query results last fetched")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "updating query results last fetched")
	}
	return nil
}

// Above any id a BIGINT UNSIGNED query_results.id will reach.
const allQueryResultsBeforeID = math.MaxInt64

func (ds *Datastore) CleanupDiscardedQueryResults(ctx context.Context) error {
	// Both reads go to the primary, and only rows stored before the run are deleted: a report whose
	// results are turned back on mid-run keeps the rows hosts store from then on.
	var maxID sql.NullInt64
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &maxID, `SELECT MAX(id) FROM query_results`); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting last query_results id")
	}
	if !maxID.Valid {
		return nil
	}
	var queryIDs []uint
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &queryIDs, `SELECT id FROM queries WHERE discard_data = 1`); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting discarded queries")
	}
	for _, queryID := range queryIDs {
		if err := ds.deleteQueryResultsBeforeID(ctx, queryID, uint(maxID.Int64)+1, false, deleteQueryResultsBatchSize); err != nil { //nolint:gosec // dismiss G115
			return ctxerr.Wrapf(ctx, err, "cleaning up discarded results of query %d", queryID)
		}
	}
	return nil
}

func (ds *Datastore) deleteStaleQueryResults(ctx context.Context, queryID, validFromID uint) error {
	if err := ds.deleteQueryResultsBeforeID(ctx, queryID, validFromID, false, deleteQueryResultsBatchSize); err != nil {
		return err
	}
	// A later edit that cleared the results again moved the cutoff, so the query stays pending.
	// updated_at is kept: this isn't an edit of the report.
	if _, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE queries SET results_cleanup_pending = 0, updated_at = updated_at WHERE id = ? AND results_valid_from_id = ?`,
		queryID, validFromID); err != nil {
		return ctxerr.Wrap(ctx, err, "clearing query results cleanup flag")
	}
	return nil
}

func (ds *Datastore) CleanupStaleQueryResults(ctx context.Context) error {
	var pending []struct {
		ID          uint `db:"id"`
		ValidFromID uint `db:"results_valid_from_id"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &pending,
		`SELECT id, results_valid_from_id FROM queries WHERE results_cleanup_pending = 1`); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting queries with stale results")
	}
	for _, q := range pending {
		if err := ds.deleteStaleQueryResults(ctx, q.ID, q.ValidFromID); err != nil {
			return ctxerr.Wrapf(ctx, err, "cleaning up stale results of query %d", q.ID)
		}
	}

	var resultQueryIDs []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &resultQueryIDs, `SELECT DISTINCT query_id FROM query_results`); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting queries with results")
	}
	// Read from the primary: a report created moments ago may not be on the replica yet, and
	// its results would look orphaned.
	existing := make(map[uint]struct{}, len(resultQueryIDs))
	for batch := range slices.Chunk(resultQueryIDs, 50000) {
		stmt, args, err := sqlx.In(`SELECT id FROM queries WHERE id IN (?)`, batch)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building existing queries statement")
		}
		var existingIDs []uint
		if err := sqlx.SelectContext(ctx, ds.writer(ctx), &existingIDs, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting existing queries")
		}
		for _, id := range existingIDs {
			existing[id] = struct{}{}
		}
	}
	for _, queryID := range resultQueryIDs {
		if _, ok := existing[queryID]; ok {
			continue
		}
		if err := ds.deleteQueryResultsBeforeID(ctx, queryID, allQueryResultsBeforeID, false, deleteQueryResultsBatchSize); err != nil {
			return ctxerr.Wrapf(ctx, err, "cleaning up results of deleted query %d", queryID)
		}
	}
	return nil
}

// CleanupExcessQueryResultRows deletes query result rows that exceed the maximum
// allowed per query. It keeps the most recent rows (by id, which correlates with insert order) up to the limit.
// Deletes are batched to avoid large binlogs and long lock times.
// This runs as a cron job to ensure the query_results table doesn't grow unbounded.
// Returns a map of query IDs to their current row count after cleanup (for syncing Redis counters).
func (ds *Datastore) CleanupExcessQueryResultRows(ctx context.Context, maxQueryReportRows int, opts ...fleet.CleanupExcessQueryResultRowsOptions) (map[uint]int, error) {
	batchSize := 500
	// Allow overriding the batch size mainly for tests.
	if len(opts) > 0 && opts[0].BatchSize > 0 {
		batchSize = opts[0].BatchSize
	}

	// Get all saved query IDs that could have query results to clean up.
	// Only saved queries (scheduled reports) store rows in query_results;
	// live queries do not, so there's nothing to clean up for them.
	var queryIDs []uint
	selectStmt := `
		SELECT id
		FROM queries
		WHERE saved = 1 AND discard_data = false AND logging_type = 'snapshot'
	`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &queryIDs, selectStmt); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query IDs for cleanup")
	}

	// Nothing to do, bail early.
	if len(queryIDs) == 0 {
		return map[uint]int{}, nil
	}

	// Get the cutoff IDs for each query in one query.
	// Cutoff is the ID of the Nth most recent row,
	// where N is the maxQueryReportRows.
	type cutoffRow struct {
		QueryID  uint `db:"query_id"`
		CutoffID uint `db:"cutoff_id"`
	}
	var queryCutoffs []cutoffRow
	cutoffStmt := `
        SELECT query_id, id as cutoff_id FROM (
            SELECT qr.query_id, qr.id,
                ROW_NUMBER() OVER (PARTITION BY qr.query_id ORDER BY qr.id DESC) as rn
            FROM query_results qr
            JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
            WHERE qr.query_id IN (?) AND qr.has_data = 1
        ) cutoff
        WHERE rn = ?
    `
	// Batch the IN clause to avoid MySQL's 65,535 placeholder limit.
	const queryIDBatchSize = 50000
	for batch := range slices.Chunk(queryIDs, queryIDBatchSize) {
		var batchCutoffs []cutoffRow
		query, args, err := sqlx.In(cutoffStmt, batch, maxQueryReportRows)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "building cutoff query")
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &batchCutoffs, ds.reader(ctx).Rebind(query), args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "selecting cutoffs")
		}
		queryCutoffs = append(queryCutoffs, batchCutoffs...)
	}

	// Delete excess rows from each query, in batches. IDs are selected first and deleted by
	// primary key: a DELETE filtered by query_id and id scans (and locks) the query's whole
	// secondary index range, stalling every host writing results for that query.
	for _, c := range queryCutoffs {
		if err := ds.deleteQueryResultsBeforeID(ctx, c.QueryID, c.CutoffID, true, batchSize); err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "cleaning up query %d", c.QueryID)
		}
	}

	// Count the results for each query.
	// This will be used to sync Redis counters.
	type countRow struct {
		QueryID uint `db:"query_id"`
		Count   int  `db:"count"`
	}
	var counts []countRow
	countStmt := `
        SELECT qr.query_id, COUNT(*) as count
        FROM query_results qr
        JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
        WHERE qr.query_id IN (?) AND qr.has_data = 1
        GROUP BY qr.query_id
    `
	for batch := range slices.Chunk(queryIDs, queryIDBatchSize) {
		var batchCounts []countRow
		query, args, err := sqlx.In(countStmt, batch)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "building count query")
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &batchCounts, ds.reader(ctx).Rebind(query), args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "selecting counts")
		}
		counts = append(counts, batchCounts...)
	}

	queryCounts := make(map[uint]int)
	for _, c := range counts {
		queryCounts[c.QueryID] = c.Count
	}

	// Include queries with 0 results
	for _, qid := range queryIDs {
		if _, ok := queryCounts[qid]; !ok {
			queryCounts[qid] = 0
		}
	}

	return queryCounts, nil
}

// hostReportAllowedOrderKeys defines the allowed order keys for ListHostReports.
// The last_fetched entry is overridden dynamically in ListHostReports with a
// direction-aware COALESCE sentinel so that NULLs sort last in both ASC and DESC
// and the expression remains a single column (required for cursor pagination).
var hostReportAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"name":         "q.name",
	"last_fetched": "qr_stats.last_result_fetched",
}

// hostReportRow is a scan target for the paginated query list in ListHostReports.
type hostReportRow struct {
	QueryID           uint         `db:"id"`
	Name              string       `db:"name"`
	Description       string       `db:"description"`
	LastResultFetched sql.NullTime `db:"last_result_fetched"`
	DiscardData       bool         `db:"discard_data"`
	LoggingType       string       `db:"logging_type"`
}

// ListHostReports returns reports associated with a host, applying
// the provided filtering, sorting, and pagination options. ReportClipped is
// left for the service layer to fill in.
func (ds *Datastore) ListHostReports(
	ctx context.Context,
	hostID uint,
	teamID *uint,
	hostPlatform string,
	opts fleet.ListHostReportsOptions,
) ([]*fleet.HostReport, int, *fleet.PaginationMetadata, error) {
	// We only care about saved queries
	whereClause := "WHERE q.saved = 1"
	var whereArgs []any

	// We also want to show queries that have not run yet, so we need
	// to figure out which queries are associated with the host based
	// on Team membership.
	switch {
	case teamID != nil:
		whereArgs = append(whereArgs, *teamID)
		whereClause += " AND (q.team_id IS NULL OR q.team_id = ?)"
	default:
		whereClause += " AND q.team_id IS NULL"
	}

	// By default, only include queries that store results (discard_data=0 AND
	// logging_type='snapshot'). When IncludeReportsDontStoreResults is set,
	// all queries are returned regardless of their storage settings.
	if !opts.IncludeReportsDontStoreResults {
		whereClause += " AND q.discard_data = 0 AND q.logging_type = 'snapshot'"
	}

	if opts.ExcludeIncludeAllQueries {
		whereClause += `
		AND NOT EXISTS (
			SELECT 1 FROM query_labels ql
			WHERE ql.query_id = q.id AND ql.require_all = 1
		)`
	}

	labelSQL, labelArgs := queryLabelScope(hostID)
	whereClause += labelSQL
	whereArgs = append(whereArgs, labelArgs...)

	// Filter by platform: include queries with no platform restriction, or
	// whose platform list contains the host's normalized platform.
	whereClause += " AND (q.platform = '' OR FIND_IN_SET(?, q.platform) > 0)"
	whereArgs = append(whereArgs, hostPlatform)

	matchQuery := strings.TrimSpace(opts.ListOptions.MatchQuery)
	if matchQuery != "" {
		whereClause, whereArgs = searchLike(whereClause, whereArgs, matchQuery, "q.name")
	}

	countStmt := "SELECT COUNT(*) FROM queries q " + whereClause

	// Do a LATERAL subquery for each row in queries q so that everything stays in index space
	listStmt := `
		SELECT q.id, q.name, q.description, q.discard_data, q.logging_type, qr_stats.last_result_fetched
		FROM queries q
		LEFT JOIN LATERAL (
			SELECT MAX(last_fetched) AS last_result_fetched
			FROM query_results
			WHERE query_id = q.id AND host_id = ? AND id >= q.results_valid_from_id
		) qr_stats ON TRUE
	` + whereClause
	listArgs := append([]any{hostID}, whereArgs...)

	// For last_fetched, replace the static allowlist entry with a direction-aware
	// COALESCE so that NULLs sort last in both ASC and DESC while keeping the
	// expression as a single column (required for cursor WHERE comparison).
	// A secondary sort by q.id breaks timestamp ties deterministically.
	allowedKeys := hostReportAllowedOrderKeys
	if opts.ListOptions.OrderKey == "last_fetched" {
		sentinel := "'9999-12-31 23:59:59'" // NULLs → max, sort last in ASC
		if opts.ListOptions.OrderDirection == fleet.OrderDescending {
			sentinel = "'0001-01-01 00:00:00'" // NULLs → min, sort last in DESC
		}
		allowedKeys = make(common_mysql.OrderKeyAllowlist, len(hostReportAllowedOrderKeys)+1)
		maps.Copy(allowedKeys, hostReportAllowedOrderKeys)
		allowedKeys["last_fetched"] = fmt.Sprintf("COALESCE(qr_stats.last_result_fetched, %s)", sentinel)
		allowedKeys["id"] = "q.id"
		opts.ListOptions.TestSecondaryOrderKey = "id"
	}

	pagedStmt, pagedArgs, err := appendListOptionsWithCursorToSQLSecure(listStmt, listArgs, &opts.ListOptions, allowedKeys)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "apply list options for host reports")
	}

	dbReader := ds.reader(ctx)

	var queryRows []hostReportRow
	if err := sqlx.SelectContext(ctx, dbReader, &queryRows, pagedStmt, pagedArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "listing host reports")
	}

	var total int
	if err := sqlx.GetContext(ctx, dbReader, &total, countStmt, whereArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "counting host reports")
	}

	metadata := &fleet.PaginationMetadata{HasPreviousResults: opts.ListOptions.Page > 0}
	if len(queryRows) > int(opts.ListOptions.PerPage) { //nolint:gosec // dismiss G115
		metadata.HasNextResults = true
		queryRows = queryRows[:len(queryRows)-1]
	}

	if len(queryRows) == 0 {
		return []*fleet.HostReport{}, total, metadata, nil
	}

	// Collect IDs for the current page.
	queryIDs := make([]uint, 0, len(queryRows))
	for _, r := range queryRows {
		queryIDs = append(queryIDs, r.QueryID)
	}

	// Fetch the host-specific result count per query, used to populate
	// NHostResults.
	type hostCountRow struct {
		QueryID      uint `db:"query_id"`
		NHostResults int  `db:"n_host_results"`
	}
	hostCountStmt, hostCountArgs, err := sqlx.In(`
		SELECT qr.query_id, COUNT(*) AS n_host_results
		FROM query_results qr
		JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
		WHERE qr.query_id IN (?) AND qr.host_id = ? AND qr.has_data = 1
		GROUP BY qr.query_id
	`, queryIDs, hostID)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "building host count query for host reports")
	}
	var hostCountRows []hostCountRow
	if err := sqlx.SelectContext(ctx, dbReader, &hostCountRows, dbReader.Rebind(hostCountStmt), hostCountArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "fetching host result counts for host reports")
	}
	nHostResultsByID := make(map[uint]int, len(hostCountRows))
	for _, r := range hostCountRows {
		nHostResultsByID[r.QueryID] = r.NHostResults
	}

	// Fetch the single most recent result row per query for this host.
	type firstDataRow struct {
		QueryID uint             `db:"query_id"`
		Data    *json.RawMessage `db:"data"`
	}
	// Rank on IDs only and join data afterwards; sorting rows that carry data
	// can exceed MySQL's sort buffer.
	firstDataStmt, firstDataArgs, err := sqlx.In(`
		SELECT qr.query_id, qr.data
		FROM (
			SELECT
				qr.id,
				ROW_NUMBER() OVER (PARTITION BY qr.query_id ORDER BY qr.last_fetched DESC) AS rn
			FROM query_results qr
			JOIN queries q ON q.id = qr.query_id AND qr.id >= q.results_valid_from_id
			WHERE qr.query_id IN (?) AND qr.host_id = ? AND qr.has_data = 1
		) ranked
		JOIN query_results qr ON qr.id = ranked.id
		WHERE ranked.rn = 1
	`, queryIDs, hostID)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "building first data query for host reports")
	}
	var firstDataRows []firstDataRow
	if err := sqlx.SelectContext(ctx, dbReader, &firstDataRows, dbReader.Rebind(firstDataStmt), firstDataArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "fetching first result data for host reports")
	}
	firstDataByQueryID := make(map[uint]*json.RawMessage, len(firstDataRows))
	for i := range firstDataRows {
		if firstDataRows[i].Data != nil {
			firstDataByQueryID[firstDataRows[i].QueryID] = firstDataRows[i].Data
		}
	}

	// Map to HostReport structs, joining in the batch-fetched metadata.
	reports := make([]*fleet.HostReport, 0, len(queryRows))
	for _, qr := range queryRows {
		r := &fleet.HostReport{
			ReportID:     qr.QueryID,
			Name:         qr.Name,
			Description:  qr.Description,
			StoreResults: !qr.DiscardData && qr.LoggingType == fleet.LoggingSnapshot,
		}
		if qr.LastResultFetched.Valid {
			t := qr.LastResultFetched.Time
			r.LastFetched = &t
		}
		r.NHostResults = nHostResultsByID[qr.QueryID]
		if data, ok := firstDataByQueryID[qr.QueryID]; ok {
			var cols map[string]string
			if err := json.Unmarshal(*data, &cols); err != nil {
				return nil, 0, nil, ctxerr.Wrap(ctx, err, "unmarshal first result data")
			}
			r.FirstResult = cols
		}
		reports = append(reports, r)
	}

	return reports, total, metadata, nil
}
