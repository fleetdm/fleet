package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20211116184029, Down_20211116184029)
}

func Up_20211116184029(tx *sql.Tx) error {
	_, err := tx.Exec("CREATE INDEX policy_membership_history_groupby_idx on policy_membership_history (host_id, policy_id, id)")
	if err != nil {
		return errors.Wrap(err, "create policy_membership_history_groupby_idx")
	}

	return nil
}

func Down_20211116184029(tx *sql.Tx) error {
	return nil
}
