package tables

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20230721161508, Down_20230721161508)
}

func _20230719152138_migrate_global_packs(tx *sql.Tx) error {
	type QueryWithScheduledFields struct {
		fleet.Query
		ScheduledQID           uint
		ScheduledQueryInterval uint
		ScheduledQSnapshot     *bool
		ScheduledQRemoved      *bool
		ScheduledQueryPlatform string
		ScheduledQueryVersion  string
		PackType               string
	}

	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						sq.id       AS scheduled_query_id,
						sq.interval AS scheduled_query_interval,
						sq.snapshot AS scheduled_query_snapshot,
						sq.removed  AS scheduled_query_removed,
						sq.platform AS scheduled_query_platform,
						sq.version  AS scheduled_query_version,
						p.pack_type AS pack_type
		FROM queries q
				INNER JOIN scheduled_queries sq ON q.team_id IS NULL AND q.name = sq.query_name
				INNER JOIN packs p ON sq.pack_id = p.id
		WHERE p.pack_type = 'global'`
	rows, err := tx.Query(selectStmt)
	if err != nil {
		return fmt.Errorf("error executing 'Query' for scheduled queries from global packs: %s", err)
	}
	defer rows.Close()

	var args []interface{}
	var nRows int

	for rows.Next() {
		nRows += 1
		query := QueryWithScheduledFields{}
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.ScheduledQID,
			&query.ScheduledQueryInterval,
			&query.ScheduledQSnapshot,
			&query.ScheduledQRemoved,
			&query.ScheduledQueryPlatform,
			&query.ScheduledQueryVersion,
			&query.PackType,
		); err != nil {
			return fmt.Errorf("error executing 'Scan' for scheduled queries from global packs: %s", err)
		}

		var loggingType string
		if query.ScheduledQSnapshot != nil && *query.ScheduledQSnapshot {
			loggingType = "snapshot"
		}
		if loggingType == "" && query.ScheduledQRemoved != nil {
			if *query.ScheduledQRemoved {
				loggingType = "differential"
			} else {
				loggingType = "differential_ignore_removals"
			}
		}

		args = append(args,
			fmt.Sprintf("%s - %d", query.Name, query.ScheduledQID),
			query.Description,
			query.Query.Query,
			query.AuthorID,
			query.Saved,
			query.ObserverCanRun,
			query.ScheduledQueryPlatform,
			query.ScheduledQueryVersion,
			query.ScheduledQueryInterval,
			loggingType,
			true,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error on 'rows' for scheduled queries from global packs: %s", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("error executing 'Close' for scheduled queries from global packs: %s", err)
	}

	if len(args) == 0 {
		return nil
	}

	insertStmt := `
		INSERT INTO queries (
			name,
			description,
			query,
			author_id,
			saved,
			observer_can_run,
			platform,
			min_osquery_version,
			schedule_interval,
			logging_type,
			automations_enabled 
		) VALUES %s`

	placeHolders := strings.TrimSuffix(strings.Repeat("( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? ),", nRows), ",")
	if _, err = tx.Exec(fmt.Sprintf(insertStmt, placeHolders), args...); err != nil {
		return fmt.Errorf("error executing 'Exec' for scheduled queries from global packs: %s", err)
	}

	return nil
}

func _20230719152138_migrate_team_packs(tx *sql.Tx) error {
	type QueryWithScheduledFields struct {
		fleet.Query
		ScheduledQID           uint
		ScheduledQueryInterval uint
		ScheduledQSnapshot     *bool
		ScheduledQRemoved      *bool
		ScheduledQueryPlatform string
		ScheduledQueryVersion  string
		PackType               string
	}

	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						sq.id         AS scheduled_query_id,
						sq.interval   AS scheduled_query_interval,
						sq.snapshot   AS scheduled_query_snapshot,
						sq.removed    AS scheduled_query_removed,
						sq.platform   AS scheduled_query_platform,
						sq.version    AS scheduled_query_version,
						p.pack_type   AS pack_type
		FROM queries q
				INNER JOIN scheduled_queries sq ON q.team_id IS NULL AND q.name = sq.query_name
				INNER JOIN packs p ON sq.pack_id = p.id
		WHERE p.pack_type <> 'global'
		AND p.pack_type IS NOT NULL`

	rows, err := tx.Query(selectStmt)
	if err != nil {
		return fmt.Errorf("error executing 'Query' for scheduled queries from team packs: %s", err)
	}
	defer rows.Close()

	var args []interface{}
	var nRows int

	for rows.Next() {
		nRows += 1
		var query QueryWithScheduledFields
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.ScheduledQID,
			&query.ScheduledQueryInterval,
			&query.ScheduledQSnapshot,
			&query.ScheduledQRemoved,
			&query.ScheduledQueryPlatform,
			&query.ScheduledQueryVersion,
			&query.PackType,
		); err != nil {
			return fmt.Errorf("error executing 'Scan' for scheduled queries from team packs: %s", err)
		}

		teamIDParts := strings.Split(query.PackType, "-")
		teamID, err := strconv.Atoi(teamIDParts[1])
		if err != nil {
			return fmt.Errorf("error parsing TeamID for scheduled queries from team packs: %s", err)
		}

		var loggingType string
		if query.ScheduledQSnapshot != nil && *query.ScheduledQSnapshot {
			loggingType = "snapshot"
		}
		if query.ScheduledQRemoved != nil {
			if *query.ScheduledQRemoved {
				loggingType = "differential"
			} else {
				loggingType = "differential_ignore_removals"
			}
		}
		args = append(args,
			teamID,
			teamIDParts[1],
			fmt.Sprintf("%s - %d", query.Name, query.ScheduledQID),
			query.Description,
			query.Query.Query,
			query.AuthorID,
			query.Saved,
			query.ObserverCanRun,
			query.ScheduledQueryPlatform,
			query.ScheduledQueryVersion,
			query.ScheduledQueryInterval,
			loggingType,
			true,
		)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error on 'rows' for scheduled queries from team packs: %s", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("error closing reader from team packs: %s", err)
	}

	if len(args) == 0 {
		return nil
	}

	insertStmt := `
		INSERT INTO queries (
			team_id,
			team_id_char,
			name,
			description,
			query,
			author_id,
			saved,
			observer_can_run,
			platform,
			min_osquery_version,
			schedule_interval,
			logging_type,
			automations_enabled 
		) VALUES %s`

	placeHolders := strings.TrimSuffix(strings.Repeat("( ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? ),", nRows), ",")
	if _, err = tx.Exec(fmt.Sprintf(insertStmt, placeHolders), args...); err != nil {
		return fmt.Errorf("error executing 'Exec' for scheduled queries from team packs: %s", err)
	}

	return nil
}

