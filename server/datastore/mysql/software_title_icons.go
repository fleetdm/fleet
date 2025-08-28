package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateOrUpdateSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
	var query string
	var args []interface{}
	query = `
		INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
		VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE
		id = LAST_INSERT_ID(id), storage_id = VALUES(storage_id), filename = VALUES(filename)
	`
	args = []interface{}{payload.TeamID, payload.TitleID, payload.StorageID, payload.Filename}

	res, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upserting software title icon")
	}

	iconInt64, err := res.LastInsertId()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting last insert id")
	}
	if iconInt64 < 0 {
		return nil, ctxerr.New(ctx, "invalid icon ID")
	}
	iconID := uint(iconInt64)

	return &fleet.SoftwareTitleIcon{
		ID:              iconID,
		TeamID:          payload.TeamID,
		SoftwareTitleID: payload.TitleID,
		StorageID:       payload.StorageID,
		Filename:        payload.Filename,
	}, nil
}

func (ds *Datastore) GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint, storageID *string) (*fleet.SoftwareTitleIcon, error) {
	args := []interface{}{teamID, titleID}
	query := `
		SELECT id, team_id, software_title_id, storage_id, filename
		FROM software_title_icons
		WHERE team_id = ? AND software_title_id = ?
	`
	if storageID != nil && *storageID != "" {
		query += " AND storage_id = ?"
		args = append(args, storageID)
	}

	var icon fleet.SoftwareTitleIcon
	err := sqlx.GetContext(ctx, ds.reader(ctx), &icon, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SoftwareTitleIcon"), "get software title icon")
		}
		return nil, ctxerr.Wrap(ctx, err, "get software title icon")
	}

	return &icon, nil
}

func (ds *Datastore) DeleteSoftwareTitleIcon(ctx context.Context, teamID, titleID uint) error {
	query := `
		DELETE FROM software_title_icons
		WHERE team_id = ? AND software_title_id = ?
	`
	_, err := ds.writer(ctx).ExecContext(ctx, query, teamID, titleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting software title icon")
	}
	return nil
}

func (ds *Datastore) CleanupUnusedSoftwareTitleIcons(ctx context.Context, iconStore fleet.SoftwareTitleIconStore, removeCreatedBefore time.Time) error {
	if iconStore == nil {
		return nil
	}

	var storageIDs []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &storageIDs, `SELECT DISTINCT storage_id FROM software_title_icons`); err != nil {
		return ctxerr.Wrap(ctx, err, "get list of software title icons in use")
	}

	_, err := iconStore.Cleanup(ctx, storageIDs, removeCreatedBefore)
	return ctxerr.Wrap(ctx, err, "cleanup unused software title icons")
}
