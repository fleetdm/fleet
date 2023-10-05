package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	// QueryResultRowLimit is the maximum number of rows that can be stored per query
	QueryResultRowLimit = 1000
)

// SaveQueryResultRow saves a query result row to the datastore and returns number of rows inserted
func (ds *Datastore) SaveQueryResultRows(ctx context.Context, rows []*fleet.ScheduledQueryResultRow) error {
	if len(rows) == 0 {
		return nil // Nothing to insert
	}

	valueStrings := make([]string, 0, len(rows))
	valueArgs := make([]interface{}, 0, len(rows)*4)

	for _, row := range rows {
		valueStrings = append(valueStrings, "(?, ?, ?, ?)")
		valueArgs = append(valueArgs, row.QueryID, row.HostID, row.LastFetched, row.Data)
	}

	insertStmt := fmt.Sprintf(`
        INSERT INTO query_results (query_id, host_id, last_fetched, data)
            VALUES %s
    `, strings.Join(valueStrings, ","))

	_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, valueArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "saving Query Result Rows")
	}

	return nil
}

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

// TODO(lucas): If we just use this for testing then we can remove it and use ExecAdhocSQL.
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

func (ds *Datastore) DeleteQueryResultsForHost(ctx context.Context, hostID, queryID uint) error {
	deleteStmt := `
		DELETE FROM query_results WHERE host_id = ? AND query_id = ?
		`
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, hostID, queryID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting query results for host")
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
