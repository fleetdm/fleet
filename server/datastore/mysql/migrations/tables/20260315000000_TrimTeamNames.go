package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260315000000, Down_20260315000000)
}

// rtrim is a SQL expression that trims leading and trailing whitespace from a
// column using REGEXP_REPLACE with the POSIX [[:space:]] class, which covers
// space, tab, newline, carriage return, vertical tab, and form feed — matching
// Go's strings.TrimSpace for ASCII whitespace.
const rtrim = `REGEXP_REPLACE(%s, '^[[:space:]]+|[[:space:]]+$', '')`

func Up_20260315000000(tx *sql.Tx) error {
	// Trim leading and trailing whitespace from team names.
	//
	// Edge cases handled:
	// 1. Whitespace-only names: trimming produces "", which is invalid. These
	//    are renamed to "Unnamed team (<id>)".
	// 2. Duplicate after trimming: e.g. "Engineering " and "Engineering" both
	//    exist. The one with whitespace is disambiguated by appending " (<id>)".
	// 3. Multiple whitespace-only names: each gets a unique name via its ID.

	trimExpr := fmt.Sprintf(rtrim, "name")
	trimT1 := fmt.Sprintf(rtrim, "t1.name")
	trimTName := fmt.Sprintf(rtrim, "t.name")

	// Step 1: Rename whitespace-only team names to "Unnamed team (<id>)".
	_, err := tx.Exec(fmt.Sprintf(`
		UPDATE teams
		SET name = CONCAT('Unnamed team (', id, ')')
		WHERE %s = ''
	`, trimExpr))
	if err != nil {
		return fmt.Errorf("rename whitespace-only team names: %w", err)
	}

	// Step 2: Find teams that would conflict after trimming and disambiguate
	// them by appending the team ID. Uses a derived table with a self-join
	// fully contained inside the subquery to avoid MySQL's restriction on
	// updating a table referenced in a subquery.
	// Note: we avoid CREATE TEMPORARY TABLE as it can fail with GTID replication.
	_, err = tx.Exec(fmt.Sprintf(`
		UPDATE teams t
		JOIN (
			SELECT DISTINCT t1.id
			FROM teams t1
			INNER JOIN teams t2 ON t2.id != t1.id AND t2.name = %s
			WHERE %s != t1.name
		) AS conflicting ON t.id = conflicting.id
		SET t.name = CONCAT(%s, ' (', t.id, ')')
	`, trimT1, trimT1, trimTName))
	if err != nil {
		return fmt.Errorf("resolve conflicting trimmed team names: %w", err)
	}

	// Step 3: Trim all remaining names that have leading/trailing whitespace.
	_, err = tx.Exec(fmt.Sprintf(`
		UPDATE teams
		SET name = %s
		WHERE %s != name
	`, trimExpr, trimExpr))
	if err != nil {
		return fmt.Errorf("trim team names: %w", err)
	}

	return nil
}

func Down_20260315000000(tx *sql.Tx) error {
	return nil
}
