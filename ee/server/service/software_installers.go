package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	// make sure all scripts use unix-style newlines to prevent errors when
	// running them, browsers use windows-style newlines, which breaks the
	// shebang when the file is directly executed.
	payload.InstallScript = file.Dos2UnixNewlines(payload.InstallScript)
	payload.PostInstallScript = file.Dos2UnixNewlines(payload.PostInstallScript)

	if _, err := svc.addMetadataToSoftwarePayload(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "adding metadata to payload")
	}

	if err := svc.storeSoftware(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "storing software installer")
	}

	// TODO: basic validation of install and post-install script (e.g., supported interpreters)?
	// TODO: any validation of pre-install query?

	installerID, err := svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
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
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for added software")
	}

	return nil
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
	if teamID != nil {
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

	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeDeletedSoftware{
		SoftwareTitle:   meta.SoftwareTitle,
		SoftwarePackage: meta.Name,
		TeamName:        teamName,
		TeamID:          meta.TeamID,
		SelfService:     meta.SelfService,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleted software")
	}

	return nil
}

func (svc *Service) GetSoftwareInstallerMetadata(ctx context.Context, titleID uint, teamID *uint) (*fleet.SoftwareInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	meta, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, titleID, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}

	return meta, nil
}

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	if teamID == nil {
		return nil, fleet.NewInvalidArgumentError("team_id", "is required")
	}

	meta, err := svc.GetSoftwareInstallerMetadata(ctx, titleID, teamID)
	if err != nil {
		return nil, err
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	_, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
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
			return svc.installSoftwareTitleUsingInstaller(ctx, host, installer)
		}
	}

	vppApp, err := svc.ds.GetVPPAppByTeamAndTitleID(ctx, host.TeamID, softwareTitleID, false)
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

	return svc.installSoftwareFromVPP(ctx, host, vppApp, mobileAppleDevice || fleet.AppleDevicePlatform(platform) == fleet.MacOSPlatform)
}

