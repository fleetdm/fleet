package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20231121140727, Down_20231121140727)
}

func Up_20231121140727(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mock_profiles (
	-- 37 and not 36, to leave 1 char for 'A' or 'W' prefix (Apple/Windows)
	profile_uuid VARCHAR(37) NOT NULL DEFAULT '' PRIMARY KEY,
	profile_id INT(10) NOT NULL DEFAULT 0,
	name VARCHAR(255) NOT NULL DEFAULT '',

	UNIQUE KEY (profile_id)
)`)
	if err != nil {
		return err
	}
	// And then to insert into that table, must be something like that:
	// INSERT INTO mock_profiles
	//  (profile_uuid, profile_id, name)
	// SELECT
	//   concat('A', uuid()),
	//   coalesce(max(profile_id), 0) + 1,
	//   'test'
	// FROM mock_profiles;

	return nil
}

func Down_20231121140727(tx *sql.Tx) error {
	return nil
}
