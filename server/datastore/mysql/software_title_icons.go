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

func (ds *Datastore) ActivityDetailsForSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) (fleet.SoftwareTitleIconActivity, error) {
	var details fleet.SoftwareTitleIconActivity
	query := `
		SELECT
			software_installer.id AS software_installer_id,
			vpp_apps.adam_id AS adam_id,
			software_titles.name AS software_title,
			software_installers.filename AS filename,
			teams.name AS team_name,
			teams.id AS team_id,
			COALESCE(software_installers.self_service, vpp_apps_teams.self_service) AS self_service,
			software_titles.id AS software_title_id
		FROM software_title_icons
		INNER JOIN software_titles ON software_title_icons.software_title_id = software_titles.id
		INNER JOIN teams ON software_title_icons.team_id = teams.id
		LEFT JOIN software_installers ON software_title_icons.software_title_id = software_installers.id
		LEFT JOIN vpp_apps ON software_title_icons.software_title_id = vpp_apps.id
		LEFT JOIN vpp_apps_teams ON software_title_icons.software_title_id = vpp_apps_teams.vpp_app_id
		WHERE software_title_icons.team_id = ? AND software_title_icons.software_title_id = ?
	`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &details, query, teamID, titleID)
	type ActivitySoftwareLabel struct {
		ID      uint   `db:"id"`
		Name    string `db:"name"`
		Exclude bool   `db:"exclude"`
	}
	if err != nil {
		return fleet.SoftwareTitleIconActivity{}, ctxerr.Wrap(ctx, err, "getting activity details for software title icon")
	}

	var labels []ActivitySoftwareLabel
	if details.SoftwareInstallerID != nil {
		labelQuery := `
			SELECT
				labels.id AS id,
				labels.name AS name,
				software_installer_labels.exclude AS exclude
			FROM software_installer_labels
			INNER JOIN labels ON software_installer_labels.label_id = labels.id
			WHERE software_installer_id = ?
		`
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, labelQuery, details.SoftwareInstallerID); err != nil {
			return fleet.SoftwareTitleIconActivity{}, ctxerr.Wrap(ctx, err, "getting labels for software title icon")
		}
	}
	if details.AdamID != nil {
		labelQuery := `
			SELECT
				labels.id AS id,
				labels.name AS name,
				vpp_app_labels.exclude AS exclude
			FROM vpp_app_labels
			INNER JOIN labels ON vpp_app_labels.label_id = labels.id
			WHERE vpp_app_id = ?
		`
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, labelQuery, details.AdamID); err != nil {
			return fleet.SoftwareTitleIconActivity{}, ctxerr.Wrap(ctx, err, "getting labels for software title icon")
		}
	}
	for _, l := range labels {
		if l.Exclude {
			details.LabelsExcludeAny = append(details.LabelsExcludeAny, fleet.ActivitySoftwareLabel{
				ID:   l.ID,
				Name: l.Name,
			})
		} else {
			details.LabelsIncludeAny = append(details.LabelsIncludeAny, fleet.ActivitySoftwareLabel{
				ID:   l.ID,
				Name: l.Name,
			})
		}
	}

	return details, nil
}
