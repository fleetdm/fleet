package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210819143446, Down_20210819143446)
}

func Up_20210819143446(tx *sql.Tx) error {
	policiesTable := `
		CREATE TABLE IF NOT EXISTS policies (
			id int(10) UNSIGNED NOT NULL AUTO_INCREMENT,
			query_id int(10) UNSIGNED NOT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			FOREIGN KEY fk_policies_query_id (query_id) REFERENCES queries(id) ON DELETE RESTRICT
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`
	policyMembershipHistoryTable := `
		CREATE TABLE IF NOT EXISTS policy_membership_history (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			policy_id INT UNSIGNED,
			host_id int(10) UNSIGNED NOT NULL,
			passes BOOL DEFAULT NULL,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			FOREIGN KEY fk_policy_membership_policy_id (policy_id) REFERENCES policies(id) ON DELETE CASCADE,
			FOREIGN KEY fk_policy_membership_host_id (host_id) REFERENCES hosts(id) ON DELETE CASCADE,
			INDEX idx_policy_membership_passes (passes),
			INDEX idx_policy_membership_policy_id (policy_id),
			INDEX idx_policy_membership_host_id_passes (host_id, passes)
		);
	`
	policyMembershipView := `
		CREATE OR REPLACE VIEW policy_membership AS select * from policy_membership_history where id in (select max(id) as id from policy_membership_history group by host_id, policy_id);
	`
	if _, err := tx.Exec(policiesTable); err != nil {
		return errors.Wrap(err, "create policies table")
	}
	if _, err := tx.Exec(policyMembershipHistoryTable); err != nil {
		return errors.Wrap(err, "create policy membership history table")
	}
	if _, err := tx.Exec(policyMembershipView); err != nil {
		return errors.Wrap(err, "create policy membership view")
	}

	return nil
}

func Down_20210819143446(tx *sql.Tx) error {
	return nil
}