func _20230719152138_migrate_non_scheduled(tx *sql.Tx) error {
	type QueryWithTeamIDs struct {
		fleet.Query
		TeamIDs string
	}

	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						GROUP_CONCAT(ut.team_id) AS team_ids
		FROM queries q
				LEFT JOIN scheduled_queries sq ON q.team_id IS NULL AND q.name = sq.query_name
				INNER JOIN user_teams ut on q.author_id = ut.user_id
		WHERE sq.id IS NULL
		GROUP BY q.id`

	rows, err := tx.Query(selectStmt)
	if err != nil {
		return fmt.Errorf("error executing 'Query' for non-scheduled queries: %s", err)
	}
	defer rows.Close()

	var args []interface{}
	var nRows int

	for rows.Next() {
		query := QueryWithTeamIDs{}
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.TeamIDs,
		); err != nil {
			return fmt.Errorf("error executing 'Scan' for non-scheduled queries: %s", err)
		}
		teamIDs := strings.Split(query.TeamIDs, ",")
		for _, teamIDStr := range teamIDs {
			nRows += 1
			teamID, err := strconv.Atoi(teamIDStr)
			if err != nil {
				return fmt.Errorf("error parsing team ID on non-scheduled queries: %s", err)
			}
			args = append(args,
				query.Name,
				query.Description,
				query.Query.Query,
				query.AuthorID,
				query.Saved,
				query.ObserverCanRun,
				teamID,
				teamIDStr,
			)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error on 'rows' for non-scheduled queries: %s", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("error executing 'Close' for non-scheduled queries: %s", err)
	}

	if len(args) == 0 {
		return nil
	}

	insertStmt := `
		INSERT INTO queries (
			name,
			description,
			query,
			author_id,
			saved,
			observer_can_run,
			team_id,
			team_id_char
		) VALUES %s`

	placeHolders := strings.TrimSuffix(strings.Repeat("( ?, ?, ?, ?, ?, ?, ?, ? ),", nRows), ",")
	if _, err = tx.Exec(fmt.Sprintf(insertStmt, placeHolders), args...); err != nil {
		return fmt.Errorf("error executing 'Exec' on non-scheduled queries: %s", err)
	}
	return nil
}

func Up_20230721161508(tx *sql.Tx) error {
	// Remove query stats
	_, err := tx.Exec(`TRUNCATE scheduled_query_stats`)
	if err != nil {
		return fmt.Errorf("error truncating 'scheduled_query_stats': %s", err)
	}
	_, err = tx.Exec(`DELETE FROM aggregated_stats WHERE type = 'query' OR type = 'scheduled_query'`)
	if err != nil {
		return fmt.Errorf("error removing aggregated_stats: %s", err)
	}

	// Migrates 'old' scheduled queries to the 'new' query schema.
	// Queries can either be:
	// 	1 - Scheduled, which can either belong to:
	//		1.1 - The global pack:
	// 			For each scheduled query create a single global scheduled query named `$query.name - $scheduled.id`.
	//		1.2 - Team pack:
	//			Create a new team query with the name of `$query.name - $scheduled.id`.
	//		1.3 - A user pack (a.k.a 2017 pack):
	// 			Do nothing.
	//	2 - Not scheduled:
	// 		2.1 - If the author belongs to the global team, do nothing.
	// 		2.2 - Otherwise, for each team the author belongs to:
	//			Create a new team query with the name of `$query.name` iff the author can run the query.
	//

	// ----------------------------------------------------------------------------
	// (2.2) Non scheduled queries, author belongs to one or more teams:
	// ----------------------------------------------------------------------------
	if err := _20230719152138_migrate_non_scheduled(tx); err != nil {
		return err
	}

	// -------------------------------------
	// (1.1) Global pack scheduled queries
	// -------------------------------------
	if err := _20230719152138_migrate_global_packs(tx); err != nil {
		return err
	}

	// -------------------------------------
	// (1.2) Team pack scheduled queries
	// -------------------------------------
	if err := _20230719152138_migrate_team_packs(tx); err != nil {
		return err
	}

	// Remove 'global' and 'team' packs
	_, err = tx.Exec(`DELETE FROM packs WHERE pack_type = 'global' OR pack_type LIKE 'team-%'`)
	if err != nil {
		return fmt.Errorf("error deleting packs: %s", err)
	}
	return nil
}

func Down_20230721161508(tx *sql.Tx) error {
	return nil
}
