package mysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetPatchDeadlineSettings(ctx context.Context, teamID uint) (*fleet.PatchDeadlineSettings, error) {
	var row struct {
		Days string `db:"days"`
		Time string `db:"time"`
	}
	err := sqlx.GetContext(ctx, ds.reader(ctx), &row, "SELECT days, `time` FROM patch_deadline_settings WHERE team_id = ?", teamID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fleet.DefaultPatchDeadlineSettings(teamID), nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get patch deadline settings")
	}
	return &fleet.PatchDeadlineSettings{TeamID: teamID, Days: strings.Split(row.Days, ","), Time: row.Time}, nil
}

func (ds *Datastore) SetPatchDeadlineSettings(ctx context.Context, settings fleet.PatchDeadlineSettings) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `
INSERT INTO patch_deadline_settings (team_id, days, `+"`time`"+`) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE days = VALUES(days), `+"`time`"+` = VALUES(`+"`time`"+`)`,
		settings.TeamID, strings.Join(settings.Days, ","), settings.Time)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "set patch deadline settings")
	}
	return nil
}
