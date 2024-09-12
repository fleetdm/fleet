package service

import (
	"bytes"
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

const maxMaintainedInstallerSizeBytes int64 = 1024 * 1024 * 1024 * 3 // 3GB

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	app, err := svc.ds.GetMaintainedAppByID(ctx, appID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	// Download installer from the URL
	slog.With("filename", "server/service/maintained_apps.go", "func", "AddFleetMaintainedApp").Info("JVE_LOG: got mapp ", "app", app.Name, "sha", app.SHA256)
	installerBytes, err := maintainedapps.Download(ctx, app.InstallerURL, maxMaintainedInstallerSizeBytes)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading app installer")
	}

	// TODO(JVE): could this be moved into Download (rename the func ofc)?
	installerReader := bytes.NewReader(installerBytes)
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:     installerReader,
		Title:             app.Name,
		UserID:            vc.UserID(),
		TeamID:            teamID,
		Version:           app.Version,
		Platform:          string(app.Platform),
		BundleIdentifier:  app.BundleIdentifier,
		StorageID:         app.SHA256,
		FleetLibraryAppID: &app.ID,
	}
	_, err = svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting downloaded installer")
	}

	if err := svc.storeSoftware(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "upload maintained app installer to S3")
	}

	// Create activity
	var teamName *string
	if payload.TeamID != nil && *payload.TeamID != 0 {
		t, err := svc.ds.Team(ctx, *payload.TeamID)
		if err != nil {
			return err
		}
		teamName = &t.Name
	}

	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeAddedSoftware{
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		TeamName:        teamName,
		TeamID:          payload.TeamID,
		SelfService:     payload.SelfService,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for added software")
	}

	return nil
}
