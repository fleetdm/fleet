package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230227095350, Down_20230227095350)
}

func Up_20230227095350(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_apple_script_status (
	status VARCHAR(20) PRIMARY KEY
)`)
	if err != nil {
		return err
	}

	// Using the same status values as mdm_apple_delivery_status, except
	// "ran" instead of "applied" on success.
	_, err = tx.Exec(`
INSERT INTO mdm_apple_script_status (status)
VALUES ('failed'), ('ran'), ('pending')
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
CREATE TABLE host_mdm_apple_scripts (
	script_id   INT(10) UNSIGNED NOT NULL,
	host_id     INT(10) UNSIGNED NOT NULL,
	status      VARCHAR(20) DEFAULT NULL,
	-- output of the last execution (if status is not 'failed' or 'ran', then
	-- the output is for the previous execution), limited to 10 000 chars.
	output      TEXT NOT NULL DEFAULT '',
	created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at  TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	PRIMARY KEY (host_id, script_id),
	FOREIGN KEY (status) REFERENCES mdm_apple_script_status (status) ON UPDATE CASCADE
)`)
	return err
}

func Down_20230227095350(tx *sql.Tx) error {
	return nil
}
