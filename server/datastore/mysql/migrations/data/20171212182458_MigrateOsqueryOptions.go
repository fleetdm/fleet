package data

import (
	"database/sql"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20171212182458, Down_20171212182458)
}

type configForExport struct {
	Options   map[string]interface{} `json:"options"`
	FilePaths map[string][]string    `json:"file_paths,omitempty"`
}

type yamlObjForExport struct {
	ApiVersion string        `json:"apiVersion"`
	Kind       string        `json:"kind"`
	Spec       specForExport `json:"spec"`
}

type specForExport struct {
	Config json.RawMessage `json:"config"`
}

func Up_20171212182458(tx *sql.Tx) error {
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	// Get basic osquery options
	query := `
		SELECT *
		FROM options
		WHERE value != "null"
	`
	// Intentionally initialize empty instead of nil so that we generate a
	// config with empty options rather than a null value.
	var opts []kolide.Option
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
	rows, err := txx.Query(query)
	if err != nil && err != sql.ErrNoRows {
		return errors.Wrap(err, "retrieving fim paths")
	}
	fimConfig := kolide.FIMSections{}
	for rows.Next() {
		var sectionName, fileName string
		err = rows.Scan(&sectionName, &fileName)
		if err != nil {
			return errors.Wrap(err, "retrieving path for fim section")
		}
		fimConfig[sectionName] = append(fimConfig[sectionName], fileName)
	}

	// Create config JSON
	config := configForExport{
		Options:   optConfig,
		FilePaths: fimConfig,
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
	if _, err = txx.Exec(query, kolide.OptionOverrideTypeDefault, "", string(confJSON)); err != nil {
		return errors.Wrap(err, "saving converted options")
	}

	return nil
}

func Down_20171212182458(tx *sql.Tx) error {
	// No down migration
	return nil
}
