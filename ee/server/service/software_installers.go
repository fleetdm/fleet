package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const softwareInstallerTokenMaxLength = 36 // UUID length

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	payload.UserID = vc.UserID()

	// make sure all scripts use unix-style newlines to prevent errors when
	// running them, browsers use windows-style newlines, which breaks the
	// shebang when the file is directly executed.
	payload.InstallScript = file.Dos2UnixNewlines(payload.InstallScript)
	payload.PostInstallScript = file.Dos2UnixNewlines(payload.PostInstallScript)
	payload.UninstallScript = file.Dos2UnixNewlines(payload.UninstallScript)

	if _, err := svc.addMetadataToSoftwarePayload(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "adding metadata to payload")
	}

	if err := svc.storeSoftware(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "storing software installer")
	}

	// TODO: basic validation of install and post-install script (e.g., supported interpreters)?
	// TODO: any validation of pre-install query?

	// Update $PACKAGE_ID in uninstall script
	preProcessUninstallScript(payload)

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{payload.InstallScript, payload.PostInstallScript, payload.UninstallScript}); err != nil {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("script", err.Error()))
	}

	installerID, titleID, err := svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "matching or creating software installer")
	}
	level.Debug(svc.logger).Log("msg", "software installer uploaded", "installer_id", installerID)

	// TODO: QA what breaks when you have a software title with no versions?

	var teamName *string
	if payload.TeamID != nil && *payload.TeamID != 0 {
		t, err := svc.ds.Team(ctx, *payload.TeamID)
		if err != nil {
			return err
		}
		teamName = &t.Name
	}

	// Create activity
	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeAddedSoftware{
		SoftwareTitle:   payload.Title,
		SoftwarePackage: payload.Filename,
		TeamName:        teamName,
		TeamID:          payload.TeamID,
		SelfService:     payload.SelfService,
		SoftwareTitleID: titleID,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for added software")
	}

	return nil
}

var packageIDRegex = regexp.MustCompile(`((("\$PACKAGE_ID")|(\$PACKAGE_ID))(?P<suffix>\W|$))|(("\${PACKAGE_ID}")|(\${PACKAGE_ID}))`)

func preProcessUninstallScript(payload *fleet.UploadSoftwareInstallerPayload) {
	// We assume that we already validated that payload.PackageIDs is not empty.
	// Replace $PACKAGE_ID in the uninstall script with the package ID(s).
	var packageID string
	switch payload.Extension {
	case "dmg", "zip":
		return
	case "pkg":
		var sb strings.Builder
		_, _ = sb.WriteString("(\n")
		for _, pkgID := range payload.PackageIDs {
			_, _ = sb.WriteString(fmt.Sprintf("  \"%s\"\n", pkgID))
		}
		_, _ = sb.WriteString(")") // no ending newline
		packageID = sb.String()
	default:
		packageID = fmt.Sprintf("\"%s\"", payload.PackageIDs[0])
	}

	payload.UninstallScript = packageIDRegex.ReplaceAllString(payload.UninstallScript, fmt.Sprintf("%s${suffix}", packageID))
}

