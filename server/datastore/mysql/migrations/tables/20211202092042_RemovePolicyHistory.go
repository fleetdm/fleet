package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211202092042, Down_20211202092042)
}

func Up_20211202092042(tx *sql.Tx) error {
	_, err := tx.Exec("DROP VIEW IF EXISTS policy_membership")
	if err != nil {
		return errors.Wrap(err, "drop view policy_membership")
	}

	policyMembershipTable := `
		CREATE TABLE IF NOT EXISTS policy_membership (
			policy_id INT UNSIGNED NOT NULL,
			host_id int(10) UNSIGNED NOT NULL,
			passes BOOL DEFAULT NULL,
			created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
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

	if _, err := tx.Exec(`insert ignore into policy_membership 
		select policy_id, host_id, passes, created_at, updated_at 
		from policy_membership_history where id in (
		    select max(id) as id from policy_membership_history group by policy_id, host_id
		)`); err != nil {
		return errors.Wrap(err, "populate policy membership table")
	}

	_, err = tx.Exec("DROP TABLE IF EXISTS policy_membership_history")
	if err != nil {
		return errors.Wrap(err, "drop table policy_membership_history")
	}

	return nil
}

func Down_20211202092042(tx *sql.Tx) error {
	return nil
}
