package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host
// in a single transaction, ensuring that the number of rows for the given query
// does not exceed the maximum allowed
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow, maxQueryReportRows int) (err error) {
	if len(rows) == 0 {
		return nil
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Since we assume all rows have the same queryID, take it from the first row
		queryID := rows[0].QueryID
		hostID := rows[0].HostID

		// Count how many rows are already in the database for the given queryID
		var countExisting int
		countStmt := `SELECT COUNT(*) FROM query_results WHERE query_id = ? AND data IS NOT NULL`
		err = sqlx.GetContext(ctx, tx, &countExisting, countStmt, queryID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "counting existing query results")
		}

		if countExisting >= maxQueryReportRows {
			// do not delete any rows if we are already at the limit
			return nil
		}

		// Delete rows based on the specific queryID and hostID
		deleteStmt := `
		DELETE FROM query_results WHERE host_id = ? AND query_id = ?
	`
		result, err := tx.ExecContext(ctx, deleteStmt, hostID, queryID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting query results for host")
		}

		// Count how many rows we deleted
		countDeleted, err := result.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "fetching deleted row count")
		}

		// Calculate how many new rows can be added given the maximum limit
		netRowsAfterDeletion := countExisting - int(countDeleted)
		allowedNewRows := maxQueryReportRows - netRowsAfterDeletion
		if allowedNewRows == 0 {
			return nil
		}

		if len(rows) > allowedNewRows {
			rows = rows[:allowedNewRows]
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
