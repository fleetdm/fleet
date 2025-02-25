package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240228082706, Down_20240228082706)
}

func Up_20240228082706(tx *sql.Tx) error {
	stmt := `
ALTER TABLE host_dep_assignments
	-- profile_uuid is the uuid of the enrollment profile that was assigned to the host (which should correspond to an entry in the mdm_apple_setup_assistants table)
	ADD COLUMN profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NULL,
	
	-- assign_profile_response is the response received for the DEP profile assignment request (e.g., 'SUCCESS', 'NOT_ACCESSIBLE', or 'FAILED')
	ADD COLUMN assign_profile_response VARCHAR(15) COLLATE utf8mb4_unicode_ci NULL,
	
	-- response_updated_at is the time the most recent DEP profile assignment response was received
	ADD COLUMN response_updated_at TIMESTAMP NULL,

	-- retry_job_id is the id of job to retry a failed DEP profile assignment
	ADD COLUMN retry_job_id int(10) UNSIGNED NOT NULL DEFAULT 0,

	ADD INDEX idx_hdep_response (assign_profile_response, response_updated_at);`

	if _, err := tx.Exec(stmt); err != nil {
		return errors.Wrap(err, "alter host_dep_assignments table")
	}

	return nil
}

func Down_20240228082706(tx *sql.Tx) error {
	return nil
}
