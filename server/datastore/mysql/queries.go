package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/text/unicode/norm"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log/level"
	"github.com/jmoiron/sqlx"
)

const (
	statsScheduledQueryType = iota
	statsLiveQueryType
)

var querySearchColumns = []string{"q.name"}

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

func (ds *Datastore) applyQueriesInTx(
	ctx context.Context,
	authorID uint,
	queries []*fleet.Query,
) (err error) {
	// First, verify all 'queries' are valid.
	for _, q := range queries {
		if err := q.Verify(); err != nil {
			return ctxerr.Wrap(ctx, err)
		}
	}

	const upsertQueriesSQL = `
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
		) VALUES %s
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
			discard_data = VALUES(discard_data)`

	// 'queries' are uniquely identified by {name, team_id}
	unqKeyGen := func(name string, teamID *uint) string {
		if teamID == nil {
			return fmt.Sprintf(":%s", name)
		}
		return fmt.Sprintf("%d:%s", *teamID, name)
	}

	batchSize := 50
	for i := 0; i < len(queries); i += batchSize {
		end := i + batchSize
		if end > len(queries) {
			end = len(queries)
		}
		batch := queries[i:end]

		// Group queries by their 'key' to make lookups more efficient.
		batchGrp := make(map[string]*fleet.Query, len(batch))
		for _, q := range batch {
			batchGrp[unqKeyGen(q.Name, q.TeamID)] = q
		}

		if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
			// For upserting
			pToInsert := make([]string, 0, len(batch))
			aToInsert := make([]interface{}, 0, len(batch)*13)

			// For fetching the ID after the upsert
			pToSelect := make([]string, 0, len(batch))
			aToSelect := make([]interface{}, 0, len(batch)*2)

			for _, q := range batch {
				pToInsert = append(pToInsert, "( ?, ?, ?, ?, true, ?, ?, ?, ?, ?, ?, ?, ?, ? )")
				aToInsert = append(aToInsert, q.Name, q.Description, q.Query, authorID, q.ObserverCanRun, q.TeamID,
					q.TeamIDStr(), q.Platform, q.MinOsqueryVersion, q.Interval, q.AutomationsEnabled, q.Logging,
					q.DiscardData)

				pToSelect = append(pToSelect, "(name = ? AND team_id_char = ?)")
				aToSelect = append(aToSelect, q.Name, q.TeamIDStr())
			}

			upsertStm := fmt.Sprintf(upsertQueriesSQL, strings.Join(pToInsert, ","))
			if _, err = tx.ExecContext(ctx, upsertStm, aToInsert...); err != nil {
				return ctxerr.Wrap(ctx, err, "bulk upserting queries")
			}

			selectStm := fmt.Sprintf(
				`SELECT id, name, team_id FROM queries WHERE %s`,
				strings.Join(pToSelect, " OR "),
			)
			rows, err := tx.QueryContext(ctx, selectStm, aToSelect...)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "select queries for update")
			}
			defer rows.Close()
			for rows.Next() {
				var id uint
				var name string
				var teamID *uint
				if err := rows.Scan(&id, &name, &teamID); err != nil {
					return ctxerr.Wrap(ctx, err, "scan existing query")
				}
				if q, ok := batchGrp[unqKeyGen(name, teamID)]; ok {
					q.ID = id
				}
			}
			if err := rows.Err(); err != nil {
				return ctxerr.Wrap(ctx, err, "fetching query IDs")
			}
			if err := rows.Close(); err != nil { //nolint:sqlclosecheck
				return ctxerr.Wrap(ctx, err, "closing query rows")
			}

			return ds.updateQueryLabelsInTx(ctx, batch, tx)
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "updating query labels")
		}
	}

	return nil
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
	queryStatement := `
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
		queryStatement,
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

	if err := ds.updateQueryLabels(ctx, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving labels for query")
	}

	return query, nil
}

func (ds *Datastore) updateQueryLabels(ctx context.Context, query *fleet.Query) error {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return ds.updateQueryLabelsInTx(ctx, []*fleet.Query{query}, tx)
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating query labels")
	}
	return nil
}

// updateQueryLabelsInTx updates the LabelsIncludeAny for a set of queries, using the string value of
// the label. Labels IDs are populated
func (ds *Datastore) updateQueryLabelsInTx(ctx context.Context, queries []*fleet.Query, tx sqlx.ExtContext) error {
	if tx == nil {
		return ctxerr.New(ctx, "updateQueryLabelsInTx called with nil tx")
	}
	if len(queries) == 0 {
		return nil
	}

	queriesIDs := make([]uint, 0, len(queries))
	for _, q := range queries {
		queriesIDs = append(queriesIDs, q.ID)
	}

	deleteQueryLabelsStm, args, err := sqlx.In(`DELETE FROM query_labels WHERE query_id IN (?)`, queriesIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting old query labels")
	}
	if _, err := tx.ExecContext(ctx, deleteQueryLabelsStm, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting old query labels")
	}

	var lblNames []interface{}
	for _, q := range queries {
		for _, lbl := range q.LabelsIncludeAny {
			lblNames = append(lblNames, lbl.LabelName)
		}
	}
	if len(lblNames) == 0 {
		return nil
	}

	// We need to figure out the label IDs for the labels we're going to add.
	stm, args, err := sqlx.In(`SELECT id, name FROM labels WHERE name IN (?)`, lblNames)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching label IDs")
	}

	rows, err := tx.QueryxContext(ctx, stm, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching label IDs")
	}
	defer rows.Close()

	lblNameToID := make(map[string]uint)
	for rows.Next() {
		var id uint
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return ctxerr.Wrap(ctx, err, "scan existing query")
		}
		lblNameToID[name] = id
	}
	if err := rows.Err(); err != nil {
		return ctxerr.Wrap(ctx, err, "fetching query IDs")
	}
	if err := rows.Close(); err != nil { //nolint:sqlclosecheck
		return ctxerr.Wrap(ctx, err, "closing query IDs")
	}

	if len(lblNameToID) < len(lblNames) {
		return ctxerr.New(ctx, "not all labels found for query")
	}

	params := make([]string, 0, len(lblNames))
	args = make([]interface{}, 0, len(lblNames)*2)
	for _, q := range queries {
		lblIdents := make([]fleet.LabelIdent, 0, len(q.LabelsIncludeAny))
		for _, lbl := range q.LabelsIncludeAny {
			if lblID, ok := lblNameToID[lbl.LabelName]; ok {
				params = append(params, "(?, ?)")
				args = append(args, q.ID, lblID)

				lblIdents = append(lblIdents, fleet.LabelIdent{
					LabelID:   lblID,
					LabelName: lbl.LabelName,
				})
			}
		}
		if len(lblIdents) != 0 {
			q.LabelsIncludeAny = lblIdents
		}
	}

	insertSQL := fmt.Sprintf(`INSERT INTO query_labels (query_id, label_id) VALUES %s`, strings.Join(params, ", "))
	if _, err := tx.ExecContext(ctx, insertSQL, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "creating query labels")
	}

	return nil
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

	if err := ds.updateQueryLabels(ctx, q); err != nil {
		return ctxerr.Wrap(ctx, err, "updaing query labels")
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
	return query(ctx, ds.reader(ctx), id)
}

func query(ctx context.Context, db sqlx.QueryerContext, id uint) (*fleet.Query, error) {
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
	if err := sqlx.GetContext(ctx, db, query, sqlQuery, false, fleet.AggregatedStatsTypeScheduledQuery, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Query").WithID(id))
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting query")
	}

	if err := loadPacksForQueries(ctx, db, []*fleet.Query{query}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading packs for queries")
	}

	if err := loadLabelsForQueries(ctx, db, []*fleet.Query{query}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "loading labels for query")
	}

	return query, nil
}

// ListQueries returns a list of queries with sort order and results limit
// determined by passed in fleet.ListOptions, count of total queries returned without limits, and
// pagination metadata
func (ds *Datastore) ListQueries(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, int, *fleet.PaginationMetadata, error) {
	getQueriesStmt := `
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

	if opt.Platform != nil {
		qs := fmt.Sprintf("%%%s%%", *opt.Platform)
		args = append(args, qs)
		whereClauses += ` AND (q.platform LIKE ? OR q.platform = '')`
	}

	// normalize the name for full Unicode support (Unicode equivalence).
	normMatch := norm.NFC.String(opt.MatchQuery)
	whereClauses, args = searchLike(whereClauses, args, normMatch, querySearchColumns...)

	getQueriesStmt += whereClauses

	// build the count statement before adding pagination constraints
	getQueriesCountStmt := fmt.Sprintf("SELECT COUNT(DISTINCT id) FROM (%s) AS s", getQueriesStmt)

	getQueriesStmt, args = appendListOptionsWithCursorToSQL(getQueriesStmt, args, &opt.ListOptions)

	dbReader := ds.reader(ctx)
	queries := []*fleet.Query{}
	if err := sqlx.SelectContext(ctx, dbReader, &queries, getQueriesStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "listing queries")
	}

	// perform a second query to grab the count
	var count int
	if err := sqlx.GetContext(ctx, dbReader, &count, getQueriesCountStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get queries count")
	}

	if err := ds.loadPacksForQueries(ctx, queries); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "loading packs for queries")
	}

	if err := ds.loadLabelsForQueries(ctx, queries); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "loading labels for queries")
	}

	var meta *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		meta = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		// `appendListOptionsWithCursorToSQL` used above to build the query statement will cause this
		// discrepancy
		if len(queries) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			meta.HasNextResults = true
			queries = queries[:len(queries)-1]
		}
	}

	return queries, count, meta, nil
}

