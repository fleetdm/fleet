package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210506095025, Down_20210506095025)
}

func Up_20210506095025(tx *sql.Tx) error {
	sql := `
		CREATE TABLE scheduled_query_stats (
			host_id int unsigned NOT NULL,
			scheduled_query_id int unsigned NOT NULL,
			average_memory int,
			denylisted tinyint(1),
			executions int,
			schedule_interval int,
			last_executed timestamp,
			output_size int,
			system_time int,
			user_time int,
			wall_time int,
			PRIMARY KEY (host_id, scheduled_query_id),
			FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE ON UPDATE CASCADE,
			FOREIGN KEY (scheduled_query_id) REFERENCES scheduled_queries (id) ON DELETE CASCADE ON UPDATE CASCADE
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create scheduled_query_stats")
	}
	return nil
}

func Down_20210506095025(tx *sql.Tx) error {
	return nil
}
