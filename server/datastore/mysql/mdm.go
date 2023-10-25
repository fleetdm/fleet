package mysql

import (
	"context"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error) {
	stmt := `
SELECT CASE
	WHEN EXISTS (SELECT 1 FROM nano_commands WHERE command_uuid = ?) THEN 'macos'
	WHEN EXISTS (SELECT 1 FROM windows_mdm_commands WHERE command_uuid = ?) THEN 'windows'
	ELSE ''
END AS platform
`

	var p string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &p, stmt, commandUUID, commandUUID); err != nil {
		return "", err
	}
	if p == "" {
		return "", ctxerr.Wrap(ctx, notFound("MDMCommand").WithName(commandUUID))
	}

	return p, nil
}

func (ds *Datastore) GetMDMCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	// check that command exists first, to return 404 on invalid commands
	// (the command may exist but have no results yet).
	p, err := ds.GetMDMCommandPlatform(ctx, commandUUID)
	if err != nil {
		return nil, err
	}

	switch p {
	case "macos":
		return ds.GetMDMAppleCommandResults(ctx, commandUUID)
	case "windows":
		return ds.GetMDMWindowsCommandResults(ctx, commandUUID)
	default:
		// this should never happen, but just in case
		return nil, errors.New("invalid platform for command_uuid")
	}
}