// loadPacksForQueries loads the user packs (aka 2017 packs) associated with the provided queries.
func (ds *Datastore) loadPacksForQueries(ctx context.Context, queries []*fleet.Query) error {
	return loadPacksForQueries(ctx, ds.reader(ctx), queries)
}

func loadPacksForQueries(ctx context.Context, db sqlx.QueryerContext, queries []*fleet.Query) error {
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

	err = sqlx.SelectContext(ctx, db, &rows, query, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "selecting load packs for queries")
	}

	for _, row := range rows {
		q := name_queries[row.QueryName]
		q.Packs = append(q.Packs, row.Pack)
	}

	return nil
}

func (ds *Datastore) loadLabelsForQueries(ctx context.Context, queries []*fleet.Query) error {
	return loadLabelsForQueries(ctx, ds.reader(ctx), queries)
}

func loadLabelsForQueries(ctx context.Context, db sqlx.QueryerContext, queries []*fleet.Query) error {
	if len(queries) == 0 {
		return nil
	}

	sql := `
		SELECT
			ql.query_id AS query_id,
			ql.label_id AS label_id,
			l.name AS label_name
		FROM query_labels ql
		INNER JOIN labels l ON l.id = ql.label_id
		WHERE ql.query_id IN (?)
	`

	queryIDs := []uint{}
	for _, query := range queries {
		query.LabelsIncludeAny = nil
		queryIDs = append(queryIDs, query.ID)
	}

	stmt, args, err := sqlx.In(sql, queryIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building query to load labels for queries")
	}

	queryMap := make(map[uint]*fleet.Query, len(queries))
	for _, query := range queries {
		queryMap[query.ID] = query
	}

	rows := []struct {
		QueryID   uint   `db:"query_id"`
		LabelID   uint   `db:"label_id"`
		LabelName string `db:"label_name"`
	}{}

	err = sqlx.SelectContext(ctx, db, &rows, stmt, args...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "selecting labels for queries")
	}

	for _, row := range rows {
		queryMap[row.QueryID].LabelsIncludeAny = append(queryMap[row.QueryID].LabelsIncludeAny, fleet.LabelIdent{LabelID: row.LabelID, LabelName: row.LabelName})
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

func (ds *Datastore) ListScheduledQueriesForAgents(ctx context.Context, teamID *uint, hostID *uint, queryReportsDisabled bool) ([]*fleet.Query, error) {
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
		)%s`

	args := []interface{}{}
	teamSQL := " team_id IS NULL"
	if teamID != nil {
		args = append(args, *teamID)
		teamSQL = " team_id = ?"
	}
	args = append(args, queryReportsDisabled, fleet.LoggingSnapshot)
	labelSQL := ""
	if hostID != nil {
		labelSQL = `
		-- Query has a tag in common with the host
		AND (EXISTS (
			SELECT 1
			FROM query_labels ql
			JOIN label_membership hl ON (hl.host_id = ? AND hl.label_id = ql.label_id)
			WHERE ql.query_id = q.id
		-- Query has no tags
		) OR NOT EXISTS (
			SELECT 1
			FROM query_labels ql
			WHERE ql.query_id = q.id
		))`
		args = append(args, hostID)
	}
	sqlStmt = fmt.Sprintf(sqlStmt, teamSQL, labelSQL)

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
		// Handle zero time value
		var lastExecuted interface{} = s.LastExecuted
		if s.LastExecuted.IsZero() {
			lastExecuted = nil
		}
		args = append(
			args, queryID, s.HostID, statsLiveQueryType, s.Executions, s.AverageMemory, s.SystemTime, s.UserTime, s.WallTime, s.OutputSize,
			0, 0, lastExecuted,
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
