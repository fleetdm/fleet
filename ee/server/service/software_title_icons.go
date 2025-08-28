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
	if icon == nil {
		vppApp, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &teamID, titleID)
		if vppApp != nil || vppApp.IconURL != nil {
			return nil, nil, nil, &fleet.VPPIconAvailableError{IconURL: *vppApp.IconURL}
		}

		return nil, nil, nil, ctxerr.Wrap(ctx, err, "getting software title icon")
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

func (svc *Service) UploadSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (fleet.SoftwareTitleIcon, error) {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return fleet.SoftwareTitleIcon{}, err
	}
	var softwareInstaller *fleet.SoftwareInstaller
	var vppApp *fleet.VPPAppStoreApp

	softwareInstaller, err = svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, &payload.TeamID, payload.TitleID, false)
	if err != nil && !fleet.IsNotFound(err) {
		return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "getting software installer")
	}
	if softwareInstaller == nil {
		vppApp, err = svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &payload.TeamID, payload.TitleID)
		if err != nil && !fleet.IsNotFound(err) {
			return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "getting VPP app")
		}
	}
	if softwareInstaller == nil && vppApp == nil {
		return fleet.SoftwareTitleIcon{}, &fleet.BadRequestError{Message: fmt.Sprintf("Software title has no software installer or VPP app: %d", payload.TitleID)}
	}

	// get sha256 of icon file
	payload.StorageID, err = file.SHA256FromTempFileReader(payload.IconFile)
	if err != nil {
		return fleet.SoftwareTitleIcon{}, err
	}

	// store icon
	exists, err := svc.softwareTitleIconStore.Exists(ctx, payload.StorageID)
	if err != nil {
		return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		if err := svc.softwareTitleIconStore.Put(ctx, payload.StorageID, payload.IconFile); err != nil {
			return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "storing icon")
		}
	}

	softwareTitleIcon, err := svc.ds.CreateOrUpdateSoftwareTitleIcon(ctx, payload)
	if err != nil {
		return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "creating or updating software title icon")
	}

	return *softwareTitleIcon, nil
}

func (svc *Service) DeleteSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) error {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	err = svc.ds.DeleteSoftwareTitleIcon(ctx, teamID, titleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting software title icon")
	}

	return nil
}
