package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

const (
	statsScheduledQueryType = iota
	statsLiveQueryType
)

func (ds *Datastore) ApplyQueries(ctx context.Context, authorID uint, queries []*fleet.Query, queriesToDiscardResults map[uint]struct{}) error {
	if err := ds.applyQueriesInTx(ctx, authorID, queries); err != nil {
		return ctxerr.Wrap(ctx, err, "apply queries in tx")
	}

	// Opportunistically delete associated query_results.
	//
	// TODO(lucas): We should run this on a transaction but we found
	// performance issues and deadlocks at scale.
	queryIDs := make([]uint, 0, len(queriesToDiscardResults))
	for queryID := range queriesToDiscardResults {
		queryIDs = append(queryIDs, queryID)
	}
	if err := ds.deleteMultipleQueryResults(ctx, queryIDs); err != nil {
		return ctxerr.Wrap(ctx, err, "delete query_results")
	}
	return nil
}

func (ds *Datastore) applyQueriesInTx(ctx context.Context, authorID uint, queries []*fleet.Query) (err error) {
	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "begin applyQueriesInTx")
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
		return ctxerr.Wrap(ctx, err, "prepare queries insert")
	}
	defer stmt.Close()

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
			return ctxerr.Wrap(ctx, err, "exec queries insert")
		}
	}

	err = tx.Commit()
	return ctxerr.Wrap(ctx, err, "commit queries tx")
}

func (ds *Datastore) deleteMultipleQueryResults(ctx context.Context, queryIDs []uint) (err error) {
	if len(queryIDs) == 0 {
		return nil
	}

	deleteQueryResultsStmt := `DELETE FROM query_results WHERE query_id IN (?)`
	query, args, err := sqlx.In(deleteQueryResultsStmt, queryIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building delete query_results stmt")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, query, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "executing delete query_results")
	}
	return nil
}

func (ds *Datastore) QueryByName(
	ctx context.Context,
	teamID *uint,
	name string,
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

func (ds *Datastore) NewQuery(
	ctx context.Context,
	query *fleet.Query,
	opts ...fleet.OptionalArg,
) (*fleet.Query, error) {
	if err := query.Verify(); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
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

	if err != nil && IsDuplicate(err) {
		return nil, ctxerr.Wrap(ctx, alreadyExists("Query", query.Name))
	} else if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating new Query")
	}

	id, _ := result.LastInsertId()
	query.ID = uint(id) //nolint:gosec // dismiss G115
	query.Packs = []fleet.Pack{}
	return query, nil
}

func (ds *Datastore) SaveQuery(ctx context.Context, q *fleet.Query, shouldDiscardResults bool, shouldDeleteStats bool) (err error) {
	if err := q.Verify(); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	updateSQL := `
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
			discard_data		= ?
		WHERE id = ?
	`
	result, err := ds.writer(ctx).ExecContext(
		ctx,
		updateSQL,
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
		q.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating query")
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "rows affected updating query")
	}
	if rows == 0 {
		return ctxerr.Wrap(ctx, notFound("Query").WithID(q.ID))
	}

	if shouldDeleteStats {
		// Delete any associated stats asynchronously.
		go ds.deleteQueryStats(context.WithoutCancel(ctx), []uint{q.ID})
	}

	// Opportunistically delete associated query_results.
	//
	// TODO(lucas): We should run this on a transaction but we found
	// performance issues and deadlocks at scale.
	if shouldDiscardResults {
		if err := ds.deleteQueryResults(ctx, q.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting query_results")
		}
	}

	return nil
}

func (ds *Datastore) deleteQueryResults(ctx context.Context, queryID uint) error {
	resultsSQL := `DELETE FROM query_results WHERE query_id = ?`
	if _, err := ds.writer(ctx).ExecContext(ctx, resultsSQL, queryID); err != nil {
		return ctxerr.Wrap(ctx, err, "executing delete query_results")
	}
	return nil
}

