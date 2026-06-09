package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260609081645, Down_20260609081645)
}

func Up_20260609081645(tx *sql.Tx) error {
	// policy_id records the team policy (with an install-software automation pointing at the same installer) that gates a
	// Windows/Linux setup-experience software item. It is NULL for un-gated items, VPP items, and Apple-platform items. The gate
	// is internal (json:"-"), so this is not an API change. ON DELETE SET NULL so deleting the policy simply un-gates the item.
	_, err := tx.Exec(`
ALTER TABLE setup_experience_status_results
	ADD COLUMN policy_id INT UNSIGNED NULL DEFAULT NULL,
	ADD CONSTRAINT fk_setup_experience_status_results_policy_id
		FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE SET NULL
`)
	if err != nil {
		return errors.Wrap(err, "add policy_id to setup_experience_status_results")
	}
	return nil
}

func Down_20260609081645(tx *sql.Tx) error {
	return nil
}
