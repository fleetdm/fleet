package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func init() {
	MigrationClient.AddMigration(Up_20260121100000, Down_20260121100000)
}

func Up_20260121100000(tx *sql.Tx) error {
	// Add encrypted column for Google Calendar API key
	_, err := tx.Exec(`
		ALTER TABLE app_config_json
		ADD COLUMN google_calendar_api_key_encrypted BLOB DEFAULT NULL
	`)
	if err != nil {
		return fmt.Errorf("failed to add google_calendar_api_key_encrypted column: %w", err)
	}

	// Check if there's existing Google Calendar configuration that needs to be migrated.
	// We need to parse the JSON to check if google_calendar has any api_key_json data.
	var jsonValue []byte
	err = tx.QueryRow(`SELECT json_value FROM app_config_json LIMIT 1`).Scan(&jsonValue)
	if err != nil {
		if err == sql.ErrNoRows {
			// No app config yet, nothing to migrate
			return nil
		}
		return fmt.Errorf("failed to read app_config_json: %w", err)
	}

	// Parse the JSON to check for existing Google Calendar config
	var config struct {
		Integrations struct {
			GoogleCalendar []struct {
				ApiKey map[string]string `json:"api_key_json"`
			} `json:"google_calendar"`
		} `json:"integrations"`
	}
	if err := json.Unmarshal(jsonValue, &config); err != nil {
		return fmt.Errorf("failed to unmarshal app_config_json: %w", err)
	}

	// Check if there's any Google Calendar integration with an API key
	hasApiKey := false
	for _, gc := range config.Integrations.GoogleCalendar {
		if len(gc.ApiKey) > 0 {
			hasApiKey = true
			break
		}
	}

	if !hasApiKey {
		// No existing Google Calendar API key to migrate
		return nil
	}

	// Queue a worker job to encrypt the existing plaintext API key.
	// We can't do it here because migrations don't have access to the server private key.
	const (
		jobName        = "db_migration"
		taskName       = "encrypt_google_calendar_api_key"
		jobStateQueued = "queued"
	)

	type migrateArgs struct {
		Task string `json:"task"`
	}
	argsJSON, err := json.Marshal(migrateArgs{Task: taskName})
	if err != nil {
		return fmt.Errorf("failed to JSON marshal the job arguments: %w", err)
	}

	// Use a fixed timestamp for stable schema.sql
	const query = `
INSERT INTO jobs (
    name,
    args,
    state,
    error,
    not_before,
    created_at,
    updated_at
)
VALUES (?, ?, ?, '', ?, ?, ?)
`
	ts := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	if _, err := tx.Exec(query, jobName, argsJSON, jobStateQueued, ts, ts, ts); err != nil {
		return fmt.Errorf("failed to insert worker job: %w", err)
	}

	return nil
}

func Down_20260121100000(tx *sql.Tx) error {
	return nil
}
