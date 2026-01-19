package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host.
// It deletes existing rows for the host/query and inserts the new rows.
// If the incoming result set has more than 1000 rows, it bails early without storing anything.
// Excess rows across all hosts are cleaned up by a separate cron job.
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) (err error) {
	if len(rows) == 0 {
		return nil
	}

	// Bail early if the incoming result set is too large (more than 1000 rows from a single host)
	if len(rows) > fleet.DefaultMaxQueryReportRows {
		return nil
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Since we assume all rows have the same queryID, take it from the first row
		queryID := rows[0].QueryID
		hostID := rows[0].HostID

		// Delete rows based on the specific queryID and hostID
		deleteStmt := `DELETE FROM query_results WHERE host_id = ? AND query_id = ?`
		_, err := tx.ExecContext(ctx, deleteStmt, hostID, queryID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting query results for host")
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

		_, err = tx.ExecContext(ctx, insertStmt, valueArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting new rows")
		}

		return nil
	})

	return ctxerr.Wrap(ctx, err, "overwriting query result rows")
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
			WHERE query_id = ? AND data IS NOT NULL AND %s
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
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND data IS NOT NULL`, queryID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query")
	}

	return count, nil
}

// ResultCountForQueryAndHost counts the query report rows for a given query and host
// excluding rows with null data
func (ds *Datastore) ResultCountForQueryAndHost(ctx context.Context, queryID, hostID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND host_id = ? AND data IS NOT NULL`, queryID, hostID)
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
// allowed per query. It keeps the most recent rows (by last_fetched) up to the limit.
// This runs as a cron job to ensure the query_results table doesn't grow unbounded.
func (ds *Datastore) CleanupExcessQueryResultRows(ctx context.Context, maxQueryReportRows int) error {
	// Get all distinct query_ids that have results and are scheduled queries with discard_data = false
	var queryIDs []uint
	selectStmt := `
		SELECT DISTINCT qr.query_id
		FROM query_results qr
		INNER JOIN queries q ON qr.query_id = q.id
		WHERE q.discard_data = false AND qr.data IS NOT NULL
	`
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &queryIDs, selectStmt); err != nil {
		return ctxerr.Wrap(ctx, err, "selecting query IDs for cleanup")
	}

	// For each query, delete rows that exceed the limit
	for _, queryID := range queryIDs {
		// Delete all rows older than the Nth most recent row (where N = maxQueryReportRows)
		// The subquery finds the id of the row at position maxQueryReportRows when ordered by last_fetched DESC.
		// All rows with id < that id are deleted.
		deleteStmt := `
			DELETE FROM query_results
			WHERE query_id = ?
			AND data IS NOT NULL
			AND id < (
				SELECT id FROM (
					SELECT id
					FROM query_results
					WHERE query_id = ?
					AND data IS NOT NULL
					ORDER BY last_fetched DESC
					LIMIT 1 OFFSET ?
				) AS t
			)
		`
		if _, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, queryID, queryID, maxQueryReportRows); err != nil {
			return ctxerr.Wrapf(ctx, err, "cleaning up excess query results for query %d", queryID)
		}
	}

	return nil
}
