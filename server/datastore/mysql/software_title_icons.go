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
	var args []any
	query = `
		INSERT INTO software_title_icons (team_id, software_title_id, storage_id, filename)
		VALUES (?, ?, ?, ?) ON DUPLICATE KEY UPDATE
		storage_id = VALUES(storage_id), filename = VALUES(filename)
	`
	args = []any{payload.TeamID, payload.TitleID, payload.StorageID, payload.Filename}

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
	args := []any{teamID, titleID}
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

func (ds *Datastore) GetTeamIdsForIconStorageId(ctx context.Context, storageID string) ([]uint, error) {
	var teamIds []uint
	query := `SELECT team_id FROM software_title_icons WHERE storage_id = ?`
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &teamIds, query, storageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking access to software title icon")
	}
	return teamIds, nil
}

func (ds *Datastore) GetSoftwareIconsByTeamAndTitleIds(ctx context.Context, teamID uint, titleIDs []uint) (map[uint]fleet.SoftwareTitleIcon, error) {
	if len(titleIDs) == 0 {
		return map[uint]fleet.SoftwareTitleIcon{}, nil
	}

	var args []any
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

	iconsBySoftwareTitleID := make(map[uint]fleet.SoftwareTitleIcon, len(icons))
	for _, icon := range icons {
		iconsBySoftwareTitleID[icon.SoftwareTitleID] = icon
	}

	return iconsBySoftwareTitleID, nil
}

func (ds *Datastore) DeleteSoftwareTitleIcon(ctx context.Context, teamID, titleID uint) error {
	query := `
		DELETE FROM software_title_icons
		WHERE team_id = ? AND software_title_id = ?
	`
	result, err := ds.writer(ctx).ExecContext(ctx, query, teamID, titleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting software title icon")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting rows affected")
	}

	if rowsAffected == 0 {
		return ctxerr.Wrap(ctx, notFound("SoftwareTitleIcon"), "software title icon not found")
	}

	return nil
}

func (ds *Datastore) DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx context.Context, teamID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM software_title_icons WHERE team_id = ?
		AND software_title_id NOT IN (SELECT title_id FROM vpp_apps va JOIN vpp_apps_teams vat 
			ON vat.adam_id = va.adam_id AND vat.platform = va.platform WHERE global_or_team_id = ?)
		AND software_title_id NOT IN (SELECT title_id FROM software_installers WHERE global_or_team_id = ?)
		AND software_title_id NOT IN (SELECT title_id FROM in_house_apps WHERE global_or_team_id = ?)`,
		teamID, teamID, teamID, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up icons not associated with software installers")
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

func (ds *Datastore) ActivityDetailsForSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) (fleet.DetailsForSoftwareIconActivity, error) {
	var details fleet.DetailsForSoftwareIconActivity
	query := `
		SELECT
			software_installers.id AS software_installer_id,
			in_house_apps.id AS in_house_app_id,
			vpp_apps.adam_id AS adam_id,
			vpp_apps_teams.id AS vpp_app_team_id,
			vpp_apps.icon_url AS vpp_icon_url,
			COALESCE(software_titles.name, vpp_apps.name) AS software_title,
			COALESCE(software_installers.filename, in_house_apps.filename) AS filename,
			teams.name AS team_name,
			COALESCE(teams.id, 0) AS team_id,
			COALESCE(software_installers.self_service, vpp_apps_teams.self_service, in_house_apps.self_service) AS self_service,
			software_titles.id AS software_title_id,
			vpp_apps.platform AS platform
		FROM software_title_icons
		INNER JOIN software_titles ON software_title_icons.software_title_id = software_titles.id
		LEFT JOIN teams ON software_title_icons.team_id = teams.id
		LEFT JOIN software_installers ON software_installers.title_id = software_titles.id
		LEFT JOIN in_house_apps ON in_house_apps.title_id = software_titles.id
		LEFT JOIN vpp_apps ON vpp_apps.title_id = software_titles.id
		LEFT JOIN vpp_apps_teams ON vpp_apps_teams.adam_id = vpp_apps.adam_id AND vpp_apps_teams.platform = vpp_apps.platform
		WHERE software_title_icons.team_id = ? AND software_title_icons.software_title_id = ?
	`
	err := sqlx.GetContext(ctx, ds.reader(ctx), &details, query, teamID, titleID)
	if err != nil {
		return fleet.DetailsForSoftwareIconActivity{}, ctxerr.Wrap(ctx, err, "getting activity details for software title icon")
	}

	type ActivitySoftwareLabel struct {
		ID      uint   `db:"id"`
		Name    string `db:"name"`
		Exclude bool   `db:"exclude"`
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
			return fleet.DetailsForSoftwareIconActivity{}, ctxerr.Wrap(ctx, err, "getting labels for software title icon")
		}
	}
	if details.AdamID != nil {
		labelQuery := `
			SELECT
				labels.id AS id,
				labels.name AS name,
				vpp_app_team_labels.exclude AS exclude
			FROM vpp_app_team_labels
			INNER JOIN labels ON vpp_app_team_labels.label_id = labels.id
			WHERE vpp_app_team_id = ?
		`
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, labelQuery, details.VPPAppTeamID); err != nil {
			return fleet.DetailsForSoftwareIconActivity{}, ctxerr.Wrap(ctx, err, "getting labels for software title icon")
		}
	}
	if details.InHouseAppID != nil {
		labelQuery := `
			SELECT
				labels.id AS id,
				labels.name AS name,
				in_house_app_labels.exclude AS exclude
			FROM in_house_app_labels
			INNER JOIN labels ON in_house_app_labels.label_id = labels.id
			WHERE in_house_app_id = ?
		`
		if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, labelQuery, details.InHouseAppID); err != nil {
			return fleet.DetailsForSoftwareIconActivity{}, ctxerr.Wrap(ctx, err, "getting labels for software title icon")
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
