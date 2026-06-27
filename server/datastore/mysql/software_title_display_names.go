package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

func updateSoftwareTitleDisplayName(ctx context.Context, tx sqlx.ExtContext, teamID *uint, titleID uint, displayName string) error {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO software_title_display_names
			(team_id, software_title_id, display_name)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
			display_name = VALUES(display_name)`, tmID, titleID, displayName)
	if err != nil {
		return err
	}

	return nil
}

func (ds *Datastore) getDisplayNamesByTeamAndTitleIds(ctx context.Context, teamID uint, titleIDs []uint) (map[uint]string, error) {
	if len(titleIDs) == 0 {
		return map[uint]string{}, nil
	}

	namesBySoftwareTitleID := make(map[uint]string, len(titleIDs))

	// Process in batches to avoid exceeding MySQL's 65,535 prepared statement
	// placeholder limit when the caller passes a large number of title IDs
	// (e.g., when per_page is not specified and defaults to 1,000,000).
	const batchSize = 32000
	err := common_mysql.BatchProcessSimple(titleIDs, batchSize, func(batch []uint) error {
		query := `
			SELECT software_title_id, display_name
			FROM software_title_display_names
			WHERE software_title_id IN (?) AND team_id = ?
		`
		query, args, err := sqlx.In(query, batch, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "building query for get software title display names")
		}

		var results []struct {
			SoftwareTitleID uint   `db:"software_title_id"`
			DisplayName     string `db:"display_name"`
		}
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, query, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "get software title display names")
		}

		for _, r := range results {
			namesBySoftwareTitleID[r.SoftwareTitleID] = r.DisplayName
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return namesBySoftwareTitleID, nil
}

func (ds *Datastore) getSoftwareTitleDisplayName(ctx context.Context, teamID uint, titleID uint) (string, error) {
	args := []any{teamID, titleID}
	query := `
		SELECT display_name
		FROM software_title_display_names
		WHERE team_id = ? AND software_title_id = ?
	`
	var displayName string
	err := sqlx.GetContext(ctx, ds.reader(ctx), &displayName, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ctxerr.Wrap(ctx, notFound("SoftwareTitleDisplayName"), "get software title display name")
		}
		return "", ctxerr.Wrap(ctx, err, "get software title display name")
	}

	return displayName, nil
}
