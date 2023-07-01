package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListWindowsUpdatesByHostID(
	ctx context.Context,
	hostID uint,
) ([]fleet.WindowsUpdate, error) {
	stmt := `
	SELECT kb_id, date_epoch
	FROM windows_updates wu
	WHERE host_id = ?
	ORDER BY date_epoch
	`
	updates := []fleet.WindowsUpdate{}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &updates, stmt, hostID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list windows updates")
	}

	return updates, nil
}

// InsertWindowsUpdates inserts one or more windows updates for the given host.
func (ds *Datastore) InsertWindowsUpdates(ctx context.Context, hostID uint, updates []fleet.WindowsUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	// The windows_updates_history table in OSQUERY is append only so we only need to figure what
	// new updates were installed since the last sync.

	var lastUpdateEpoch uint
	var args []interface{}
	var placeholders []string

	lastUpdateSmt := `SELECT date_epoch FROM windows_updates WHERE host_id = ? ORDER BY date_epoch DESC LIMIT 1`
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &lastUpdateEpoch, lastUpdateSmt, hostID); err != nil {
		if err != sql.ErrNoRows {
			return ctxerr.Wrap(ctx, err, "inserting windows updates")
		}
		lastUpdateEpoch = 0
	}

	for _, v := range updates {
		if v.DateEpoch > lastUpdateEpoch {
			placeholders = append(placeholders, "(?,?,?)")
			args = append(args, hostID, v.DateEpoch, v.KBID)
		}
	}

	if len(args) > 0 {
		smt := fmt.Sprintf(
			`INSERT IGNORE INTO windows_updates (host_id, date_epoch, kb_id) VALUES %s`,
			strings.Join(placeholders, ","),
		)

		if _, err := ds.writer(ctx).ExecContext(ctx, smt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "inserting windows updates")
		}

	}

	return nil
}
