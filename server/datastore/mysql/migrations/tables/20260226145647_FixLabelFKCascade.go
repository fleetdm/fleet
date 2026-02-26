package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260226145647, Down_20260226145647)
}

func Up_20260226145647(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(`
ALTER TABLE policy_labels
  DROP FOREIGN KEY policy_labels_label_id,
  ADD CONSTRAINT policy_labels_label_id
    FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT`,
			"changing policy_labels.label_id FK from CASCADE to RESTRICT"),
		basicMigrationStep(`
ALTER TABLE query_labels
  DROP FOREIGN KEY query_labels_label_id,
  ADD CONSTRAINT query_labels_label_id
    FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE RESTRICT`,
			"changing query_labels.label_id FK from CASCADE to RESTRICT"),
	}, tx)
}

func Down_20260226145647(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		basicMigrationStep(`
ALTER TABLE policy_labels
  DROP FOREIGN KEY policy_labels_label_id,
  ADD CONSTRAINT policy_labels_label_id
    FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE CASCADE`,
			"reverting policy_labels.label_id FK from RESTRICT to CASCADE"),
		basicMigrationStep(`
ALTER TABLE query_labels
  DROP FOREIGN KEY query_labels_label_id,
  ADD CONSTRAINT query_labels_label_id
    FOREIGN KEY (label_id) REFERENCES labels (id) ON DELETE CASCADE`,
			"reverting query_labels.label_id FK from RESTRICT to CASCADE"),
	}, tx)
}
