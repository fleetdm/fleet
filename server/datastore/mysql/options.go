package mysql

import (
	"database/sql"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
)

func (d *Datastore) OptionByName(name string) (*kolide.Option, error) {
	sqlStatement := `
			SELECT *
			FROM options
			WHERE name = ?
		`
	var option kolide.Option
	if err := d.db.Get(&option, sqlStatement, name); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Option")
		}
		return nil, errors.Wrap(err, sqlStatement)
	}
	return &option, nil
}

func (d *Datastore) SaveOptions(opts []kolide.Option) (err error) {
	sqlStatement := `
		UPDATE options
		SET value = ?
		WHERE id = ? AND type = ? AND NOT read_only
	`
	txn, err := d.db.Begin()
	if err != nil {
		return errors.Wrap(err, "update options begin transaction")
	}
	var success bool
	defer func() {
		if success {
			if err = txn.Commit(); err == nil {
				return
			}
		}
		txn.Rollback()
	}()

	for _, opt := range opts {
		result, err := txn.Exec(sqlStatement, opt.Value, opt.ID, opt.Type)
		if err != nil {
			return errors.Wrap(err, "update options")
		}
		rowsChanged, err := result.RowsAffected()
		if err != nil {
			return errors.Wrap(err, "option rows affected")
		}
		if rowsChanged != 1 {
			return notFound("Option").WithID(opt.ID)
		}
	}
	// If all the updates succeed, set the success flag, this will cause the
	// function we defined in defer to commit the transaction. Otherwise, all
	// changes will be rolled back
	success = true
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