func (svc *Service) UpdateSoftwareInstaller(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) (*fleet.SoftwareInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	payload.UserID = vc.UserID()

	if payload.TeamID == nil {
		return nil, &fleet.BadRequestError{Message: "team_id is required; enter 0 for no team"}
	}

	var teamName *string
	if *payload.TeamID != 0 {
		t, err := svc.ds.Team(ctx, *payload.TeamID)
		if err != nil {
			return nil, err
		}
		teamName = &t.Name
	}

	var scripts []string

	if payload.InstallScript != nil {
		scripts = append(scripts, *payload.InstallScript)
	}
	if payload.PostInstallScript != nil {
		scripts = append(scripts, *payload.PostInstallScript)
	}
	if payload.UninstallScript != nil {
		scripts = append(scripts, *payload.UninstallScript)
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, scripts); err != nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("script", err.Error()))
	}

	// get software by ID, fail if it does not exist or does not have an existing installer
	software, err := svc.ds.SoftwareTitleByID(ctx, payload.TitleID, payload.TeamID, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software title by id")
	}

	// TODO when we start supporting multiple installers per title X team, need to rework how we determine installer to edit
	if software.SoftwareInstallersCount != 1 {
		return nil, &fleet.BadRequestError{
			Message: "There are no software installers defined yet for this title and team. Please add an installer instead of attempting to edit.",
		}
	}

	existingInstaller, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, payload.TeamID, payload.TitleID, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting existing installer")
	}

	if payload.SelfService == nil && payload.InstallerFile == nil && payload.PreInstallQuery == nil &&
		payload.InstallScript == nil && payload.PostInstallScript == nil && payload.UninstallScript == nil {
		return existingInstaller, nil // no payload, noop
	}

	payload.InstallerID = existingInstaller.InstallerID
	dirty := make(map[string]bool)

	if payload.SelfService != nil && *payload.SelfService != existingInstaller.SelfService {
		dirty["SelfService"] = true
	}

	activity := fleet.ActivityTypeEditedSoftware{
		SoftwareTitle:   existingInstaller.SoftwareTitle,
		TeamName:        teamName,
		TeamID:          payload.TeamID,
		SelfService:     existingInstaller.SelfService,
		SoftwarePackage: &existingInstaller.Name,
	}

	var payloadForNewInstallerFile *fleet.UploadSoftwareInstallerPayload
	if payload.InstallerFile != nil {
		payloadForNewInstallerFile = &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: payload.InstallerFile,
			Filename:      payload.Filename,
		}

		newInstallerExtension, err := svc.addMetadataToSoftwarePayload(ctx, payloadForNewInstallerFile)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "extracting updated installer metadata")
		}

		if newInstallerExtension != existingInstaller.Extension {
			return nil, &fleet.BadRequestError{
				Message:     "The selected package is for a different file type.",
				InternalErr: ctxerr.Wrap(ctx, err, "installer extension mismatch"),
			}
		}

		if payloadForNewInstallerFile.Title != software.Name {
			return nil, &fleet.BadRequestError{
				Message:     "The selected package is for different software.",
				InternalErr: ctxerr.Wrap(ctx, err, "installer software title mismatch"),
			}
		}

		if payloadForNewInstallerFile.StorageID != existingInstaller.StorageID {
			activity.SoftwarePackage = &payload.Filename
			payload.StorageID = payloadForNewInstallerFile.StorageID
			payload.Filename = payloadForNewInstallerFile.Filename
			payload.Version = payloadForNewInstallerFile.Version
			payload.PackageIDs = payloadForNewInstallerFile.PackageIDs

			dirty["Package"] = true
		} else { // noop if uploaded installer is identical to previous installer
			payloadForNewInstallerFile = nil
			payload.InstallerFile = nil
		}
	}

	if payload.InstallerFile == nil { // fill in existing existingInstaller data to payload
		payload.StorageID = existingInstaller.StorageID
		payload.Filename = existingInstaller.Name
		payload.Version = existingInstaller.Version
		payload.PackageIDs = existingInstaller.PackageIDs()
	}

	// default pre-install query is blank, so blanking out the query doesn't have a semantic meaning we have to take care of
	if payload.PreInstallQuery != nil && *payload.PreInstallQuery != existingInstaller.PreInstallQuery {
		dirty["PreInstallQuery"] = true
	}

	if payload.InstallScript != nil {
		installScript := file.Dos2UnixNewlines(*payload.InstallScript)
		if installScript == "" {
			installScript = file.GetInstallScript(existingInstaller.Extension)
		}

		if installScript != existingInstaller.InstallScript {
			dirty["InstallScript"] = true
		}
		payload.InstallScript = &installScript
	}

	if payload.PostInstallScript != nil {
		postInstallScript := file.Dos2UnixNewlines(*payload.PostInstallScript)
		if postInstallScript != existingInstaller.PostInstallScript {
			dirty["PostInstallScript"] = true
		}
		payload.PostInstallScript = &postInstallScript
	}

	if payload.UninstallScript != nil {
		uninstallScript := file.Dos2UnixNewlines(*payload.UninstallScript)
		if uninstallScript == "" { // extension can't change on an edit so we can generate off of the existing file
			uninstallScript = file.GetUninstallScript(existingInstaller.Extension)
		}

		payloadForUninstallScript := &fleet.UploadSoftwareInstallerPayload{
			Extension:       existingInstaller.Extension,
			UninstallScript: uninstallScript,
			PackageIDs:      existingInstaller.PackageIDs(),
		}
		if payloadForNewInstallerFile != nil {
			payloadForUninstallScript.PackageIDs = payloadForNewInstallerFile.PackageIDs
		}

		preProcessUninstallScript(payloadForUninstallScript)
		if payloadForUninstallScript.UninstallScript != existingInstaller.UninstallScript {
			dirty["UninstallScript"] = true
		}
		uninstallScript = payloadForUninstallScript.UninstallScript
		payload.UninstallScript = &uninstallScript
	}

	// persist changes starting here, now that we've done all the validation/diffing we can
	if len(dirty) > 0 {
		if len(dirty) == 1 && dirty["SelfService"] { // only self-service changed; use lighter update function
			if err := svc.ds.UpdateInstallerSelfServiceFlag(ctx, *payload.SelfService, existingInstaller.InstallerID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "updating installer self service flag")
			}
		} else {
			if payloadForNewInstallerFile != nil {
				if err := svc.storeSoftware(ctx, payloadForNewInstallerFile); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "storing software installer")
				}
			}

			// fill in values from existing installer if they weren't supplied
			if payload.InstallScript == nil {
				payload.InstallScript = &existingInstaller.InstallScript
			}
			if payload.UninstallScript == nil {
				payload.UninstallScript = &existingInstaller.UninstallScript
			}
			if payload.PostInstallScript == nil && !dirty["PostInstallScript"] {
				payload.PostInstallScript = &existingInstaller.PostInstallScript
			}
			if payload.PreInstallQuery == nil {
				payload.PreInstallQuery = &existingInstaller.PreInstallQuery
			}
			if payload.SelfService == nil {
				payload.SelfService = &existingInstaller.SelfService
			}

			if err := svc.ds.SaveInstallerUpdates(ctx, payload); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "saving installer updates")
			}

			// if we're updating anything other than self-service, we cancel pending installs/uninstalls,
			// and if we're updating the package we reset counts. This is run in its own transaction internally
			// for consistency, but independent of the installer update query as the main update should stick
			// even if side effects fail.
			if err := svc.ds.ProcessInstallerUpdateSideEffects(ctx, existingInstaller.InstallerID, true, dirty["Package"]); err != nil {
				return nil, err
			}
		}

		if err := svc.NewActivity(ctx, vc.User, activity); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "creating activity for edited software")
		}
	}

	// re-pull installer from database to ensure any side effects are accounted for; may be able to optimize this out later
	updatedInstaller, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, payload.TeamID, payload.TitleID, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "re-hydrating updated installer metadata")
	}

	statuses, err := svc.ds.GetSummaryHostSoftwareInstalls(ctx, updatedInstaller.InstallerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting updated installer statuses")
	}
	updatedInstaller.Status = statuses

	return updatedInstaller, nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) error {
	if teamID == nil {
		return fleet.NewInvalidArgumentError("team_id", "is required")
	}

	// we authorize with SoftwareInstaller here, but it uses the same AuthzType
	// as VPPApp, so this is correct for both software installers and VPP apps.
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// first, look for a software installer
	meta, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, titleID, false)
	if err != nil {
		if fleet.IsNotFound(err) {
			// no software installer, look for a VPP app
			meta, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting software app metadata")
			}
			return svc.deleteVPPApp(ctx, teamID, meta)
		}
		return ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}
	return svc.deleteSoftwareInstaller(ctx, meta)
}

func (svc *Service) deleteVPPApp(ctx context.Context, teamID *uint, meta *fleet.VPPAppStoreApp) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	if err := svc.ds.DeleteVPPAppFromTeam(ctx, teamID, meta.VPPAppID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting VPP app")
	}

	var teamName *string
	if teamID != nil && *teamID != 0 {
		t, err := svc.ds.Team(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team name for deleted VPP app")
		}
		teamName = &t.Name
	}

	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityDeletedAppStoreApp{
		AppStoreID:    meta.AdamID,
		SoftwareTitle: meta.Name,
		TeamName:      teamName,
		TeamID:        teamID,
		Platform:      meta.Platform,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleted VPP app")
	}

	return nil
}

