package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host.
// It deletes existing rows for the host/query and inserts the new rows.
// If the incoming result set has more than the row limit, it bails early without storing anything.
// Excess rows across all hosts are cleaned up by a separate cron job.
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) (rowsAdded int, err error) {
	if len(rows) == 0 {
		return 0, nil
	}

	// Bail early if the incoming result set is too large (more than the row limit from a single host)
	if len(rows) > 1000 {
		return 0, nil
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Since we assume all rows have the same queryID, take it from the first row
		queryID := rows[0].QueryID
		hostID := rows[0].HostID

		// Delete rows based on the specific queryID and hostID
		deleteStmt := `DELETE FROM query_results WHERE host_id = ? AND query_id = ?`
		result, err := tx.ExecContext(ctx, deleteStmt, hostID, queryID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting query results for host")
		}
		deletedRows, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting rows affected for delete")
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

		result, err = tx.ExecContext(ctx, insertStmt, valueArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting new rows")
		}
		insertedRows, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting rows affected for insert")
		}

		rowsAdded = int(insertedRows - deletedRows)
		return nil
	})

	return rowsAdded, ctxerr.Wrap(ctx, err, "overwriting query result rows")
}

// TODO(lucas): Any chance we can store hostname in the query_results table?
// (to avoid having to left join hosts).
// QueryResultRows returns the query result rows for a given query
func (ds *Datastore) QueryResultRows(ctx context.Context, queryID uint, filter fleet.TeamFilter) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := fmt.Sprintf(`
		SELECT qr.query_id, qr.host_id, qr.last_fetched, qr.data,
			h.hostname, h.computer_name, h.hardware_model, h.hardware_serial
			FROM query_results qr
			LEFT JOIN hosts h ON (qr.host_id=h.id)
			WHERE query_id = ? AND has_data = 1 AND %s
		`, ds.whereFilterHostsByTeams(filter, "h"))

	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows")
	}

	return results, nil
}

// ResultCountForQuery counts the query report rows for a given query
// excluding rows with null data
func (ds *Datastore) ResultCountForQuery(ctx context.Context, queryID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND has_data = 1`, queryID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query")
	}

	return count, nil
}

// ResultCountForQueryAndHost counts the query report rows for a given query and host
// excluding rows with null data
func (ds *Datastore) ResultCountForQueryAndHost(ctx context.Context, queryID, hostID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND host_id = ? AND has_data = 1`, queryID, hostID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query and host")
	}

	return count, nil
}

// QueryResultRowsForHost returns the query result rows for a given query and host
// including rows with null data
func (ds *Datastore) QueryResultRowsForHost(ctx context.Context, queryID, hostID uint) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := `
               SELECT query_id, host_id, last_fetched, data FROM query_results
                       WHERE query_id = ? AND host_id = ?
               `
	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID, hostID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows for host")
	}

	return results, nil
}

