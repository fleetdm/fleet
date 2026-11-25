package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

// scdConvertedExpr is true for a host_scd_change_state row whose conversion is complete.
const scdConvertedExpr = `(convert_cursor IS NOT NULL AND convert_cursor >= changes_from)`

// scdRetireBatch caps how many host_scd_data rows one RetireSCDData call deletes. var (not const) so tests can
// shrink it.
var scdRetireBatch = 1000

// SCDHistoryConverted reports whether ConvertSCDHistory has converted all of the dataset's host_scd_data history
// from before the change storage started, and false when the change storage has not started. It reads the
// primary: the collector decides from it whether to keep writing and whether to delete host_scd_data rows.
func (ds *Datastore) SCDHistoryConverted(ctx context.Context, dataset string) (bool, error) {
	ctx = ctxdb.RequirePrimary(ctx, true)
	var converted []bool
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &converted,
		`SELECT `+scdConvertedExpr+` FROM host_scd_change_state WHERE dataset = ?`,
		dataset); err != nil {
		return false, ctxerr.Wrap(ctx, err, "read chart conversion state")
	}
	return len(converted) > 0 && converted[0], nil
}

// RetireSCDData deletes up to scdRetireBatch of the dataset's host_scd_data rows once its history is converted to
// changes, and reports whether more may remain. Before that it deletes nothing: the conversion and the requests the
// change storage does not cover yet read those rows.
func (ds *Datastore) RetireSCDData(ctx context.Context, dataset string) (bool, error) {
	converted, err := ds.SCDHistoryConverted(ctx, dataset)
	if err != nil || !converted {
		return false, err
	}
	res, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM host_scd_data WHERE dataset = ? LIMIT ?`, dataset, scdRetireBatch)
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "delete retired chart rows")
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, ctxerr.Wrap(ctx, err, "delete retired chart rows affected")
	}
	return n == int64(scdRetireBatch), nil
}
