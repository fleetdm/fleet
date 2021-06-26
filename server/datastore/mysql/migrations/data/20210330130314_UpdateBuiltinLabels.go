package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210330130314, Down_20210330130314)
}

func Up_20210330130314(tx *sql.Tx) error {
	// Update labels to set platform to empty. Previously the platform meant
	// that if a host changed platform (say by installing a new OS and still
	// having the same hardware UUID) the old label query would never run again
	// and a host could show up in multiple of the built-in platform labels.
	sql := "UPDATE labels SET platform = '' WHERE label_type = ?"
	if _, err := tx.Exec(sql, fleet.LabelTypeBuiltIn); err != nil {
		return errors.Wrap(err, "update labels")
	}

	// Insert Red Hat label
	sql = `
		INSERT INTO labels (
			name,
			description,
			query,
			platform,
			label_type,
			label_membership_type
		) VALUES (?, ?, ?, ?, ?, ?)
`
	if _, err := tx.Exec(
		sql,
		"Red Hat Linux",
		"All Red Hat Enterprise Linux hosts",
		"SELECT 1 FROM os_version WHERE name LIKE '%red hat%'",
		"",
		fleet.LabelTypeBuiltIn,
		fleet.LabelMembershipTypeDynamic,
	); err != nil {
		return errors.Wrap(err, "add red hat label")
	}

	return nil
}

func Down_20210330130314(tx *sql.Tx) error {
	return nil
}
