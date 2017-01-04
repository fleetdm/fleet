package mysql

import (
	"fmt"

	"github.com/pkg/errors"
)

func (d *Datastore) deleteEntity(dbTable string, id uint) error {
	deleteStmt := fmt.Sprintf(
		`
		UPDATE %s SET deleted_at = ?, deleted = TRUE
			WHERE id = ?
	`, dbTable)
	result, err := d.db.Exec(deleteStmt, d.clock.Now(), id)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("delete %s", dbTable))
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		return notFound(dbTable).WithID(id)
	}
	return nil
}
