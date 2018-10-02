package mysql

import (
	"database/sql"

	"github.com/kolide/fleet/server/datastore/internal/appstate"
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

// ResetOptions note we use named return values so we can preserve and return
// errors in our defer function
func (d *Datastore) ResetOptions() (opts []kolide.Option, err error) {
	// Atomically remove all existing options, reset auto increment so id's will be the
	// same as original defaults, and re-insert defaults in option table.
	var txn *sql.Tx
	txn, err = d.db.Begin()
	if err != nil {
		return nil, errors.Wrap(err, "reset options begin transaction")
	}

	defer func() {
		if err != nil {
			if txErr := txn.Rollback(); txErr != nil {
				err = errors.Wrapf(err, "reset options failed, transaction rollback failed with error: %s", txErr)
			}
		}
	}()
	_, err = txn.Exec("DELETE FROM options")
	if err != nil {
		return nil, errors.Wrap(err, "deleting options in reset options")
	}
	// Reset auto increment
	_, err = txn.Exec("ALTER TABLE `options` AUTO_INCREMENT = 1")
	if err != nil {
		return nil, errors.Wrap(err, "resetting auto increment counter in reset options")
	}
	sqlStatement := `
		INSERT INTO options (
			name,
			type,
			value,
			read_only
		) VALUES (?, ?, ?, ?)
	`
	for _, defaultOpt := range appstate.Options() {
		opt := kolide.Option{
			Name:     defaultOpt.Name,
			ReadOnly: defaultOpt.ReadOnly,
			Type:     defaultOpt.Type,
			Value: kolide.OptionValue{
				Val: defaultOpt.Value,
			},
		}
		dbResponse, err := txn.Exec(
			sqlStatement,
			opt.Name,
			opt.Type,
			opt.Value,
			opt.ReadOnly,
		)
		if err != nil {
			return nil, errors.Wrap(err, "inserting default option in reset options")
		}
		id, err := dbResponse.LastInsertId()
		if err != nil {
			return nil, errors.Wrap(err, "fetching id in reset options")
		}
		opt.ID = uint(id)
		opts = append(opts, opt)
	}
	err = txn.Commit()
	if err != nil {
		return nil, errors.Wrap(err, "committing reset options")
	}

	return opts, nil
}

func (d *Datastore) OptionByName(name string, args ...kolide.OptionalArg) (*kolide.Option, error) {
	db := d.getTransaction(args)
	sqlStatement := `
			SELECT *
			FROM options
			WHERE name = ?
		`
	var option kolide.Option
	if err := db.Get(&option, sqlStatement, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Option")
		}
		return nil, errors.Wrap(err, sqlStatement)
	}
	return &option, nil
}

func (d *Datastore) SaveOptions(opts []kolide.Option, args ...kolide.OptionalArg) (err error) {
	db := d.getTransaction(args)

	sqlStatement := `
		UPDATE options
		SET value = ?
		WHERE id = ? AND type = ? AND NOT read_only
	`

	for _, opt := range opts {
		resultInfo, err := db.Exec(sqlStatement, opt.Value, opt.ID, opt.Type)
		if err != nil {
			return errors.Wrap(err, "update options")
		}
		rowsMatched, err := resultInfo.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "update options reading rows matched")
		}
		if rowsMatched == 0 {
			return notFound("Option").WithID(opt.ID)
		}
	}
	return err
}

func (d *Datastore) Option(id uint) (*kolide.Option, error) {
	sqlStatement := `
		SELECT *
		FROM options
		WHERE id = ?
	`
	var opt kolide.Option
	if err := d.db.Get(&opt, sqlStatement, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Option").WithID(id)
		}
		return nil, errors.Wrap(err, "select option by ID")
	}
	return &opt, nil
}

func (d *Datastore) ListOptions() ([]kolide.Option, error) {
	sqlStatement := `
    SELECT *
    FROM options
    ORDER BY name ASC
  `
	var opts []kolide.Option
	if err := d.db.Select(&opts, sqlStatement); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Option")
		}
		return nil, errors.Wrap(err, "select from options")
	}
	return opts, nil
}

func (d *Datastore) GetOsqueryConfigOptions() (map[string]interface{}, error) {
	// Retrieve all the options that are set. The value field is JSON formatted so
	// to retrieve options that are set, we check JSON null keyword
	sqlStatement := `
		SELECT *
		FROM options
		WHERE value != "null"
	`
	var opts []kolide.Option
	if err := d.db.Select(&opts, sqlStatement); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Option")
		}
		return nil, errors.Wrap(err, "select from options")
	}
	optConfig := map[string]interface{}{}
	for _, opt := range opts {
		optConfig[opt.Name] = opt.GetValue()
	}
	return optConfig, nil
}