func (svc *Service) deleteSoftwareInstaller(ctx context.Context, meta *fleet.SoftwareInstaller) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	if err := svc.ds.DeleteSoftwareInstaller(ctx, meta.InstallerID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting software installer")
	}

	var teamName *string
	if meta.TeamID != nil {
		t, err := svc.ds.Team(ctx, *meta.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team name for deleted software")
		}
		teamName = &t.Name
	}

	var teamID *uint
	switch {
	case meta.TeamID == nil:
		teamID = ptr.Uint(0)
	case meta.TeamID != nil:
		teamID = meta.TeamID
	}

	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeDeletedSoftware{
		SoftwareTitle:   meta.SoftwareTitle,
		SoftwarePackage: meta.Name,
		TeamName:        teamName,
		TeamID:          teamID,
		SelfService:     meta.SelfService,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleted software")
	}

	return nil
}

func (svc *Service) GetSoftwareInstallerMetadata(ctx context.Context, skipAuthz bool, titleID uint, teamID *uint) (*fleet.SoftwareInstaller,
	error,
) {
	if !skipAuthz {
		if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	meta, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, titleID, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}

	return meta, nil
}

func (svc *Service) GenerateSoftwareInstallerToken(ctx context.Context, alt string, titleID uint, teamID *uint) (string, error) {
	downloadRequested := alt == "media"
	if !downloadRequested {
		svc.authz.SkipAuthorization(ctx)
		return "", fleet.NewInvalidArgumentError("alt", "only alt=media is supported")
	}

	if teamID == nil {
		svc.authz.SkipAuthorization(ctx)
		return "", fleet.NewInvalidArgumentError("team_id", "is required")
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionRead); err != nil {
		return "", err
	}

	meta := fleet.SoftwareInstallerTokenMetadata{
		TitleID: titleID,
		TeamID:  *teamID,
	}
	metaByte, err := json.Marshal(meta)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "marshaling software installer metadata")
	}

	// Generate token and store in Redis
	token := uuid.NewString()
	const tokenExpirationMs = 10 * 60 * 1000 // 10 minutes
	ok, err := svc.distributedLock.SetIfNotExist(ctx, fmt.Sprintf("software_installer_token:%s", token), string(metaByte),
		tokenExpirationMs)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "saving software installer token")
	}
	if !ok {
		// Should not happen since token is unique
		return "", ctxerr.Errorf(ctx, "failed to save software installer token")
	}

	return token, nil
}

func (svc *Service) GetSoftwareInstallerTokenMetadata(ctx context.Context, token string,
	titleID uint,
) (*fleet.SoftwareInstallerTokenMetadata, error) {
	// We will manually authorize this endpoint based on the token.
	svc.authz.SkipAuthorization(ctx)

	if len(token) > softwareInstallerTokenMaxLength {
		return nil, fleet.NewPermissionError("invalid token")
	}

	metaStr, err := svc.distributedLock.GetAndDelete(ctx, fmt.Sprintf("software_installer_token:%s", token))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer token metadata")
	}
	if metaStr == nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewPermissionError("invalid token"))
	}

	var meta fleet.SoftwareInstallerTokenMetadata
	if err := json.Unmarshal([]byte(*metaStr), &meta); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshaling software installer token metadata")
	}

	if titleID != meta.TitleID {
		return nil, ctxerr.Wrap(ctx, fleet.NewPermissionError("invalid token"))
	}

	// The token is valid.
	return &meta, nil
}

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, skipAuthz bool, alt string, titleID uint,
	teamID *uint,
) (*fleet.DownloadSoftwareInstallerPayload, error) {
	downloadRequested := alt == "media"
	if !downloadRequested {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("alt", "only alt=media is supported")
	}

	if teamID == nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, fleet.NewInvalidArgumentError("team_id", "is required")
	}

	meta, err := svc.GetSoftwareInstallerMetadata(ctx, skipAuthz, titleID, teamID)
	if err != nil {
		return nil, err
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	access, err := svc.ds.ValidateOrbitSoftwareInstallerAccess(ctx, host.ID, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "check software installer access")
	}

	if !access {
		return nil, fleet.NewUserMessageError(errors.New("Host doesn't have access to this installer"), http.StatusForbidden)
	}

	// get the installer's metadata
	meta, err := svc.ds.GetSoftwareInstallerMetadataByID(ctx, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}

	// Note that we do allow downloading an installer that is on a different team
	// than the host's team, because the install request might have come while
	// the host was on that team, and then the host got moved to a different team
	// but the request is still pending execution.

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) getSoftwareInstallerBinary(ctx context.Context, storageID string, filename string) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// check if the installer exists in the store
	exists, err := svc.softwareInstallStore.Exists(ctx, storageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		return nil, ctxerr.Wrap(ctx, notFoundError{}, "does not exist in software installer store")
	}

	// get the installer from the store
	installer, size, err := svc.softwareInstallStore.Get(ctx, storageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting installer from store")
	}

	return &fleet.DownloadSoftwareInstallerPayload{
		Filename:  filename,
		Installer: installer,
		Size:      size,
	}, nil
}

