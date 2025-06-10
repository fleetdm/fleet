package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250424153059, Down_20250424153059)
}

func Up_20250424153059(tx *sql.Tx) error {
	stmt := `
CREATE TABLE batch_script_executions (
  id int unsigned NOT NULL AUTO_INCREMENT,
  script_id int unsigned NOT NULL,
  execution_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  user_id int unsigned DEFAULT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_batch_script_executions_execution_id (execution_id),
  CONSTRAINT batch_script_executions_script_id FOREIGN KEY (script_id) REFERENCES scripts (id) ON DELETE CASCADE
);

CREATE TABLE batch_script_execution_host_results (
  id int unsigned NOT NULL AUTO_INCREMENT,
  batch_execution_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  host_id int unsigned NOT NULL,
  host_execution_id varchar(255) COLLATE utf8mb4_unicode_ci,
  error varchar(255) COLLATE utf8mb4_unicode_ci,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_batch_script_execution_host_result_execution_id (batch_execution_id),
  CONSTRAINT batch_script_batch_id FOREIGN KEY (batch_execution_id) REFERENCES batch_script_executions (execution_id) ON DELETE CASCADE
)
`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("creating batch script tables: %w", err)
	}

	return nil
}

func Down_20250424153059(tx *sql.Tx) error {
	return nil
}
