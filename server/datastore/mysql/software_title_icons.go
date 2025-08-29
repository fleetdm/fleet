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
		storage_id = VALUES(storage_id), filename = VALUES(filename)
	`
	args = []interface{}{payload.TeamID, payload.TitleID, payload.StorageID, payload.Filename}

	_, err := ds.writer(ctx).ExecContext(ctx, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upserting software title icon")
	}

	return &fleet.SoftwareTitleIcon{
		TeamID:          payload.TeamID,
		SoftwareTitleID: payload.TitleID,
		StorageID:       payload.StorageID,
		Filename:        payload.Filename,
	}, nil
}

func (ds *Datastore) GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) (*fleet.SoftwareTitleIcon, error) {
	args := []interface{}{teamID, titleID}
	query := `
		SELECT team_id, software_title_id, storage_id, filename
		FROM software_title_icons
		WHERE team_id = ? AND software_title_id = ?
	`
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

func (ds *Datastore) GetSoftwareIconsByTeamAndTitleIds(ctx context.Context, teamID uint, titleIDs []uint) ([]fleet.SoftwareTitleIcon, error) {
	var args []interface{}
	query := `
		SELECT team_id, software_title_id, storage_id, filename
		FROM software_title_icons
		WHERE software_title_id IN (?) AND team_id = ?
	`
	query, args, err := sqlx.In(query, titleIDs, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for get software title icons")
	}

	var icons []fleet.SoftwareTitleIcon
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &icons, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software title icons")
	}

	return icons, nil
}

func (ds *Datastore) GetSoftwareTitleIconsByTeamAndAdamIDs(ctx context.Context, teamID uint, adamIDs []string) (map[string]fleet.SoftwareTitleIcon, error) {
	var args []interface{}

	query := `
		SELECT
			software_title_icons.team_id,
			software_title_icons.software_title_id,
			software_title_icons.storage_id,
			software_title_icons.filename,
			vpp_apps.adam_id
		FROM software_title_icons
		JOIN vpp_apps ON vpp_apps.title_id = software_title_icons.software_title_id
		JOIN vpp_apps_teams ON vpp_apps_teams.vpp_app_id = vpp_apps.id
		WHERE vpp_apps.adam_id IN (?) AND vpp_apps_teams.team_id = ? AND software_title_icons.team_id = ?
	`
	query, args, err := sqlx.In(query, adamIDs, teamID, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building query for get software title icons")
	}

	type IconsWithAdamId struct {
		fleet.SoftwareTitleIcon
		AdamID string `db:"adam_id"`
	}
	var iconsWithAdamId []IconsWithAdamId
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &iconsWithAdamId, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software title icons")
	}

	icons := make(map[string]fleet.SoftwareTitleIcon, len(iconsWithAdamId))
	for _, icon := range iconsWithAdamId {
		icons[icon.AdamID] = icon.SoftwareTitleIcon
	}

	return icons, nil
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
