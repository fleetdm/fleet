package mysql

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// deleteEntity deletes an entity with the given id from the given DB table,
// returning a notFound error if appropriate.
func (d *Datastore) deleteEntity(ctx context.Context, dbTable entity, id uint) error {
	deleteStmt := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, dbTable.name)
	result, err := d.writer.ExecContext(ctx, deleteStmt, id)
	if err != nil {
		return errors.Wrapf(err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return notFound(dbTable.name).WithID(id)
	}
	return nil
}

// deleteEntityByName deletes an entity with the given name from the given DB
// table, returning a notFound error if appropriate.
func (d *Datastore) deleteEntityByName(ctx context.Context, dbTable entity, name string) error {
	deleteStmt := fmt.Sprintf("DELETE FROM %s WHERE name = ?", dbTable.name)
	result, err := d.writer.ExecContext(ctx, deleteStmt, name)
	if err != nil {
		if isMySQLForeignKey(err) {
			return foreignKey(dbTable.name, name)
		}
		return errors.Wrapf(err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return notFound(dbTable.name).WithName(name)
	}
	return nil
}

// deleteEntities deletes the existing entity objects with the provided IDs.
// The number of deleted entities is returned along with any error.
func (d *Datastore) deleteEntities(ctx context.Context, dbTable entity, ids []uint) (uint, error) {
	deleteStmt := fmt.Sprintf(`DELETE FROM %s WHERE id IN (?)`, dbTable.name)

	query, args, err := sqlx.In(deleteStmt, ids)
	if err != nil {
		return 0, errors.Wrapf(err, "building delete entities query %s", dbTable)
	}

	result, err := d.writer.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, errors.Wrapf(err, "executing delete entities query %s", dbTable)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrapf(err, "fetching delete entities query rows affected %s", dbTable)
	}

	return uint(deleted), nil
}
