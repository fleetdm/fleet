package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// ListScheduledQueriesInPackWithStats loads a pack's scheduled queries and its aggregated stats.
func (ds *Datastore) ListScheduledQueriesInPackWithStats(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
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
			JSON_EXTRACT(ag.json_value, '$.user_time_p50') as user_time_p50,
			JSON_EXTRACT(ag.json_value, '$.user_time_p95') as user_time_p95,
			JSON_EXTRACT(ag.json_value, '$.system_time_p50') as system_time_p50,
			JSON_EXTRACT(ag.json_value, '$.system_time_p95') as system_time_p95,
			JSON_EXTRACT(ag.json_value, '$.total_executions') as total_executions
		FROM scheduled_queries sq
		JOIN queries q ON (sq.query_name = q.name)
		LEFT JOIN aggregated_stats ag ON (ag.id=sq.id AND ag.type='scheduled_query')
		WHERE sq.pack_id = ?
	`
	query = appendListOptionsToSQL(query, opts)
	results := []*fleet.ScheduledQuery{}

	if err := sqlx.SelectContext(ctx, ds.reader, &results, query, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing scheduled queries")
	}

	return results, nil
}

// ListScheduledQueriesInPack lists all the scheduled queries of a pack.
func (ds *Datastore) ListScheduledQueriesInPack(ctx context.Context, id uint) ([]*fleet.ScheduledQuery, error) {
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
			q.id AS query_id
		FROM scheduled_queries sq
		JOIN queries q ON (sq.query_name = q.name)
		WHERE sq.pack_id = ?
	`
	results := []*fleet.ScheduledQuery{}
	if err := sqlx.SelectContext(ctx, ds.reader, &results, query, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing scheduled queries")
	}

	return results, nil
}

func (ds *Datastore) NewScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery, opts ...fleet.OptionalArg) (*fleet.ScheduledQuery, error) {
	return insertScheduledQueryDB(ctx, ds.writer, sq)
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

func (ds *Datastore) SaveScheduledQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	return saveScheduledQueryDB(ctx, ds.writer, sq)
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

func (ds *Datastore) DeleteScheduledQuery(ctx context.Context, id uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM scheduled_queries WHERE id = ?`, id)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete scheduled_queries")
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete scheduled_queries: rows affeted")
		}
		if rowsAffected == 0 {
			return ctxerr.Wrap(ctx, notFound("ScheduledQuery").WithID(id))
		}
		_, err = tx.ExecContext(ctx, `DELETE FROM scheduled_query_stats WHERE scheduled_query_id = ?`, id)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "delete scheduled_queries_stats")
		}
		return nil
	})
}

func (ds *Datastore) ScheduledQuery(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
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
	if err := sqlx.GetContext(ctx, ds.reader, sq, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("ScheduledQuery").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "select scheduled query")
	}

	return sq, nil
}

func (ds *Datastore) ScheduledQueryIDsByName(ctx context.Context, batchSize int, packAndSchedQueryNames ...[2]string) ([]uint, error) {
	const (
		stmt = `
    SELECT sqn.idx, sq.id
      FROM scheduled_queries sq
      INNER JOIN packs p ON sq.pack_id = p.id
      INNER JOIN (
        SELECT ? as idx, ? as pack_name, ? as scheduled_query_name
        %s
      ) AS sqn ON (p.name, sq.name) = (sqn.pack_name, sqn.scheduled_query_name)
`
		additionalRows = `UNION SELECT ?, ?, ? `
	)

	type idxAndID struct {
		IDX int  `db:"idx"`
		ID  uint `db:"id"`
	}

	if batchSize <= 0 {
		batchSize = fleet.DefaultScheduledQueryIDsByNameBatchSize
	}

	// all provided names have a corresponding scheduled query ID in the result,
	// even if it doesn't exist for some reason (in which case it will be 0).
	result := make([]uint, len(packAndSchedQueryNames))

	var indexOffset int
	for len(packAndSchedQueryNames) > 0 {
		max := len(packAndSchedQueryNames)
		if max > batchSize {
			max = batchSize
		}

		args := make([]interface{}, 0, max*3)
		for i, psn := range packAndSchedQueryNames[:max] {
			args = append(args, indexOffset+i, psn[0], psn[1])
		}
		packAndSchedQueryNames = packAndSchedQueryNames[max:]
		indexOffset += max

		stmt := fmt.Sprintf(stmt, strings.Repeat(additionalRows, max-1))
		var rows []idxAndID
		if err := ds.writer.SelectContext(ctx, &rows, stmt, args...); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "select scheduled query IDs by name")
		}
		for _, row := range rows {
			result[row.IDX] = row.ID
		}
	}
	return result, nil
}

func (ds *Datastore) AsyncBatchSaveHostsScheduledQueryStats(ctx context.Context, stats map[uint][]fleet.ScheduledQueryStats, batchSize int) (int, error) {
	// NOTE: this implementation must be kept in sync with the non-async version
	// in SaveHostPackStats (in hosts.go) - that is, the behaviour per host must
	// be the same.

	const (
		stmt = `
			INSERT INTO scheduled_query_stats (
				host_id,
				scheduled_query_id,
				average_memory,
				denylisted,
				executions,
				schedule_interval,
				last_executed,
				output_size,
				system_time,
				user_time,
				wall_time
			)
			VALUES %s
      ON DUPLICATE KEY UPDATE
				average_memory = VALUES(average_memory),
				denylisted = VALUES(denylisted),
				executions = VALUES(executions),
				schedule_interval = VALUES(schedule_interval),
				last_executed = VALUES(last_executed),
				output_size = VALUES(output_size),
				system_time = VALUES(system_time),
				user_time = VALUES(user_time),
				wall_time = VALUES(wall_time)
`

		values = `(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?),`
	)

	var countExecs int

	// inserting sorted by host id (the first key in the PK) apparently helps
	// mysql with locking
	hostIDs := make([]uint, 0, len(stats))
	for k := range stats {
		hostIDs = append(hostIDs, k)
	}
	sort.Slice(hostIDs, func(i, j int) bool {
		return hostIDs[i] < hostIDs[j]
	})

	var batchArgs []interface{}
	var batchCount int
	for _, hostID := range hostIDs {
		hostStats := stats[hostID]

		for _, stat := range hostStats {
			batchArgs = append(batchArgs,
				hostID,
				stat.ScheduledQueryID,
				stat.AverageMemory,
				stat.Denylisted,
				stat.Executions,
				stat.Interval,
				stat.LastExecuted,
				stat.OutputSize,
				stat.SystemTime,
				stat.UserTime,
				stat.WallTime)
			batchCount++

			if batchCount >= batchSize {
				stmt := fmt.Sprintf(stmt, strings.TrimSuffix(strings.Repeat(values, batchCount), ","))
				if _, err := ds.writer.ExecContext(ctx, stmt, batchArgs...); err != nil {
					return countExecs, ctxerr.Wrap(ctx, err, "insert batch of scheduled query stats")
				}
				countExecs++
				batchArgs = batchArgs[:0]
				batchCount = 0
			}
		}
	}

	if batchCount > 0 {
		stmt := fmt.Sprintf(stmt, strings.TrimSuffix(strings.Repeat(values, batchCount), ","))
		if _, err := ds.writer.ExecContext(ctx, stmt, batchArgs...); err != nil {
			return countExecs, ctxerr.Wrap(ctx, err, "insert batch of scheduled query stats")
		}
		countExecs++
	}

	return countExecs, nil
}
