package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261019212319, Down_20261019212319)
}

func Up_20261019212319(tx *sql.Tx) error {
	// Drop the table first so a rerun after a failed backfill starts from an empty table.
	_, err := tx.Exec(`DROP TABLE IF EXISTS host_policy_automation_run_counts`)
	if err != nil {
		return fmt.Errorf("drop host_policy_automation_run_counts table: %w", err)
	}

	// Store activity_type as an ENUM, with VARCHAR(255) in the primary key the backfill inserts took twice as long.
	_, err = tx.Exec(`
CREATE TABLE host_policy_automation_run_counts (
	policy_id     INT UNSIGNED NOT NULL,
	activity_type ENUM('installed_app_store_app', 'installed_software', 'ran_script') NOT NULL,
	host_id       INT UNSIGNED NOT NULL,
	success_count INT UNSIGNED NOT NULL DEFAULT 0,
	fail_count    INT UNSIGNED NOT NULL DEFAULT 0,
	PRIMARY KEY (policy_id, activity_type, host_id),
	CONSTRAINT fk_host_policy_automation_run_counts_policy_id FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_policy_automation_run_counts table: %w", err)
	}

	err = backfillHostPolicyAutomationRunCounts(tx)
	if err != nil {
		return err
	}

	// Add the host_id index after the backfill, building it in one pass is faster than updating it for every inserted row.
	_, err = tx.Exec(`ALTER TABLE host_policy_automation_run_counts ADD KEY idx_host_policy_automation_run_counts_host_id (host_id)`)
	if err != nil {
		return fmt.Errorf("add host_policy_automation_run_counts host_id index: %w", err)
	}
	return nil
}

func backfillHostPolicyAutomationRunCounts(tx *sql.Tx) error {
	// Sort each insert in primary key order so rows append to the end of the table instead of splitting random pages.
	_, err := tx.Exec(`
INSERT INTO host_policy_automation_run_counts (policy_id, activity_type, host_id, success_count, fail_count)
SELECT policy_id, 'ran_script', host_id, SUM(exit_code = 0), SUM(exit_code != 0)
FROM host_script_results
WHERE policy_id IS NOT NULL
	AND host_deleted_at IS NULL
	AND canceled = 0
	AND exit_code IS NOT NULL
GROUP BY policy_id, host_id
ORDER BY policy_id, host_id`)
	if err != nil {
		return fmt.Errorf("backfill script runs: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO host_policy_automation_run_counts (policy_id, activity_type, host_id, success_count, fail_count)
SELECT policy_id, 'installed_software', host_id, SUM(execution_status = 'installed'), SUM(execution_status = 'failed_install')
FROM host_software_installs
WHERE policy_id IS NOT NULL
	AND host_deleted_at IS NULL
	AND canceled = 0
	AND uninstall = 0
	AND execution_status IN ('installed', 'failed_install')
GROUP BY policy_id, host_id
ORDER BY policy_id, host_id`)
	if err != nil {
		return fmt.Errorf("backfill software installs: %w", err)
	}

	_, err = tx.Exec(`
INSERT INTO host_policy_automation_run_counts (policy_id, activity_type, host_id, success_count, fail_count)
SELECT policy_id, 'installed_app_store_app', host_id, SUM(verification_at IS NOT NULL), SUM(verification_failed_at IS NOT NULL)
FROM host_vpp_software_installs
WHERE policy_id IS NOT NULL
	AND canceled = 0
	AND (verification_at IS NOT NULL OR verification_failed_at IS NOT NULL)
GROUP BY policy_id, host_id
ORDER BY policy_id, host_id`)
	if err != nil {
		return fmt.Errorf("backfill VPP installs: %w", err)
	}

	return nil
}

func Down_20261019212319(tx *sql.Tx) error {
	return nil
}
