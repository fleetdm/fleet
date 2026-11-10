package tables

import (
	"database/sql"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20261109113017, Down_20261109113017)
}

// Built-in labels are no longer valid report targets. Removing them intentionally changes which
// hosts some reports target (e.g. a report scoped only to "macOS" now targets all hosts).
func Up_20261109113017(tx *sql.Tx) error {
	queryIDs, err := queryIDsTargetingBuiltinLabels(tx)
	if err != nil {
		return err
	}
	if len(queryIDs) == 0 {
		return nil
	}

	if _, err := tx.Exec(`
		DELETE ql FROM query_labels ql
		JOIN labels l ON l.id = ql.label_id
		WHERE l.label_type = 1`); err != nil {
		return fmt.Errorf("delete built-in labels from query_labels: %w", err)
	}
	logger.Warn.Printf("Removed built-in label targets from reports, which may now target more or fewer hosts. Review the targets of report IDs: %s\n", strings.Join(queryIDs, ", "))
	return nil
}

func queryIDsTargetingBuiltinLabels(tx *sql.Tx) ([]string, error) {
	rows, err := tx.Query(`
		SELECT DISTINCT ql.query_id FROM query_labels ql
		JOIN labels l ON l.id = ql.label_id
		WHERE l.label_type = 1
		ORDER BY ql.query_id`)
	if err != nil {
		return nil, fmt.Errorf("select reports targeting built-in labels: %w", err)
	}
	defer rows.Close()

	var queryIDs []string
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan report id: %w", err)
		}
		queryIDs = append(queryIDs, fmt.Sprint(id))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reports targeting built-in labels: %w", err)
	}
	return queryIDs, nil
}

func Down_20261109113017(tx *sql.Tx) error {
	return nil
}
