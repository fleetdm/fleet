package data

import (
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20170314151620, Down_20170314151620)
}

func Up_20170314151620(tx *sql.Tx) error {
	// Fix for osquery not correctly reporting platform for CentOS6
	label_query := `select 1 from os_version where platform = 'centos' or name like '%centos%'`
	sql := `
                UPDATE labels
                SET query = ?, platform = ''
                WHERE name = 'CentOS Linux' AND label_type = ?
`

	_, err := tx.Exec(sql, label_query, fleet.LabelTypeBuiltIn)
	if err != nil {
		return err
	}

	return nil
}

func Down_20170314151620(tx *sql.Tx) error {
	// Not reversible
	return nil
}
