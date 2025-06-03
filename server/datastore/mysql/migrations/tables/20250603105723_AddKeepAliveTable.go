package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250603105723, Down_20250603105723)
}

func Up_20250603105723(tx *sql.Tx) error {
	stmt := `CREATE TABLE keep_alive (
	last_server_instance_checkin DATETIME(6) NOT NULL DEFAULT NOW(6),
	PRIMARY KEY (last_server_instance_checkin)
);
	INSERT INTO keep_alive VALUE (NOW());
`
	// TODO - initial value NULL

	_, err := tx.Exec(stmt)
	if err != nil {
		return fmt.Errorf("creating keep_alive table: %w", err)
	}
	return nil
}

func Down_20250603105723(tx *sql.Tx) error {
	return nil
}
