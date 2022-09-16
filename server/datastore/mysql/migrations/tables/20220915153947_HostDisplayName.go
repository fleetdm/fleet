package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220915153947, Down_20220915153947)
}

func Up_20220915153947(tx *sql.Tx) error {
	for _, change := range []struct{ name, sql string }{
		{"delete index", `ALTER TABLE hosts DROP INDEX hosts_search`},
		{"create index", `CREATE FULLTEXT INDEX hosts_search ON hosts(hostname, uuid, computer_name)`},
		{"new table", `
			CREATE TABLE hosts_display_name (
			    host_id int(10) unsigned NOT NULL,
			    display_name varchar(255) NOT NULL,
			    PRIMARY KEY (host_id),
			    FULLTEXT KEY (display_name)
			);
		`},
		{"migrate data", `
			INSERT INTO hosts_display_name (
				SELECT id host_id, IF(computer_name="", hostname, computer_name) display_name FROM hosts
			)
		`},
		{"insert trigger", `
				CREATE TRIGGER host_display_name_insert AFTER INSERT ON hosts FOR EACH ROW
				    INSERT INTO hosts_display_name (host_id, display_name) VALUES (NEW.id, IF(NEW.computer_name="", NEW.hostname, NEW.computer_name));
		`},
		{"update trigger", `
				CREATE TRIGGER host_display_name_update AFTER UPDATE ON hosts FOR EACH ROW
				    UPDATE hosts_display_name SET display_name = IF(NEW.computer_name="", NEW.hostname, NEW.computer_name)
				        WHERE NEW.id = OLD.id
				          AND NEW.id = host_id
				          AND IF(OLD.computer_name="", OLD.hostname, OLD.computer_name) != IF(NEW.computer_name="", NEW.hostname, NEW.computer_name);
		`},
		{"delete", `
				CREATE TRIGGER host_display_name_delete AFTER DELETE ON hosts FOR EACH ROW
				    DELETE FROM hosts_display_name WHERE host_id = OLD.id;
		`},
	} {
		if _, err := tx.Exec(change.sql); err != nil {
			return errors.Wrapf(err, "upHostDisplayName: %s", change.name)
		}

	}
	return nil
}

func Down_20220915153947(tx *sql.Tx) error {
	return nil
}
