package mysql

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type optionsRow struct {
	ID                 int                       `db:"id"`
	OverrideType       kolide.OptionOverrideType `db:"override_type"`
	OverrideIdentifier string                    `db:"override_identifier"`
	Options            string                    `db:"options"`
}

func (d *Datastore) ApplyOptions(spec *kolide.OptionsSpec) (err error) {
	tx, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "begin ApplyOptions transaction")
	}

	defer func() {
		if err != nil {
			rbErr := tx.Rollback()
			// It seems possible that there might be a case in
			// which the error we are dealing with here was thrown
			// by the call to tx.Commit(), and the docs suggest
			// this call would then result in sql.ErrTxDone.
			if rbErr != nil && rbErr != sql.ErrTxDone {
				panic(fmt.Sprintf("got err '%s' rolling back after err '%s'", rbErr, err))
			}
		}
	}()

	// Clear all the existing options
	_, err = tx.Exec("DELETE FROM osquery_options")
	if err != nil {
		return errors.Wrap(err, "delete existing options")
	}

	// Save new options
	sql := `
		INSERT INTO osquery_options (
			override_type, override_identifier, options
		) VALUES (?, ?, ?)
	`

	// Default options
	_, err = tx.Exec(sql, kolide.OptionOverrideTypeDefault, "", string(spec.Config))
	if err != nil {
		return errors.Wrap(err, "saving default config")
	}

	// Platform overrides
	for platform, opts := range spec.Overrides.Platforms {
		_, err = tx.Exec(sql, kolide.OptionOverrideTypePlatform, platform, string(opts))
		if err != nil {
			return errors.Wrapf(err, "saving %s platform config", platform)
		}

	}

	// Success!
	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "commit ApplyOptions transaction")
	}

	return nil
}

func (d *Datastore) GetOptions() (*kolide.OptionsSpec, error) {
	var rows []optionsRow
	if err := d.db.Select(&rows, "SELECT * FROM osquery_options"); err != nil {
		return nil, errors.Wrap(err, "selecting options")
	}

	spec := &kolide.OptionsSpec{
		Overrides: kolide.OptionsOverrides{
			Platforms: make(map[string]json.RawMessage),
		},
	}
	for _, row := range rows {
		switch row.OverrideType {
		case kolide.OptionOverrideTypeDefault:
			spec.Config = json.RawMessage(row.Options)

		case kolide.OptionOverrideTypePlatform:
			spec.Overrides.Platforms[row.OverrideIdentifier] = json.RawMessage(row.Options)

		default:
			level.Info(d.logger).Log(
				"err", "ignoring unkown override type",
				"type", row.OverrideType,
			)
		}
	}

	return spec, nil
}

func (d *Datastore) OptionsForPlatform(platform string) (json.RawMessage, error) {
	// SQL uses a custom ordering function to return the single correct
	// config with the highest precedence override (the FIELD function
	// defines this ordering). If there is no override, it returns the
	// default.
	sql := `
		SELECT * FROM osquery_options
		WHERE override_type = ? OR
			(override_type = ? AND override_identifier = ?)
		ORDER BY FIELD(override_type, ?, ?)
		LIMIT 1
		`
	var row optionsRow
	err := d.db.Get(
		&row, sql,
		kolide.OptionOverrideTypeDefault,
		kolide.OptionOverrideTypePlatform, platform,
		// Order of the following arguments defines precedence of
		// overrides.
		kolide.OptionOverrideTypePlatform, kolide.OptionOverrideTypeDefault,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving osquery options for platform '%s'", platform)
	}

	return json.RawMessage(row.Options), nil
}