func (ds *Datastore) CleanupDiscardedQueryResults(ctx context.Context) error {
	deleteStmt := `
		DELETE FROM query_results
		WHERE query_id IN
			(SELECT id FROM queries WHERE discard_data = true)
		`
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up discarded query results")
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
            SELECT query_id, id,
                ROW_NUMBER() OVER (PARTITION BY query_id ORDER BY id DESC) as rn
            FROM query_results
            WHERE query_id IN (?) AND has_data = 1
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

	// Delete excess rows from each query, in batches.
	if len(queryCutoffs) > 0 {
		for _, c := range queryCutoffs {
			deleteStmt := `
                DELETE FROM query_results
                WHERE query_id = ? AND id < ? AND has_data = 1
                LIMIT ?
            `
			for {
				result, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, c.QueryID, c.CutoffID, batchSize)
				if err != nil {
					return nil, ctxerr.Wrapf(ctx, err, "cleaning up query %d", c.QueryID)
				}
				rowsAffected, _ := result.RowsAffected()
				if rowsAffected == 0 {
					break
				}
			}
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
        SELECT query_id, COUNT(*) as count
        FROM query_results
        WHERE query_id IN (?) AND has_data = 1
        GROUP BY query_id
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
// the provided filtering, sorting, and pagination options. maxQueryReportRows
// is the configured report cap; a query whose total result count (across all
// hosts) meets or exceeds this value is considered clipped.
func (ds *Datastore) ListHostReports(
	ctx context.Context,
	hostID uint,
	teamID *uint,
	hostPlatform string,
	opts fleet.ListHostReportsOptions,
	maxQueryReportRows int,
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

	// labels_include_all is a premium-only feature. On free tier, hide any
	// query that has include_all labels (require_all=1) from the reports
	// list, even if such rows pre-exist (e.g. after a tier downgrade).
	// Mirrors the server-side write gate so include_all data never surfaces
	// via the reports API on free tier.
	if !license.IsPremium(ctx) {
		whereClause += `
		AND NOT EXISTS (
			SELECT 1 FROM query_labels ql
			WHERE ql.query_id = q.id AND ql.require_all = 1
		)`
	}

	// Filter by label membership. Two scopes coexist on query_labels via
	// require_all:
	//   include_any (require_all=0): host must be in at least ONE of the labels
	//   include_all (require_all=1): host must be in EVERY one of the labels
	whereClause += `
		AND (
			NOT EXISTS (
				SELECT 1 FROM query_labels ql
				WHERE ql.query_id = q.id AND ql.require_all = 0
			)
			OR EXISTS (
				SELECT 1 FROM query_labels ql
				JOIN label_membership lm ON lm.label_id = ql.label_id AND lm.host_id = ?
				WHERE ql.query_id = q.id AND ql.require_all = 0
			)
		)
		AND (
			NOT EXISTS (
				SELECT 1 FROM query_labels ql
				WHERE ql.query_id = q.id AND ql.require_all = 1
			)
			OR (
				SELECT COUNT(*) FROM query_labels ql
				WHERE ql.query_id = q.id AND ql.require_all = 1
			) = (
				SELECT COUNT(*) FROM query_labels ql
				JOIN label_membership lm ON lm.label_id = ql.label_id AND lm.host_id = ?
				WHERE ql.query_id = q.id AND ql.require_all = 1
			)
		)`
	whereArgs = append(whereArgs, hostID, hostID)

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
			WHERE query_id = q.id AND host_id = ?
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

	// Fetch the total non-null result count per query across all hosts, used to
	// determine report_clipped.
	type totalCountRow struct {
		QueryID       uint `db:"query_id"`
		NQueryResults int  `db:"n_query_results"`
	}
	totalStmt, totalArgs, err := sqlx.In(`
		SELECT query_id, COUNT(*) AS n_query_results
		FROM query_results
		WHERE query_id IN (?) AND has_data = 1
		GROUP BY query_id
	`, queryIDs)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "building total count query for host reports")
	}
	var totalCountRows []totalCountRow
	if err := sqlx.SelectContext(ctx, dbReader, &totalCountRows, dbReader.Rebind(totalStmt), totalArgs...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "fetching total result counts for host reports")
	}
	nQueryResultsByID := make(map[uint]int, len(totalCountRows))
	for _, r := range totalCountRows {
		nQueryResultsByID[r.QueryID] = r.NQueryResults
	}

	// Fetch the host-specific result count per query, used to populate
	// NHostResults.
	type hostCountRow struct {
		QueryID      uint `db:"query_id"`
		NHostResults int  `db:"n_host_results"`
	}
	hostCountStmt, hostCountArgs, err := sqlx.In(`
		SELECT query_id, COUNT(*) AS n_host_results
		FROM query_results
		WHERE query_id IN (?) AND host_id = ? AND has_data = 1
		GROUP BY query_id
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
	firstDataStmt, firstDataArgs, err := sqlx.In(`
		SELECT query_id, data
		FROM (
			SELECT
				query_id,
				data,
				ROW_NUMBER() OVER (PARTITION BY query_id ORDER BY last_fetched DESC) AS rn
			FROM query_results
			WHERE query_id IN (?) AND host_id = ? AND has_data = 1
		) ranked
		WHERE rn = 1
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
		r.ReportClipped = nQueryResultsByID[qr.QueryID] >= maxQueryReportRows
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
