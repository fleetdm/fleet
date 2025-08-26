package service

import (
	"context"
	"fmt"
	"io"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) ([]byte, *int64, *string, error) {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, nil, err
	}

	icon, err := svc.ds.GetSoftwareTitleIcon(ctx, teamID, titleID, nil)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "getting software title icon")
	}
	vppApp, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &teamID, titleID)
	if vppApp.IconURL != nil {
		return nil, nil, nil, &fleet.VPPIconAvailableError{IconURL: *vppApp.IconURL}
	}

	iconData, size, err := svc.softwareTitleIconStore.Get(ctx, icon.StorageID)
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "getting software title icon data")
	}
	defer iconData.Close()
	imageBytes, err := io.ReadAll(iconData)
	if err != nil {
		return nil, nil, nil, ctxerr.Wrap(ctx, err, "reading icon data")
	}

	return imageBytes, &size, &icon.Filename, nil
}

func (svc *Service) UploadSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (*fleet.SoftwareTitleIcon, error) {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}
	var softwareInstaller *fleet.SoftwareInstaller
	var vppApp *fleet.VPPAppStoreApp

	softwareInstaller, err = svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &payload.TeamID, payload.TitleID, false)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, ctxerr.Wrap(ctx, err, "getting software installer")
	}
	if softwareInstaller == nil {
		vppApp, err = svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &payload.TeamID, payload.TitleID)
		if err != nil && !fleet.IsNotFound(err) {
			return nil, ctxerr.Wrap(ctx, err, "getting VPP app")
		}
	}
	if softwareInstaller == nil && vppApp == nil {
		return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Software title has no software installer or VPP app: %d", payload.TitleID)}
	}

	// get sha256 of icon file
	payload.StorageID, err = file.SHA256FromTempFileReader(payload.IconFile)
	if err != nil {
		return nil, err
	}

	// store icon
	exists, err := svc.softwareTitleIconStore.Exists(ctx, payload.StorageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		if _, err := payload.IconFile.Seek(0, 0); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "seeking back to start")
		}

		if err := svc.softwareTitleIconStore.Put(ctx, payload.StorageID, payload.IconFile); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "storing icon")
		}
	}

	softwareTitleIcon, err := svc.ds.CreateOrUpdateSoftwareTitleIcon(ctx, payload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating or updating software title icon")
	}

	return softwareTitleIcon, nil
}
