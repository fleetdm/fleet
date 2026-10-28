package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261028232258, Down_20261028232258)
}

func Up_20261028232258(tx *sql.Tx) error {
	// backfill marks a session that had been running for a while when Fleet first stored it, whose calls are read a
	// few at a time. actions and event_count through events_empty_reads summarize the session's stored events, so
	// picking the sessions that still need events at each detail refresh reads only this table.

	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS host_ai_sessions (
	id                            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	host_id                       INT UNSIGNED NOT NULL,
	agent                         VARCHAR(32) NOT NULL,
	agent_session_id              VARCHAR(255) NOT NULL,
	agent_version                 VARCHAR(64) NOT NULL DEFAULT '',
	username                      VARCHAR(255) NOT NULL DEFAULT '',
	account_identifier            VARCHAR(255) NOT NULL DEFAULT '',
	model                         VARCHAR(255) NOT NULL DEFAULT '',
	project_path                  TEXT NOT NULL,
	git_repository                VARCHAR(255) NOT NULL DEFAULT '',
	git_branch                    VARCHAR(255) NOT NULL DEFAULT '',
	started_at                    DATETIME(6) NULL DEFAULT NULL,
	ended_at                      DATETIME(6) NULL DEFAULT NULL,
	user_message_count            INT UNSIGNED NOT NULL DEFAULT 0,
	assistant_message_count       INT UNSIGNED NOT NULL DEFAULT 0,
	tool_call_count               INT UNSIGNED NOT NULL DEFAULT 0,
	input_token_count             BIGINT UNSIGNED NOT NULL DEFAULT 0,
	output_token_count            BIGINT UNSIGNED NOT NULL DEFAULT 0,
	cache_read_token_count        BIGINT UNSIGNED NOT NULL DEFAULT 0,
	cache_write_token_count       BIGINT UNSIGNED NOT NULL DEFAULT 0,
	actions                       SET('read', 'write', 'execute', 'search', 'network', 'mcp', 'other') NOT NULL DEFAULT '',
	transcript_path               TEXT NOT NULL,
	transcript_size               BIGINT UNSIGNED NOT NULL DEFAULT 0,
	backfill                        TINYINT(1) NOT NULL DEFAULT 0,
	event_count                     INT UNSIGNED NOT NULL DEFAULT 0,
	first_undecided_tool_call_index INT UNSIGNED NULL DEFAULT NULL,
	events_ended_at                 DATETIME(6) NULL DEFAULT NULL,
	events_empty_reads              TINYINT UNSIGNED NOT NULL DEFAULT 0,
	created_at                      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at                      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (id),
	UNIQUE KEY idx_host_ai_sessions_host_agent_session (host_id, agent, agent_session_id),
	KEY idx_host_ai_sessions_host_ended_at (host_id, ended_at),
	KEY idx_host_ai_sessions_started_at (started_at),
	KEY idx_host_ai_sessions_updated_at_started_at (updated_at, started_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_ai_sessions table: %w", err)
	}

	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS host_ai_session_events (
	id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
	ai_session_id   BIGINT UNSIGNED NOT NULL,
	tool_call_index INT UNSIGNED NOT NULL,
	action          VARCHAR(16) NOT NULL DEFAULT '',
	target          VARCHAR(256) NOT NULL DEFAULT '',
	decision        VARCHAR(16) NOT NULL DEFAULT '',
	executed_at     DATETIME(6) NULL DEFAULT NULL,
	payload         JSON NOT NULL,
	content_length  INT UNSIGNED NOT NULL DEFAULT 0,
	truncated       TINYINT(1) NOT NULL DEFAULT 0,

	created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	PRIMARY KEY (id),
	UNIQUE KEY idx_host_ai_session_events_session_call (ai_session_id, tool_call_index),
	CONSTRAINT fk_host_ai_session_events_ai_session_id FOREIGN KEY (ai_session_id) REFERENCES host_ai_sessions (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("create host_ai_session_events table: %w", err)
	}
	return nil
}

func Down_20261028232258(tx *sql.Tx) error {
	return nil
}
