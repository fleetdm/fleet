package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240320145650, Down_20240320145650)
}

func Up_20240320145650(tx *sql.Tx) error {
	// This migration is to re-generate and re-register with Apple the DEP
	// enrollment profile(s) so that await_device_configured is set to true.
	// We do this by doing the equivalent of:
	//
	// 	 worker.QueueMacosSetupAssistantJob(ctx, ds, logger,
	//     worker.MacosSetupAssistantUpdateAllProfiles, nil)
	//
	// but without calling that function, in case the code changes in the future,
	// breaking this migration. Instead we insert directly the job in the
	// database, and the worker will process it shortly after Fleet restarts.

	const (
		jobName        = "macos_setup_assistant"
		taskName       = "update_all_profiles"
		jobStateQueued = "queued"
	)

	type macosSetupAssistantArgs struct {
		Task              string   `json:"task"`
		TeamID            *uint    `json:"team_id,omitempty"`
		HostSerialNumbers []string `json:"host_serial_numbers,omitempty"`
	}
	argsJSON, err := json.Marshal(macosSetupAssistantArgs{Task: taskName})
	if err != nil {
		return fmt.Errorf("failed to JSON marshal the job arguments: %w", err)
	}

	const query = `
INSERT INTO jobs (
    name,
    args,
    state,
		error,
    not_before
)
VALUES (?, ?, ?, '', NOW())
`
	if _, err := tx.Exec(query, jobName, argsJSON, jobStateQueued); err != nil {
		return fmt.Errorf("failed to insert worker job: %w", err)
	}
	return nil
}

func Down_20240320145650(tx *sql.Tx) error {
	return nil
}
