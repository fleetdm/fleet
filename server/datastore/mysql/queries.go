package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ApplyQueries(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]bool) (err error) {
	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "begin ApplyQueries transaction")
	}

	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			// It seems possible that there might be a case in
			// which the error we are dealing with here was thrown
			// by the call to tx.Commit(), and the docs suggest
			// this call would then result in sql.ErrTxDone.
			if rbErr != nil && rbErr != sql.ErrTxDone {
				panic(fmt.Sprintf("got err '%s' rolling back after err '%s'", rbErr, err))
			}
		}
	}()

	insertSql := `
		INSERT INTO queries (
			name,
			description,
			query,
			author_id,
			saved,
			observer_can_run,
			team_id,
			team_id_char,
			platform,
			min_osquery_version,
			schedule_interval,
			automations_enabled,
			logging_type,
			discard_data
		) VALUES ( ?, ?, ?, ?, true, ?, ?, ?, ?, ?, ?, ?, ?, ? )
		ON DUPLICATE KEY UPDATE
			name = VALUES(name),
			description = VALUES(description),
			query = VALUES(query),
			author_id = VALUES(author_id),
			saved = VALUES(saved),
			observer_can_run = VALUES(observer_can_run),
			team_id = VALUES(team_id),
			team_id_char = VALUES(team_id_char),
			platform = VALUES(platform),
			min_osquery_version = VALUES(min_osquery_version),
			schedule_interval = VALUES(schedule_interval),
			automations_enabled = VALUES(automations_enabled),
			logging_type = VALUES(logging_type),
			discard_data = VALUES(discard_data)
	`
	stmt, err := tx.PrepareContext(ctx, insertSql)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare ApplyQueries insert")
	}
	defer stmt.Close()

	var resultsStmt *sql.Stmt
	if len(queriesToDiscardResults) > 0 {
		resultsSql := `DELETE FROM query_results WHERE query_id = ?`
		resultsStmt, err = tx.PrepareContext(ctx, resultsSql)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare ApplyQueries delete query results")
		}
		defer resultsStmt.Close()

	}

	for _, q := range queries {
		if err := q.Verify(); err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		_, err := stmt.ExecContext(
			ctx,
			q.Name,
			q.Description,
			q.Query,
			authorID,
			q.ObserverCanRun,
			q.TeamID,
			q.TeamIDStr(),
			q.Platform,
			q.MinOsqueryVersion,
			q.Interval,
			q.AutomationsEnabled,
			q.Logging,
			q.DiscardData,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "exec ApplyQueries insert")
		}
	}

	for id := range queriesToDiscardResults {
		_, err := resultsStmt.ExecContext(
			ctx,
			id,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "exec ApplyQueries delete query results")
		}
	}

	err = tx.Commit()
	return ctxerr.Wrap(ctx, err, "commit ApplyQueries transaction")
}

func (ds *Datastore) QueryByName(
	ctx context.Context,
	teamID *uint,
	name string,
	opts ...fleet.OptionalArg,
) (*fleet.Query, error) {
	stmt := `
		SELECT 
			id,
			team_id,
			name,
			description,
			query,
			author_id,
			saved,
			observer_can_run,
			schedule_interval,
			platform,
			min_osquery_version,
			automations_enabled,
			logging_type,
			discard_data,
			created_at,
			updated_at
		FROM queries
		WHERE name = ?
	`
	args := []interface{}{name}
	whereClause := " AND team_id_char = ''"
	if teamID != nil {
		args = append(args, fmt.Sprint(*teamID))
		whereClause = " AND team_id_char = ?"
	}

	stmt += whereClause
	var query fleet.Query
	err := sqlx.GetContext(ctx, ds.reader(ctx), &query, stmt, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Query").WithName(name))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting query by name")
	}

	if err := ds.loadPacksForQueries(ctx, []*fleet.Query{&query}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading packs for query")
	}

	return &query, nil
}

// NewQuery creates a New Query.
func (ds *Datastore) NewQuery(
	ctx context.Context,
	query *fleet.Query,
	opts ...fleet.OptionalArg,
) (*fleet.Query, error) {
	sqlStatement := `
		INSERT INTO queries (
			name,
			description,
			query,
			saved,
			author_id,
			observer_can_run,
			team_id,
			team_id_char,
			platform,
			min_osquery_version,
			schedule_interval,
			automations_enabled,
			logging_type,
			discard_data
		) VALUES ( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? )
	`
	result, err := ds.writer(ctx).ExecContext(
		ctx,
		sqlStatement,
		query.Name,
		query.Description,
		query.Query,
		query.Saved,
		query.AuthorID,
		query.ObserverCanRun,
		query.TeamID,
		query.TeamIDStr(),
		query.Platform,
		query.MinOsqueryVersion,
		query.Interval,
		query.AutomationsEnabled,
		query.Logging,
		query.DiscardData,
	)

	if err != nil && isDuplicate(err) {
		return nil, ctxerr.Wrap(ctx, alreadyExists("Query", query.Name))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating new Query")
	}

	id, _ := result.LastInsertId()
	query.ID = uint(id)
	query.Packs = []fleet.Pack{}
	return query, nil
}

