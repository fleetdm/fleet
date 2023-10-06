package mysql

import (
	"context"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// OverwriteQueryResultRows overwrites the query result rows for a given query and host
// in a single transaction
func (ds *Datastore) OverwriteQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow) error {
	if len(rows) == 0 {
		return nil // Nothing to overwrite
	}

	// Start a transaction
	tx, err := ds.writer(ctx).BeginTx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "starting a transaction")
	}
	defer func() {
		err := tx.Rollback()
		if err != nil {
			ds.logger.Log("err", err, "msg", "rolling back transaction")
		}
	}()

	// Since we assume all rows have the same queryID and hostID, take it from the first row
	queryID := rows[0].QueryID
	hostID := rows[0].HostID

	// Delete existing rows based on the common query_id and host_id
	deleteStmt := `
		DELETE FROM query_results WHERE host_id = ? AND query_id = ?
	`
	_, err = tx.ExecContext(ctx, deleteStmt, hostID, queryID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting query results for host")
	}

	// Prepare data for insertion
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
		return ctxerr.Wrap(ctx, err, "saving Query Result Rows")
	}

	// Commit the transaction
	err = tx.Commit()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "committing the transaction")
	}

	return nil
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
