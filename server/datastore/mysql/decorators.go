package mysql

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/kolide/fleet/server/kolide"
)

func (ds *Datastore) SaveDecorator(dec *kolide.Decorator, opts ...kolide.OptionalArg) error {
	db := ds.getTransaction(opts)
	sqlStatement :=
		"UPDATE decorators SET " +
			"`name` = ?, " +
			"`query` = ?, " +
			"`type` = ?, " +
			"`interval` = ? " +
			"WHERE id = ?"
	_, err := db.Exec(
		sqlStatement,
		dec.Name,
		dec.Query,
		dec.Type,
		dec.Interval,
		dec.ID,
	)
	if err != nil {
		return errors.Wrap(err, "saving decorator")
	}
	return nil
}

func (ds *Datastore) NewDecorator(decorator *kolide.Decorator, opts ...kolide.OptionalArg) (*kolide.Decorator, error) {
	db := ds.getTransaction(opts)
	sqlStatement :=
		"INSERT INTO decorators (" +
			"`name`," +
			"`query`," +
			"`type`," +
			"`interval` ) " +
			"VALUES (?, ?, ?, ?)"
	result, err := db.Exec(sqlStatement, decorator.Name, decorator.Query, decorator.Type, decorator.Interval)

	if err != nil {
		return nil, errors.Wrap(err, "creating decorator")
	}
	id, _ := result.LastInsertId()
	decorator.ID = uint(id)
	return decorator, nil
}

func (ds *Datastore) DeleteDecorator(id uint) error {
	sqlStatement := `
    DELETE FROM decorators
      WHERE id = ?
  `
	res, err := ds.db.Exec(sqlStatement, id)
	if err != nil {
		return errors.Wrap(err, "deleting decorator")
	}
	deleted, _ := res.RowsAffected()
	if deleted < 1 {
		return notFound("Decorator").WithID(id)
	}
	return nil
}

func (ds *Datastore) Decorator(id uint) (*kolide.Decorator, error) {
	sqlStatement := `
    SELECT *
      FROM decorators
      WHERE id = ?
  `
	var result kolide.Decorator
	err := ds.db.Get(&result, sqlStatement, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Decorator").WithID(id)
		}
		return nil, errors.Wrap(err, "retrieving decorator")
	}
	return &result, nil
}

func (ds *Datastore) ListDecorators(opts ...kolide.OptionalArg) ([]*kolide.Decorator, error) {
	db := ds.getTransaction(opts)
	sqlStatement := `
    SELECT *
      FROM decorators
      ORDER by built_in DESC, name ASC
  `
	var results []*kolide.Decorator
	err := db.Select(&results, sqlStatement)
	if err != nil {
		return nil, errors.Wrap(err, "listing decorators")
	}
	return results, nil
}