// SaveQuery saves changes to a Query.
func (ds *Datastore) SaveQuery(ctx context.Context, q *fleet.Query, shouldDiscardResults bool) (err error) {
	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "begin SaveQuery transaction")
	}

	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			// It seems possible that there might be a case in
			// which the error we are dealing with here was thrown
			// by the call to tx.Commit(), and the docs suggest
			// this call would then result in sql.ErrTxDone.
			if rbErr != nil && rbErr != sql.ErrTxDone {
				panic(fmt.Sprintf("got err '%s' rolling back after err '%s'", rbErr, err))
			}
		}
	}()

	updateSql := `
		UPDATE queries
		SET name                = ?,
			description         = ?,
			query               = ?,
			author_id           = ?,
			saved               = ?,
			observer_can_run    = ?,
			team_id             = ?,
			team_id_char        = ?,
			platform            = ?,
			min_osquery_version = ?,
			schedule_interval   = ?,
			automations_enabled = ?,
			logging_type        = ?,
			discard_data        = ?
		WHERE id = ?
	`

	stmt, err := tx.PrepareContext(ctx, updateSql)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare SaveQuery update")
	}
	defer stmt.Close()

	var resultsStmt *sql.Stmt
	if shouldDiscardResults {
		resultsSql := `DELETE FROM query_results WHERE query_id = ?`
		resultsStmt, err = tx.PrepareContext(ctx, resultsSql)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "prepare SaveQuery delete query results")
		}
		defer resultsStmt.Close()

	}

	_, err = stmt.ExecContext(
		ctx,
		q.Name,
		q.Description,
		q.Query,
		q.AuthorID,
		q.Saved,
		q.ObserverCanRun,
		q.TeamID,
		q.TeamIDStr(),
		q.Platform,
		q.MinOsqueryVersion,
		q.Interval,
		q.AutomationsEnabled,
		q.Logging,
		q.DiscardData,
		q.ID,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "exec SaveQuery update")
	}

	if resultsStmt != nil {
		_, err := resultsStmt.ExecContext(
			ctx,
			q.ID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "exec SaveQuery delete query results")
		}
	}

	err = tx.Commit()
	return ctxerr.Wrap(ctx, err, "commit SaveQuery transaction")
}

func (ds *Datastore) DeleteQuery(
	ctx context.Context,
	teamID *uint,
	name string,
) error {
	stmt := "DELETE FROM queries WHERE name = ?"

	args := []interface{}{name}
	whereClause := " AND team_id_char = ''"
	if teamID != nil {
		args = append(args, fmt.Sprint(*teamID))
		whereClause = " AND team_id_char = ?"
	}
	stmt += whereClause

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		if isMySQLForeignKey(err) {
			return ctxerr.Wrap(ctx, foreignKey("queries", name))
		}
		return ctxerr.Wrap(ctx, err, "delete queries")
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ctxerr.Wrap(ctx, notFound("queries").WithName(name))
	}
	return nil
}

// DeleteQueries deletes the existing query objects with the provided IDs. The
// number of deleted queries is returned along with any error.
func (ds *Datastore) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	return ds.deleteEntities(ctx, queriesTable, ids)
}

// Query returns a single Query identified by id, if such exists.
func (ds *Datastore) Query(ctx context.Context, id uint) (*fleet.Query, error) {
	sqlQuery := `
		SELECT 
			q.id,
			q.team_id,
			q.name,
			q.description,
			q.query,
			q.author_id,
			q.saved,
			q.observer_can_run,
			q.schedule_interval,
			q.platform,
			q.min_osquery_version,
			q.automations_enabled,
			q.logging_type,
			q.discard_data,
			q.created_at,
			q.updated_at,
			q.discard_data,
			COALESCE(NULLIF(u.name, ''), u.email, '') AS author_name, 
			COALESCE(u.email, '') AS author_email,
			JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
			JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
			JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
			JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
			JSON_EXTRACT(json_value, '$.total_executions') as total_executions
		FROM queries q
		LEFT JOIN users u
			ON q.author_id = u.id
		LEFT JOIN aggregated_stats ag
			ON (ag.id = q.id AND ag.global_stats = ? AND ag.type = ?)
		WHERE q.id = ?
	`
	query := &fleet.Query{}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), query, sqlQuery, false, aggregatedStatsTypeScheduledQuery, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Query").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting query")
	}

	if err := ds.loadPacksForQueries(ctx, []*fleet.Query{query}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading packs for queries")
	}

	return query, nil
}

