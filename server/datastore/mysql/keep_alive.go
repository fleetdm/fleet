package mysql

import (
	"context"
)

func (ds *Datastore) UpdateKeepAlive(ctx context.Context) error {
	stmt := `UPDATE keep_alive SET last_server_instance_checkin=NOW();`

	_, err := ds.writer(ctx).ExecContext(ctx, stmt)
	if err != nil {
		return err
	}
	return nil
}