func (svc *Service) InstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error {
	// we need to use ds.Host because ds.HostLite doesn't return the orbit
	// node key
	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		// if error is because the host does not exist, check first if the user
		// had access to install software (to prevent leaking valid host ids).
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionWrite); err != nil {
				return err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "get host")
	}

	platform := host.FleetPlatform()
	mobileAppleDevice := fleet.AppleDevicePlatform(platform) == fleet.IOSPlatform || fleet.AppleDevicePlatform(platform) == fleet.IPadOSPlatform

	if !mobileAppleDevice && (host.OrbitNodeKey == nil || *host.OrbitNodeKey == "") {
		// fleetd is required to install software so if the host is
		// enrolled via plain osquery we return an error
		svc.authz.SkipAuthorization(ctx)
		return fleet.NewUserMessageError(errors.New("Host doesn't have fleetd installed"), http.StatusUnprocessableEntity)
	}

	// authorize with the host's team
	if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if !mobileAppleDevice {
		installer, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID, false)
		if err != nil {
			if !fleet.IsNotFound(err) {
				return ctxerr.Wrap(ctx, err, "finding software installer for title")
			}
			installer = nil
		}

		// if we found an installer, use that
		if installer != nil {
			lastInstallRequest, err := svc.ds.GetHostLastInstallData(ctx, host.ID, installer.InstallerID)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "getting last install data for host %d and installer %d", host.ID, installer.InstallerID)
			}
			if lastInstallRequest != nil && lastInstallRequest.Status != nil &&
				(*lastInstallRequest.Status == fleet.SoftwareInstallPending || *lastInstallRequest.Status == fleet.SoftwareUninstallPending) {
				return &fleet.BadRequestError{
					Message: "Couldn't install software. Host has a pending install/uninstall request.",
					InternalErr: ctxerr.WrapWithData(
						ctx, err, "host already has a pending install/uninstall for this installer",
						map[string]any{
							"host_id":               host.ID,
							"software_installer_id": installer.InstallerID,
							"team_id":               host.TeamID,
							"title_id":              softwareTitleID,
						},
					),
				}
			}
			return svc.installSoftwareTitleUsingInstaller(ctx, host, installer)
		}
	}

	vppApp, err := svc.ds.GetVPPAppByTeamAndTitleID(ctx, host.TeamID, softwareTitleID)
	if err != nil {
		// if we couldn't find an installer or a VPP app, return a bad
		// request error
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Couldn't install software. Software title is not available for install. Please add software package or App Store app to install.",
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "couldn't find an installer or VPP app for software title",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}

		return ctxerr.Wrap(ctx, err, "finding VPP app for title")
	}

	_, err = svc.installSoftwareFromVPP(ctx, host, vppApp, mobileAppleDevice || fleet.AppleDevicePlatform(platform) == fleet.MacOSPlatform, false)
	return err
}

func (svc *Service) installSoftwareFromVPP(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, appleDevice bool, selfService bool) (string, error) {
	if !appleDevice {
		return "", &fleet.BadRequestError{
			Message: "VPP apps can only be installed only on Apple hosts.",
			InternalErr: ctxerr.NewWithData(
				ctx, "invalid host platform for requested installer",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": vppApp.TitleID},
			),
		}
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "fetching config to check MDM status")
	}

	if !config.MDM.EnabledAndConfigured {
		return "", fleet.NewUserMessageError(errors.New("Couldn't install. MDM is turned off. Please make sure that MDM is turned on to install App Store apps."), http.StatusUnprocessableEntity)
	}

	mdmConnected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "checking MDM status for host %d", host.ID)
	}

	if !mdmConnected {
		return "", &fleet.BadRequestError{
			Message: "Error: Couldn't install. To install App Store app, turn on MDM for this host.",
			InternalErr: ctxerr.NewWithData(
				ctx, "VPP install attempted on non-MDM host",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": vppApp.TitleID},
			),
		}
	}

	token, err := svc.getVPPToken(ctx, host.TeamID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting VPP token")
	}

	// at this moment, neither the UI or the back-end are prepared to
	// handle [asyncronous errors][1] on assignment, so before assigning a
	// device to a license, we need to:
	//
	// 1. Check if the app is already assigned to the serial number.
	// 2. If it's not assigned yet, check if we have enough licenses.
	//
	// A race still might happen, so async error checking needs to be
	// implemented anyways at some point.
	//
	// [1]: https://developer.apple.com/documentation/devicemanagement/app_and_book_management/handling_error_responses#3729433
	assignments, err := vpp.GetAssignments(token, &vpp.AssignmentFilter{AdamID: vppApp.AdamID, SerialNumber: host.HardwareSerial})
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting assignments from VPP API")
	}

	var eventID string

	// this app is not assigned to this device, check if we have licenses
	// left and assign it.
	if len(assignments) == 0 {
		assets, err := vpp.GetAssets(token, &vpp.AssetFilter{AdamID: vppApp.AdamID})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting assets from VPP API")
		}

		if len(assets) == 0 {
			level.Debug(svc.logger).Log(
				"msg", "trying to assign VPP asset to host",
				"adam_id", vppApp.AdamID,
				"host_serial", host.HardwareSerial,
			)
			return "", &fleet.BadRequestError{
				Message:     "Couldn't add software. <app_store_id> isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.",
				InternalErr: ctxerr.Errorf(ctx, "VPP API didn't return any assets for adamID %s", vppApp.AdamID),
			}
		}

		if len(assets) > 1 {
			return "", ctxerr.Errorf(ctx, "VPP API returned more than one asset for adamID %s", vppApp.AdamID)
		}

		if assets[0].AvailableCount <= 0 {
			return "", &fleet.BadRequestError{
				Message: "Couldn't install. No available licenses. Please purchase license in Apple Business Manager and try again.",
				InternalErr: ctxerr.NewWithData(
					ctx, "license available count <= 0",
					map[string]any{
						"host_id": host.ID,
						"team_id": host.TeamID,
						"adam_id": vppApp.AdamID,
						"count":   assets[0].AvailableCount,
					},
				),
			}
		}

		eventID, err = vpp.AssociateAssets(token, &vpp.AssociateAssetsRequest{Assets: assets, SerialNumbers: []string{host.HardwareSerial}})
		if err != nil {
			return "", ctxerr.Wrapf(ctx, err, "associating asset with adamID %s to host %s", vppApp.AdamID, host.HardwareSerial)
		}

	}

	// add command to install
	cmdUUID := uuid.NewString()
	err = svc.mdmAppleCommander.InstallApplication(ctx, []string{host.UUID}, cmdUUID, vppApp.AdamID)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "sending command to install VPP %s application to host with serial %s", vppApp.AdamID, host.HardwareSerial)
	}

	err = svc.ds.InsertHostVPPSoftwareInstall(ctx, host.ID, vppApp.VPPAppID, cmdUUID, eventID, selfService)
	if err != nil {
		return "", ctxerr.Wrapf(ctx, err, "inserting host vpp software install for host with serial %s and app with adamID %s", host.HardwareSerial, vppApp.AdamID)
	}

	return cmdUUID, nil
}