// ListQueries returns a list of queries with sort order and results limit
// determined by passed in fleet.ListOptions
func (ds *Datastore) ListQueries(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, error) {
	sql := `
		SELECT
			q.id,
			q.team_id,
			q.name,
			q.description,
			q.query,
			q.author_id,
			q.saved,
			q.observer_can_run,
			q.schedule_interval,
			q.platform,
			q.min_osquery_version,
			q.automations_enabled,
			q.logging_type,
			q.discard_data,
			q.created_at,
			q.updated_at,
			q.discard_data,
			COALESCE(u.name, '<deleted>') AS author_name,
			COALESCE(u.email, '') AS author_email,
			JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
			JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
			JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
			JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
			JSON_EXTRACT(json_value, '$.total_executions') as total_executions
		FROM queries q
		LEFT JOIN users u ON (q.author_id = u.id)
		LEFT JOIN aggregated_stats ag ON (ag.id = q.id AND ag.global_stats = ? AND ag.type = ?)
	`

	args := []interface{}{false, aggregatedStatsTypeScheduledQuery}
	whereClauses := "WHERE saved = true"

	if opt.OnlyObserverCanRun {
		whereClauses += " AND q.observer_can_run=true"
	}

	if opt.TeamID != nil {
		args = append(args, *opt.TeamID)
		whereClauses += " AND team_id = ?"
	} else {
		whereClauses += " AND team_id IS NULL"
	}

	if opt.IsScheduled != nil {
		if *opt.IsScheduled {
			whereClauses += " AND (q.schedule_interval>0 AND q.automations_enabled=1)"
		} else {
			whereClauses += " AND (q.schedule_interval=0 OR q.automations_enabled=0)"
		}
	}

	sql += whereClauses
	sql = appendListOptionsToSQL(sql, &opt.ListOptions)

	results := []*fleet.Query{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing queries")
	}

	if err := ds.loadPacksForQueries(ctx, results); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading packs for queries")
	}

	return results, nil
}

// loadPacksForQueries loads the user packs (aka 2017 packs) associated with the provided queries.
func (ds *Datastore) loadPacksForQueries(ctx context.Context, queries []*fleet.Query) error {
	if len(queries) == 0 {
		return nil
	}

	// packs.pack_type is NULL for user created packs (aka 2017 packs).
	sql := `
		SELECT p.*, sq.query_name AS query_name
		FROM packs p
		JOIN scheduled_queries sq
			ON p.id = sq.pack_id
		WHERE query_name IN (?) AND p.pack_type IS NULL
	`

	// Used to map the results
	name_queries := map[string]*fleet.Query{}
	// Used for the IN clause
	names := []string{}
	for _, q := range queries {
		q.Packs = make([]fleet.Pack, 0)
		names = append(names, q.Name)
		name_queries[q.Name] = q
	}

	query, args, err := sqlx.In(sql, names)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query in load packs for queries")
	}

	rows := []struct {
		QueryName string `db:"query_name"`
		fleet.Pack
	}{}

	err = sqlx.SelectContext(ctx, ds.reader(ctx), &rows, query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "selecting load packs for queries")
	}

	for _, row := range rows {
		q := name_queries[row.QueryName]
		q.Packs = append(q.Packs, row.Pack)
	}

	return nil
}

func (ds *Datastore) ObserverCanRunQuery(ctx context.Context, queryID uint) (bool, error) {
	sql := `
		SELECT observer_can_run
		FROM queries
		WHERE id = ?
	`
	var observerCanRun bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &observerCanRun, sql, queryID)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "selecting observer_can_run")
	}

	return observerCanRun, nil
}

func (ds *Datastore) ListScheduledQueriesForAgents(ctx context.Context, teamID *uint) ([]*fleet.Query, error) {
	sql := `
		SELECT
			q.name,
			q.query,
			q.team_id,
			q.schedule_interval,
			q.platform,
			q.min_osquery_version,
			q.automations_enabled,
			q.logging_type
		FROM queries q
		WHERE q.saved = true 
			AND (q.schedule_interval > 0 AND q.automations_enabled = 1)
	`

	args := []interface{}{}
	if teamID != nil {
		args = append(args, *teamID)
		sql += " AND team_id = ?"
	} else {
		sql += " AND team_id IS NULL"
	}

	results := []*fleet.Query{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, sql, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list scheduled queries for agents")
	}

	return results, nil
}

func (ds *Datastore) CleanupGlobalDiscardQueryResults(ctx context.Context) error {
	deleteStmt := "DELETE FROM query_results"
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "delete all from query_result")
	}

	return nil
}
