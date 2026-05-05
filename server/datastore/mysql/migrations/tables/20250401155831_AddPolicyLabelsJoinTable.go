package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250401155831, Down_20250401155831)
}

func Up_20250401155831(tx *sql.Tx) error {
	stmt := `
CREATE TABLE policy_labels (
  id int unsigned NOT NULL AUTO_INCREMENT,
  policy_id int unsigned NOT NULL,
  label_id int unsigned NOT NULL,
  exclude tinyint(1) NOT NULL DEFAULT '0',
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT policy_labels_policy_id FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE CASCADE,
  CONSTRAINT policy_labels_label_id FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE CASCADE,
  UNIQUE KEY idx_policy_labels_policy_label (policy_id, label_id)
)
`
	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("creating policy_labels table: %w", err)
	}

	return nil
}

func Down_20250401155831(tx *sql.Tx) error {
	return nil
}
