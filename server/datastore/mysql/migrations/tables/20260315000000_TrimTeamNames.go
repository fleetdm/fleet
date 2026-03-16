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

	// needsTrim matches names with leading or trailing whitespace (including
	// tabs, newlines, etc.). Used instead of comparing trimmed != original to
	// avoid PAD SPACE collation issues where "Finance " = "Finance".
	const needsTrim = `%s REGEXP '^[[:space:]]|[[:space:]]$'`

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
	//
	// The join compares trimmed(t1) = trimmed(t2) instead of t2.name = trimmed(t1)
	// so that two teams that both need trimming to the same value (e.g.,
	// "Finance " and "  Finance" → both "Finance") are both caught.
	trimT2 := fmt.Sprintf(rtrim, "t2.name")
	_, err = tx.Exec(fmt.Sprintf(`
		UPDATE teams t
		JOIN (
			SELECT DISTINCT t1.id
			FROM teams t1
			INNER JOIN teams t2 ON t2.id != t1.id AND %s = %s
			WHERE `+needsTrim+`
		) AS conflicting ON t.id = conflicting.id
		SET t.name = CONCAT(%s, ' (', t.id, ')')
	`, trimT2, trimT1, "t1.name", trimTName))
	if err != nil {
		return fmt.Errorf("resolve conflicting trimmed team names: %w", err)
	}

	// Step 3: Trim all remaining names that have leading/trailing whitespace.
	_, err = tx.Exec(fmt.Sprintf(`
		UPDATE teams
		SET name = %s
		WHERE `+needsTrim, trimExpr, "name"))
	if err != nil {
		return fmt.Errorf("trim team names: %w", err)
	}

	return nil
}

func Down_20260315000000(tx *sql.Tx) error {
	return nil
}
