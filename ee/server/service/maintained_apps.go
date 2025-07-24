package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/go-kit/kit/log/level"
)

// noCheckHash is used by homebrew to signal that a hash shouldn't be checked, and FMA carries this convention over
const noCheckHash = "no_check"

func (svc *Service) AddFleetMaintainedApp(
	ctx context.Context,
	teamID *uint,
	appID uint,
	installScript, preInstallQuery, postInstallScript, uninstallScript string,
	selfService bool, automaticInstall bool,
	labelsIncludeAny, labelsExcludeAny []string,
) (titleID uint, err error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return 0, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return 0, fleet.ErrNoContext
	}

	// validate labels before we do anything else
	validatedLabels, err := ValidateSoftwareLabels(ctx, svc, labelsIncludeAny, labelsExcludeAny)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "validating software labels")
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{installScript, postInstallScript, uninstallScript}); err != nil {
		// We redo the validation on each script to find out which script has the missing secret.
		// This is done to provide a more informative error message to the UI user.
		var argErr *fleet.InvalidArgumentError
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "install script", &installScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "post-install script", &postInstallScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "uninstall script", &uninstallScript, argErr)
		if argErr != nil {
			return 0, argErr
		}
		// We should not get to this point. If we did, it means we have another issue, such as large read replica latency.
		return 0, ctxerr.Wrap(ctx, err, "transient server issue validating embedded secrets")
	}

	app, err := svc.ds.GetMaintainedAppByID(ctx, appID, teamID)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	app, err = maintained_apps.Hydrate(ctx, app)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "hydrating app from manifest")
	}

	// Download installer from the URL
	timeout := maintained_apps.InstallerTimeout
	if v := os.Getenv("FLEET_DEV_MAINTAINED_APPS_INSTALLER_TIMEOUT"); v != "" {
		timeout, _ = time.ParseDuration(v)
	}

	client := fleethttp.NewClient(fleethttp.WithTimeout(timeout))
	installerTFR, filename, err := maintained_apps.DownloadInstaller(ctx, app.InstallerURL, client)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "downloading app installer")
	}
	defer installerTFR.Close()

	gotHash, err := maintained_apps.SHA256FromInstallerFile(installerTFR)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "calculating SHA256 hash")
	}

	// Validate the bytes we got are what we expected, if a valid SHA is supplied
	if app.SHA256 != noCheckHash {
		if gotHash != app.SHA256 {
			return 0, ctxerr.New(ctx, "mismatch in maintained app SHA256 hash")
		}
	} else { // otherwise set the app hash to what we downloaded so storage writes correctly
		app.SHA256 = gotHash
	}

	extension := strings.TrimLeft(filepath.Ext(filename), ".")

	installScript = file.Dos2UnixNewlines(installScript)
	if installScript == "" {
		installScript = app.InstallScript
	}

	uninstallScript = file.Dos2UnixNewlines(uninstallScript)
	if uninstallScript == "" {
		uninstallScript = app.UninstallScript
	}

	maintainedAppID := &app.ID
	if strings.TrimSpace(installScript) != strings.TrimSpace(app.InstallScript) ||
		strings.TrimSpace(uninstallScript) != strings.TrimSpace(app.UninstallScript) {
		maintainedAppID = nil // don't set app as maintained if scripts have been modified
	}

	// For platforms other than macOS, installer name has to match what we see in software inventory,
	// so we have the UniqueIdentifier field to indicate what that should be (independent of the name we
	// display when listing the FMA). For macOS, unique identifier is bundle name, and we use bundle
	// identifier to link installers with inventory, so we set the name to the FMA's display name instead.
	appName := app.UniqueIdentifier
	if app.Platform == "darwin" || appName == "" {
		appName = app.Name
	}

	version := app.Version
	if version == "latest" { // download URL isn't version-pinned; extract version from installer
		meta, err := file.ExtractInstallerMetadata(installerTFR)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "extracting installer metadata")
		}

		// reset the reader (it was consumed to extract metadata)
		if err := installerTFR.Rewind(); err != nil {
			return 0, ctxerr.Wrap(ctx, err, "resetting installer file reader")
		}

		version = meta.Version
	}

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:         installerTFR,
		Title:                 appName,
		UserID:                vc.UserID(),
		TeamID:                teamID,
		Version:               version,
		Filename:              filename,
		Platform:              app.Platform,
		Source:                app.Source(),
		Extension:             extension,
		BundleIdentifier:      app.BundleIdentifier(),
		StorageID:             app.SHA256,
		FleetMaintainedAppID:  maintainedAppID,
		PreInstallQuery:       preInstallQuery,
		PostInstallScript:     postInstallScript,
		SelfService:           selfService,
		InstallScript:         installScript,
		UninstallScript:       uninstallScript,
		ValidatedLabels:       validatedLabels,
		AutomaticInstall:      automaticInstall,
		AutomaticInstallQuery: app.AutomaticInstallQuery,
		Categories:            app.Categories,
		URL:                   app.InstallerURL,
	}

	payload.Categories = server.RemoveDuplicatesFromSlice(payload.Categories)
	catIDs, err := svc.ds.GetSoftwareCategoryIDs(ctx, payload.Categories)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "getting software category ids")
	}

	if len(catIDs) != len(payload.Categories) {
		return 0, &fleet.BadRequestError{
			Message:     "some or all of the categories provided don't exist",
			InternalErr: fmt.Errorf("categories provided: %v", payload.Categories),
		}
	}

	payload.CategoryIDs = catIDs

	// Create record in software installers table
	_, titleID, err = svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "setting downloaded installer")
	}

	// Save in S3
	if err := svc.storeSoftware(ctx, payload); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "upload maintained app installer to S3")
	}

	// Create activity
	var teamName *string
	if payload.TeamID != nil && *payload.TeamID != 0 {
		t, err := svc.ds.Team(ctx, *payload.TeamID)
		if err != nil {
			return 0, ctxerr.Wrap(ctx, err, "getting team")
		}
		teamName = &t.Name
	}

	actLabelsIncl, actLabelsExcl := activitySoftwareLabelsFromValidatedLabels(payload.ValidatedLabels)
	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeAddedSoftware{
		SoftwareTitle:    payload.Title,
		SoftwarePackage:  payload.Filename,
		TeamName:         teamName,
		TeamID:           payload.TeamID,
		SelfService:      payload.SelfService,
		SoftwareTitleID:  titleID,
		LabelsIncludeAny: actLabelsIncl,
		LabelsExcludeAny: actLabelsExcl,
	}); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "creating activity for added software")
	}

	if automaticInstall && payload.AddedAutomaticInstallPolicy != nil {
		policyAct := fleet.ActivityTypeCreatedPolicy{
			ID:   payload.AddedAutomaticInstallPolicy.ID,
			Name: payload.AddedAutomaticInstallPolicy.Name,
		}

		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), policyAct); err != nil {
			level.Warn(svc.logger).Log("msg", "failed to create activity for create automatic install policy for FMA", "err", err)
		}
	}

	return titleID, nil
}