func (svc *Service) installSoftwareTitleUsingInstaller(ctx context.Context, host *fleet.Host, installer *fleet.SoftwareInstaller) error {
	ext := filepath.Ext(installer.Name)
	requiredPlatform := packageExtensionToPlatform(ext)
	if requiredPlatform == "" {
		// this should never happen
		return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
	}

	if host.FleetPlatform() != requiredPlatform {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("Package (%s) can be installed only on %s hosts.", ext, requiredPlatform),
			InternalErr: ctxerr.NewWithData(
				ctx, "invalid host platform for requested installer",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": installer.TitleID},
			),
		}
	}

	_, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, false, nil)
	return ctxerr.Wrap(ctx, err, "inserting software install request")
}

func (svc *Service) UninstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error {
	// we need to use ds.Host because ds.HostLite doesn't return the orbit node key
	host, err := svc.ds.Host(ctx, hostID)
	if err != nil {
		// if error is because the host does not exist, check first if the user
		// had access to install/uninstall software (to prevent leaking valid host ids).
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionWrite); err != nil {
				return err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return ctxerr.Wrap(ctx, err, "get host")
	}

	if host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" {
		// fleetd is required to install software so if the host is enrolled via plain osquery we return an error
		svc.authz.SkipAuthorization(ctx)
		return fleet.NewUserMessageError(errors.New("host does not have fleetd installed"), http.StatusUnprocessableEntity)
	}

	// If scripts are disabled (according to the last detail query), we return an error.
	// host.ScriptsEnabled may be nil for older orbit versions.
	if host.ScriptsEnabled != nil && !*host.ScriptsEnabled {
		svc.authz.SkipAuthorization(ctx)
		return fleet.NewUserMessageError(errors.New(fleet.RunScriptsOrbitDisabledErrMsg), http.StatusUnprocessableEntity)
	}

	// authorize with the host's team
	if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	installer, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID, false)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Couldn't uninstall software. Software title is not available for uninstall. Please add software package to install/uninstall.",
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "couldn't find an installer for software title",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}
		return ctxerr.Wrap(ctx, err, "finding software installer for title")
	}

	lastInstallRequest, err := svc.ds.GetHostLastInstallData(ctx, host.ID, installer.InstallerID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "getting last install data for host %d and installer %d", host.ID, installer.InstallerID)
	}
	if lastInstallRequest != nil && lastInstallRequest.Status != nil &&
		(*lastInstallRequest.Status == fleet.SoftwareInstallPending || *lastInstallRequest.Status == fleet.SoftwareUninstallPending) {
		return &fleet.BadRequestError{
			Message: "Couldn't uninstall software. Host has a pending install/uninstall request.",
			InternalErr: ctxerr.WrapWithData(
				ctx, err, "host already has a pending install/uninstall for this installer",
				map[string]any{
					"host_id":               host.ID,
					"software_installer_id": installer.InstallerID,
					"team_id":               host.TeamID,
					"title_id":              softwareTitleID,
					"status":                *lastInstallRequest.Status,
				},
			),
		}
	}

	// Validate platform
	ext := filepath.Ext(installer.Name)
	requiredPlatform := packageExtensionToPlatform(ext)
	if requiredPlatform == "" {
		// this should never happen
		return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
	}

	if host.FleetPlatform() != requiredPlatform {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("Package (%s) can be uninstalled only on %s hosts.", ext, requiredPlatform),
			InternalErr: ctxerr.NewWithData(
				ctx, "invalid host platform for requested uninstall",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": installer.TitleID},
			),
		}
	}

	// Get the uninstall script and use the standard script infrastructure to run it.
	contents, err := svc.ds.GetAnyScriptContents(ctx, installer.UninstallScriptContentID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError("software_title_id", `No uninstall script exists for the provided "software_title_id".`).
					WithStatus(http.StatusNotFound), "getting uninstall script contents")
		}
		return err
	}

	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}
	// create the script execution request; the host will be notified of the
	// script execution request via the orbit config's Notifications mechanism.
	request := fleet.HostScriptRequestPayload{
		HostID:          host.ID,
		ScriptContents:  string(contents),
		ScriptContentID: installer.UninstallScriptContentID,
		TeamID:          teamID,
	}
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		request.UserID = &ctxUser.ID
	}
	scriptResult, err := svc.ds.NewInternalScriptExecutionRequest(ctx, &request)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create script execution request")
	}

	// Update the host software installs table with the uninstall request.
	// Pending uninstalls will automatically show up in the UI Host Details -> Activity -> Upcoming tab.
	if err = svc.insertSoftwareUninstallRequest(ctx, scriptResult.ExecutionID, host, installer); err != nil {
		return err
	}

	return nil
}

func (svc *Service) insertSoftwareUninstallRequest(ctx context.Context, executionID string, host *fleet.Host,
	installer *fleet.SoftwareInstaller,
) error {
	if err := svc.ds.InsertSoftwareUninstallRequest(ctx, executionID, host.ID, installer.InstallerID); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting software uninstall request")
	}
	return nil
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	// Basic auth check
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	res, err := svc.ds.GetSoftwareInstallResults(ctx, resultUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionRead); err != nil {
				return nil, err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get software install result")
	}

	if res.HostDeletedAt == nil {
		// host is not deleted, get it and authorize for the host's team
		host, err := svc.ds.HostLite(ctx, res.HostID)
		// if error is because the host does not exist, check first if the user
		// had access to run a script (to prevent leaking valid host ids).
		if err != nil {
			if fleet.IsNotFound(err) {
				if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionRead); err != nil {
					return nil, err
				}
			}
			svc.authz.SkipAuthorization(ctx)
			return nil, ctxerr.Wrap(ctx, err, "get host lite")
		}
		// Team specific auth check
		if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: host.TeamID}, fleet.ActionRead); err != nil {
			return nil, err
		}
	} else {
		// host was deleted, authorize for no-team as a fallback
		if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	res.EnhanceOutputDetails()
	return res, nil
}

