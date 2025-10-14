package service

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) updateInHouseAppInstaller(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload, vc viewer.Viewer, teamName *string, software *fleet.SoftwareTitle) (*fleet.SoftwareInstaller, error) {
	existingInstaller, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, payload.TeamID, payload.TitleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting existing installer")
	}

	if payload.SelfService == nil && payload.InstallerFile == nil && payload.PreInstallQuery == nil &&
		payload.InstallScript == nil && payload.PostInstallScript == nil && payload.UninstallScript == nil &&
		payload.LabelsIncludeAny == nil && payload.LabelsExcludeAny == nil {
		return existingInstaller, nil // no payload, noop
	}

	payload.InstallerID = existingInstaller.InstallerID

	_, validatedLabels, err := ValidateSoftwareLabelsForUpdate(ctx, svc, existingInstaller, payload.LabelsIncludeAny, payload.LabelsExcludeAny)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating software labels for update")
	}
	payload.ValidatedLabels = validatedLabels

	// activity team ID must be null if no team, not zero
	var actTeamID *uint
	if payload.TeamID != nil && *payload.TeamID != 0 {
		actTeamID = payload.TeamID
	}
	activity := fleet.ActivityTypeEditedSoftware{
		SoftwareTitle:   existingInstaller.SoftwareTitle,
		TeamName:        teamName,
		TeamID:          actTeamID,
		SoftwarePackage: &existingInstaller.Name,
		SoftwareTitleID: payload.TitleID,
		SoftwareIconURL: existingInstaller.IconUrl,
	}

	var payloadForNewInstallerFile *fleet.UploadSoftwareInstallerPayload
	if payload.InstallerFile != nil {
		payloadForNewInstallerFile = &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: payload.InstallerFile,
			Filename:      payload.Filename,
		}

		newInstallerExtension, err := svc.addMetadataToSoftwarePayload(ctx, payloadForNewInstallerFile, false)
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

		} else { // noop if uploaded installer is identical to previous installer
			payloadForNewInstallerFile = nil
			payload.InstallerFile = nil
		}
	}

	if payload.InstallerFile == nil { // fill in existing existingInstaller data to payload
		payload.StorageID = existingInstaller.StorageID
		payload.Filename = existingInstaller.Name
		payload.Version = existingInstaller.Version
	}

	// persist changes starting here, now that we've done all the validation/diffing we can
	if payloadForNewInstallerFile != nil {
		if err := svc.storeSoftware(ctx, payloadForNewInstallerFile); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "storing software installer")
		}
	}

	if err := svc.ds.SaveInHouseAppUpdates(ctx, payload); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "saving installer updates")
	}

	if err := svc.ds.RemovePendingInHouseAppInstalls(ctx, existingInstaller.InstallerID); err != nil {
		return nil, err
	}

	// now that the payload has been updated with any patches, we can set the
	// final fields of the activity
	actLabelsIncl, actLabelsExcl := activitySoftwareLabelsFromSoftwareScopeLabels(
		existingInstaller.LabelsIncludeAny, existingInstaller.LabelsExcludeAny)
	if payload.ValidatedLabels != nil {
		actLabelsIncl, actLabelsExcl = activitySoftwareLabelsFromValidatedLabels(payload.ValidatedLabels)
	}
	activity.LabelsIncludeAny = actLabelsIncl
	activity.LabelsExcludeAny = actLabelsExcl
	if err := svc.NewActivity(ctx, vc.User, activity); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating activity for edited in house app")
	}

	// re-pull installer from database to ensure any side effects are accounted for; may be able to optimize this out later
	updatedInstaller, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, payload.TeamID, payload.TitleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "re-hydrating updated installer metadata")
	}

	statuses, err := svc.ds.GetSummaryInHouseAppInstalls(ctx, payload.TeamID, updatedInstaller.InstallerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting updated installer statuses")
	}
	updatedInstaller.Status = statuses

	return updatedInstaller, nil
}

func (svc *Service) GetInHouseAppManifest(ctx context.Context, titleID uint, teamID *uint) ([]byte, error) {
	// TODO(JVE): use time-based JWT auth here, this is just for testing
	svc.authz.SkipAuthorization(ctx)

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get in house app manifest: get app config")
	}

	var tid uint
	if teamID != nil {
		tid = *teamID
	}
	downloadUrl := fmt.Sprintf("%s/api/latest/fleet/software/titles/%d/in_house_app?team_id=%d", appConfig.ServerSettings.ServerURL, titleID, tid)

	meta, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get in house app manifest: get in house app metadata")
	}

	tmpl := template.Must(template.New("").Parse(`
<plist version="1.0">
  <dict>
    <key>items</key>
    <array>
      <dict>
        <key>assets</key>
        <array>
          <dict>
            <key>kind</key>
            <string>software-package</string>
            <key>url</key>
            <string>{{ .URL }}</string>
          </dict>
          <dict>
            <key>kind</key>
            <string>display-image</string>
            <key>needs-shine</key>
            <true/>
            <key>url</key>
            <string/>
          </dict>
        </array>
        <key>metadata</key>
        <dict>
          <key>bundle-identifier</key>
          <string>{{ .BundleID }}</string>
          <key>bundle-version</key>
          <string>{{ .Version }}</string>
          <key>kind</key>
          <string>software</string>
          <key>title</key>
          <string>{{ .Name }}</string>
        </dict>
      </dict>
    </array>
  </dict>
</plist>`))

	buf := bytes.NewBuffer([]byte{})

	err = tmpl.Execute(buf, struct {
		BundleID string
		Version  string
		Name     string
		URL      string
	}{meta.BundleIdentifier, meta.Version, meta.SoftwareTitle, downloadUrl})

	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "rendering app manifest")
	}

	return buf.Bytes(), nil
}

func (svc *Service) GetInHouseAppPackage(ctx context.Context, titleID uint, teamID *uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// TODO(JVE): JWT with expiration for auth
	svc.authz.SkipAuthorization(ctx)

	meta, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get in house app package: get in house app metadata")
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, "installer.ipa")
}
