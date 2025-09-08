package service

import (
	"context"
	"fmt"
	"io"
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) ([]byte, int64, string, error) {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, 0, "", err
	}

	icon, err := svc.ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon")
	}
	if icon == nil {
		vppApp, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &teamID, titleID)
		if vppApp != nil && vppApp.IconURL != nil {
			return nil, 0, "", &fleet.VPPIconAvailable{IconURL: *vppApp.IconURL}
		}

		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon")
	}

	iconData, size, err := svc.softwareTitleIconStore.Get(ctx, icon.StorageID)
	if err != nil {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon data")
	}
	defer iconData.Close()
	imageBytes, err := io.ReadAll(iconData)
	if err != nil {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "reading icon data")
	}

	return imageBytes, size, icon.Filename, nil
}

func (svc *Service) UploadSoftwareTitleIcon(ctx context.Context, payload *fleet.UploadSoftwareTitleIconPayload) (fleet.SoftwareTitleIcon, error) {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return fleet.SoftwareTitleIcon{}, err
	}
	var softwareInstaller *fleet.SoftwareInstaller
	var vppApp *fleet.VPPAppStoreApp

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.SoftwareTitleIcon{}, fleet.ErrNoContext
	}
	user := vc.User

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

	icon, err := svc.ds.GetSoftwareTitleIcon(ctx, payload.TeamID, payload.TitleID)
	if err != nil && !fleet.IsNotFound(err) {
		return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "getting software title icon")
	}

	if payload.IconFile != nil {
		// get sha256 of icon file
		payload.StorageID, err = file.SHA256FromTempFileReader(payload.IconFile)
		if err != nil {
			return fleet.SoftwareTitleIcon{}, err
		}
	}

	if icon == nil || icon.StorageID != payload.StorageID {
		var exists bool

		hasAccess, err := userHasAccessToStorageID(ctx, svc, payload.TeamID, payload.StorageID)
		if err != nil {
			return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "checking access to existing software title icon")
		}

		if hasAccess {
			exists, err = svc.softwareTitleIconStore.Exists(ctx, payload.StorageID)
			if err != nil {
				return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "checking if software title icon exists")
			}
		}

		if !exists {
			if payload.IconFile != nil {
				if err := svc.softwareTitleIconStore.Put(ctx, payload.StorageID, payload.IconFile); err != nil {
					return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "storing icon")
				}
			} else {
				return fleet.SoftwareTitleIcon{}, ctxerr.New(ctx, fmt.Sprintf("software title icon with hash '%s' does not exist", payload.StorageID))
			}
		}
	}

	softwareTitleIcon, err := svc.ds.CreateOrUpdateSoftwareTitleIcon(ctx, payload)
	if err != nil {
		return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "creating or updating software title icon")
	}

	// if anything on the icon has changed, we need to generate a new activity
	if icon == nil || icon.StorageID != softwareTitleIcon.StorageID || icon.Filename != softwareTitleIcon.Filename {
		iconUrl := fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", softwareTitleIcon.SoftwareTitleID, softwareTitleIcon.TeamID)
		activityDetailsForSoftwareTitleIcon, err := svc.ds.ActivityDetailsForSoftwareTitleIcon(ctx, payload.TeamID, payload.TitleID)
		if err != nil {
			return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "fetching software title icon activity details")
		}
		err = generateEditActivityForSoftwareTitleIcon(ctx, svc, user, iconUrl, activityDetailsForSoftwareTitleIcon)
		if err != nil {
			return fleet.SoftwareTitleIcon{}, ctxerr.Wrap(ctx, err, "generating edit activity for software title icon")
		}
	}

	return *softwareTitleIcon, nil
}

