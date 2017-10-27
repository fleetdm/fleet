package data

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20171027173700, Down_20171027173700)
}

func Up_20171027173700(tx *sql.Tx) error {
	sql := "UPDATE decorators SET name=? where query=?"

	rows := []struct {
		name  string
		query string
	}{
		{"Host UUID", "SELECT uuid AS host_uuid FROM system_info;"},
		{"Hostname", "SELECT hostname AS hostname FROM system_info;"},
	}
	for _, row := range rows {
		_, err := tx.Exec(sql, row.name, row.query)
		if err != nil {
			return err
		}
	}
	return nil
}

func Down_20171027173700(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE decorators SET name='' WHERE built_in = TRUE")
	return err
}
