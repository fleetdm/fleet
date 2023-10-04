package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

const (
	// QueryResultRowLimit is the maximum number of rows that can be stored per query
	QueryResultRowLimit = 1000
)

func (ds *Datastore) SaveQueryResultRow(ctx context.Context, row *fleet.ScheduledQueryResultRow) (*fleet.ScheduledQueryResultRow, error) {
	insertStmt := `
		INSERT INTO query_results (query_id, host_id, last_fetched, data)
			VALUES (?, ?, ?, ?)
		`
	_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, row.QueryID, row.HostID, row.LastFetched, row.Data)
	if err != nil && isDuplicate(err) {
		return nil, ctxerr.Wrap(ctx, alreadyExists("Query Result Row", row.QueryID+row.HostID))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving Query Result Row")
	}

	return row, nil
}

func (ds *Datastore) QueryResultRows(ctx context.Context, queryID, hostID uint) ([]*fleet.ScheduledQueryResultRow, error) {
	selectStmt := `
		SELECT query_id, host_id, last_fetched, data FROM query_results
			WHERE query_id = ? AND host_id = ?
		`
	results := []*fleet.ScheduledQueryResultRow{}
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, selectStmt, queryID, hostID)
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
		return 0, ctxerr.Wrap(ctx, err, "counting query results")
	}

	return count, nil
}