func (svc *Service) DeleteSoftwareTitleIcon(ctx context.Context, teamID uint, titleID uint) error {
	var err error
	if err = svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	user := vc.User
	activityDetailsForSoftwareTitleIcon, err := svc.ds.ActivityDetailsForSoftwareTitleIcon(ctx, teamID, titleID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching software title icon activity details")
	}
	var iconUrl string
	if activityDetailsForSoftwareTitleIcon.VPPIconUrl != nil {
		iconUrl = *activityDetailsForSoftwareTitleIcon.VPPIconUrl
	}

	err = svc.ds.DeleteSoftwareTitleIcon(ctx, teamID, titleID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "deleting software title icon")
	}

	// since delete is idempotent, we only want to generate an activity if the
	// software title icon was actually deleted. This error will be a not found error,
	// so if it exists, skip the activity generation
	if err == nil {
		err = generateEditActivityForSoftwareTitleIcon(ctx, svc, user, iconUrl, activityDetailsForSoftwareTitleIcon)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "generating edit activity for software title icon")
		}
	}

	return nil
}

func userHasAccessToStorageID(ctx context.Context, svc *Service, teamId uint, storageID string) (bool, error) {
	teamIds, err := svc.ds.GetTeamIdsForIconStorageId(ctx, storageID)
	if err != nil {
		return false, err
	}
	if slices.Contains(teamIds, teamId) {
		return true, nil
	}

	for _, tmID := range teamIds {
		if authErr := svc.authz.Authorize(ctx, &fleet.SoftwareTitleIcon{TeamID: tmID}, fleet.ActionWrite); authErr != nil {
			continue
		}

		return true, nil
	}

	return false, nil
}

func generateEditActivityForSoftwareTitleIcon(ctx context.Context, svc *Service, user *fleet.User, iconUrl string, activityDetailsForSoftwareTitleIcon fleet.DetailsForSoftwareIconActivity) error {
	if activityDetailsForSoftwareTitleIcon.AdamID != nil {
		if err := svc.NewActivity(ctx, user, fleet.ActivityEditedAppStoreApp{
			SoftwareTitle:    activityDetailsForSoftwareTitleIcon.SoftwareTitle,
			SoftwareTitleID:  activityDetailsForSoftwareTitleIcon.SoftwareTitleID,
			AppStoreID:       *activityDetailsForSoftwareTitleIcon.AdamID,
			TeamName:         activityDetailsForSoftwareTitleIcon.TeamName,
			TeamID:           &activityDetailsForSoftwareTitleIcon.TeamID,
			Platform:         *activityDetailsForSoftwareTitleIcon.Platform,
			SelfService:      activityDetailsForSoftwareTitleIcon.SelfService,
			SoftwareIconURL:  &iconUrl,
			LabelsIncludeAny: activityDetailsForSoftwareTitleIcon.LabelsIncludeAny,
			LabelsExcludeAny: activityDetailsForSoftwareTitleIcon.LabelsExcludeAny,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "creating activity for software title icon")
		}

		return nil
	}

	if activityDetailsForSoftwareTitleIcon.SoftwareInstallerID != nil {
		if err := svc.NewActivity(ctx, user, fleet.ActivityTypeEditedSoftware{
			SoftwareTitle:    activityDetailsForSoftwareTitleIcon.SoftwareTitle,
			SoftwarePackage:  activityDetailsForSoftwareTitleIcon.Filename,
			TeamName:         activityDetailsForSoftwareTitleIcon.TeamName,
			TeamID:           &activityDetailsForSoftwareTitleIcon.TeamID,
			SelfService:      activityDetailsForSoftwareTitleIcon.SelfService,
			SoftwareIconURL:  &iconUrl,
			LabelsIncludeAny: activityDetailsForSoftwareTitleIcon.LabelsIncludeAny,
			LabelsExcludeAny: activityDetailsForSoftwareTitleIcon.LabelsExcludeAny,
			SoftwareTitleID:  activityDetailsForSoftwareTitleIcon.SoftwareTitleID,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "creating activity for software title icon")
		}

		return nil
	}

	return ctxerr.New(ctx, "no software installer or VPP app found for software title icon")
}