func (ds *Datastore) DeleteQuery(ctx context.Context, teamID *uint, name string) error {
	selectStmt := "SELECT id FROM queries WHERE name = ?"
	args := []interface{}{name}
	whereClause := " AND team_id_char = ''"
	if teamID != nil {
		args = append(args, fmt.Sprint(*teamID))
		whereClause = " AND team_id_char = ?"
	}
	selectStmt += whereClause
	var queryID uint
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &queryID, selectStmt, args...); err != nil {
		if err == sql.ErrNoRows {
			return ctxerr.Wrap(ctx, notFound("queries").WithName(name))
		}
		return ctxerr.Wrap(ctx, err, "getting query to delete")
	}

	deleteStmt := "DELETE FROM queries WHERE id = ?"
	result, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, queryID)
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

	// Delete any associated stats asynchronously.
	go ds.deleteQueryStats(context.WithoutCancel(ctx), []uint{queryID})

	// Opportunistically delete associated query_results.
	//
	// TODO(lucas): We should run this on a transaction but we found
	// performance issues and deadlocks at scale.
	if err := ds.deleteQueryResults(ctx, queryID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting query_results")
	}

	return nil
}

// DeleteQueries deletes the existing query objects with the provided IDs. The
// number of deleted queries is returned along with any error.
func (ds *Datastore) DeleteQueries(ctx context.Context, ids []uint) (uint, error) {
	deleted, err := ds.deleteEntities(ctx, queriesTable, ids)
	if err != nil {
		return deleted, err
	}

	// Delete any associated stats asynchronously.
	go ds.deleteQueryStats(context.WithoutCancel(ctx), ids)

	// Opportunistically delete associated query_results.
	//
	// TODO(lucas): We should run this on a transaction but we found
	// performance issues and deadlocks at scale.
	if err := ds.deleteMultipleQueryResults(ctx, ids); err != nil {
		return deleted, ctxerr.Wrap(ctx, err, "delete multiple query_results")
	}
	return deleted, nil
}

// deleteQueryStats deletes query stats and aggregated stats for saved queries.
// Errors are logged and not returned.
func (ds *Datastore) deleteQueryStats(ctx context.Context, queryIDs []uint) {
	// Delete stats for each host.
	stmt := "DELETE FROM scheduled_query_stats WHERE scheduled_query_id IN (?)"
	stmt, args, err := sqlx.In(stmt, queryIDs)
	if err != nil {
		level.Error(ds.logger).Log("msg", "error creating delete query stats statement", "err", err)
	} else {
		_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
		if err != nil {
			level.Error(ds.logger).Log("msg", "error deleting query stats", "err", err)
		}
	}

	// Delete aggregated stats
	stmt = fmt.Sprintf("DELETE FROM aggregated_stats WHERE type = '%s' AND id IN (?)", fleet.AggregatedStatsTypeScheduledQuery)
	stmt, args, err = sqlx.In(stmt, queryIDs)
	if err != nil {
		level.Error(ds.logger).Log("msg", "error creating delete aggregated stats statement", "err", err)
	} else {
		_, err = ds.writer(ctx).ExecContext(ctx, stmt, args...)
		if err != nil {
			level.Error(ds.logger).Log("msg", "error deleting aggregated stats", "err", err)
		}
	}
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
	if err := sqlx.GetContext(ctx, ds.reader(ctx), query, sqlQuery, false, fleet.AggregatedStatsTypeScheduledQuery, id); err != nil {
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

	args := []interface{}{false, fleet.AggregatedStatsTypeScheduledQuery}
	whereClauses := "WHERE saved = true"

	switch {
	case opt.TeamID != nil && opt.MergeInherited:
		args = append(args, *opt.TeamID)
		whereClauses += " AND (team_id = ? OR team_id IS NULL)"
	case opt.TeamID != nil:
		args = append(args, *opt.TeamID)
		whereClauses += " AND team_id = ?"
	default:
		whereClauses += " AND team_id IS NULL"
	}

	if opt.IsScheduled != nil {
		if *opt.IsScheduled {
			whereClauses += " AND (q.schedule_interval>0 AND q.automations_enabled=1)"
		} else {
			whereClauses += " AND (q.schedule_interval=0 OR q.automations_enabled=0)"
		}
	}

	if opt.MatchQuery != "" {
		whereClauses += " AND q.name = ?"
		args = append(args, opt.MatchQuery)
	}

	sql += whereClauses
	sql, args = appendListOptionsWithCursorToSQL(sql, args, &opt.ListOptions)

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

func (ds *Datastore) ListScheduledQueriesForAgents(ctx context.Context, teamID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
	sqlStmt := `
		SELECT
			q.name,
			q.query,
			q.team_id,
			q.schedule_interval,
			q.platform,
			q.min_osquery_version,
			q.automations_enabled,
			q.logging_type,
			q.discard_data
		FROM queries q
		WHERE q.saved = true 
			AND (
				q.schedule_interval > 0 AND
				%s AND
				(
					q.automations_enabled
					OR
					(NOT q.discard_data AND NOT ? AND q.logging_type = ?)
				)
			)
	`

	args := []interface{}{}
	teamSQL := " team_id IS NULL"
	if teamID != nil {
		args = append(args, *teamID)
		teamSQL = " team_id = ?"
	}
	sqlStmt = fmt.Sprintf(sqlStmt, teamSQL)
	args = append(args, queryReportsDisabled, fleet.LoggingSnapshot)

	results := []*fleet.Query{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, sqlStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list scheduled queries for agents")
	}

	return results, nil
}

func (ds *Datastore) CleanupGlobalDiscardQueryResults(ctx context.Context) error {
	deleteStmt := "DELETE FROM query_results"
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "delete all from query_results")
	}

	return nil
}

