package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211202092042, Down_20211202092042)
}

func Up_20211202092042(tx *sql.Tx) error {
	_, err := tx.Exec("DROP VIEW policy_membership")
	if err != nil {
		return errors.Wrap(err, "drop table policy_membership_history")
	}

	_, err = tx.Exec("DROP TABLE policy_membership_history")
	if err != nil {
		return errors.Wrap(err, "drop table policy_membership_history")
	}

	policyMembershipTable := `
		CREATE TABLE IF NOT EXISTS policy_membership (
			policy_id INT UNSIGNED,
			host_id int(10) UNSIGNED NOT NULL,
			passes BOOL DEFAULT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (policy_id,host_id),
			FOREIGN KEY fk_policy_membership_policy_id (policy_id) REFERENCES policies(id) ON DELETE CASCADE,
			FOREIGN KEY fk_policy_membership_host_id (host_id) REFERENCES hosts(id) ON DELETE CASCADE,
			INDEX idx_policy_membership_passes (passes),
			INDEX idx_policy_membership_policy_id (policy_id),
			INDEX idx_policy_membership_host_id_passes (host_id, passes)
		);
	`

	if _, err := tx.Exec(policyMembershipTable); err != nil {
		return errors.Wrap(err, "create policy membership table")
	}

	return nil
}

func Down_20211202092042(tx *sql.Tx) error {
	return nil
}
