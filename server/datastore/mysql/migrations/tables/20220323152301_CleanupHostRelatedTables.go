package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220323152301, Down_20220323152301)
}

func Up_20220323152301(tx *sql.Tx) error {
	var hostRefs = []string{
		"host_seen_times",
		"host_software",
		"host_users",
		"host_emails",
		"host_additional",
		"scheduled_query_stats",
		"label_membership",
		"policy_membership",
		"host_mdm",
		"host_munki_info",
		"host_device_auth",
	}

	const delStmt = `
    DELETE target
    FROM %s target
    WHERE NOT EXISTS (
      SELECT 1
      FROM
        hosts h
      WHERE
        h.id = target.host_id
    )`

	for _, hostRef := range hostRefs {
		_, err := tx.Exec(fmt.Sprintf(delStmt, hostRef))
		if err != nil {
			return errors.Wrapf(err, "delete from %s", hostRef)
		}
	}
	return nil
}

func Down_20220323152301(tx *sql.Tx) error {
	return nil
}