// IsSavedQuery returns true if the given query is a saved query.
func (ds *Datastore) IsSavedQuery(ctx context.Context, queryID uint) (bool, error) {
	stmt := `
		SELECT saved
		FROM queries
		WHERE id = ?
	`
	var result bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, queryID)
	return result, err
}

// GetLiveQueryStats returns the live query stats for the given query and hosts.
func (ds *Datastore) GetLiveQueryStats(ctx context.Context, queryID uint, hostIDs []uint) ([]*fleet.LiveQueryStats, error) {
	stmt, args, err := sqlx.In(
		`SELECT host_id, average_memory, executions, system_time, user_time, wall_time, output_size, last_executed
		FROM scheduled_query_stats
		WHERE host_id IN (?) AND scheduled_query_id = ? AND query_type = ?
	`, hostIDs, queryID, statsLiveQueryType,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building get live query stats stmt")
	}

	results := []*fleet.LiveQueryStats{}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get live query stats")
	}
	return results, nil
}

// UpdateLiveQueryStats writes new stats as a batch
func (ds *Datastore) UpdateLiveQueryStats(ctx context.Context, queryID uint, stats []*fleet.LiveQueryStats) error {
	if len(stats) == 0 {
		return nil
	}

	// Bulk insert/update
	const valueStr = "(?,?,?,?,?,?,?,?,?,?,?,?),"
	stmt := "REPLACE INTO scheduled_query_stats (scheduled_query_id, host_id, query_type, executions, average_memory, system_time, user_time, wall_time, output_size, denylisted, schedule_interval, last_executed) VALUES " +
		strings.Repeat(valueStr, len(stats))
	stmt = strings.TrimSuffix(stmt, ",")

	var args []interface{}
	for _, s := range stats {
		args = append(
			args, queryID, s.HostID, statsLiveQueryType, s.Executions, s.AverageMemory, s.SystemTime, s.UserTime, s.WallTime, s.OutputSize,
			0, 0, s.LastExecuted,
		)
	}
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "update live query stats")
	}
	return nil
}

func numSavedQueriesDB(ctx context.Context, db sqlx.QueryerContext) (int, error) {
	var count int
	const stmt = `
		SELECT count(*) FROM queries WHERE saved
  	`
	if err := sqlx.GetContext(ctx, db, &count, stmt); err != nil {
		return 0, err
	}

	return count, nil
}
