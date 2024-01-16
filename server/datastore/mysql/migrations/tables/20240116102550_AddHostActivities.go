package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240116102550, Down_20240116102550)
}

func Up_20240116102550(tx *sql.Tx) error {
	// add user_id to host_script_results so that we can display the "actor" in
	// the upcoming activities for a host (who requested the script execution).
	// Unlike for activities, we don't copy over all the user's information,
	// instead we just link to the existing user and set it to NULL if the user
	// gets deleted. This is because the script executions are expected to run
	// soon after the request is made, it should be a rare occurrence for the
	// requesting user to be deleted before it runs.
	//
	// sync_request indicates if the script execution was requested via the
	// synchronous API. We need this information to generate the proper activity
	// details later on when the results are received.
	const alterStmt = `
		ALTER TABLE host_script_results
		ADD COLUMN user_id INT(10) UNSIGNED DEFAULT NULL,
		ADD COLUMN sync_request TINYINT(1) NOT NULL DEFAULT '0',
		ADD CONSTRAINT fk_host_script_results_user_id FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;
	`
	if _, err := tx.Exec(alterStmt); err != nil {
		return fmt.Errorf("add user_id to host_script_results: %w", err)
	}

	// Note that we don't create FKs to hosts for performance reasons (ingestion
	// of data at scale). FK is created for activities, those entries should be
	// deleted if for some reason the activity is deleted.
	const hostActivitiesStmt = `
    CREATE TABLE IF NOT EXISTS host_activities (
			host_id     INT(10) UNSIGNED NOT NULL,
			activity_id INT(10) UNSIGNED NOT NULL,

			PRIMARY KEY (host_id, activity_id),
			FOREIGN KEY fk_host_activities_activity_id (activity_id) REFERENCES activities (id) ON DELETE CASCADE
    );
	`
	if _, err := tx.Exec(hostActivitiesStmt); err != nil {
		return errors.Wrap(err, "create host_activities table")
	}

	return nil
}

func Down_20240116102550(tx *sql.Tx) error {
	return nil
}
