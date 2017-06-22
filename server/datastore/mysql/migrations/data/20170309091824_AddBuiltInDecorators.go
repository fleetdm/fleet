package data

import (
	"database/sql"

	"github.com/kolide/fleet/server/kolide"
)

func init() {
	MigrationClient.AddMigration(Up_20170309091824, Down_20170309091824)
}

func Up_20170309091824(tx *sql.Tx) error {
	sql := "INSERT INTO decorators (" +
		"`type`, " +
		"`query`, " +
		"`built_in`, " +
		"`interval`" +
		") VALUES ( ?, ?, TRUE, 0 )"

	rows := []struct {
		t kolide.DecoratorType
		q string
	}{
		{kolide.DecoratorLoad, "SELECT uuid AS host_uuid FROM system_info;"},
		{kolide.DecoratorLoad, "SELECT hostname AS hostname FROM system_info;"},
	}
	for _, row := range rows {
		_, err := tx.Exec(sql, row.t, row.q)
		if err != nil {
			return err
		}
	}
	return nil
}

func Down_20170309091824(tx *sql.Tx) error {
	_, err := tx.Exec("DELETE FROM decorators WHERE built_in = TRUE")
	return err
}
