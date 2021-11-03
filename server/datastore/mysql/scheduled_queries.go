package mysql

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (d *Datastore) ListScheduledQueriesInPack(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	query := `
		SELECT
			sq.id,
			sq.pack_id,
			sq.name,
			sq.query_name,
			sq.description,
			sq.interval,
			sq.snapshot,
			sq.removed,
			sq.platform,
			sq.version,
			sq.shard,
			sq.denylist,
			q.query,
			q.id AS query_id,
			JSON_EXTRACT(json_value, "$.user_time_p50") as user_time_p50,
			JSON_EXTRACT(json_value, "$.user_time_p95") as user_time_p95,
			JSON_EXTRACT(json_value, "$.system_time_p50") as system_time_p50,
			JSON_EXTRACT(json_value, "$.system_time_p95") as system_time_p95,
			JSON_EXTRACT(json_value, "$.total_executions") as total_executions
		FROM scheduled_queries sq
		JOIN queries q ON (sq.query_name = q.name)
		LEFT JOIN aggregated_stats ag ON (ag.id=sq.id AND ag.type="scheduled_query")
		WHERE sq.pack_id = ?
	`
	query = appendListOptionsToSQL(query, opts)
	results := []*fleet.ScheduledQuery{}

	if err := sqlx.SelectContext(ctx, d.reader, &results, query, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing scheduled queries")
	}

	return results, nil
}

func (d *Datastore) NewScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
	return insertScheduledQueryDB(ctx, d.writer, sq)
}

func insertScheduledQueryDB(ctx context.Context, q sqlx.ExtContext, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	// This query looks up the query name using the ID (for backwards
	// compatibility with the UI)
	query := `
		INSERT INTO scheduled_queries (
			query_name,
			query_id,
			name,
			pack_id,
			snapshot,
			removed,
			` + "`interval`" + `,
			platform,
			version,
			shard,
			denylist
		)
		SELECT name, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		FROM queries
		WHERE id = ?
		`
	result, err := q.ExecContext(ctx, query, sq.QueryID, sq.Name, sq.PackID, sq.Snapshot, sq.Removed, sq.Interval, sq.Platform, sq.Version, sq.Shard, sq.Denylist, sq.QueryID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert scheduled query")
	}

	id, _ := result.LastInsertId()
	sq.ID = uint(id)

	query = `SELECT query, name FROM queries WHERE id = ? LIMIT 1`
	metadata := []struct {
		Query string
		Name  string
	}{}

	err = sqlx.SelectContext(ctx, q, &metadata, query, sq.QueryID)
	if err != nil && err == sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, notFound("Query").WithID(sq.QueryID))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select query by ID")
	}

	if len(metadata) != 1 {
		return nil, ctxerr.Wrap(ctx, err, "wrong number of results returned from database")
	}

	sq.Query = metadata[0].Query
	sq.QueryName = metadata[0].Name

	return sq, nil
}

func (d *Datastore) SaveScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	return saveScheduledQueryDB(ctx, d.writer, sq)
}

func saveScheduledQueryDB(ctx context.Context, exec sqlx.ExecerContext, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	query := `
		UPDATE scheduled_queries
			SET pack_id = ?, query_id = ?, ` + "`interval`" + ` = ?, snapshot = ?, removed = ?, platform = ?, version = ?, shard = ?, denylist = ?
			WHERE id = ?
	`
	result, err := exec.ExecContext(ctx, query, sq.PackID, sq.QueryID, sq.Interval, sq.Snapshot, sq.Removed, sq.Platform, sq.Version, sq.Shard, sq.Denylist, sq.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving a scheduled query")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "rows affected saving a scheduled query")
	}
	if rows == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("ScheduledQueries").WithID(sq.ID))
	}
	return sq, nil
}

func (d *Datastore) DeleteScheduledQuery(ctx context.Context, id uint) error {
	return d.deleteEntity(ctx, scheduledQueriesTable, id)
}

func (d *Datastore) ScheduledQuery(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
	query := `
		SELECT
			sq.id,
			sq.created_at,
			sq.updated_at,
			sq.pack_id,
			sq.interval,
			sq.snapshot,
			sq.removed,
			sq.platform,
			sq.version,
			sq.shard,
			sq.query_name,
			sq.description,
			sq.denylist,
			q.query,
			q.name,
			q.id AS query_id
		FROM scheduled_queries sq
		JOIN queries q
		ON sq.query_name = q.name
		WHERE sq.id = ?
	`
	sq := &fleet.ScheduledQuery{}
	if err := sqlx.GetContext(ctx, d.reader, sq, query, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select scheduled query")
	}

	return sq, nil
}

func (d *Datastore) CleanupOrphanScheduledQueryStats(ctx context.Context) error {
	_, err := d.writer.ExecContext(ctx, `DELETE FROM scheduled_query_stats where scheduled_query_id not in (select id from scheduled_queries where id=scheduled_query_id)`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning orphan scheduled_query_stats by scheduled_query")
	}
	_, err = d.writer.ExecContext(ctx, `DELETE FROM scheduled_query_stats where host_id not in (select id from hosts where id=host_id)`)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning orphan scheduled_query_stats by host")
	}
	return nil
}