func (svc *Service) installSoftwareFromVPP(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, appleDevice bool) error {
	if !appleDevice {
		return &fleet.BadRequestError{
			Message: "VPP apps can only be installed only on Apple hosts.",
			InternalErr: ctxerr.NewWithData(
				ctx, "invalid host platform for requested installer",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": vppApp.TitleID},
			),
		}
	}

	config, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching config to check MDM status")
	}

	if !config.MDM.EnabledAndConfigured {
		return fleet.NewUserMessageError(errors.New("Couldn't install. MDM is turned off. Please make sure that MDM is turned on to install App Store apps."), http.StatusUnprocessableEntity)
	}

	mdmConnected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "checking MDM status for host %d", host.ID)
	}

	if !mdmConnected {
		return &fleet.BadRequestError{
			Message: "VPP apps can only be installed only on hosts enrolled in MDM.",
			InternalErr: ctxerr.NewWithData(
				ctx, "VPP install attempted on non-MDM host",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": vppApp.TitleID},
			),
		}
	}

	token, err := svc.getVPPToken(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting VPP token")
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
		return ctxerr.Wrap(ctx, err, "getting assignments from VPP API")
	}

	var eventID string

	// this app is not assigned to this device, check if we have licenses
	// left and assign it.
	if len(assignments) == 0 {
		assets, err := vpp.GetAssets(token, &vpp.AssetFilter{AdamID: vppApp.AdamID})
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting assets from VPP API")
		}

		if len(assets) == 0 {
			level.Debug(svc.logger).Log(
				"msg", "trying to assign VPP asset to host",
				"adam_id", vppApp.AdamID,
				"host_serial", host.HardwareSerial,
			)
			return &fleet.BadRequestError{
				Message:     "Couldn't add software. <app_store_id> isn't available in Apple Business Manager. Please purchase license in Apple Business Manager and try again.",
				InternalErr: ctxerr.Errorf(ctx, "VPP API didn't return any assets for adamID %s", vppApp.AdamID),
			}
		}

		if len(assets) > 1 {
			return ctxerr.Errorf(ctx, "VPP API returned more than one asset for adamID %s", vppApp.AdamID)
		}

		if assets[0].AvailableCount <= 0 {
			return &fleet.BadRequestError{
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
			return ctxerr.Wrapf(ctx, err, "associating asset with adamID %s to host %s", vppApp.AdamID, host.HardwareSerial)
		}

	}

	user := authz.UserFromContext(ctx)

	// add command to install
	cmdUUID := uuid.NewString()
	err = svc.mdmAppleCommander.InstallApplication(ctx, []string{host.UUID}, cmdUUID, vppApp.AdamID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "sending command to install VPP %s application to host with serial %s", vppApp.AdamID, host.HardwareSerial)
	}

	err = svc.ds.InsertHostVPPSoftwareInstall(ctx, host.ID, user.ID, vppApp.VPPAppID, cmdUUID, eventID)
	if err != nil {
		return ctxerr.Wrapf(ctx, err, "inserting host vpp software install for host with serial %s and app with adamID %s", host.HardwareSerial, vppApp.AdamID)
	}

	return nil
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

	_, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, false)
	return ctxerr.Wrap(ctx, err, "inserting software install request")
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
				Message:     "Couldn't edit software. File type not supported. The file should be .pkg, .msi, .exe or .deb.",
				InternalErr: ctxerr.Wrap(ctx, err, "extracting metadata from installer"),
			}
		}
		return "", ctxerr.Wrap(ctx, err, "extracting metadata from installer")
	}
	payload.Title = meta.Name
	if payload.Title == "" {
		// use the filename if no title from metadata
		payload.Title = payload.Filename
	}
	payload.Version = meta.Version
	payload.StorageID = hex.EncodeToString(meta.SHASum)
	payload.BundleIdentifier = meta.BundleIdentifier

	// reset the reader (it was consumed to extract metadata)
	if _, err := payload.InstallerFile.Seek(0, 0); err != nil {
		return "", ctxerr.Wrap(ctx, err, "resetting installer file reader")
	}

	if payload.InstallScript == "" {
		payload.InstallScript = file.GetInstallScript(meta.Extension)
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

const maxInstallerSizeBytes int64 = 1024 * 1024 * 500

func (svc *Service) BatchSetSoftwareInstallers(ctx context.Context, tmName string, payloads []fleet.SoftwareInstallerPayload, dryRun bool) error {
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return err
	}

	var teamID *uint
	if tmName != "" {
		tm, err := svc.ds.TeamByName(ctx, tmName)
		if err != nil {
			// If this is a dry run, the team may not have been created yet
			if dryRun && fleet.IsNotFound(err) {
				return nil
			}
			return err
		}
		teamID = &tm.ID
	}

	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "validating authorization")
	}

	g, workerCtx := errgroup.WithContext(ctx)
	g.SetLimit(3)
	// critical to avoid data race, the slice is pre-allocated and each
	// goroutine only writes to its index.
	installers := make([]*fleet.UploadSoftwareInstallerPayload, len(payloads))

	client := fleethttp.NewClient()
	client.Transport = fleethttp.NewSizeLimitTransport(maxInstallerSizeBytes)
	for i, p := range payloads {
		i, p := i, p

		g.Go(func() error {
			// validate the URL before doing the request
			_, err := url.ParseRequestURI(p.URL)
			if err != nil {
				return fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q) is invalid", p.URL),
				)
			}

			req, err := http.NewRequestWithContext(workerCtx, http.MethodGet, p.URL, nil)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "creating request for URL %s", p.URL)
			}

			resp, err := client.Do(req)
			if err != nil {
				var maxBytesErr *http.MaxBytesError
				if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
					return fleet.NewInvalidArgumentError(
						"software.url",
						fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %d MB", p.URL, maxInstallerSizeBytes/(1024*1024)),
					)
				}

				return ctxerr.Wrapf(ctx, err, "performing request for URL %s", p.URL)
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				return fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q) doesn't exist. Please make sure that URLs are publicy accessible to the internet.", p.URL),
				)
			}

			// Allow all 2xx and 3xx status codes in this pass.
			if resp.StatusCode > 400 {
				return fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q) received response status code %d.", p.URL, resp.StatusCode),
				)
			}

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				// the max size error can be received either at client.Do or here when
				// reading the body if it's caught via a limited body reader.
				var maxBytesErr *http.MaxBytesError
				if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
					return fleet.NewInvalidArgumentError(
						"software.url",
						fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %d MB", p.URL, maxInstallerSizeBytes/(1024*1024)),
					)
				}
				return ctxerr.Wrapf(ctx, err, "reading installer %q contents", p.URL)
			}

			installer := &fleet.UploadSoftwareInstallerPayload{
				TeamID:            teamID,
				InstallScript:     p.InstallScript,
				PreInstallQuery:   p.PreInstallQuery,
				PostInstallScript: p.PostInstallScript,
				InstallerFile:     bytes.NewReader(bodyBytes),
				SelfService:       p.SelfService,
			}

			// set the filename before adding metadata, as it is used as fallback
			var filename string
			cdh, ok := resp.Header["Content-Disposition"]
			if ok && len(cdh) > 0 {
				_, params, err := mime.ParseMediaType(cdh[0])
				if err == nil {
					filename = params["filename"]
				}
			}
			installer.Filename = filename

			ext, err := svc.addMetadataToSoftwarePayload(ctx, installer)
			if err != nil {
				return err
			}

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

	if err := g.Wait(); err != nil {
		// NOTE: intentionally not wrapping to avoid polluting user
		// errors.
		return err
	}

	if dryRun {
		return nil
	}

	for _, payload := range installers {
		if err := svc.storeSoftware(ctx, payload); err != nil {
			return ctxerr.Wrap(ctx, err, "storing software installer")
		}
	}

	if err := svc.ds.BatchSetSoftwareInstallers(ctx, teamID, installers); err != nil {
		return ctxerr.Wrap(ctx, err, "batch set software installers")
	}

	// Note: per @noahtalerman we don't want activity items for CLI actions
	// anymore, so that's intentionally skipped.

	return nil
}

func (svc *Service) SelfServiceInstallSoftwareTitle(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	installer, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID, false)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Software title has no package added. Please add software package to install.",
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "couldn't find an installer for software title",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}

		return ctxerr.Wrap(ctx, err, "finding software installer for title")
	}

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

	_, err = svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, true)
	return ctxerr.Wrap(ctx, err, "inserting self-service software install request")
}

// packageExtensionToPlatform returns the platform name based on the
// package extension. Returns an empty string if there is no match.
func packageExtensionToPlatform(ext string) string {
	var requiredPlatform string
	switch ext {
	case ".msi", ".exe":
		requiredPlatform = "windows"
	case ".pkg":
		requiredPlatform = "darwin"
	case ".deb":
		requiredPlatform = "linux"
	default:
		return ""
	}

	return requiredPlatform
}
