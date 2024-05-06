package service

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log/level"
)

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}
	if payload == nil {
		return ctxerr.New(ctx, "payload is required")
	}

	if payload.InstallerFile == nil {
		return ctxerr.New(ctx, "installer file is required")
	}

	title, vers, hash, err := file.ExtractInstallerMetadata(payload.Filename, payload.InstallerFile)
	if err != nil {
		// TODO: confirm error handling
		if strings.Contains(err.Error(), "unsupported file type") {
			return &fleet.BadRequestError{
				Message:     "The file should be .pkg, .msi, .exe or .deb.",
				InternalErr: ctxerr.Wrap(ctx, err, "extracting metadata from installer"),
			}
		}
		return ctxerr.Wrap(ctx, err, "extracting metadata from installer")
	}
	payload.Title = title
	payload.Version = vers
	payload.StorageID = hex.EncodeToString(hash)

	// checck if exists in the installer store
	exists, err := svc.softwareInstallStore.Exists(ctx, payload.StorageID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		// reset the reader before storing (it was consumed to extract metadata)
		if _, err := payload.InstallerFile.Seek(0, 0); err != nil {
			return ctxerr.Wrap(ctx, err, "resetting installer file reader")
		}
		if err := svc.softwareInstallStore.Put(ctx, payload.StorageID, payload.InstallerFile); err != nil {
			return ctxerr.Wrap(ctx, err, "storing installer")
		}
	}

	if payload.InstallScript == "" {
		payload.InstallScript = file.GetInstallScript(payload.Filename)
	}

	// TODO: basic validation of install and post-install script (e.g., supported interpreters)?

	// TODO: any validation of pre-install query?

	source, err := fleet.SofwareInstallerSourceFromFilename(payload.Filename)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "determining source from filename")
	}
	payload.Source = source

	installerID, err := svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "matching or creating software installer")
	}
	level.Debug(svc.logger).Log("msg", "software installer uploaded", "installer_id", installerID)

	// TODO: QA what breaks when you have a software title with no versions?

	return nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, id uint) error {
	// get the software installer to have its team id
	meta, err := svc.ds.GetSoftwareInstallerMetadata(ctx, id)
	if err != nil {
		if fleet.IsNotFound(err) {
			// couldn't get the metadata to have its team, authorize with a no-team
			// as a fallback - the requested installer does not exist so there's
			// no way to know what team it would be for, and returning a 404 without
			// authorization would leak the existing/non existing ids.
			if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{}, fleet.ActionWrite); err != nil {
				return err
			}
			return ctxerr.Wrap(ctx, err, "getting software installer metadata")
		}
	}

	// do the actual authorization with the software installer's team id
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: meta.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	if err := svc.ds.DeleteSoftwareInstaller(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting software installer")
	}

	return nil
}

func (svc *Service) GetSoftwareInstallerMetadata(ctx context.Context, installerID uint) (*fleet.SoftwareInstaller, error) {
	// first do a basic authorization check, any logged in user can read teams
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// get the installer's metadata
	meta, err := svc.ds.GetSoftwareInstallerMetadata(ctx, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}

	// authorize with the software installer's team id
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: meta.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return meta, nil
}

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	meta, err := svc.GetSoftwareInstallerMetadata(ctx, installerID)
	if err != nil {
		return nil, err
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	// TODO: confirm error handling

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	// get the installer's metadata
	meta, err := svc.ds.GetSoftwareInstallerMetadata(ctx, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer metadata")
	}

	// ensure it cannot get access to a different team's installer
	var hTeamID uint
	if host.TeamID != nil {
		hTeamID = *host.TeamID
	}
	if (meta.TeamID != nil && *meta.TeamID != hTeamID) || (meta.TeamID == nil && hTeamID != 0) {
		return nil, ctxerr.Wrap(ctx, fleet.OrbitError{}, "host team does not match installer team")
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) getSoftwareInstallerBinary(ctx context.Context, storageID string, filename string) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// check if the installer exists in the store
	exists, err := svc.softwareInstallStore.Exists(ctx, storageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		return nil, ctxerr.Wrap(ctx, err, "does not exist in software installer store")
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

	if host.OrbitNodeKey == nil || *host.OrbitNodeKey == "" {
		// fleetd is required to install software so if the host is
		// enrolled via plain osquery we return an error
		svc.authz.SkipAuthorization(ctx)
		// TODO(roberto): for cleanup task, confirm with product error message.
		return fleet.NewUserMessageError(errors.New("Host doesn't have fleetd installed"), http.StatusUnprocessableEntity)
	}

	// authorize with the host's team
	if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return err
	}

	installer, err := svc.ds.GetSoftwareInstallerForTitle(ctx, softwareTitleID, host.TeamID)
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

	ext := filepath.Ext(installer.Name)
	var requiredPlatform string
	switch ext {
	case ".msi", ".exe":
		requiredPlatform = "windows"
	case ".pkg":
		requiredPlatform = "darwin"
	case ".deb":
		requiredPlatform = "linux"
	default:
		// this should never happen
		return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
	}

	hostPlatform := host.FleetPlatform()
	if hostPlatform != requiredPlatform {
		return &fleet.BadRequestError{
			Message: fmt.Sprintf("Package (%s) can be installed only on %s hosts.", ext, hostPlatform),
			InternalErr: ctxerr.WrapWithData(
				ctx, err, "invalid host platform for requested installer",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
			),
		}
	}

	err = svc.ds.InsertSoftwareInstallRequest(ctx, hostID, installer.ID)
	return ctxerr.Wrap(ctx, err, "inserting software install request")
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	// Basic auth check
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	res, err := svc.ds.GetSoftwareInstallResults(ctx, resultUUID)
	if err != nil {
		return nil, err
	}

	// Team specific auth check
	if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: res.HostTeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return res, nil
}
