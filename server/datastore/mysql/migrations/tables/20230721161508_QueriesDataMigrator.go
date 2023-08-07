package tables

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func init() {
	MigrationClient.AddMigration(Up_20230721161508, Down_20230721161508)
}

// This is meant to future-proof this migration, this type is based on the fleet.Query type.
type _20230719152138_Query struct {
	TeamID                  *uint
	TeamIDChar              string
	ObserverCanRun          bool
	ScheduleInterval        uint
	Platform                string
	MinOsqueryVersion       string
	AutomationsEnabled      bool
	LoggingType             string
	Name                    string
	Description             string
	Query                   string
	Saved                   bool
	AuthorID                *uint
	ScheduledQID            uint
	ScheduledQueryInterval  uint
	ScheduledQSnapshot      *bool
	ScheduledQRemoved       *bool
	ScheduledQueryPlatform  string
	ScheduledQueryVersion   string
	ScheduledQueryTimestamp time.Time
	PackType                string
	TeamRoles               string
}

func _20230719152138_QueryName(q _20230719152138_Query) string {
	return fmt.Sprintf("%s - %d - %s", q.Name, q.ScheduledQID, q.ScheduledQueryTimestamp.Format("Jan _2 15:04:05.000"))
}

func _20230719152138_migrate_global_packs(tx *sql.Tx) error {
	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						sq.id                     AS scheduled_query_id,
						sq.interval               AS scheduled_query_interval,
						sq.snapshot               AS scheduled_query_snapshot,
						sq.removed                AS scheduled_query_removed,
						COALESCE(sq.platform, '') AS scheduled_query_platform,
						COALESCE(sq.version, '')  AS scheduled_query_version,
						sq.created_at             AS scheduled_created_at,
						p.pack_type               AS pack_type
		FROM queries q
				INNER JOIN scheduled_queries sq ON q.name = sq.query_name
				INNER JOIN packs p ON sq.pack_id = p.id
		WHERE p.pack_type = 'global' AND q.team_id IS NULL`
	rows, err := tx.Query(selectStmt)
	if err != nil {
		return fmt.Errorf("error executing 'Query' for scheduled queries from global packs: %s", err)
	}
	defer rows.Close()

	var args []interface{}
	var nRows int

	for rows.Next() {
		nRows += 1
		query := _20230719152138_Query{}
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.ScheduledQID,
			&query.ScheduledQueryInterval,
			&query.ScheduledQSnapshot,
			&query.ScheduledQRemoved,
			&query.ScheduledQueryPlatform,
			&query.ScheduledQueryVersion,
			&query.ScheduledQueryTimestamp,
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
			_20230719152138_QueryName(query),
			query.Description,
			query.Query,
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
	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						sq.id                      AS scheduled_query_id,
						sq.interval                AS scheduled_query_interval,
						sq.snapshot                AS scheduled_query_snapshot,
						sq.removed                 AS scheduled_query_removed,
						COALESCE(sq.platform, '')  AS scheduled_query_platform,
						COALESCE(sq.version, '')   AS scheduled_query_version,
						sq.created_at              AS scheduled_created_at,
						p.pack_type                AS pack_type
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
		var query _20230719152138_Query
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.ScheduledQID,
			&query.ScheduledQueryInterval,
			&query.ScheduledQSnapshot,
			&query.ScheduledQRemoved,
			&query.ScheduledQueryPlatform,
			&query.ScheduledQueryVersion,
			&query.ScheduledQueryTimestamp,
			&query.PackType,
		); err != nil {
			return fmt.Errorf("error executing 'Scan' for scheduled queries from team packs: %s", err)
		}

		teamIDParts := strings.Split(query.PackType, "-")
		if len(teamIDParts) != 2 {
			return fmt.Errorf("invalid pack_type value %s", query.PackType)
		}
		teamID, err := strconv.Atoi(teamIDParts[1])
		if err != nil {
			return fmt.Errorf("error parsing TeamID for scheduled queries from team packs: %s", err)
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
			teamID,
			teamIDParts[1],
			_20230719152138_QueryName(query),
			query.Description,
			query.Query,
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
	// If the query is not scheduled, then it stays global except if it was created by a team user,
	// in which case the query is duplicated as a team query iff the user is an admin or mantainer of the team.
	selectStmt := `
		SELECT DISTINCT q.name,
						q.description,
						q.query,
						q.author_id,
						q.saved,
						q.observer_can_run,
						GROUP_CONCAT(CONCAT(ut.team_id, ':', ut.role)) AS team_roles
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
		query := _20230719152138_Query{}
		if err := rows.Scan(
			&query.Name,
			&query.Description,
			&query.Query,
			&query.AuthorID,
			&query.Saved,
			&query.ObserverCanRun,
			&query.TeamRoles,
		); err != nil {
			return fmt.Errorf("error executing 'Scan' for non-scheduled queries: %s", err)
		}
		teamRoles := strings.Split(query.TeamRoles, ",")
		for _, teamRole := range teamRoles {
			teamRoleParts := strings.Split(teamRole, ":")

			role := teamRoleParts[1]
			if role == "observer" || role == "observer_plus" {
				continue
			}
			nRows += 1
			teamID, err := strconv.Atoi(teamRoleParts[0])
			if err != nil {
				return fmt.Errorf("error parsing team ID on non-scheduled queries: %s", err)
			}
			args = append(args,
				query.Name,
				query.Description,
				query.Query,
				query.AuthorID,
				query.Saved,
				query.ObserverCanRun,
				teamID,
				teamRoleParts[0],
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

func _20230719152138_clean_up(tx *sql.Tx) error {
	// Remove query stats
	if _, err := tx.Exec(`TRUNCATE scheduled_query_stats`); err != nil {
		return fmt.Errorf("error truncating 'scheduled_query_stats': %s", err)
	}
	if _, err := tx.Exec(`DELETE FROM aggregated_stats WHERE type = 'query' OR type = 'scheduled_query'`); err != nil {
		return fmt.Errorf("error removing aggregated_stats: %s", err)
	}

	// Delete queries that only belong to 'global' and 'team' packs
	if _, err := tx.Exec(`
DELETE
FROM queries
WHERE name IN (SELECT query_name
               FROM (SELECT query_name
                     FROM scheduled_queries
                              INNER JOIN packs p on scheduled_queries.pack_id = p.id
                     WHERE p.pack_type = 'global'
                     UNION
                     SELECT query_name
                     FROM scheduled_queries
                              INNER JOIN packs p on scheduled_queries.pack_id = p.id
                     WHERE p.pack_type LIKE 'team-%') r
               WHERE query_name NOT IN (SELECT query_name
                                        FROM scheduled_queries
                                                 INNER JOIN packs p on scheduled_queries.pack_id = p.id
                                        WHERE p.pack_type IS NULL))	
	
	`); err != nil {
		return fmt.Errorf("error deleting queries that belong only to global / team packs: %s", err)
	}

	// Remove 'global' and 'team' packs ... relevant rows in the 'scheduled_queries' table should be
	// deleted because of the on cascade delete on the pack_id FK.
	if _, err := tx.Exec(`DELETE FROM packs WHERE pack_type = 'global' OR pack_type LIKE 'team-%'`); err != nil {
		return fmt.Errorf("error deleting packs: %s", err)
	}

	return nil
}

func Up_20230721161508(tx *sql.Tx) error {
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

	//-------------------------------------------------------
	// Remove stats, global packs and team packs and queries
	// that are only used in global/team packs.
	//-------------------------------------------------------
	if err := _20230719152138_clean_up(tx); err != nil {
		return err
	}

	return nil
}

func Down_20230721161508(tx *sql.Tx) error {
	return nil
}
