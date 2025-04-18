package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250416152709, Down_20250416152709)
}

func Up_20250416152709(tx *sql.Tx) error {
	stmt := `
CREATE TABLE batch_script_executions (
  id int unsigned NOT NULL AUTO_INCREMENT,
  execution_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY idx_batch_script_executions_execution_id (execution_id)
);

CREATE TABLE batch_script_execution_host_results (
  id int unsigned NOT NULL AUTO_INCREMENT,
  batch_execution_id int unsigned NOT NULL,
  host_execution_id varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  error varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  created_at timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  CONSTRAINT batch_script_batch_id FOREIGN KEY (batch_execution_id) REFERENCES batch_script_executions (id) ON DELETE CASCADE,
  CONSTRAINT batch_script_host_execution_id FOREIGN KEY (host_execution_id) REFERENCES host_script_results (execution_id) ON DELETE CASCADE
)
`

	// TODO ADD KEY FOR EXECUTION ID!

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("creating batch script tables: %w", err)
	}

	return nil
}

func Down_20250416152709(tx *sql.Tx) error {
	return nil
}
