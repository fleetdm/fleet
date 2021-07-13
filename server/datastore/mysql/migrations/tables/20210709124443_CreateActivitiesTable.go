package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210709124443, Down_20210709124443)
}

func Up_20210709124443(tx *sql.Tx) error {
	sql := `
		CREATE TABLE IF NOT EXISTS activities (
			id int(10) unsigned NOT NULL AUTO_INCREMENT,
			created_at timestamp DEFAULT CURRENT_TIMESTAMP,
			user_id int(10) unsigned,
			user_name varchar(255),
			activity_type varchar(255) NOT NULL,
			details json DEFAULT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY fk_activities_user_id (user_id) REFERENCES users (id) ON DELETE SET NULL 
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create activities")
	}

	return nil
}

func Down_20210709124443(tx *sql.Tx) error {
	return nil
}