func (svc *Service) ListFleetMaintainedApps(ctx context.Context, teamID *uint, opts fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	var authErr error
	// viewing the maintained app list without showing team-specific info can be done by anyone who can view individual FMAs
	if teamID == nil {
		authErr = svc.authz.Authorize(ctx, &fleet.MaintainedApp{}, fleet.ActionRead)
	} else { // viewing the maintained app list when showing team-specific info requires access to that team
		authErr = svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionRead)
	}

	if authErr != nil {
		return nil, nil, authErr
	}

	opts.IncludeMetadata = true
	avail, meta, err := svc.ds.ListAvailableFleetMaintainedApps(ctx, teamID, opts)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "listing available fleet maintained apps")
	}

	return avail, meta, nil
}

func (svc *Service) GetFleetMaintainedApp(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
	var authErr error
	// viewing the maintained app without showing team-specific info can be done by anyone who can view individual FMAs
	if teamID == nil {
		authErr = svc.authz.Authorize(ctx, &fleet.MaintainedApp{}, fleet.ActionRead)
	} else { // viewing the maintained app when showing team-specific info requires access to that team
		authErr = svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionRead)
	}

	if authErr != nil {
		return nil, authErr
	}

	app, err := svc.ds.GetMaintainedAppByID(ctx, appID, teamID)
	if err != nil {
		return nil, err
	}

	return maintained_apps.Hydrate(ctx, app)
}
