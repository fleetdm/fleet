package mysql

import (
	"database/sql"

	"github.com/pkg/errors"

	"github.com/kolide/kolide/server/kolide"
)

func (ds *Datastore) NewDecorator(decorator *kolide.Decorator) (*kolide.Decorator, error) {
	sqlStatement :=
		"INSERT INTO decorators (" +
			"`query`," +
			"`type`," +
			"`interval` ) " +
			"VALUES (?, ?, ?)"
	result, err := ds.db.Exec(sqlStatement, decorator.Query, decorator.Type, decorator.Interval)
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

func (ds *Datastore) ListDecorators() ([]*kolide.Decorator, error) {
	sqlStatement := `
    SELECT *
      FROM decorators
  `
	var results []*kolide.Decorator
	err := ds.db.Select(&results, sqlStatement)
	if err != nil {
		return nil, errors.Wrap(err, "listing decorators")
	}
	return results, nil
}
