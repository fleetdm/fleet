package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240219133527, Down_20240219133527)
}

func Up_20240219133527(tx *sql.Tx) error {
	stmt := `
ALTER TABLE host_dep_assignments
	-- profile_uuid is the uuid of the enrollment profile that was assigned to the host (which should correspond to an entry in the mdm_apple_setup_assistants table)
	ADD COLUMN profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NULL,
	-- assign_profile_response is the response received for the DEP profile assignment request (e.g., 'success', 'not_accessible', or 'failed')
	ADD COLUMN assign_profile_response VARCHAR(15) COLLATE utf8mb4_unicode_ci NULL;`

	if _, err := tx.Exec(stmt); err != nil {
		return errors.Wrap(err, "alter host_dep_assignments table")
	}

	return nil
}

func Down_20240219133527(tx *sql.Tx) error {
	return nil
}
