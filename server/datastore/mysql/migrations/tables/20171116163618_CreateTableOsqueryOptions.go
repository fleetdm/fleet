package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20171116163618, Down_20171116163618)
}

func Up_20171116163618(tx *sql.Tx) error {
	sqlStatement := "CREATE TABLE `osquery_options` (" +
		"`id` INT(10) UNSIGNED NOT NULL AUTO_INCREMENT," +
		"`override_type` INT(1) NOT NULL, " +
		"`override_identifier` VARCHAR(255) NOT NULL DEFAULT ''," +
		"`options` JSON NOT NULL," +
		"PRIMARY KEY (`id`)" +
		") ENGINE=InnoDB DEFAULT CHARSET=utf8;"
	_, err := tx.Exec(sqlStatement)
	if err != nil {
		return errors.Wrap(err, "create table osquery_options")
	}

	// Check whether there are existing options to migrate, or we should insert
	// a new default set of options
	var count int
	err = tx.QueryRow("SELECT count(1) FROM options").Scan(&count)
	if err != nil {
		return errors.Wrap(err, "get options count")
	}

	if count > 0 {
		// Migrate existing options
		err = migrateOptions(tx)
		if err != nil {
			return errors.Wrap(err, "migrate options")
		}
	} else {
		// Insert default options
		_, err = tx.Exec("INSERT INTO `osquery_options`" +
			"(override_type, override_identifier, options)" +
			`VALUES (0, '', '{"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/v1/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}')`,
		)
		if err != nil {
			return errors.Wrap(err, "insert options")
		}
	}

	return nil
}

func migrateOptions(tx *sql.Tx) error {
	// This migration uses the deprecated types in deprecated_types.go

	type configForExport struct {
		Options    map[string]interface{} `json:"options"`
		FilePaths  map[string][]string    `json:"file_paths,omitempty"`
		Decorators decorators             `json:"decorators"`
	}

	// Migrate pre fleetctl osquery options to the new osquery options
	// formats.
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Get basic osquery options
	query := `
		SELECT *
		FROM options
		WHERE value != 'null'
	`
	// Intentionally initialize empty instead of nil so that we generate a
	// config with empty options rather than a null value.
	var opts []option
	if err := txx.Select(&opts, query); err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "getting options")
	}
	optConfig := map[string]interface{}{}
	for _, opt := range opts {
		optConfig[opt.Name] = opt.GetValue()
	}

	// Get FIM paths from fim table
	query = `
		SELECT fim.section_name, mf.file
		FROM file_integrity_monitorings AS fim
		INNER JOIN file_integrity_monitoring_files AS mf
		ON (fim.id = mf.file_integrity_monitoring_id)
	`
	rows, err := txx.Query(query) //nolint
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "retrieving fim paths")
	}
	fimConfig := map[string][]string{}
	for rows.Next() {
		var sectionName, fileName string
		err = rows.Scan(&sectionName, &fileName)
		if err != nil {
			return errors.Wrap(err, "retrieving path for fim section")
		}
		fimConfig[sectionName] = append(fimConfig[sectionName], fileName)
	}

	query = `
		SELECT *
		FROM decorators
		ORDER by built_in DESC, name ASC
	`
	var decs []*decorator
	err = txx.Select(&decs, query)
	if err != nil {
		return errors.Wrap(err, "retrieving decorators")
	}

	decConfig := decorators{
		Interval: make(map[string][]string),
	}
	for _, dec := range decs {
		switch dec.Type {
		case decoratorLoad:
			decConfig.Load = append(decConfig.Load, dec.Query)
		case decoratorAlways:
			decConfig.Always = append(decConfig.Always, dec.Query)
		case decoratorInterval:
			key := fmt.Sprint(dec.Interval)
			decConfig.Interval[key] = append(decConfig.Interval[key], dec.Query)
		default:
			fmt.Printf("Unable to migrate decorator. Please migrate manually: '%s'\n", dec.Query)
		}
	}

	// Create config JSON
	config := configForExport{
		Options:    optConfig,
		FilePaths:  fimConfig,
		Decorators: decConfig,
	}
	confJSON, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshal config JSON")
	}

	// Save config JSON
	query = `
		INSERT INTO osquery_options (
			override_type, override_identifier, options
		) VALUES (?, ?, ?)
	`
	if _, err = txx.Exec(query, 0, "", string(confJSON)); err != nil {
		return errors.Wrap(err, "saving converted options")
	}

	return nil
}

func Down_20171116163618(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `osquery_options`;")
	return err
}