func (svc *Service) storeSoftware(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	// check if exists in the installer store
	exists, err := svc.softwareInstallStore.Exists(ctx, payload.StorageID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		if err := svc.softwareInstallStore.Put(ctx, payload.StorageID, payload.InstallerFile); err != nil {
			return ctxerr.Wrap(ctx, err, "storing installer")
		}
	}

	return nil
}

func (svc *Service) addMetadataToSoftwarePayload(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (extension string, err error) {
	if payload == nil {
		return "", ctxerr.New(ctx, "payload is required")
	}

	if payload.InstallerFile == nil {
		return "", ctxerr.New(ctx, "installer file is required")
	}

	meta, err := file.ExtractInstallerMetadata(payload.InstallerFile)
	if err != nil {
		if errors.Is(err, file.ErrUnsupportedType) {
			return "", &fleet.BadRequestError{
				Message:     "Couldn't edit software. File type not supported. The file should be .pkg, .msi, .exe, .deb or .rpm.",
				InternalErr: ctxerr.Wrap(ctx, err, "extracting metadata from installer"),
			}
		}
		return "", ctxerr.Wrap(ctx, err, "extracting metadata from installer")
	}

	if meta.Version == "" {
		return "", &fleet.BadRequestError{
			Message:     fmt.Sprintf("Couldn't add. Fleet couldn't read the version from %s.", payload.Filename),
			InternalErr: ctxerr.New(ctx, "extracting version from installer metadata"),
		}
	}

	if len(meta.PackageIDs) == 0 {
		return "", &fleet.BadRequestError{
			Message:     fmt.Sprintf("Couldn't add. Fleet couldn't read the package IDs, product code, or name from %s.", payload.Filename),
			InternalErr: ctxerr.New(ctx, "extracting package IDs from installer metadata"),
		}
	}

	payload.Title = meta.Name
	if payload.Title == "" {
		// use the filename if no title from metadata
		payload.Title = payload.Filename
	}
	payload.Version = meta.Version
	payload.StorageID = hex.EncodeToString(meta.SHASum)
	payload.BundleIdentifier = meta.BundleIdentifier
	payload.PackageIDs = meta.PackageIDs
	payload.Extension = meta.Extension

	// reset the reader (it was consumed to extract metadata)
	if err := payload.InstallerFile.Rewind(); err != nil {
		return "", ctxerr.Wrap(ctx, err, "resetting installer file reader")
	}

	if payload.InstallScript == "" {
		payload.InstallScript = file.GetInstallScript(meta.Extension)
	}

	if payload.UninstallScript == "" {
		payload.UninstallScript = file.GetUninstallScript(meta.Extension)
	}

	source, err := fleet.SofwareInstallerSourceFromExtensionAndName(meta.Extension, meta.Name)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "determining source from extension and name")
	}
	payload.Source = source

	platform, err := fleet.SofwareInstallerPlatformFromExtension(meta.Extension)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "determining platform from extension")
	}
	payload.Platform = platform

	return meta.Extension, nil
}

const (
	batchSoftwarePrefix = "software_batch_"
)

func (svc *Service) BatchSetSoftwareInstallers(
	ctx context.Context, tmName string, payloads []fleet.SoftwareInstallerPayload, dryRun bool,
) (string, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return "", err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return "", fleet.ErrNoContext
	}

	var teamID *uint
	if tmName != "" {
		tm, err := svc.ds.TeamByName(ctx, tmName)
		if err != nil {
			// If this is a dry run, the team may not have been created yet
			if dryRun && fleet.IsNotFound(err) {
				return "", nil
			}
			return "", err
		}
		teamID = &tm.ID
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return "", ctxerr.Wrap(ctx, err, "validating authorization")
	}

	var allScripts []string

	// Verify payloads first, to prevent starting the download+upload process if the data is invalid.
	for _, payload := range payloads {
		if len(payload.URL) > fleet.SoftwareInstallerURLMaxLength {
			return "", fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("software URL is too long, must be %d characters or less", fleet.SoftwareInstallerURLMaxLength),
			)
		}
		if _, err := url.ParseRequestURI(payload.URL); err != nil {
			return "", fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't edit software. URL (%q) is invalid", payload.URL),
			)
		}
		allScripts = append(allScripts, payload.InstallScript, payload.PostInstallScript, payload.UninstallScript)
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, allScripts); err != nil {
		return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("script", err.Error()))
	}

	// keyExpireTime is the current maximum time supported for retrieving
	// the result of a software by batch operation.
	const keyExpireTime = 24 * time.Hour

	requestUUID := uuid.NewString()
	if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID, batchSetProcessing, keyExpireTime); err != nil {
		return "", ctxerr.Wrapf(ctx, err, "failed to set key as %s", batchSetProcessing)
	}

	svc.logger.Log(
		"msg", "software batch start",
		"request_uuid", requestUUID,
		"team_id", teamID,
		"payloads", len(payloads),
	)

	go svc.softwareBatchUpload(
		requestUUID,
		teamID,
		vc.UserID(),
		payloads,
		dryRun,
	)

	return requestUUID, nil
}

const (
	batchSetProcessing   = "processing"
	batchSetCompleted    = "completed"
	batchSetFailedPrefix = "failed:"
)

