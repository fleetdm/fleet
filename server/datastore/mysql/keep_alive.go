package mysql

import (
	"context"
)

func (ds *Datastore) UpdateKeepAlive(ctx context.Context) error {
	// convert time.Time to appropriate format for DB write

	// 	-- Try to insert a new record

	// INSERT INTO your_table (column1, column2, ...)

	// VALUES (value1, value2, ...)

	// -- If there's a duplicate, update the existing record

	// ON DUPLICATE KEY UPDATE column1 = value1, column2 = value2, ...;
	stmt := `INSERT INTO keep_alive VALUES (NOW()) ON DUPLICATE KEY UPDATE last_server_instance_checkin=NOW()`

	// TODO - parse time into string?
	_, err := ds.writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		return err
	}
	return nil
}
