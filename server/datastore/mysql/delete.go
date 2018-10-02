package mysql

import (
	"fmt"

	"github.com/pkg/errors"
)

func (d *Datastore) deleteEntity(dbTable string, id uint) error {
	deleteStmt := fmt.Sprintf(
		`
		UPDATE %s SET deleted_at = ?, deleted = TRUE
			WHERE id = ? AND NOT deleted
	`, dbTable)
	result, err := d.db.Exec(deleteStmt, d.clock.Now(), id)
	if err != nil {
		return errors.Wrapf(err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return notFound(dbTable).WithID(id)
	}
	return nil
}

// deleteEntityByName hard deletes an entity with the given name from the given
// DB table, returning a notFound error if appropriate.
// Note: deleteEntity uses soft deletion, but we are moving off this pattern.
func (d *Datastore) deleteEntityByName(dbTable string, name string) error {
	deleteStmt := fmt.Sprintf("DELETE FROM %s WHERE name = ?", dbTable)
	result, err := d.db.Exec(deleteStmt, name)
	if err != nil {
		if isMySQLForeignKey(err) {
			return foreignKey(dbTable, name)
		}
		return errors.Wrapf(err, "delete %s", dbTable)
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return notFound(dbTable).WithName(name)
	}
	return nil
}
