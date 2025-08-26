package mysql

import (
	"context"
	"database/sql"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) CreateOrUpdateSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
	icon, err := ds.GetSoftwareTitleIcon(ctx, payload.TeamID, payload.TitleID, &payload.StorageID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "getting software title icon")
	}

	var query string
	var args []interface{}
	if icon == nil {
		query = `
			INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
			VALUES (?, ?, ?, ?)
		`
		args = []interface{}{payload.TeamID, payload.TitleID, payload.StorageID, payload.Filename}
	} else {
		query = `
			UPDATE software_title_icons
			SET filename = ?, storage_id = ?
			WHERE id = ?
		`
		args = []interface{}{payload.Filename, payload.StorageID, icon.ID}
	}
	res, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upserting software title icon")
	}

	var iconID uint
	if icon == nil {
		iconInt64, err := res.LastInsertId()
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting last insert id")
		}
		iconID = uint(iconInt64)
	} else {
		iconID = icon.ID
	}

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

// func (ds *Datastore) CleanupUnusedSoftwareTitleIcons(ctx context.Context, softwareInstallStore fleet.SoftwareInstallerStore, removeCreatedBefore time.Time) error {
// 	if softwareInstallStore == nil {
// 		// no-op in this case, possible if not running with a Premium license
// 		return nil
// 	}

// 	// get the list of software installers hashes that are in use
// 	var storageIDs []string
// 	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &storageIDs, `SELECT DISTINCT storage_id FROM software_installers`); err != nil {
// 		return ctxerr.Wrap(ctx, err, "get list of software installers in use")
// 	}

// 	_, err := softwareInstallStore.Cleanup(ctx, storageIDs, removeCreatedBefore)
// 	return ctxerr.Wrap(ctx, err, "cleanup unused software installers")
// }
