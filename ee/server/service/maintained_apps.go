package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
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

	h := sha256.New()
	_, _ = io.Copy(h, installerTFR) // writes to a Hash can never fail
	gotHash := hex.EncodeToString(h.Sum(nil))

	// Validate the bytes we got are what we expected, if a valid SHA is supplied
	if app.SHA256 != noCheckHash {
		if gotHash != app.SHA256 {
			return 0, ctxerr.New(ctx, "mismatch in maintained app SHA256 hash")
		}
	} else { // otherwise set the app hash to what we downloaded so storage writes correctly
		app.SHA256 = gotHash
	}

	if err := installerTFR.Rewind(); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "rewind installer reader")
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

	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallerFile:         installerTFR,
		Title:                 app.Name,
		UserID:                vc.UserID(),
		TeamID:                teamID,
		Version:               app.Version,
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
	}

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
