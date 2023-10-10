package mysql

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host
// in a single transaction, ensuring that the number of rows for the given query
// does not exceed the maximum allowed
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow) (err error) {
	if len(rows) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := ds.writer(ctx).BeginTx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "starting a transaction")
	}

	// Since we assume all rows have the same queryID, take it from the first row
	queryID := rows[0].QueryID
	hostID := rows[0].HostID

	defer func() {
		if err != nil {
			err := tx.Rollback()
			if err != nil {
				ds.logger.Log("err", err, "msg", "rolling back transaction", "query_id", queryID, "host_id", hostID)
			}
		}
	}()

	// Count how many rows are already in the database for the given queryID
	var countExisting int
	countStmt := `
		SELECT COUNT(*) FROM query_results WHERE query_id = ?
	`
	err = tx.QueryRowContext(ctx, countStmt, queryID).Scan(&countExisting)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "counting existing query results")
	}

	if countExisting == fleet.MaxQueryReportRows {
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
	allowedNewRows := fleet.MaxQueryReportRows - netRowsAfterDeletion
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
		INSERT INTO query_results (query_id, host_id, last_fetched, data) VALUES
	` + strings.Join(valueStrings, ",")

	_, err = tx.ExecContext(ctx, insertStmt, valueArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting new rows")
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "committing the transaction")
	}

	return nil
}

// TODO(lucas): Any chance we can store hostname in the query_results table?
// (to avoid having to left join hosts).
func (ds *Datastore) QueryResultRows(ctx context.Context, queryID uint) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := `
		SELECT qr.query_id, qr.host_id, h.hostname, qr.last_fetched, qr.data
			FROM query_results qr
			LEFT JOIN hosts h ON (qr.host_id=h.id)
			WHERE query_id = ?
		`
	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting query result rows")
	}

	return results, nil
}

func (ds *Datastore) ResultCountForQuery(ctx context.Context, queryID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `select count(*) from query_results where query_id = ?`, queryID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query")
	}

	return count, nil
}

func (ds *Datastore) ResultCountForQueryAndHost(ctx context.Context, queryID, hostID uint) (int, error) {
	var count int
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, `select count(*) from query_results where query_id = ? AND host_id = ?`, queryID, hostID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "counting query results for query and host")
	}

	return count, nil
}