func (svc *Service) softwareBatchUpload(
	requestUUID string,
	teamID *uint,
	userID uint,
	payloads []fleet.SoftwareInstallerPayload,
	dryRun bool,
) {
	var batchErr error

	// We do not use the request ctx on purpose because this method runs in the background.
	ctx := context.Background()

	defer func(start time.Time) {
		status := batchSetCompleted
		if batchErr != nil {
			status = fmt.Sprintf("%s%s", batchSetFailedPrefix, batchErr)
		}
		logger := log.With(svc.logger,
			"request_uuid", requestUUID,
			"team_id", teamID,
			"payloads", len(payloads),
			"status", status,
			"took", time.Since(start),
		)
		logger.Log("msg", "software batch done")
		// Give 10m for the client to read the result (it overrides the previos expiration time).
		if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID, status, 10*time.Minute); err != nil {
			logger.Log("msg", "failed to set result", "err", err)
		}
	}(time.Now())

	downloadURLFn := func(ctx context.Context, url string) (http.Header, *fleet.TempFileReader, error) {
		client := fleethttp.NewClient()
		client.Transport = fleethttp.NewSizeLimitTransport(fleet.MaxSoftwareInstallerSize)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("creating request for URL %q: %w", url, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			var maxBytesErr *http.MaxBytesError
			if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
				return nil, nil, fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %d GB", url, fleet.MaxSoftwareInstallerSize/(1000*1024*1024)),
				)
			}

			return nil, nil, fmt.Errorf("performing request for URL %q: %w", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotFound {
			return nil, nil, fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't edit software. URL (%q) returned \"Not Found\". Please make sure that URLs are reachable from your Fleet server.", url),
			)
		}

		// Allow all 2xx and 3xx status codes in this pass.
		if resp.StatusCode >= 400 {
			return nil, nil, fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't edit software. URL (%q) received response status code %d.", url, resp.StatusCode),
			)
		}

		tfr, err := fleet.NewTempFileReader(resp.Body, nil)
		if err != nil {
			// the max size error can be received either at client.Do or here when
			// reading the body if it's caught via a limited body reader.
			var maxBytesErr *http.MaxBytesError
			if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
				return nil, nil, fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %d GB", url, fleet.MaxSoftwareInstallerSize/(1000*1024*1024)),
				)
			}
			return nil, nil, fmt.Errorf("reading installer %q contents: %w", url, err)
		}

		return resp.Header, tfr, nil
	}

	var g errgroup.Group
	g.SetLimit(1) // TODO: consider whether we can increase this limit, see https://github.com/fleetdm/fleet/issues/22704#issuecomment-2397407837
	// critical to avoid data race, the slice is pre-allocated and each
	// goroutine only writes to its index.
	installers := make([]*fleet.UploadSoftwareInstallerPayload, len(payloads))

	for i, p := range payloads {
		i, p := i, p

		g.Go(func() error {
			headers, tfr, err := downloadURLFn(ctx, p.URL)
			if err != nil {
				return err
			}

			// NOTE: cannot defer tfr.Close() here because the reader needs to be
			// available after the goroutine completes. Instead, all temp file
			// readers will have their Close deferred after the join/wait of
			// goroutines.
			installer := &fleet.UploadSoftwareInstallerPayload{
				TeamID:             teamID,
				InstallScript:      p.InstallScript,
				PreInstallQuery:    p.PreInstallQuery,
				PostInstallScript:  p.PostInstallScript,
				UninstallScript:    p.UninstallScript,
				InstallerFile:      tfr,
				SelfService:        p.SelfService,
				UserID:             userID,
				URL:                p.URL,
				InstallDuringSetup: p.InstallDuringSetup,
			}

			// set the filename before adding metadata, as it is used as fallback
			var filename string
			cdh, ok := headers["Content-Disposition"]
			if ok && len(cdh) > 0 {
				_, params, err := mime.ParseMediaType(cdh[0])
				if err == nil {
					filename = params["filename"]
				}
			}
			installer.Filename = filename

			ext, err := svc.addMetadataToSoftwarePayload(ctx, installer)
			if err != nil {
				_ = tfr.Close() // closing the temp file here since it will not be available after the goroutine completes
				return err
			}

			// Update $PACKAGE_ID in uninstall script
			preProcessUninstallScript(installer)

			// if filename was empty, try to extract it from the URL with the
			// now-known extension
			if filename == "" {
				filename = file.ExtractFilenameFromURLPath(p.URL, ext)
			}
			// if empty, resort to a default name
			if filename == "" {
				filename = fmt.Sprintf("package.%s", ext)
			}
			installer.Filename = filename
			if installer.Title == "" {
				installer.Title = filename
			}

			installers[i] = installer

			return nil
		})
	}

	waitErr := g.Wait()

	// defer close for any valid temp file reader
	for _, payload := range installers {
		if payload != nil && payload.InstallerFile != nil {
			defer payload.InstallerFile.Close()
		}
	}

	if waitErr != nil {
		// NOTE: intentionally not wrapping to avoid polluting user errors.
		batchErr = waitErr
		return
	}

	if dryRun {
		return
	}

	for _, payload := range installers {
		if err := svc.storeSoftware(ctx, payload); err != nil {
			batchErr = fmt.Errorf("storing software installer %q: %w", payload.Filename, err)
			return
		}
	}

	if err := svc.ds.BatchSetSoftwareInstallers(ctx, teamID, installers); err != nil {
		batchErr = fmt.Errorf("batch set software installers: %w", err)
		return
	}

	// Note: per @noahtalerman we don't want activity items for CLI actions
	// anymore, so that's intentionally skipped.
}

func (svc *Service) GetBatchSetSoftwareInstallersResult(ctx context.Context, tmName string, requestUUID string, dryRun bool) (string, string, []fleet.SoftwarePackageResponse, error) {
	// We've already authorized in the POST /api/latest/fleet/software/batch,
	// but adding it here so we don't need to worry about a special case endpoint.
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return "", "", nil, err
	}

	result, err := svc.keyValueStore.Get(ctx, batchSoftwarePrefix+requestUUID)
	if err != nil {
		return "", "", nil, ctxerr.Wrap(ctx, err, "failed to get result")
	}
	if result == nil {
		return "", "", nil, ctxerr.Wrap(ctx, notFoundError{}, "request_uuid not found")
	}

	switch {
	case *result == batchSetCompleted:
		if dryRun {
			return fleet.BatchSetSoftwareInstallersStatusCompleted, "", nil, nil
		} // this will fall through to retrieving software packages if not a dry run.
	case *result == batchSetProcessing:
		return fleet.BatchSetSoftwareInstallersStatusProcessing, "", nil, nil
	case strings.HasPrefix(*result, batchSetFailedPrefix):
		message := strings.TrimPrefix(*result, batchSetFailedPrefix)
		return fleet.BatchSetSoftwareInstallersStatusFailed, message, nil, nil
	default:
		return "", "", nil, ctxerr.New(ctx, "invalid status")
	}

	var (
		teamID    uint  // GetSoftwareInstallers uses 0 for "No team"
		ptrTeamID *uint // Authorize uses *uint for "No team" teamID
	)
	if tmName != "" {
		team, err := svc.ds.TeamByName(ctx, tmName)
		if err != nil {
			return "", "", nil, ctxerr.Wrap(ctx, err, "load team by name")
		}
		teamID = team.ID
		ptrTeamID = &team.ID
	}

	// We've already authorized in the POST /api/latest/fleet/software/batch,
	// but adding it here so we don't need to worry about a special case endpoint.
	//
	// We use fleet.ActionWrite because this method is the counterpart of the POST
	// /api/latest/fleet/software/batch.
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: ptrTeamID}, fleet.ActionWrite); err != nil {
		return "", "", nil, ctxerr.Wrap(ctx, err, "validating authorization")
	}

	softwarePackages, err := svc.ds.GetSoftwareInstallers(ctx, teamID)
	if err != nil {
		return "", "", nil, ctxerr.Wrap(ctx, err, "get software installers")
	}

	return fleet.BatchSetSoftwareInstallersStatusCompleted, "", softwarePackages, nil
}

