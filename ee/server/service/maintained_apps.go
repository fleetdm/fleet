package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint, installScript, preInstallQuery, postInstallScript string, selfService bool) error {
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
	installerBytes, filename, err := maintainedapps.DownloadInstaller(ctx, app.InstallerURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading app installer")
	}

	// Validate the bytes we got are what we expected
	h := sha256.New()
	_, err = h.Write(installerBytes)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating SHA256 of maintained app installer")
	}
	gotHash := hex.EncodeToString(h.Sum(nil))

	if gotHash != app.SHA256 {
		return ctxerr.New(ctx, "mismatch in maintained app SHA256 hash")
	}

	// Fall back to the filename if we weren't able to extract a filename from the installer response
	if filename == "" {
		filename = app.Name
	}

	installerReader := bytes.NewReader(installerBytes)
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:     installerReader,
		Title:             app.Name,
		UserID:            vc.UserID(),
		TeamID:            teamID,
		Version:           app.Version,
		Filename:          filename,
		Platform:          string(app.Platform),
		BundleIdentifier:  app.BundleIdentifier,
		StorageID:         app.SHA256,
		FleetLibraryAppID: &app.ID,
		PreInstallQuery:   preInstallQuery,
		PostInstallScript: postInstallScript,
		SelfService:       selfService,
		InstallScript:     installScript,
	}

	// Create record in software installers table
	_, err = svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting downloaded installer")
	}

	// Save in S3
	if err := svc.storeSoftware(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "upload maintained app installer to S3")
	}

	// Create activity
	var teamName *string
	if payload.TeamID != nil && *payload.TeamID != 0 {
		t, err := svc.ds.Team(ctx, *payload.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team")
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

func (svc *Service) GetFleetMaintainedApp(ctx context.Context, appID uint) (*fleet.MaintainedApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: nil,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return nil, nil
}
