package service

import (
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
		installerType := file.InstallerType(strings.TrimPrefix(filepath.Ext(payload.Filename), "."))
		installerPath := "some path" // TODO: where does this come from?
		payload.InstallScript = file.GetInstallScript(installerType, installerPath)
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

	err = svc.ds.InsertSoftwareInstallRequest(ctx, hostID, softwareTitleID, host.TeamID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message:     "The software title provided doesn't have an installer",
				InternalErr: ctxerr.Wrapf(ctx, err, "couldn't find an installer for software title"),
			}
		}

		return ctxerr.Wrap(ctx, err, "inserting software install request")
	}

	return nil
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	// TODO(JVE): check the host is in the right team?
	if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	slog.With("filename", "ee/server/service/software_installers.go", "func", "GetSoftwareInstallResults").Info("JVE_LOG: we out here\n\n\n\n ")

	res, err := svc.ds.GetSoftwareInstallResults(ctx, resultUUID)
	if err != nil {
		return nil, err
	}

	return res, nil
}