func (svc *Service) SelfServiceInstallSoftwareTitle(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	installer, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID, false)
	if err != nil {
		if !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "finding software installer for title")
		}
		installer = nil
	}

	if installer != nil {
		if !installer.SelfService {
			return &fleet.BadRequestError{
				Message: "Software title is not available through self-service",
				InternalErr: ctxerr.NewWithData(
					ctx, "software title not available through self-service",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}

		ext := filepath.Ext(installer.Name)
		requiredPlatform := packageExtensionToPlatform(ext)
		if requiredPlatform == "" {
			// this should never happen
			return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
		}

		if host.FleetPlatform() != requiredPlatform {
			return &fleet.BadRequestError{
				Message: fmt.Sprintf("Package (%s) can be installed only on %s hosts.", ext, requiredPlatform),
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "invalid host platform for requested installer",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}

		_, err = svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, true, nil)
		return ctxerr.Wrap(ctx, err, "inserting self-service software install request")
	}

	vppApp, err := svc.ds.GetVPPAppByTeamAndTitleID(ctx, host.TeamID, softwareTitleID)
	if err != nil {
		// if we couldn't find an installer or a VPP app, return a bad
		// request error
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Couldn't install software. Software title is not available for install. Please add software package or App Store app to install.",
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "couldn't find an installer or VPP app for software title",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}

		return ctxerr.Wrap(ctx, err, "finding VPP app for title")
	}

	if !vppApp.SelfService {
		return &fleet.BadRequestError{
			Message: "Software title is not available through self-service",
			InternalErr: ctxerr.NewWithData(
				ctx, "software title not available through self-service",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
			),
		}
	}

	platform := host.FleetPlatform()
	mobileAppleDevice := fleet.AppleDevicePlatform(platform) == fleet.IOSPlatform || fleet.AppleDevicePlatform(platform) == fleet.IPadOSPlatform

	_, err = svc.installSoftwareFromVPP(ctx, host, vppApp, mobileAppleDevice || fleet.AppleDevicePlatform(platform) == fleet.MacOSPlatform, true)
	return err
}

// packageExtensionToPlatform returns the platform name based on the
// package extension. Returns an empty string if there is no match.
func packageExtensionToPlatform(ext string) string {
	var requiredPlatform string
	switch ext {
	case ".msi", ".exe":
		requiredPlatform = "windows"
	case ".pkg", ".dmg", ".zip":
		requiredPlatform = "darwin"
	case ".deb", ".rpm":
		requiredPlatform = "linux"
	default:
		return ""
	}

	return requiredPlatform
}

func UninstallSoftwareMigration(
	ctx context.Context,
	ds fleet.Datastore,
	softwareInstallStore fleet.SoftwareInstallerStore,
	logger kitlog.Logger,
) error {
	// Find software installers without package_id
	idMap, err := ds.GetSoftwareInstallersWithoutPackageIDs(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting software installers without package_id")
	}
	if len(idMap) == 0 {
		return nil
	}

	// Download each package and parse it
	for id, storageID := range idMap {
		// check if the installer exists in the store
		exists, err := softwareInstallStore.Exists(ctx, storageID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking if installer exists")
		}
		if !exists {
			level.Warn(logger).Log("msg", "software installer not found in store", "software_installer_id", id, "storage_id", storageID)
			continue
		}

		// get the installer from the store
		installer, _, err := softwareInstallStore.Get(ctx, storageID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting installer from store")
		}

		tfr, err := fleet.NewTempFileReader(installer, nil)
		_ = installer.Close()
		if err != nil {
			level.Warn(logger).Log("msg", "extracting metadata from installer", "software_installer_id", id, "storage_id", storageID, "err",
				err)
			continue
		}
		meta, err := file.ExtractInstallerMetadata(tfr)
		_ = tfr.Close() // best-effort closing and deleting of temp file
		if err != nil {
			level.Warn(logger).Log("msg", "extracting metadata from installer", "software_installer_id", id, "storage_id", storageID, "err",
				err)
			continue
		}
		if len(meta.PackageIDs) == 0 {
			level.Warn(logger).Log("msg", "no package_id found in metadata", "software_installer_id", id, "storage_id", storageID)
			continue
		}
		if meta.Extension == "" {
			level.Warn(logger).Log("msg", "no extension found in metadata", "software_installer_id", id, "storage_id", storageID)
			continue
		}
		payload := fleet.UploadSoftwareInstallerPayload{
			PackageIDs: meta.PackageIDs,
			Extension:  meta.Extension,
		}
		payload.UninstallScript = file.GetUninstallScript(payload.Extension)

		// Update $PACKAGE_ID in uninstall script
		preProcessUninstallScript(&payload)

		// Update the package_id and extension in the software installer and the uninstall script
		if err := ds.UpdateSoftwareInstallerWithoutPackageIDs(ctx, id, payload); err != nil {
			return ctxerr.Wrap(ctx, err, "updating package_id in software installer")
		}
	}

	return nil
}
