package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// deleteEntity deletes an entity with the given id from the given DB table,
// returning a notFound error if appropriate.
func (ds *Datastore) deleteEntity(ctx context.Context, dbTable entity, id uint) error {
	deleteStmt := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, dbTable.name)
	result, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, id)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ctxerr.Wrap(ctx, notFound(dbTable.name).WithID(id))
	}
	return nil
}

// deleteEntityByName deletes an entity with the given name from the given DB
// table, returning a notFound error if appropriate.
func (ds *Datastore) deleteEntityByName(ctx context.Context, dbTable entity, name string) error {
	deleteStmt := fmt.Sprintf("DELETE FROM %s WHERE name = ?", dbTable.name)
	result, err := ds.writer(ctx).ExecContext(ctx, deleteStmt, name)
	if err != nil {
		if isMySQLForeignKey(err) {
			return ctxerr.Wrap(ctx, foreignKey(dbTable.name, name))
		}
		return ctxerr.Wrapf(ctx, err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return ctxerr.Wrap(ctx, notFound(dbTable.name).WithName(name))
	}
	return nil
}

// deleteEntities deletes the existing entity objects with the provided IDs.
// The number of deleted entities is returned along with any error.
func (ds *Datastore) deleteEntities(ctx context.Context, dbTable entity, ids []uint) (uint, error) {
	deleteStmt := fmt.Sprintf(`DELETE FROM %s WHERE id IN (?)`, dbTable.name)

	query, args, err := sqlx.In(deleteStmt, ids)
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "building delete entities query %s", dbTable)
	}

	result, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "executing delete entities query %s", dbTable)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrapf(ctx, err, "fetching delete entities query rows affected %s", dbTable)
	}

	return uint(deleted), nil //nolint:gosec // dismiss G115
}
