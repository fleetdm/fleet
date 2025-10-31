package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) getDisplayNamesByTeamAndTitleIds(ctx context.Context, teamID uint, titleIDs []uint) (map[uint]string, error) {
	if len(titleIDs) == 0 {
		return map[uint]string{}, nil
	}

	var args []interface{}
	query := `
		SELECT software_title_id, display_name
		FROM software_title_display_names
		WHERE software_title_id IN (?) AND team_id = ?
	`
	query, args, err := sqlx.In(query, titleIDs, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for get software title display names")
	}

	var results []struct {
		SoftwareTitleID uint   `db:"software_title_id"`
		DisplayName     string `db:"display_name"`
	}
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software title display names")
	}

	iconsBySoftwareTitleID := make(map[uint]string, len(results))
	for _, r := range results {
		iconsBySoftwareTitleID[r.SoftwareTitleID] = r.DisplayName
	}

	return iconsBySoftwareTitleID, nil
}
