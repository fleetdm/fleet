package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetTeamAppleSerialNumbers(ctx context.Context, teamID uint) ([]string, error) {
	stmt := `
SELECT
  hardware_serial
FROM
  hosts
WHERE
  platform = 'darwin'
AND
  team_id = ?
`

	var serialNumbers []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &serialNumbers, stmt, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unable to retrieve team serial numbers")
	}

	return serialNumbers, nil
}
