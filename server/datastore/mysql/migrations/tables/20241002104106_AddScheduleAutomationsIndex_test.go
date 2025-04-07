package tables

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241002104106(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...

	// Apply current migration.
	applyNext(t, db)

	// Assert the index was created.
	rows, err := db.Query("SHOW INDEX FROM queries WHERE Key_name = 'idx_queries_schedule_automations'")
	require.NoError(t, err)
	defer rows.Close()

	var indexCount int
	for rows.Next() {
		indexCount++
	}

	require.NoError(t, rows.Err())
	require.Greater(t, indexCount, 0)

	//
	// Assert the index is used when there are rows in the queries table
	// (wrong index is used when there are no rows in the queries table)
	//

	stmtPrefix := "INSERT INTO `queries` (`saved`, `name`, `description`, `query`, `author_id`, `observer_can_run`, `team_id`, `team_id_char`, `platform`, `min_osquery_version`, `schedule_interval`, `automations_enabled`, `logging_type`, `discard_data`) VALUES "
	stmtSuffix := ";"

	var valueStrings []string
	var valueArgs []interface{}

	// Generate 10 records
	for i := 0; i < 10; i++ {
		queryID := i + 1
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, 0, fmt.Sprintf("query_%d", queryID), "", "SELECT * FROM processes;", 1, 0, nil, "", "", "", 0, 0, "snapshot", 0)
	}

	// Disable foreign key checks to improve performance
	_, err = db.Exec("SET FOREIGN_KEY_CHECKS=0")
	require.NoError(t, err)

	// Construct and execute the batch insert
	stmt := stmtPrefix + strings.Join(valueStrings, ",") + stmtSuffix
	_, err = db.Exec(stmt, valueArgs...)
	require.NoError(t, err)

	// Re-enable foreign key checks
	_, err = db.Exec(`SET FOREIGN_KEY_CHECKS=1`)
	require.NoError(t, err)

	result := struct {
		ID           int     `db:"id"`
		SelectType   string  `db:"select_type"`
		Table        string  `db:"table"`
		Type         string  `db:"type"`
		PossibleKeys *string `db:"possible_keys"`
		Key          *string `db:"key"`
		KeyLen       *int    `db:"key_len"`
		Ref          *string `db:"ref"`
		Rows         int     `db:"rows"`
		Filtered     float64 `db:"filtered"`
		Extra        *string `db:"Extra"`
		Partitions   *string `db:"partitions"`
	}{}

	// Query based on loadHostScheduledQueryStatsDB in server/datastore/mysql/hosts.go
	err = db.Get(&result, `
    EXPLAIN
    SELECT
        q.id
    FROM
        queries q
    WHERE (q.platform = ''
        OR q.platform IS NULL
        OR FIND_IN_SET('darwin', q.platform) != 0)
    AND q.is_scheduled = 1
    AND(q.automations_enabled IS TRUE
        OR(q.discard_data IS FALSE
            AND q.logging_type = 'snapshot'))
    AND(q.team_id IS NULL
        OR q.team_id = 0)
    GROUP BY
        q.id
`)
	require.NoError(t, err)

	// Assert the correct index is used
	require.Equal(t, *result.Key, "idx_queries_schedule_automations")
}
