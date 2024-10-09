package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// noCheckHash is used by homebrew to signal that a hash shouldn't be checked.
const noCheckHash = "no_check"

func (svc *Service) AddFleetMaintainedApp(
	ctx context.Context,
	teamID *uint,
	appID uint,
	installScript, preInstallQuery, postInstallScript, uninstallScript string,
	selfService bool,
) error {
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
	timeout := maintainedapps.InstallerTimeout
	if v := os.Getenv("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT"); v != "" {
		timeout, _ = time.ParseDuration(v)
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(timeout))
	installerBytes, filename, err := maintainedapps.DownloadInstaller(ctx, app.InstallerURL, client)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "downloading app installer")
	}

	extension, err := maintainedapps.ExtensionForBundleIdentifier(app.BundleIdentifier)
	if err != nil {
		return ctxerr.Errorf(ctx, "getting extension from bundle identifier %q", app.BundleIdentifier)
	}

	// Validate the bytes we got are what we expected, if homebrew supports
	// it, the string "no_check" is a special token used to signal that the
	// hash shouldn't be checked.
	if app.SHA256 != noCheckHash {
		h := sha256.New()
		_, err = h.Write(installerBytes)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating SHA256 of maintained app installer")
		}
		gotHash := hex.EncodeToString(h.Sum(nil))

		if gotHash != app.SHA256 {
			return ctxerr.New(ctx, "mismatch in maintained app SHA256 hash")
		}
	}

	// Fall back to the filename if we weren't able to extract a filename from the installer response
	if filename == "" {
		filename = app.Name
	}

	// The UI requires all filenames to have extensions. If we couldn't get
	// one, use the extension we extracted prior
	if filepath.Ext(filename) == "" {
		filename = filename + "." + extension
	}

	installScript = file.Dos2UnixNewlines(installScript)
	if installScript == "" {
		installScript = app.InstallScript
	}

	uninstallScript = file.Dos2UnixNewlines(uninstallScript)
	if uninstallScript == "" {
		uninstallScript = app.UninstallScript
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
		Source:            "apps",
		Extension:         extension,
		BundleIdentifier:  app.BundleIdentifier,
		StorageID:         app.SHA256,
		FleetLibraryAppID: &app.ID,
		PreInstallQuery:   preInstallQuery,
		PostInstallScript: postInstallScript,
		SelfService:       selfService,
		InstallScript:     installScript,
		UninstallScript:   uninstallScript,
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

func (svc *Service) ListFleetMaintainedApps(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{
		TeamID: &teamID,
	}, fleet.ActionRead); err != nil {
		return nil, nil, err
	}

	avail, meta, err := svc.ds.ListAvailableFleetMaintainedApps(ctx, teamID, opts)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "listing available fleet managed apps")
	}

	return avail, meta, nil
}

func (svc *Service) GetFleetMaintainedApp(ctx context.Context, appID uint) (*fleet.MaintainedApp, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{
		TeamID: nil,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	app, err := svc.ds.GetMaintainedAppByID(ctx, appID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get fleet maintained app")
	}

	return app, nil
}
