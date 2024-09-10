package service

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	app, err := svc.ds.GetMaintainedAppById(ctx, appID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	// Download installer from the URL
	slog.With("filename", "server/service/maintained_apps.go", "func", "AddFleetMaintainedApp").Info("JVE_LOG: got mapp ", "app", app.Name)
	d := maintainedapps.NewAppDownloader(ctx, svc.softwareInstallStore, svc.logger)
	if err := d.Download(ctx, app.InstallerURL); err != nil {
		return ctxerr.Wrap(ctx, err, "downloading app installer")
	}

	// Insert into software_installers

	return nil
}
