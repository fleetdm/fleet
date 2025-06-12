package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250306144233, Down_20250306144233)
}

func Up_20250306144233(tx *sql.Tx) error {
	stmt := `
CREATE TABLE query_labels (
  id int unsigned NOT NULL AUTO_INCREMENT,
  query_id int unsigned NOT NULL,
  label_id int unsigned NOT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT query_labels_query_id FOREIGN KEY (query_id) REFERENCES queries (id) ON DELETE CASCADE,
  CONSTRAINT query_labels_label_id FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE CASCADE,
  UNIQUE KEY idx_query_labels_query_label (query_id, label_id)
)`

	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("creating query_labels table: %w", err)
	}

	return nil
}

func Down_20250306144233(tx *sql.Tx) error {
	return nil
}
