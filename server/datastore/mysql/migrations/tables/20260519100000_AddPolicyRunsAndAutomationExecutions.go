package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260519100000, Down_20260519100000)
}

func Up_20260519100000(tx *sql.Tx) error {
	// policy_runs records a policy run, old_status stores the policy status
	// before the run and new_status stores the status after the run.
	//
	// We currently only store the 'latest' run result of the policy run, i.e.,
	// either the first time the policy flipped to pass or the
	// first time the policy failed or flipped to failure. This is why we have
	// the unique contraint in place.
	//
	// consecutive_failures stores, welp the number of consecutive failures.
	if _, err := tx.Exec(`
	CREATE TABLE policy_runs (
		id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		policy_id            INT UNSIGNED NOT NULL,
		host_id              INT UNSIGNED NOT NULL,
		old_status           TINYINT(1) NULL,
		new_status           TINYINT(1) NOT NULL,
		consecutive_failures INT UNSIGNED NOT NULL DEFAULT 0,
		created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY uk_policy_run (policy_id, host_id),
		KEY idx_policy_runs_host_id (host_id),
		CONSTRAINT fk_policy_runs_policy FOREIGN KEY (policy_id) REFERENCES policies (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		return fmt.Errorf("create policy_runs table: %w", err)
	}

	// policy_runs_to_policy_automation_executions is a joint table relating
	// each policy run to the automation batch(es) that processed it. The PK
	// includes batch_id so that re-dispatches across cron ticks (e.g. a
	// webhook POST that 5xx'd, was re-read on the next tick, and dispatched
	// again with a new batch_id) accumulate one row per attempt — full audit
	// trail instead of a unique-key violation.
	if _, err := tx.Exec(`
	CREATE TABLE policy_runs_to_policy_automation_executions (
		policy_run_id   BIGINT UNSIGNED NOT NULL,
		automation_type ENUM('webhook','jira','zendesk','calendar','conditional_access') NOT NULL,
		batch_id        BINARY(16) NOT NULL,
		created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (policy_run_id, automation_type, batch_id),
		KEY idx_batch_id (batch_id),
		CONSTRAINT fk_policy_runs_join_tbl_policy FOREIGN KEY (policy_run_id) REFERENCES policy_runs (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		return fmt.Errorf("create policy_runs_to_policy_automation_executions table: %w", err)
	}

	// policy_automation_executions records each attempt at a failing-policy
	// automation.
	// Almost all automations are batched (one webhook POST covering N hosts, one Jira
	// ticket covering N hosts, one calendar event covering N policies, etc), so that's why we
	// need the concept of a batch_id on this table.
	if _, err := tx.Exec(`
	CREATE TABLE policy_automation_executions (
		batch_id        BINARY(16) NOT NULL PRIMARY KEY,
		status          ENUM('pending','success','failure') NOT NULL DEFAULT 'pending',
		error_message   TEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
		created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`); err != nil {
		return fmt.Errorf("create policy_automation_executions table: %w", err)
	}

	// Existing per-host automation result tables get a nullable policy_run_id
	// stamp so the policy status page can attribute the result row to the
	// originating run. ON DELETE SET NULL preserves the result history when
	// policy reset removes the policy_runs row.
	//
	// NOTE: host_calendar_events does NOT get a policy_run_id column.
	// Calendar automation outcomes are tracked exclusively via
	// policy_automation_executions rows of type='calendar' linked through
	// policy_run_id, because a single host_calendar_events row reacts to
	// MANY failing policies (the calendar event covers all of them with a
	// single event body that lists each failure).
	if _, err := tx.Exec(`
	ALTER TABLE host_script_results
		ADD COLUMN policy_run_id BIGINT UNSIGNED NULL DEFAULT NULL,
		ADD KEY idx_host_script_results_policy_run (policy_run_id),
		ADD CONSTRAINT fk_host_script_results_policy_run
			FOREIGN KEY (policy_run_id) REFERENCES policy_runs (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("add policy_run_id to host_script_results: %w", err)
	}

	if _, err := tx.Exec(`
	ALTER TABLE host_software_installs
		ADD COLUMN policy_run_id BIGINT UNSIGNED NULL DEFAULT NULL,
		ADD KEY idx_host_software_installs_policy_run (policy_run_id),
		ADD CONSTRAINT fk_host_software_installs_policy_run
			FOREIGN KEY (policy_run_id) REFERENCES policy_runs (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("add policy_run_id to host_software_installs: %w", err)
	}

	// The upcoming-activities tables carry policy_run_id alongside the
	// existing policy_id so the activateNext*Activity promotion can copy it
	// forward into host_script_results / host_software_installs.
	if _, err := tx.Exec(`
	ALTER TABLE script_upcoming_activities
		ADD COLUMN policy_run_id BIGINT UNSIGNED NULL DEFAULT NULL,
		ADD KEY idx_script_upcoming_activities_policy_run (policy_run_id),
		ADD CONSTRAINT fk_script_upcoming_activities_policy_run
			FOREIGN KEY (policy_run_id) REFERENCES policy_runs (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("add policy_run_id to script_upcoming_activities: %w", err)
	}

	if _, err := tx.Exec(`
	ALTER TABLE software_install_upcoming_activities
		ADD COLUMN policy_run_id BIGINT UNSIGNED NULL DEFAULT NULL,
		ADD KEY idx_software_install_upcoming_activities_policy_run (policy_run_id),
		ADD CONSTRAINT fk_software_install_upcoming_activities_policy_run
			FOREIGN KEY (policy_run_id) REFERENCES policy_runs (id) ON DELETE SET NULL
	`); err != nil {
		return fmt.Errorf("add policy_run_id to software_install_upcoming_activities: %w", err)
	}

	return nil
}

func Down_20260519100000(tx *sql.Tx) error {
	return nil
}
