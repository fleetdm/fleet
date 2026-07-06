package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/retry"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

const softwareInstallerTokenMaxLength = 36 // UUID length

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (*fleet.SoftwareInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: payload.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if payload.AutomaticInstall {
		// Currently, same write permissions are applied on software and policies,
		// but leaving this here in case it changes in the future.
		if err := svc.authz.Authorize(ctx, &fleet.Policy{PolicyData: fleet.PolicyData{TeamID: payload.TeamID}}, fleet.ActionWrite); err != nil {
			return nil, err
		}
	}

	// validate labels before we do anything else
	validatedLabels, err := ValidateSoftwareLabels(ctx, svc, payload.TeamID, payload.LabelsIncludeAny, payload.LabelsExcludeAny, payload.LabelsIncludeAll)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating software labels")
	}
	payload.ValidatedLabels = validatedLabels

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	payload.UserID = vc.UserID()

	// make sure all scripts use unix-style newlines to prevent errors when
	// running them, browsers use windows-style newlines, which breaks the
	// shebang when the file is directly executed.
	payload.InstallScript = file.Dos2UnixNewlines(payload.InstallScript)
	payload.PostInstallScript = file.Dos2UnixNewlines(payload.PostInstallScript)
	payload.UninstallScript = file.Dos2UnixNewlines(payload.UninstallScript)

	failOnBlankScript := !strings.HasSuffix(payload.Filename, ".ipa")

	if _, err := svc.addMetadataToSoftwarePayload(ctx, payload, failOnBlankScript); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "adding metadata to payload")
	}

	// Validate iOS/iPadOS managed app configuration up-front. For non-.ipa extensions, silently drop.
	if payload.Extension == "ipa" {
		if len(payload.Configuration) > 0 {
			if err := fleet.ValidateAppleAppConfiguration(payload.Configuration); err != nil {
				return nil, err
			}
		}
	} else {
		payload.Configuration = nil
	}

	// A script package's install script is the uploaded file, validated in
	// addScriptPackageMetadata, so only post-install/uninstall are checked here.
	scriptsToValidate := []struct {
		name    string
		content string
	}{
		{"post-install script", payload.PostInstallScript},
		{"uninstall script", payload.UninstallScript},
	}
	if !fleet.IsScriptPackage(payload.Extension) {
		scriptsToValidate = append(scriptsToValidate, struct {
			name    string
			content string
		}{"install script", payload.InstallScript})
	}
	for _, scriptVal := range scriptsToValidate {
		if err := fleet.ValidateSoftwareInstallerScript(scriptVal.content, payload.Platform); err != nil {
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("Couldn't add. %s validation failed: %s", scriptVal.name, err.Error()),
			}
		}
	}

	if payload.AutomaticInstall && payload.AutomaticInstallQuery == "" {
		switch {
		//
		// For "msi", addMetadataToSoftwarePayload fails before this point if product code cannot be extracted.
		//
		case payload.Extension == "exe" || payload.Extension == "tar.gz" || fleet.IsScriptPackage(payload.Extension):
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("Couldn't add. Fleet can't create a policy to detect existing installations for .%s packages. Please add the software, add a custom policy, and enable the install software policy automation.", payload.Extension),
			}
		case payload.Extension == "pkg" && payload.BundleIdentifier == "":
			// For pkgs without bundle identifier the request usually fails before reaching this point,
			// but addMetadataToSoftwarePayload may not fail if the package has "package IDs" but not a "bundle identifier",
			// in which case we want to fail here because we cannot generate a policy without a bundle identifier.
			return nil, &fleet.BadRequestError{
				Message: "Couldn't add. Policy couldn't be created because bundle identifier can't be extracted.",
			}
		}
	}

	if err := svc.storeSoftware(ctx, payload); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "storing software installer")
	}

	// Update $PACKAGE_ID/$UPGRADE_CODE in uninstall script
	if err := preProcessUninstallScript(payload); err != nil {
		return nil, &fleet.BadRequestError{
			Message: fmt.Sprintf("Couldn't add software: %s", err),
		}
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{payload.InstallScript, payload.PostInstallScript, payload.UninstallScript}); err != nil {
		// We redo the validation on each script to find out which script has the missing secret.
		// This is done to provide a more informative error message to the UI user.
		var argErr *fleet.InvalidArgumentError
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "install script", &payload.InstallScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "post-install script", &payload.PostInstallScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "uninstall script", &payload.UninstallScript, argErr)
		if argErr != nil {
			return nil, argErr
		}
		// We should not get to this point. If we did, it means we have another issue, such as large read replica latency.
		return nil, ctxerr.Wrap(ctx, err, "transient server issue validating embedded secrets")
	}
	if err := svc.ds.ValidateReferencedCustomHostVitals(ctx, []string{payload.InstallScript, payload.PostInstallScript, payload.UninstallScript}); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	installerID, titleID, err := svc.ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "matching or creating software installer")
	}
	svc.logger.DebugContext(ctx, "software installer uploaded", "installer_id", installerID)

	var teamName *string
	if payload.TeamID != nil && *payload.TeamID != 0 {
		t, err := svc.ds.TeamLite(ctx, *payload.TeamID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting team name on upload software installer")
		}
		teamName = &t.Name
	}

	actLabelsInclAny, actLabelsExclAny, actLabelsInclAll := activitySoftwareLabelsFromValidatedLabels(payload.ValidatedLabels)
	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeAddedSoftware{
		SoftwareTitle:    payload.Title,
		SoftwarePackage:  payload.Filename,
		TeamName:         teamName,
		TeamID:           payload.TeamID,
		SelfService:      payload.SelfService,
		SoftwareTitleID:  titleID,
		LabelsIncludeAny: actLabelsInclAny,
		LabelsExcludeAny: actLabelsExclAny,
		LabelsIncludeAll: actLabelsInclAll,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating activity for added software")
	}

	// get values for response object
	var tmID uint
	if payload.TeamID != nil {
		tmID = *payload.TeamID
	}

	if payload.Extension == "ipa" {
		addedInstaller, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, &tmID, titleID)
		if err != nil {
			return nil, err
		}
		// Wrap iOS / iPadOS plist as a JSON string for the response.
		if len(addedInstaller.Configuration) > 0 {
			wrapped, err := json.Marshal(string(addedInstaller.Configuration))
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "wrapping configuration for response")
			}
			addedInstaller.Configuration = wrapped
		}
		return addedInstaller, nil
	}

	addedInstaller, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctxdb.RequirePrimary(ctx, true), &tmID, titleID, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting added software installer")
	}

	if payload.AutomaticInstall {
		policyAct := fleet.ActivityTypeCreatedPolicy{
			ID:   addedInstaller.AutomaticInstallPolicies[0].ID,
			Name: addedInstaller.AutomaticInstallPolicies[0].Name,
		}

		if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), policyAct); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "create activity for create automatic install policy for custom package")
		}
	}

	return addedInstaller, nil
}

func ValidateSoftwareLabels(ctx context.Context, svc fleet.Service, teamID *uint, labelsIncludeAny, labelsExcludeAny, labelsIncludeAll []string) (*fleet.LabelIdentsWithScope, error) {
	if authctx, ok := authz_ctx.FromContext(ctx); !ok {
		return nil, fleet.NewAuthRequiredError("validate software labels: missing authorization context")
	} else if !authctx.Checked() {
		return nil, fleet.NewAuthRequiredError("validate software labels: method requires previous authorization")
	}

	var count int
	for _, set := range [][]string{labelsIncludeAny, labelsExcludeAny, labelsIncludeAll} {
		if len(set) > 0 {
			count++
		}
	}
	if count > 1 {
		return nil, &fleet.BadRequestError{Message: `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included.`}
	}

	var names []string
	var scope fleet.LabelScope
	switch {
	case len(labelsIncludeAny) > 0:
		names = labelsIncludeAny
		scope = fleet.LabelScopeIncludeAny
	case len(labelsExcludeAny) > 0:
		names = labelsExcludeAny
		scope = fleet.LabelScopeExcludeAny
	case len(labelsIncludeAll) > 0:
		names = labelsIncludeAll
		scope = fleet.LabelScopeIncludeAll
	}

	if len(names) == 0 {
		// nothing to validate, return empty result
		return &fleet.LabelIdentsWithScope{}, nil
	}

	byName, err := svc.BatchValidateLabels(ctx, teamID, names)
	if err != nil {
		var missingLabelErr *fleet.MissingLabelError
		if errors.As(err, &missingLabelErr) {
			return nil, &fleet.BadRequestError{
				InternalErr: missingLabelErr,
				Message:     fmt.Sprintf("Couldn't update. Label %q doesn't exist. Please remove the label from the software.", missingLabelErr.MissingLabelName),
			}
		}
		return nil, err
	}

	return &fleet.LabelIdentsWithScope{
		LabelScope: scope,
		ByName:     byName,
	}, nil
}

func preProcessUninstallScript(payload *fleet.UploadSoftwareInstallerPayload) error {
	if len(payload.PackageIDs) == 0 {
		// do nothing, this could be a FMA which won't include the installer when editing the scripts
		return nil
	}

	// dmg and zip don't use template variable substitution
	switch payload.Extension {
	case "dmg", "zip":
		return nil
	}

	// Only validate and substitute $PACKAGE_ID if it appears in the script
	if file.PackageIDRegex.MatchString(payload.UninstallScript) {
		if err := file.ValidatePackageIdentifiers(payload.PackageIDs, ""); err != nil {
			return err
		}

		var packageID string
		switch payload.Extension {
		case "pkg":
			var sb strings.Builder
			_, _ = sb.WriteString("(\n")
			for _, pkgID := range payload.PackageIDs {
				_, _ = sb.WriteString(fmt.Sprintf("  '%s'\n", pkgID))
			}
			_, _ = sb.WriteString(")") // no ending newline
			packageID = sb.String()
		default:
			packageID = fmt.Sprintf("'%s'", payload.PackageIDs[0])
		}

		payload.UninstallScript = file.PackageIDRegex.ReplaceAllString(payload.UninstallScript, fmt.Sprintf("%s${suffix}", packageID))
	}

	// Only validate and substitute $UPGRADE_CODE if the template variable appears in the script
	if file.UpgradeCodeRegex.MatchString(payload.UninstallScript) {
		if payload.UpgradeCode == "" {
			return errors.New("$UPGRADE_CODE variable was used in uninstall script but package does not have an UpgradeCode")
		}
		if err := file.ValidatePackageIdentifiers(nil, payload.UpgradeCode); err != nil {
			return err
		}

		payload.UninstallScript = file.UpgradeCodeRegex.ReplaceAllString(payload.UninstallScript, fmt.Sprintf("'%s'${suffix}", payload.UpgradeCode))
	}

	return nil
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
		return nil, &fleet.BadRequestError{Message: "fleet_id is required; enter 0 for unassigned"}
	}

	var teamName *string
	if *payload.TeamID != 0 {
		t, err := svc.ds.TeamLite(ctx, *payload.TeamID)
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
		// We redo the validation on each script to find out which script has the missing secret.
		// This is done to provide a more informative error message to the UI user.
		var argErr *fleet.InvalidArgumentError
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "install script", payload.InstallScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "post-install script", payload.PostInstallScript, argErr)
		argErr = svc.validateEmbeddedSecretsOnScript(ctx, "uninstall script", payload.UninstallScript, argErr)
		if argErr != nil {
			return nil, argErr
		}
		// We should not get to this point. If we did, it means we have another issue, such as large read replica latency.
		return nil, ctxerr.Wrap(ctx, err, "transient server issue validating embedded secrets")
	}
	if err := svc.ds.ValidateReferencedCustomHostVitals(ctx, scripts); err != nil {
		return nil, fleet.NewInvalidArgumentError("script", err.Error())
	}

	// get software by ID, fail if it does not exist or does not have an existing installer
	software, err := svc.ds.SoftwareTitleByID(ctx, payload.TitleID, payload.TeamID, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting software title by id")
	}

	dirty := make(map[string]bool)

	if payload.Categories != nil {
		categories, catIDs, err := svc.removeDuplicateOrMissingCategories(ctx, ptr.ValOrZero(payload.TeamID), payload.Categories)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "filtering software installer categories")
		}
		payload.Categories = categories
		payload.CategoryIDs = catIDs
		dirty["Categories"] = true
	}

	// Handle in house apps separately
	if software.InHouseAppCount == 1 {
		return svc.updateInHouseAppInstaller(ctx, payload, vc, teamName, software)
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

	if payload.IsNoopPayload(software) {
		return existingInstaller, nil // no payload, noop
	}

	payload.InstallerID = existingInstaller.InstallerID

	if payload.DisplayName != nil && *payload.DisplayName != software.DisplayName {
		trimmed := strings.TrimSpace(*payload.DisplayName)
		if trimmed == "" && *payload.DisplayName != "" {
			return nil, fleet.NewInvalidArgumentError("display_name", "Cannot have a display name that is all whitespace.")
		}

		*payload.DisplayName = trimmed
		dirty["DisplayName"] = true
	}

	if payload.SelfService != nil && *payload.SelfService != existingInstaller.SelfService {
		dirty["SelfService"] = true
	}

	shouldUpdateLabels, validatedLabels, err := ValidateSoftwareLabelsForUpdate(ctx, svc, existingInstaller, payload.LabelsIncludeAny, payload.LabelsExcludeAny, payload.LabelsIncludeAll)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating software labels for update")
	}
	if shouldUpdateLabels {
		dirty["Labels"] = true
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
		SelfService:     existingInstaller.SelfService,
		SoftwarePackage: &existingInstaller.Name,
		SoftwareTitleID: payload.TitleID,
		SoftwareIconURL: existingInstaller.IconUrl,
	}

	if payload.SelfService != nil && *payload.SelfService != existingInstaller.SelfService {
		dirty["SelfService"] = true
		activity.SelfService = *payload.SelfService
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
			payload.PackageIDs = payloadForNewInstallerFile.PackageIDs
			payload.UpgradeCode = payloadForNewInstallerFile.UpgradeCode

			dirty["Package"] = true

			// For script packages the uploaded file's contents are the install
			// script, so replacing the file must update install_script too.
			if fleet.IsScriptPackage(existingInstaller.Extension) {
				payload.InstallScript = &payloadForNewInstallerFile.InstallScript
				if payloadForNewInstallerFile.InstallScript != existingInstaller.InstallScript {
					dirty["InstallScript"] = true
				}
			}
		} else { // noop if uploaded installer is identical to previous installer
			payloadForNewInstallerFile = nil
			payload.InstallerFile = nil
		}

		if existingInstaller.FleetMaintainedAppID != nil {
			return nil, &fleet.BadRequestError{
				Message:     "Couldn't update. The package can't be changed for Fleet-maintained apps.",
				InternalErr: ctxerr.Wrap(ctx, err, "installer file changed for fleet maintained app installer"),
			}
		}
	}

	if payload.InstallerFile == nil { // fill in existing existingInstaller data to payload
		payload.StorageID = existingInstaller.StorageID
		payload.Filename = existingInstaller.Name
		payload.Version = existingInstaller.Version
		payload.PackageIDs = existingInstaller.PackageIDs()
		payload.UpgradeCode = existingInstaller.UpgradeCode
	}

	isScriptPackage := fleet.IsScriptPackage(existingInstaller.Extension)

	// default pre-install query is blank, so blanking out the query doesn't have a semantic meaning we have to take care of
	if payload.PreInstallQuery != nil {
		if *payload.PreInstallQuery != existingInstaller.PreInstallQuery {
			dirty["PreInstallQuery"] = true
		}
	}

	if payload.InstallScript != nil {
		if isScriptPackage {
			// A script package's install script comes from the uploaded file.
			// Ignore a user-provided install_script value, but keep one derived
			// from a newly uploaded file (set above).
			if payloadForNewInstallerFile == nil {
				payload.InstallScript = nil
			}
		} else {
			installScript := file.Dos2UnixNewlines(*payload.InstallScript)
			installScript = getInstallScript(existingInstaller.Extension, existingInstaller.PackageIDs(), installScript)
			if installScript == "" {
				return nil, &fleet.BadRequestError{
					Message: fmt.Sprintf("Couldn't edit. Install script is required for .%s packages.", strings.ToLower(existingInstaller.Extension)),
				}
			}

			if err := fleet.ValidateSoftwareInstallerScript(installScript, existingInstaller.Platform); err != nil {
				return nil, &fleet.BadRequestError{
					Message: fmt.Sprintf("Couldn't edit. install script validation failed: %s", err.Error()),
				}
			}

			if installScript != existingInstaller.InstallScript {
				dirty["InstallScript"] = true
			}
			payload.InstallScript = &installScript
		}
	}

	if payload.PostInstallScript != nil {
		postInstallScript := file.Dos2UnixNewlines(*payload.PostInstallScript)

		if err := fleet.ValidateSoftwareInstallerScript(postInstallScript, existingInstaller.Platform); err != nil {
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("Couldn't edit. post-install script validation failed: %s", err.Error()),
			}
		}

		if postInstallScript != existingInstaller.PostInstallScript {
			dirty["PostInstallScript"] = true
		}
		payload.PostInstallScript = &postInstallScript
	}

	if payload.UninstallScript != nil {
		uninstallScript := file.Dos2UnixNewlines(*payload.UninstallScript)
		// Script packages have no default uninstall script and may leave it empty;
		// other types fall back to a default and require one.
		if !isScriptPackage {
			if uninstallScript == "" { // extension can't change on an edit so we can generate off of the existing file
				uninstallScript = file.GetUninstallScript(existingInstaller.Extension)
				if payload.UpgradeCode != "" {
					uninstallScript = file.UninstallMsiWithUpgradeCodeScript
				}
			}
			if uninstallScript == "" {
				return nil, &fleet.BadRequestError{
					Message: fmt.Sprintf("Couldn't edit. Uninstall script is required for .%s packages.", strings.ToLower(existingInstaller.Extension)),
				}
			}
		}

		if err := fleet.ValidateSoftwareInstallerScript(uninstallScript, existingInstaller.Platform); err != nil {
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("Couldn't edit. uninstall script validation failed: %s", err.Error()),
			}
		}

		payloadForUninstallScript := &fleet.UploadSoftwareInstallerPayload{
			Extension:       existingInstaller.Extension,
			UninstallScript: uninstallScript,
			PackageIDs:      existingInstaller.PackageIDs(),
			UpgradeCode:     existingInstaller.UpgradeCode,
		}
		if payloadForNewInstallerFile != nil {
			payloadForUninstallScript.PackageIDs = payloadForNewInstallerFile.PackageIDs
			payloadForUninstallScript.UpgradeCode = payloadForNewInstallerFile.UpgradeCode
		}

		if err := preProcessUninstallScript(payloadForUninstallScript); err != nil {
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("Couldn't edit software: %s", err),
			}
		}

		if payloadForUninstallScript.UninstallScript != existingInstaller.UninstallScript {
			dirty["UninstallScript"] = true
		}
		uninstallScript = payloadForUninstallScript.UninstallScript
		payload.UninstallScript = &uninstallScript
	}

	// switch active installer to one that matches the pinned version
	var activeInstallerID uint
	if payload.PinnedVersion != nil {
		if existingInstaller.FleetMaintainedAppID == nil {
			return nil, &fleet.BadRequestError{
				Message: `Couldn't update. "version" can be only specified for a software title that has a Fleet-maintained app.`,
			}
		}

		if len(dirty) > 0 {
			return nil, &fleet.BadRequestError{
				Message: `Couldn't update. "version" can't be changed at the same time as other fields.`,
			}
		}

		*payload.PinnedVersion = strings.TrimSpace(*payload.PinnedVersion)
		majorVersionString, usesCaret, err := parsePinnedVersion(ctx, *payload.PinnedVersion)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "reading Fleet-maintained app pinned version")
		}

		versions, err := svc.ds.GetFleetMaintainedVersionsByTitleID(ctx, payload.TeamID, payload.TitleID, true)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "getting Fleet-maintained app versions")
		}
		if len(versions) == 0 {
			return nil, ctxerr.New(ctx, "no cached versions for Fleet-maintained app")
		}

		switch {
		case *payload.PinnedVersion == "": // Latest
			activeInstallerID = versions[0].ID
		case usesCaret:
			for _, v := range versions {
				if versionMatchesMajor(v.Version, majorVersionString) {
					activeInstallerID = v.ID
					break
				}
			}
			if activeInstallerID == 0 {
				activeInstallerID = versions[0].ID
			}
		default: // literal version
			for _, v := range versions {
				if v.Version == *payload.PinnedVersion {
					activeInstallerID = v.ID
					break
				}
			}
			if activeInstallerID == 0 {
				return nil, fleet.NewUserMessageError(errVersionNotFound, http.StatusNotFound)
			}
		}

		// The active-installer flip is applied in the dirty section below.
		dirty["PinnedVersion"] = true
	}

	fieldsShouldSideEffect := map[string]struct{}{
		"InstallerFile":     {},
		"InstallScript":     {},
		"UninstallScript":   {},
		"PostInstallScript": {},
		"PreInstallQuery":   {},
		"Package":           {},
		"Labels":            {},
	}
	var shouldDoSideEffects bool
	// persist changes starting here, now that we've done all the validation/diffing we can
	if len(dirty) > 0 {
		switch {
		case len(dirty) == 1 && dirty["SelfService"]: // only self-service changed; use lighter update function
			if err := svc.ds.UpdateInstallerSelfServiceFlag(ctx, *payload.SelfService, existingInstaller.InstallerID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "updating installer self service flag")
			}
		case len(dirty) == 1 && dirty["PinnedVersion"]: // only the pinned version changed; flip the active installer rather than rewriting it
			if err := svc.ds.SetFleetMaintainedAppActiveInstaller(ctx, payload, activeInstallerID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "pinning Fleet-maintained app version")
			}

			// cancel pending installs of the version we pinned away from
			if activeInstallerID != existingInstaller.InstallerID {
				if err := svc.ds.ProcessInstallerUpdateSideEffects(ctx, existingInstaller.InstallerID, true, false); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "processing side effects for version pin")
				}
			}
		default:
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

			// Get the hosts that are NOT in label scope currently (before the update happens)
			var hostsNotInScope map[uint]struct{}
			if dirty["Labels"] {
				hostsNotInScope, err = svc.ds.GetExcludedHostIDMapForSoftwareInstaller(ctx, payload.InstallerID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "getting hosts not in scope for installer")
				}
			}

			if err := svc.ds.SaveInstallerUpdates(ctx, payload); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "saving installer updates")
			}

			if dirty["Labels"] {
				// Get the hosts that are now IN label scope (after the update)
				hostsInScope, err := svc.ds.GetIncludedHostIDMapForSoftwareInstaller(ctx, payload.InstallerID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "getting hosts in scope for installer")
				}

				var hostsToClear []uint
				for id := range hostsInScope {
					if _, ok := hostsNotInScope[id]; ok {
						// it was not in scope but now it is, so we should clear policy status
						hostsToClear = append(hostsToClear, id)
					}
				}

				// We clear the policy status here because otherwise the policy automation machinery
				// won't pick this up and the software won't install.
				if err := svc.ds.ClearSoftwareInstallerAutoInstallPolicyStatusForHosts(ctx, payload.InstallerID, hostsToClear); err != nil {
					return nil, ctxerr.Wrap(ctx, err, "failed to clear auto install policy status for host")
				}
			}

			for field := range dirty {
				if _, ok := fieldsShouldSideEffect[field]; ok {
					shouldDoSideEffects = true
					break
				}
			}

			// if we're updating anything other than self-service, we cancel pending installs/uninstalls,
			// and if we're updating the package we reset counts. This is run in its own transaction internally
			// for consistency, but independent of the installer update query as the main update should stick
			// even if side effects fail.
			if err := svc.ds.ProcessInstallerUpdateSideEffects(ctx, existingInstaller.InstallerID, shouldDoSideEffects, dirty["Package"]); err != nil {
				return nil, err
			}
		}

		// now that the payload has been updated with any patches, we can set the
		// final fields of the activity
		actLabelsInclAny, actLabelsExclAny, actLabelsInclAll := activitySoftwareLabelsFromSoftwareScopeLabels(
			existingInstaller.LabelsIncludeAny, existingInstaller.LabelsExcludeAny, existingInstaller.LabelsIncludeAll)
		if payload.ValidatedLabels != nil {
			actLabelsInclAny, actLabelsExclAny, actLabelsInclAll = activitySoftwareLabelsFromValidatedLabels(payload.ValidatedLabels)
		}
		activity.LabelsIncludeAny = actLabelsInclAny
		activity.LabelsExcludeAny = actLabelsExclAny
		activity.LabelsIncludeAll = actLabelsInclAll
		if payload.SelfService != nil {
			activity.SelfService = *payload.SelfService
		}
		if payload.DisplayName != nil {
			activity.SoftwareDisplayName = *payload.DisplayName
		}
		if payload.PinnedVersion != nil && *payload.PinnedVersion != "" {
			activity.PinnedVersion = payload.PinnedVersion
		}
		if err := svc.NewActivity(ctx, vc.User, activity); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "creating activity for edited software")
		}
	}

	// re-pull installer from database to ensure any side effects are accounted for; may be able to optimize this out later
	updatedInstaller, err := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctxdb.RequirePrimary(ctx, true), payload.TeamID, payload.TitleID, true)
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

func (svc *Service) validateEmbeddedSecretsOnScript(ctx context.Context, scriptName string, script *string,
	argErr *fleet.InvalidArgumentError,
) *fleet.InvalidArgumentError {
	if script != nil {
		if errScript := svc.ds.ValidateEmbeddedSecrets(ctx, []string{*script}); errScript != nil {
			if argErr != nil {
				argErr.Append(scriptName, errScript.Error())
			} else {
				argErr = fleet.NewInvalidArgumentError(scriptName, errScript.Error())
			}
		}
	}
	return argErr
}

func ValidateSoftwareLabelsForUpdate(ctx context.Context, svc fleet.Service, existingInstaller *fleet.SoftwareInstaller, includeAny, excludeAny, includeAll []string) (shouldUpdate bool, validatedLabels *fleet.LabelIdentsWithScope, err error) {
	if authctx, ok := authz_ctx.FromContext(ctx); !ok {
		return false, nil, fleet.NewAuthRequiredError("batch validate labels: missing authorization context")
	} else if !authctx.Checked() {
		return false, nil, fleet.NewAuthRequiredError("batch validate labels: method requires previous authorization")
	}

	if existingInstaller == nil {
		return false, nil, errors.New("existing installer must be provided")
	}

	if includeAny == nil && excludeAny == nil && includeAll == nil {
		// nothing to do
		return false, nil, nil
	}

	incoming, err := ValidateSoftwareLabels(ctx, svc, existingInstaller.TeamID, includeAny, excludeAny, includeAll)
	if err != nil {
		return false, nil, err
	}

	var prevScope fleet.LabelScope
	var prevLabels []fleet.SoftwareScopeLabel
	switch {
	case len(existingInstaller.LabelsIncludeAny) > 0:
		prevScope = fleet.LabelScopeIncludeAny
		prevLabels = existingInstaller.LabelsIncludeAny
	case len(existingInstaller.LabelsExcludeAny) > 0:
		prevScope = fleet.LabelScopeExcludeAny
		prevLabels = existingInstaller.LabelsExcludeAny
	case len(existingInstaller.LabelsIncludeAll) > 0:
		prevScope = fleet.LabelScopeIncludeAll
		prevLabels = existingInstaller.LabelsIncludeAll
	}

	prevByName := make(map[string]fleet.LabelIdent, len(prevLabels))
	for _, pl := range prevLabels {
		prevByName[pl.LabelName] = fleet.LabelIdent{
			LabelID:   pl.LabelID,
			LabelName: pl.LabelName,
		}
	}

	if prevScope != incoming.LabelScope {
		return true, incoming, nil
	}

	if len(prevByName) != len(incoming.ByName) {
		return true, incoming, nil
	}

	// compare labels by name
	for n, il := range incoming.ByName {
		pl, ok := prevByName[n]
		if !ok || pl != il {
			return true, incoming, nil
		}
	}

	return false, nil, nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) error {
	if teamID == nil {
		return fleet.NewInvalidArgumentError("fleet_id", "is required")
	}

	// we authorize with SoftwareInstaller here, but it uses the same AuthzType
	// as VPPApp, so this is correct for both software installers and VPP apps.
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	// first, look for a software installer
	metaInstaller, errInstaller := svc.ds.GetSoftwareInstallerMetadataByTeamAndTitleID(ctx, teamID, titleID, false)
	metaVPP, errVPP := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, teamID, titleID)
	metaInHouse, errInHouse := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, teamID, titleID)

	switch {
	case errInstaller != nil && !fleet.IsNotFound(errInstaller):
		return ctxerr.Wrap(ctx, errInstaller, "getting software installer metadata")
	case errVPP != nil && !fleet.IsNotFound(errVPP):
		return ctxerr.Wrap(ctx, errVPP, "getting vpp app metadata")
	case errInHouse != nil && !fleet.IsNotFound(errInHouse):
		return ctxerr.Wrap(ctx, errInHouse, "getting in house app metadata")
	}

	switch {
	case metaInstaller != nil:
		return svc.deleteSoftwareInstaller(ctx, metaInstaller)
	case metaVPP != nil:
		return svc.deleteVPPApp(ctx, teamID, metaVPP)
	case metaInHouse != nil:
		return svc.deleteSoftwareInstaller(ctx, metaInHouse)
	}
	return ctxerr.Wrap(ctx, &notFoundError{}, "getting software installer")
}

func (svc *Service) deleteVPPApp(ctx context.Context, teamID *uint, meta *fleet.VPPAppStoreApp) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	var androidHostsUUIDToPolicyID map[string]string
	if meta.Platform == fleet.AndroidPlatform {
		// if this is an Android app we're deleting, collect the host uuids that should have it removed
		// (as we uninstall Android apps on delete). We can't do this in the worker as it will be too late,
		// the vpp_apps_teams entry will have been deleted.
		hosts, err := svc.ds.GetIncludedHostUUIDMapForAppStoreApp(ctx, meta.VPPAppsTeamsID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "delete app store app: getting android hosts in scope")
		}
		androidHostsUUIDToPolicyID = hosts
	}

	if err := svc.ds.DeleteVPPAppFromTeam(ctx, teamID, meta.VPPAppID); err != nil {
		return ctxerr.Wrap(ctx, err, "deleting VPP app")
	}

	// if this is an android app, remove the self-service app from the managed Google Play store
	// and uninstall it from the hosts.
	if meta.Platform == fleet.AndroidPlatform && len(androidHostsUUIDToPolicyID) > 0 {
		enterprise, err := svc.ds.GetEnterprise(ctx)
		if err != nil {
			return &fleet.BadRequestError{Message: "Android MDM is not enabled", InternalErr: err}
		}
		err = worker.QueueMakeAndroidAppUnavailableJob(ctx, svc.ds, svc.logger, meta.VPPAppID.AdamID, androidHostsUUIDToPolicyID, enterprise.Name(), svc.config.MDM.AndroidBatchSize)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "enqueuing job to make android app unavailable")
		}
	}

	var teamName *string
	if teamID != nil && *teamID != 0 {
		t, err := svc.ds.TeamLite(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team name for deleted VPP app")
		}
		teamName = &t.Name
	}

	actLabelsInclAny, actLabelsExclAny, actLabelsInclAll := activitySoftwareLabelsFromSoftwareScopeLabels(meta.LabelsIncludeAny, meta.LabelsExcludeAny, meta.LabelsIncludeAll)

	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityDeletedAppStoreApp{
		AppStoreID:       meta.AdamID,
		SoftwareTitle:    meta.Name,
		TeamName:         teamName,
		TeamID:           teamID,
		Platform:         meta.Platform,
		LabelsIncludeAny: actLabelsInclAny,
		LabelsExcludeAny: actLabelsExclAny,
		LabelsIncludeAll: actLabelsInclAll,
		SoftwareIconURL:  meta.IconURL,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleted VPP app")
	}

	if teamID != nil && meta.IconURL != nil && *meta.IconURL != "" {
		err := svc.ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, *teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, fmt.Sprintf("failed to delete unused software icons for team %d", *teamID))
		}
	}

	return nil
}

func (svc *Service) deleteSoftwareInstaller(ctx context.Context, meta *fleet.SoftwareInstaller) error {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}

	switch {
	case meta.Extension == "ipa":
		if err := svc.ds.DeleteInHouseApp(ctx, meta.InstallerID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting in house app")
		}
	case meta.FleetMaintainedAppID != nil:
		// For FMA installers there may be multiple cached versions (active + up to
		// N-1 inactive ones). Delete the active version first so that the
		// policy-automation and setup-experience guard-rails are enforced, then
		// sweep up any remaining inactive cached versions.
		if err := svc.ds.DeleteSoftwareInstaller(ctx, meta.InstallerID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting active FMA installer version")
		}
		// After the active row is gone, fetch whatever cached versions remain and
		// delete them.  GetFleetMaintainedVersionsByTitleID queries the live DB, so
		// it will not return the row we just deleted.
		if meta.TitleID != nil {
			cachedVersions, err := svc.ds.GetFleetMaintainedVersionsByTitleID(ctx, meta.TeamID, *meta.TitleID, false)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting cached FMA versions for cleanup")
			}
			for _, v := range cachedVersions {
				if err := svc.ds.DeleteSoftwareInstaller(ctx, v.ID); err != nil && !fleet.IsNotFound(err) {
					return ctxerr.Wrap(ctx, err, "deleting cached FMA version")
				}
			}
			// The pin row is keyed by (team, title) and is not cascade-deleted when
			// installer rows go away (only when the title row is deleted), so clear
			// it explicitly to avoid a stale pin surviving a delete + re-add.
			if err := svc.ds.DeletePinnedVersion(ctx, meta.TeamID, *meta.TitleID); err != nil {
				return ctxerr.Wrap(ctx, err, "deleting pinned version after FMA removal")
			}
		}
	default:
		if err := svc.ds.DeleteSoftwareInstaller(ctx, meta.InstallerID); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting software installer")
		}
	}

	var teamName *string
	if meta.TeamID != nil {
		t, err := svc.ds.TeamLite(ctx, *meta.TeamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting team name for deleted software")
		}
		teamName = &t.Name
	}

	actLabelsInclAny, actLabelsExclAny, actLabelsInclAll := activitySoftwareLabelsFromSoftwareScopeLabels(meta.LabelsIncludeAny, meta.LabelsExcludeAny, meta.LabelsIncludeAll)
	if err := svc.NewActivity(ctx, vc.User, fleet.ActivityTypeDeletedSoftware{
		SoftwareTitle:    meta.SoftwareTitle,
		SoftwarePackage:  meta.Name,
		TeamName:         teamName,
		TeamID:           meta.TeamID,
		SelfService:      meta.SelfService,
		LabelsIncludeAny: actLabelsInclAny,
		LabelsExcludeAny: actLabelsExclAny,
		LabelsIncludeAll: actLabelsInclAll,
		SoftwareIconURL:  meta.IconUrl,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating activity for deleted software")
	}

	if meta.IconUrl != nil && *meta.IconUrl != "" {
		var teamIDForCleanup uint
		if meta.TeamID != nil {
			teamIDForCleanup = *meta.TeamID
		}
		err := svc.ds.DeleteIconsAssociatedWithTitlesWithoutInstallers(ctx, teamIDForCleanup)
		if err != nil {
			return ctxerr.Wrap(ctx, fmt.Errorf("failed to delete unused software icons for team %d: %w", teamIDForCleanup, err))
		}
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
		return "", fleet.NewInvalidArgumentError("fleet_id", "is required")
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
		return nil, fleet.NewInvalidArgumentError("fleet_id", "is required")
	}

	meta, err := svc.GetSoftwareInstallerMetadata(ctx, skipAuthz, titleID, teamID)
	if err != nil {
		return nil, err
	}

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) GetSoftwareInstallDetails(ctx context.Context, installUUID string) (*fleet.SoftwareInstallDetails, error) {
	// Call the base (non-premium) service to get the software install details
	details, err := svc.Service.GetSoftwareInstallDetails(ctx, installUUID)
	if err != nil {
		return nil, err
	}

	// SoftwareInstallersCloudFrontSigner can only be set if license.IsPremium()
	if svc.config.S3.SoftwareInstallersCloudFrontSigner != nil {
		// Sign the URL for the installer
		installerURL, err := svc.getSoftwareInstallURL(ctx, details.InstallerID)
		if err != nil {
			// We log the error but continue to return the details without the signed URL because orbit can still
			// try to download the installer via Fleet server.
			svc.logger.ErrorContext(ctx, "error getting software installer URL; check CloudFront configuration", "err", err)
		} else {
			details.SoftwareInstallerURL = installerURL
		}
	}

	return details, nil
}

func (svc *Service) getSoftwareInstallURL(ctx context.Context, installerID uint) (*fleet.SoftwareInstallerURL, error) {
	meta, err := svc.validateAndGetSoftwareInstallerMetadata(ctx, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating software installer metadata for download")
	}

	// Note: we could check if the installer exists in the S3 store.
	// However, if we fail and don't return a URL installer, the Orbit client will still try to download the installer via the Fleet server,
	// and we will end up checking if the installer exists in the S3 store again.
	// So, to reduce server load and speed up the "happy path" software install, we skip the check here and risk returning a URL that doesn't work.
	// If CloudFront is misconfigured, the server and Orbit clients will experience a greater load since they'll be doing throw-away work.

	// Get the signed URL
	signedURL, err := svc.softwareInstallStore.Sign(ctx, meta.StorageID, fleet.SoftwareInstallerSignedURLExpiry)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "signing software installer URL")
	}
	return &fleet.SoftwareInstallerURL{
		URL:      signedURL,
		Filename: meta.Name,
	}, nil
}

func (svc *Service) OrbitDownloadSoftwareInstaller(ctx context.Context, installerID uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	meta, err := svc.validateAndGetSoftwareInstallerMetadata(ctx, installerID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating software installer metadata for download")
	}

	// Note that we do allow downloading an installer that is on a different team
	// than the host's team, because the install request might have come while
	// the host was on that team, and then the host got moved to a different team
	// but the request is still pending execution.

	return svc.getSoftwareInstallerBinary(ctx, meta.StorageID, meta.Name)
}

func (svc *Service) validateAndGetSoftwareInstallerMetadata(ctx context.Context, installerID uint) (*fleet.SoftwareInstaller, error) {
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
	return meta, nil
}

func (svc *Service) getSoftwareInstallerBinary(ctx context.Context, storageID string, filename string) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// check if the installer exists in the store
	exists, err := svc.softwareInstallStore.Exists(ctx, storageID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking if installer exists")
	}
	if !exists {
		return nil, ctxerr.Wrapf(ctx, &notFoundError{}, "%s with filename %s does not exist in software installer store", storageID,
			filename)
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
	mobileAppleDevice := fleet.InstallableDevicePlatform(platform) == fleet.IOSPlatform || fleet.InstallableDevicePlatform(platform) == fleet.IPadOSPlatform

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

	if mobileAppleDevice {
		iha, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID)
		if err != nil && !fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, err, "install in house app: get metadata")
		}

		if iha != nil {
			scoped, err := svc.ds.IsInHouseAppLabelScoped(ctx, iha.InstallerID, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "checking label scoping during in-house app install attempt")
			}

			if !scoped {
				return &fleet.BadRequestError{
					Message: "Couldn't install. This host isn't a member of the labels defined for this software title.",
				}
			}

			opts := fleet.HostSoftwareInstallOptions{SelfService: false}
			cfg, err := svc.ds.GetInHouseAppConfiguration(ctx, iha.InstallerID)
			if err != nil && !fleet.IsNotFound(err) {
				return ctxerr.Wrap(ctx, err, "get in-house app configuration for pre-flight check")
			}
			switch err := svc.precheckAppConfigResolvable(ctx, host, cfg); {
			case errors.Is(err, apple_mdm.ErrUnresolvableAppConfigVar):
				return svc.recordFailedInHouseInstall(ctx, host.ID, iha.InstallerID, opts, unresolvableAppConfigFailureReason(err))
			case err != nil:
				return ctxerr.Wrap(ctx, err, "pre-flight substitute fleet variables in in-house app configuration")
			}

			err = svc.ds.InsertHostInHouseAppInstall(ctx, host.ID, iha.InstallerID, softwareTitleID, uuid.NewString(), opts)
			return ctxerr.Wrap(ctx, err, "insert in house app install")
		}
		// it's OK if we didn't find an in-house app; this might be a VPP app, so continue on
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
			// check the label scoping for this installer and host
			scoped, err := svc.ds.IsSoftwareInstallerLabelScoped(ctx, installer.InstallerID, hostID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "checking label scoping during software install attempt")
			}

			if !scoped {
				return &fleet.BadRequestError{
					Message: "Couldn't install. Host isn't member of the labels defined for this software title.",
				}
			}

			lastInstallRequest, err := svc.ds.GetHostLastInstallData(ctx, host.ID, installer.InstallerID)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "getting last install data for host %d and installer %d", host.ID, installer.InstallerID)
			}
			if lastInstallRequest != nil && lastInstallRequest.Status != nil &&
				(*lastInstallRequest.Status == fleet.SoftwareInstallPending || *lastInstallRequest.Status == fleet.SoftwareUninstallPending) {
				return &fleet.BadRequestError{
					Message: "Couldn't install. Host already has a pending install/uninstall for this installer.",
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
	// User-enrolled (BYOD) iOS/iPadOS hosts are no longer blocked here. The
	// downstream VPP install path provisions the per-user VPP user (#44003),
	// associates the asset via clientUserIds (#44004), and emits an
	// InstallApplication command without ChangeManagementState (#44005).

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

	// check the label scoping for this VPP app and host
	scoped, err := svc.ds.IsVPPAppLabelScoped(ctx, vppApp.VPPAppTeam.AppTeamID, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking label scoping during vpp software install attempt")
	}

	if !scoped {
		return &fleet.BadRequestError{
			Message: "Couldn't install. This host isn't a member of the labels defined for this software title.",
		}
	}

	_, err = svc.installSoftwareFromVPP(ctx, host, vppApp, mobileAppleDevice || fleet.InstallableDevicePlatform(platform) == fleet.MacOSPlatform, fleet.HostSoftwareInstallOptions{
		SelfService: false,
	})
	return err
}

func (svc *Service) installSoftwareFromVPP(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, appleDevice bool, opts fleet.HostSoftwareInstallOptions) (string, error) {
	token, err := svc.GetVPPTokenIfCanInstallVPPApps(ctx, appleDevice, host)
	if err != nil {
		return "", err
	}

	return svc.InstallVPPAppPostValidation(ctx, host, vppApp, token, opts)
}

func (svc *Service) GetVPPTokenIfCanInstallVPPApps(ctx context.Context, appleDevice bool, host *fleet.Host) (string, error) {
	if !appleDevice {
		return "", &fleet.BadRequestError{
			Message: "VPP apps can only be installed only on Apple hosts.",
			InternalErr: ctxerr.NewWithData(
				ctx, "invalid host platform for requested installer",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID},
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
				map[string]any{"host_id": host.ID, "team_id": host.TeamID},
			),
		}
	}

	token, err := svc.getVPPToken(ctx, host.TeamID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting VPP token")
	}

	return token, nil
}

// fleetVarInErrRe matches the $FLEET_VAR_* token embedded in the error
// returned by apple_mdm.SubstituteFleetVarsInAppConfig, so we can name the
// offending variable in the failure reason shown to the admin.
var fleetVarInErrRe = regexp.MustCompile(`\$FLEET_VAR_[A-Z_]+`)

// unresolvableAppConfigFailureReason builds the user-facing reason surfaced in
// the activity feed and Install Details modal when a managed app configuration
// references a Fleet variable that can't be resolved for the host. Prefers the
// typed per-variable detail (same wording configuration-profile delivery uses)
// and falls back to a generic sentence that names the variable.
func unresolvableAppConfigFailureReason(err error) string {
	// All current call sites gate on errors.Is(err, …ErrUnresolvableAppConfigVar)
	// before invoking, so err is non-nil in practice. Defensive nil-guard so
	// nilaway can prove this and to keep the helper safe if reused.
	if err == nil {
		return ""
	}
	var typed *apple_mdm.UnresolvableAppConfigVarError
	if errors.As(err, &typed) && typed.Detail != "" {
		return typed.Detail
	}
	if v := fleetVarInErrRe.FindString(err.Error()); v != "" {
		return fmt.Sprintf("The app's managed configuration references %s, which Fleet couldn't populate for this host.", v)
	}
	return "The app's managed configuration references a Fleet variable that can't be resolved for this host."
}

// precheckAppConfigResolvable resolves the managed app configuration's Fleet
// variables for the host without mutating anything. It returns the substitution
// error unchanged (callers check errors.Is(err, apple_mdm.ErrUnresolvableAppConfigVar)).
// cfg may be empty (no managed config), in which case it's a no-op.
func (svc *Service) precheckAppConfigResolvable(ctx context.Context, host *fleet.Host, cfg []byte) error {
	if len(cfg) == 0 {
		return nil
	}
	_, err := apple_mdm.SubstituteFleetVarsInAppConfig(ctx, svc.ds, cfg, apple_mdm.AppConfigSubstitutionHost{
		UUID:           host.UUID,
		HardwareSerial: host.HardwareSerial,
		Platform:       host.Platform,
	})
	return err
}

// recordFailedVPPInstall records a pre-flight-failed VPP install (no license
// reserved, no command enqueued) and emits the failed-install activity.
//
// For admin / self-service / policy / auto-update paths, returning a nil error
// is intentional — the activity carries the outcome and the API responds 2xx.
// For setup-experience (opts.ForSetupExperience=true) we have to surface a
// non-nil error so the setup-experience driver transitions the step out of
// Running; otherwise it would stash the unused command UUID and wait forever
// for an MDM command result that will never arrive.
func (svc *Service) recordFailedVPPInstall(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, opts fleet.HostSoftwareInstallOptions, reason string) (string, error) {
	cmdUUID := uuid.NewString()
	user, act, err := svc.ds.RecordFailedVPPAppInstall(ctx, host.ID, vppApp.VPPAppID, cmdUUID, reason, opts)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "record failed vpp install")
	}
	if act != nil {
		if err := svc.NewActivity(ctx, user, act); err != nil {
			return "", ctxerr.Wrap(ctx, err, "create activity for failed vpp install")
		}
	}
	if opts.ForSetupExperience {
		return cmdUUID, &fleet.PreflightInstallFailedError{Reason: reason}
	}
	return cmdUUID, nil
}

// recordFailedInHouseInstall is the in-house (.ipa) counterpart of
// recordFailedVPPInstall.
func (svc *Service) recordFailedInHouseInstall(ctx context.Context, hostID, inHouseAppID uint, opts fleet.HostSoftwareInstallOptions, reason string) error {
	cmdUUID := uuid.NewString()
	user, act, err := svc.ds.RecordFailedInHouseAppInstall(ctx, hostID, inHouseAppID, cmdUUID, reason, opts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "record failed in-house install")
	}
	if act != nil {
		if err := svc.NewActivity(ctx, user, act); err != nil {
			return ctxerr.Wrap(ctx, err, "create activity for failed in-house install")
		}
	}
	return nil
}

func (svc *Service) InstallVPPAppPostValidation(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, token string, opts fleet.HostSoftwareInstallOptions) (string, error) {
	// Pre-flight: resolve the managed app configuration's Fleet variables for
	// this host BEFORE anything irreversible (reserving a VPP license, enqueuing
	// the command). iOS/iPadOS only — macOS VPP installs drop the configuration.
	// If a variable can't be resolved for this host (e.g. an IdP variable on a
	// host with no IdP linkage), record a failed install and emit the
	// failed-install activity instead of rejecting the request, so the failure
	// is visible in the activity feed and Install Details modal. Doing this
	// before AssociateAssets also avoids leaking a VPP license.
	if vppApp.Platform == fleet.IOSPlatform || vppApp.Platform == fleet.IPadOSPlatform {
		cfg, err := svc.ds.GetVPPAppConfiguration(ctx, vppApp.Platform, vppApp.AdamID, ptr.ValOrZero(host.TeamID))
		if err != nil && !fleet.IsNotFound(err) {
			return "", ctxerr.Wrap(ctx, err, "get vpp app configuration for pre-flight check")
		}
		switch err := svc.precheckAppConfigResolvable(ctx, host, cfg); {
		case errors.Is(err, apple_mdm.ErrUnresolvableAppConfigVar):
			return svc.recordFailedVPPInstall(ctx, host, vppApp, opts, unresolvableAppConfigFailureReason(err))
		case err != nil:
			return "", ctxerr.Wrap(ctx, err, "pre-flight substitute fleet variables in vpp app configuration")
		}
	}

	// at this moment, neither the UI nor the back-end are prepared to
	// handle [asyncronous errors][1] on assignment, so before assigning a
	// device to a license, we need to:
	//
	// 1. Check if the app is already assigned to the serial number (or
	//    Managed Apple ID, for User Enrollments).
	// 2. If it's not assigned yet, check if we have enough licenses.
	//
	// A race still might happen, so async error checking needs to be
	// implemented anyways at some point.
	//
	// [1]: https://developer.apple.com/documentation/devicemanagement/app_and_book_management/handling_error_responses#3729433

	// Resolve enrollment style first so the assignment query can address the
	// right principal — serial for device-scoped licensing, clientUserId for
	// user-scoped (BYOD) licensing. Without this branch the existing
	// SerialNumber filter always returns empty for User Enrollments, which
	// makes Fleet enter the AvailableCount check on every retry and produces
	// false-positive "no available licenses" errors when the user is just
	// adding their Nth (≤5) device under one Managed Apple ID.
	hostMDM, err := svc.ds.GetHostMDM(ctx, host.ID)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "looking up host MDM info for VPP install")
	}
	isPersonal := hostMDM != nil && hostMDM.IsPersonalEnrollment

	var clientUserID string
	if isPersonal {
		// Token-selection policy (per #44009): use the team's default token —
		// `GetVPPTokenByTeamID` already returns the first token for the team
		// (existing behavior). Multi-location support is deferred unless a
		// customer hits the edge case.
		personalTokenDB, err := svc.ds.GetVPPTokenByTeamID(ctx, host.TeamID)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "fetching VPP token DB row for user-enrolled install")
		}
		clientUserID, err = svc.ensureVPPClientUser(ctx, host, personalTokenDB)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "ensure VPP client user")
		}
	}

	assignmentFilter := &vpp.AssignmentFilter{AdamID: vppApp.AdamID}
	if isPersonal {
		assignmentFilter.ClientUserID = clientUserID
	} else {
		assignmentFilter.SerialNumber = host.HardwareSerial
	}
	assignments, err := vpp.GetAssignments(ctx, token, assignmentFilter)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "getting assignments from VPP API")
	}

	var eventID string

	// assocReq is non-nil once we reserve a license below; it lets us release
	// the seat (DisassociateAssets) if a later step fails, avoiding a leak.
	var assocReq *vpp.AssociateAssetsRequest

	// this app is not assigned to this device (or this user, for BYOD), check
	// if we have licenses left and assign it.
	if len(assignments) == 0 {
		assets, err := vpp.GetAssets(ctx, token, &vpp.AssetFilter{AdamID: vppApp.AdamID})
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "getting assets from VPP API")
		}

		if len(assets) == 0 {
			svc.logger.DebugContext(ctx, "trying to assign VPP asset to host",
				"adam_id", vppApp.AdamID,
				"host_serial", host.HardwareSerial,
			)
			return "", &fleet.BadRequestError{
				Message:     "Couldn't add software. <app_store_id> isn't available in Apple Business. Please purchase license in Apple Business and try again.",
				InternalErr: ctxerr.Errorf(ctx, "VPP API didn't return any assets for adamID %s", vppApp.AdamID),
			}
		}

		if len(assets) > 1 {
			return "", ctxerr.Errorf(ctx, "VPP API returned more than one asset for adamID %s", vppApp.AdamID)
		}

		if assets[0].AvailableCount <= 0 {
			return "", &fleet.BadRequestError{
				Message: "Couldn't install. No available licenses. Please purchase license in Apple Business and try again.",
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

		req := &vpp.AssociateAssetsRequest{Assets: assets}
		if isPersonal {
			req.ClientUserIds = []string{clientUserID}
		} else {
			req.SerialNumbers = []string{host.HardwareSerial}
		}

		eventID, err = vpp.AssociateAssets(ctx, token, req)
		if err == nil {
			// We reserved a seat; remember the request so we can release it if a
			// later step fails.
			assocReq = req
		}
		if err != nil {
			// Apple rejects the per-user device cap (≤5 devices per Managed
			// Apple ID per license). Surface it cleanly so admins can act on
			// it without having to decode raw VPP error numbers.
			if vpp.IsMaxDevicesPerUserError(err) {
				return "", &fleet.BadRequestError{
					Message:     "Couldn't install. This user has reached the maximum number of devices for this app license.",
					InternalErr: ctxerr.WrapWithData(ctx, err, "associate asset rejected by Apple per-user device cap", map[string]any{"host_id": host.ID, "team_id": host.TeamID, "adam_id": vppApp.AdamID}),
				}
			}

			return "", ctxerr.Wrapf(ctx, err, "associating asset with adamID %s to host %s", vppApp.AdamID, host.HardwareSerial)
		}
	}

	// TODO(mna): should we associate the device (give the license) only when the
	// upcoming activity is ready to run? I don't think so, because then it could
	// fail when it's ready to run which is probably a worse UX as once enqueued
	// you expect it to succeed. But eventually, we should do better management
	// of the licenses, e.g. if the upcoming activity gets cancelled, it should
	// release the reserved license.
	//
	// But the command is definitely not enqueued now, only when activating the
	// activity.

	// enqueue the VPP app command to install
	cmdUUID := uuid.NewString()
	err = svc.ds.InsertHostVPPSoftwareInstall(ctx, host.ID, vppApp.VPPAppID, cmdUUID, eventID, opts)
	if err != nil {
		// The install didn't persist, so if we reserved a license seat above we
		// must release it — otherwise the seat leaks (no install will ever use
		// it). Best-effort: log and continue returning the original error.
		if assocReq != nil {
			if _, dErr := vpp.DisassociateAssets(token, assocReq); dErr != nil {
				svc.logger.ErrorContext(ctx, "failed to release reserved VPP license after install insert failure",
					"err", dErr, "host_id", host.ID, "adam_id", vppApp.AdamID)
			}
		}
		return "", ctxerr.Wrapf(ctx, err, "inserting host vpp software install for host with serial %s and app with adamID %s", host.HardwareSerial, vppApp.AdamID)
	}

	return cmdUUID, nil
}

func (svc *Service) installSoftwareTitleUsingInstaller(ctx context.Context, host *fleet.Host, installer *fleet.SoftwareInstaller) error {
	ext, requiredPlatform := installerRequiredPlatform(installer)
	if requiredPlatform == "" {
		// this should never happen
		return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
	}

	if host.FleetPlatform() != requiredPlatform {
		// Allow .sh scripts for any unix-like platform (linux and darwin)
		if !(ext == ".sh" && fleet.IsUnixLike(host.Platform)) {
			return &fleet.BadRequestError{
				Message: fmt.Sprintf("Package (%s) can be installed only on %s hosts.", ext, requiredPlatform),
				InternalErr: ctxerr.NewWithData(
					ctx, "invalid host platform for requested installer",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": installer.TitleID},
				),
			}
		}
	}

	// Reset old attempts so the new install starts fresh at attempt 1.
	if err := svc.ds.ResetNonPolicyInstallAttempts(ctx, host.ID, installer.InstallerID); err != nil {
		return ctxerr.Wrap(ctx, err, "reset install attempts before new install")
	}

	_, err := svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, fleet.HostSoftwareInstallOptions{
		SelfService: false,
		WithRetries: true,
	})
	return ctxerr.Wrap(ctx, err, "inserting software install request")
}

func (svc *Service) UninstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error {
	// we need to use ds.Host because ds.HostLite doesn't return the orbit node key
	host, err := svc.ds.Host(ctx, hostID)

	fromMyDevicePage := svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken) ||
		svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceCertificate) ||
		svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceURL)

	if err != nil {
		// if error is because the host does not exist, check first if the user
		// had access to install/uninstall software (to prevent leaking valid host ids).
		if fleet.IsNotFound(err) {
			if !fromMyDevicePage {
				if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{}, fleet.ActionWrite); err != nil {
					return err
				}
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
	if !fromMyDevicePage {
		if err := svc.authz.Authorize(ctx, &fleet.HostSoftwareInstallerResultAuthz{HostTeamID: host.TeamID}, fleet.ActionWrite); err != nil {
			return err
		}
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
	ext, requiredPlatform := installerRequiredPlatform(installer)
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

	// Get the uninstall script to validate there is one, will use the standard
	// script infrastructure to run it.
	_, err = svc.ds.GetAnyScriptContents(ctx, installer.UninstallScriptContentID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError("software_title_id", `No uninstall script exists for the provided "software_title_id".`).
					WithStatus(http.StatusNotFound), "getting uninstall script contents")
		}
		return err
	}

	// Pending uninstalls will automatically show up in the UI Host Details -> Activity -> Upcoming tab.
	execID := uuid.NewString()
	if err = svc.insertSoftwareUninstallRequest(ctx, execID, host, installer, fromMyDevicePage); err != nil {
		return err
	}
	return nil
}

func (svc *Service) insertSoftwareUninstallRequest(ctx context.Context, executionID string, host *fleet.Host,
	installer *fleet.SoftwareInstaller, selfService bool,
) error {
	if err := svc.ds.InsertSoftwareUninstallRequest(ctx, executionID, host.ID, installer.InstallerID, selfService); err != nil {
		return ctxerr.Wrap(ctx, err, "inserting software uninstall request")
	}
	return nil
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	if svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceToken) ||
		svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceCertificate) ||
		svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceURL) {
		return svc.getDeviceSoftwareInstallResults(ctx, resultUUID)
	}

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

func (svc *Service) getDeviceSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	res, err := svc.ds.GetSoftwareInstallResults(ctx, resultUUID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get software install result")
	} else if res.HostID != host.ID { // hosts can't see other hosts' executions
		return nil, ctxerr.Wrap(ctx, common_mysql.NotFound("HostSoftwareInstallerResult"), "get host software installer results")
	}

	res.EnhanceOutputDetails()
	return res, nil
}

func (svc *Service) GetSelfServiceUninstallScriptResult(ctx context.Context, host *fleet.Host, execID string) (*fleet.HostScriptResult, error) {
	scriptResult, err := svc.ds.GetSelfServiceUninstallScriptExecutionResult(ctx, execID, host.ID)
	if err != nil {
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get script result")
	}

	scriptResult.Hostname = host.DisplayName()

	return scriptResult, nil
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

func (svc *Service) addMetadataToSoftwarePayload(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload, failOnBlankScript bool) (extension string, err error) {
	if payload == nil {
		return "", ctxerr.New(ctx, "payload is required")
	}

	if payload.InstallerFile == nil {
		return "", ctxerr.New(ctx, "installer file is required")
	}

	ext := strings.ToLower(filepath.Ext(payload.Filename))
	ext = strings.TrimPrefix(ext, ".")

	if fleet.IsScriptPackage(ext) {
		if err := svc.addScriptPackageMetadata(ctx, payload, ext); err != nil {
			return "", err
		}
		return ext, nil
	}

	// Handle Windows zip files specially since they require scripts (like exe)
	// and share magic bytes with IPA files, so we check the extension first
	if ext == "zip" {
		platform, err := fleet.SoftwareInstallerPlatformFromExtension(ext)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "determining platform for zip file")
		}
		if platform == "windows" {
			// For Windows zip files, create basic metadata manually
			// since they require custom install/uninstall scripts
			if err := svc.addZipPackageMetadata(ctx, payload); err != nil {
				return "", err
			}
			// Validate that install and uninstall scripts are provided
			if failOnBlankScript {
				if payload.InstallScript == "" {
					return "", &fleet.BadRequestError{
						Message: "Couldn't add. Install script is required for .zip packages.",
					}
				}
				if payload.UninstallScript == "" {
					return "", &fleet.BadRequestError{
						Message: "Couldn't add. Uninstall script is required for .zip packages.",
					}
				}
			}
			return ext, nil
		}
		// For non-Windows zip files (e.g., macOS), let ExtractInstallerMetadata handle it
		// (it will detect it as IPA due to shared magic bytes, but that's handled elsewhere)
	}

	meta, err := file.ExtractInstallerMetadata(payload.InstallerFile)
	if err != nil {
		if errors.Is(err, file.ErrUnsupportedType) {
			return "", &fleet.BadRequestError{
				Message:     "Couldn't edit software. File type not supported. The file should be .pkg, .msi, .exe, .zip, .deb, .rpm, .tar.gz, .sh, .ipa or .ps1.",
				InternalErr: ctxerr.Wrap(ctx, err, "extracting metadata from installer"),
			}
		}
		if errors.Is(err, file.ErrInvalidTarball) {
			return "", &fleet.BadRequestError{
				Message:     "Couldn't edit software. Uploaded file is not a valid .tar.gz archive.",
				InternalErr: ctxerr.Wrap(ctx, err, "extracting metadata from installer"),
			}
		}
		return "", ctxerr.Wrap(ctx, err, "extracting metadata from installer")
	}

	if len(meta.PackageIDs) == 0 && meta.Extension != "tar.gz" && meta.Extension != "zip" {
		return "", &fleet.BadRequestError{
			Message:     "Couldn't add. Unable to extract necessary metadata.",
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
	payload.UpgradeCode = meta.UpgradeCode

	// reset the reader (it was consumed to extract metadata)
	if err := payload.InstallerFile.Rewind(); err != nil {
		return "", ctxerr.Wrap(ctx, err, "resetting installer file reader")
	}

	payload.InstallScript = getInstallScript(meta.Extension, meta.PackageIDs, payload.InstallScript)

	// Software edits validate non-empty scripts later, so set failOnBlankScript to false
	if payload.InstallScript == "" && failOnBlankScript && payload.Extension != "ipa" {
		ext := strings.ToLower(payload.Extension)
		if ext == "zip" {
			return "", &fleet.BadRequestError{
				Message: "Couldn't add. Install script is required for .zip packages.",
			}
		}
		return "", &fleet.BadRequestError{
			Message: fmt.Sprintf("Couldn't add. Install script is required for .%s packages.", ext),
		}
	}

	defaultUninstallScript := file.GetUninstallScript(meta.Extension)
	if payload.UninstallScript == "" || payload.UninstallScript == defaultUninstallScript || payload.UninstallScript == file.UninstallMsiWithUpgradeCodeScript {
		payload.UninstallScript = defaultUninstallScript
		if payload.UpgradeCode != "" {
			payload.UninstallScript = file.UninstallMsiWithUpgradeCodeScript
		}
	}
	if payload.UninstallScript == "" && failOnBlankScript && payload.Extension != "ipa" {
		return "", &fleet.BadRequestError{
			Message: fmt.Sprintf("Couldn't add. Uninstall script is required for .%s packages.", strings.ToLower(payload.Extension)),
		}
	}

	platform, err := fleet.SoftwareInstallerPlatformFromExtension(meta.Extension)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "determining platform from extension")
	}
	payload.Platform = platform

	switch {
	case payload.Extension == "ipa":
		if payload.Platform == "ipados" {
			payload.Source = "ipados_apps"
		} else {
			payload.Source = "ios_apps"
		}
	case payload.BundleIdentifier != "":
		payload.Source = "apps"
	default:
		source, err := fleet.SofwareInstallerSourceFromExtensionAndName(meta.Extension, meta.Name)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "determining source from extension and name")
		}
		payload.Source = source
	}

	return meta.Extension, nil
}

func (svc *Service) addScriptPackageMetadata(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload, extension string) error {
	if payload == nil {
		return ctxerr.New(ctx, "payload is required")
	}

	if payload.InstallerFile == nil {
		return ctxerr.New(ctx, "installer file is required")
	}

	scriptBytes, err := io.ReadAll(payload.InstallerFile)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading script file")
	}

	if err := payload.InstallerFile.Rewind(); err != nil {
		return ctxerr.Wrap(ctx, err, "resetting script file reader")
	}

	scriptContents := string(scriptBytes)

	if err := fleet.ValidateHostScriptContents(scriptContents, true); err != nil {
		return &fleet.BadRequestError{
			Message:     fmt.Sprintf("Couldn't add. Script validation failed: %s", err.Error()),
			InternalErr: ctxerr.Wrap(ctx, err, "validating script contents"),
		}
	}

	// Validate that the shebang matches the file extension
	kind, directExecute, err := fleet.ShebangInfo(scriptContents)
	if err != nil {
		return &fleet.BadRequestError{
			Message:     fmt.Sprintf("Couldn't add. Script validation failed: %s", err.Error()),
			InternalErr: ctxerr.Wrap(ctx, err, "validating script shebang"),
		}
	}
	switch extension {
	case "sh":
		// allow no shebang (defaults to /bin/sh), or a supported shell shebang.
		if directExecute && kind != fleet.ShebangShell {
			return &fleet.BadRequestError{
				Message:     fmt.Sprintf("Couldn't add. Script validation failed: %s", fleet.ErrUnsupportedInterpreter.Error()),
				InternalErr: ctxerr.New(ctx, "shell script with non-shell shebang"),
			}
		}
	case "py":
		// python scripts must be directly executable (via a python shebang).
		if !directExecute || kind != fleet.ShebangPython {
			return &fleet.BadRequestError{
				Message:     "Couldn't add. Script validation failed: Python scripts must start with a python shebang (for example, \"#!/usr/bin/env python3\").",
				InternalErr: ctxerr.New(ctx, "python script without python shebang"),
			}
		}
	case "ps1":
		// PowerShell scripts are executed via powershell.exe, shebangs are not supported.
		if directExecute {
			return &fleet.BadRequestError{
				Message:     "Couldn't add. Script validation failed: PowerShell scripts must not start with a shebang (\"#!\").",
				InternalErr: ctxerr.New(ctx, "powershell script with shebang"),
			}
		}
	}

	shaSum, err := file.SHA256FromTempFileReader(payload.InstallerFile)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "calculating script SHA256")
	}

	if payload.Title == "" {
		base := filepath.Base(payload.Filename)
		payload.Title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	payload.Version = ""
	payload.InstallScript = scriptContents
	payload.StorageID = shaSum
	payload.BundleIdentifier = ""
	payload.PackageIDs = nil
	payload.Extension = extension
	switch extension {
	case "sh":
		payload.Source = "sh_packages"
	case "ps1":
		payload.Source = "ps1_packages"
	}

	platform, err := fleet.SoftwareInstallerPlatformFromExtension(extension)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "determining platform from extension")
	}
	payload.Platform = platform

	return nil
}

func (svc *Service) addZipPackageMetadata(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	if payload == nil {
		return ctxerr.New(ctx, "payload is required")
	}

	if payload.InstallerFile == nil {
		return ctxerr.New(ctx, "installer file is required")
	}

	shaSum, err := file.SHA256FromTempFileReader(payload.InstallerFile)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "calculating zip SHA256")
	}

	if err := payload.InstallerFile.Rewind(); err != nil {
		return ctxerr.Wrap(ctx, err, "resetting zip file reader")
	}

	if payload.Title == "" {
		base := filepath.Base(payload.Filename)
		payload.Title = strings.TrimSuffix(base, filepath.Ext(base))
	}

	platform, err := fleet.SoftwareInstallerPlatformFromExtension("zip")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "determining platform from extension")
	}

	// Don't overwrite version if it's already set (e.g., from Fleet Maintained App manifest)
	// Zip files don't have extractable version metadata, so preserve any existing version
	payload.StorageID = shaSum
	payload.BundleIdentifier = ""
	payload.PackageIDs = nil // Zip files require scripts, so no package IDs extracted
	payload.Extension = "zip"
	payload.Source = "programs" // Same as exe and msi
	payload.Platform = platform

	return nil
}

const (
	batchSoftwarePrefix = "software_batch_"
	// batchSoftwareDeletedSuffix is appended to the batch status key to form the key holding
	// the JSON-encoded list of packages the batch will delete (or, on a dry run, would delete).
	batchSoftwareDeletedSuffix = ":deleted"
	// batchSoftwareCategoriesSuffix is appended to the batch status key to form the key holding
	// the JSON-encoded list of self-service categories this batch added. This is required because
	// we can only be certain of all categories after downloading all FMA manifests and seeing
	// which default categories we might need to add.
	batchSoftwareCategoriesSuffix = ":categories"
	// keyExpireTime serves as a timeout for each step of the batch upload process (initial checks, download for
	// a package from source, upload for a package to object storage) for each package. This timeout is refreshed
	// at each step. If the timeout is reached, they key expires in Redis and the batch process is considered
	// abandoned by clients checking in on it.
	keyExpireTime = 4 * time.Minute
)

func (svc *Service) BatchSetSoftwareInstallers(
	ctx context.Context, tmName string, payloads []*fleet.SoftwareInstallerPayload, dryRun bool,
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

	// Same pattern as the dry-run + team-not-found short-circuit above. Empty payload
	// + dry-run has nothing to validate or stage, so skip the async round-trip — but
	// only when the team also has no installers: an empty payload deletes every
	// existing package, and the dry run must report each one. The client handles an
	// empty UUID response gracefully.
	if dryRun && len(payloads) == 0 {
		pendingDeletion, err := svc.ds.GetSoftwareInstallersPendingDeletion(ctx, teamID, nil)
		if err != nil {
			return "", ctxerr.Wrap(ctx, err, "checking for software installers pending deletion")
		}
		if len(pendingDeletion) == 0 {
			svc.logger.DebugContext(ctx, "software batch dry-run skipped: empty payload and no existing installers",
				"team_id", teamID,
			)
			return "", nil
		}
	}

	var allScripts []string
	var categoryNames []string

	// Verify payloads first, to prevent starting the download+upload process if the data is invalid.
	for _, payload := range payloads {
		if payload.Slug != nil && *payload.Slug != "" {
			err := svc.softwareInstallerPayloadFromSlug(ctx, payload, teamID)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "getting fleet maintained software installer payload from slug")
			}
		}

		if payload.URL == "" && payload.SHA256 == "" {
			return "", fleet.NewInvalidArgumentError(
				"software",
				"Couldn't edit software. One or more software packages is missing url or hash_sha256 fields.",
			)
		}
		if payload.AlwaysDownload && payload.SHA256 != "" {
			return "", fleet.NewInvalidArgumentError(
				"software",
				"Couldn't edit software. The 'always_download' option cannot be used with 'hash_sha256'.",
			)
		}
		if len(payload.URL) > fleet.SoftwareInstallerURLMaxLength {
			return "", fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("software URL is too long, must be %d characters or less", fleet.SoftwareInstallerURLMaxLength),
			)
		}

		// Skip URL validation when it is empty or when it is for a script-only package,
		// which uses a "script://" URL scheme to pass the filename
		if payload.URL != "" && !strings.HasPrefix(payload.URL, "script://") {
			if _, err := url.ParseRequestURI(payload.URL); err != nil {
				return "", fleet.NewInvalidArgumentError(
					"software.url",
					fmt.Sprintf("Couldn't edit software. URL (%q) is invalid", payload.URL),
				)
			}
		}
		if !dryRun {
			validatedLabels, err := ValidateSoftwareLabels(ctx, svc, teamID, payload.LabelsIncludeAny, payload.LabelsExcludeAny, payload.LabelsIncludeAll)
			if err != nil {
				return "", err
			}
			payload.ValidatedLabels = validatedLabels
		}
		allScripts = append(allScripts, payload.InstallScript, payload.PostInstallScript, payload.UninstallScript)

		if err := trimAndValidateCategories(ctx, payload.Categories.Value); err != nil {
			return "", ctxerr.Wrap(ctx, err, "validating software categories")
		}
		categoryNames = append(categoryNames, payload.Categories.Value...)
	}

	categories, err := svc.batchAddSelfServiceCategories(ctx, teamID, categoryNames, dryRun)
	if err != nil {
		return "", err
	}

	if !dryRun {
		// presence of these secrets are validated on the gitops side,
		// we only want to ensure that secrets are in the database on the
		// non-dry run case.
		if err := svc.ds.ValidateEmbeddedSecrets(ctx, allScripts); err != nil {
			return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("script", err.Error()))
		}
		if err := svc.ds.ValidateReferencedCustomHostVitals(ctx, allScripts); err != nil {
			return "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("script", err.Error()))
		}
	}

	requestUUID := uuid.NewString()
	if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID, batchSetProcessing, keyExpireTime); err != nil {
		return "", ctxerr.Wrapf(ctx, err, "failed to set key as %s", batchSetProcessing)
	}

	categoriesJSON, err := json.Marshal(categories)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "marshal self-service categories result")
	}
	if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID+batchSoftwareCategoriesSuffix, string(categoriesJSON), 10*time.Minute); err != nil {
		return "", ctxerr.Wrap(ctx, err, "failed to set self-service categories result")
	}

	svc.logger.InfoContext(ctx, "software batch start",
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

var (
	errEmptyCaretVersion    = errors.New("a major version must be specified after the caret (^). For example, \"^32\".")
	errNonMajorVersion      = errors.New("only the major version can be specified with a caret (^), without including minor and patch versions. For example, \"^32\".")
	errMajorVersionNotFound = errors.New("specified major version is not available. Available versions are listed in the Fleet UI under Actions > Edit software.")
	errVersionNotFound      = errors.New("specified version is not available. Available versions are listed in the Fleet UI under Actions > Edit software.")
)

func (svc *Service) softwareInstallerPayloadFromSlug(ctx context.Context, payload *fleet.SoftwareInstallerPayload, teamID *uint) error {
	slug := payload.Slug
	if slug == nil || *slug == "" {
		return nil
	}

	// convert nil teamID to 0 to get correct titleID
	tmID := ptr.ValOrZero(teamID)
	app, err := svc.ds.GetMaintainedAppBySlug(ctx, *slug, &tmID)
	if err != nil {
		// Return user-friendly message for generic not found error
		if fleet.IsNotFound(err) {
			// Must return low-level error in order to be properly handled upstream
			return fleet.NewUserMessageError(
				fmt.Errorf("%s isn't a supported Fleet-maintained app. See supported apps: https://fleetdm.com/learn-more-about/supported-fleet-maintained-app-slugs", *slug),
				http.StatusNotFound,
			)
		}
		return err
	}

	payload.RollbackVersion = strings.TrimSpace(payload.RollbackVersion)
	majorVersionString, usesCaret, err := parsePinnedVersion(ctx, payload.RollbackVersion)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "reading Fleet-maintained app pinned version")
	}

	// use a temporary string for calling hydrate, so we download the latest manifest but keep the
	// version in the db later for the auto update cron job
	hydrateVersion := payload.RollbackVersion
	if usesCaret {
		hydrateVersion = ""
	}

	_, err = maintained_apps.Hydrate(ctx, app, hydrateVersion, teamID, svc.ds)
	if err != nil {
		return err
	}

	if usesCaret {
		if !versionMatchesMajor(app.Version, majorVersionString) {
			// We cannot use the FMA we just got the manifest for since it is on a different major
			// version, so we try to find the latest cached version and use that instead.
			if app.TitleID == nil {
				return fleet.NewUserMessageError(errMajorVersionNotFound, http.StatusNotFound)
			}
			versions, err := svc.ds.GetFleetMaintainedVersionsByTitleID(ctx, teamID, *app.TitleID, true)
			if err != nil {
				return fleet.NewUserMessageError(errMajorVersionNotFound, http.StatusNotFound)
			}

			// This is a bit inefficient as we are duplicating strings for categories and install/uninstall scripts,
			// but it can be optimized in softwareBatchUpload if it accepted only passing category and script content IDs.
			installer, err := svc.ds.GetCachedFMAInstallerMetadata(ctx, teamID, app.ID, versions[0].Version)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting software installer")
			}

			app.Version = installer.Version
			app.InstallerURL = installer.InstallerURL
			app.SHA256 = installer.SHA256
			app.InstallScript = installer.InstallScript
			app.UninstallScript = installer.UninstallScript
			app.Categories = installer.Categories
			app.PatchQuery = installer.PatchQuery
		}
	}

	payload.URL = app.InstallerURL
	if app.SHA256 != noCheckHash {
		payload.SHA256 = app.SHA256
	}
	if payload.InstallScript == "" {
		payload.InstallScript = app.InstallScript
	}
	if payload.UninstallScript == "" {
		payload.UninstallScript = app.UninstallScript
	}
	payload.FleetMaintained = true
	payload.MaintainedApp = app
	if !payload.Categories.Set {
		payload.Categories = optjson.SetSlice(app.Categories)
	}
	payload.MaintainedApp.PatchQuery = app.PatchQuery

	return nil
}

const (
	batchSetProcessing   = "processing"
	batchSetCompleted    = "completed"
	batchSetFailedPrefix = "failed:"
)

// downloadInstallerURL downloads an installer from a URL. If ifNoneMatch is
// non-empty, the request includes an If-None-Match header for conditional GET.
//
// On 304 Not Modified, returns (resp, nil, nil): resp has StatusCode 304 and a
// closed body, tfr is nil. Callers MUST check resp.StatusCode before using tfr.
func downloadInstallerURL(ctx context.Context, downloadURL string, ifNoneMatch string, maxInstallerSize int64) (*http.Response, *fleet.TempFileReader, error) {
	client := fleethttp.NewClient()
	client.Transport = fleethttp.NewSizeLimitTransport(maxInstallerSize)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("creating request for URL %q: %w", downloadURL, err)
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	resp, err := client.Do(req)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.Is(err, fleethttp.ErrMaxSizeExceeded) || errors.As(err, &maxBytesErr) {
			return nil, nil, fleet.NewInvalidArgumentError(
				"software.url",
				fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %s", downloadURL, installersize.Human(maxInstallerSize)),
			)
		}

		return nil, nil, fmt.Errorf("performing request for URL %q: %w", downloadURL, err)
	}

	// 304 Not Modified: content unchanged, return response with no body.
	// Set Body to http.NoBody after closing so downstream Close() calls are safe.
	if resp.StatusCode == http.StatusNotModified {
		resp.Body.Close()
		resp.Body = http.NoBody
		return resp, nil, nil
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil, fleet.NewInvalidArgumentError(
			"software.url",
			fmt.Sprintf("Couldn't edit software. URL (%q) returned \"Not Found\". Please make sure that URLs are reachable from your Fleet server.", downloadURL),
		)
	}

	// Allow all 2xx and 3xx status codes in this pass.
	if resp.StatusCode >= 400 {
		return nil, nil, fleet.NewInvalidArgumentError(
			"software.url",
			fmt.Sprintf("Couldn't edit software. URL (%q) received response status code %d.", downloadURL, resp.StatusCode),
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
				fmt.Sprintf("Couldn't edit software. URL (%q). The maximum file size is %s", downloadURL, installersize.Human(maxInstallerSize)),
			)
		}
		return nil, nil, fmt.Errorf("reading installer %q contents: %w", downloadURL, err)
	}

	return resp, tfr, nil
}

func (svc *Service) softwareBatchUpload(
	requestUUID string,
	teamID *uint,
	userID uint,
	payloads []*fleet.SoftwareInstallerPayload,
	dryRun bool,
) {
	var batchErr error
	// deletedPackagesJSON holds the JSON-encoded list of packages this batch will
	// delete (dry run: would delete), recorded in Redis on completion.
	var deletedPackagesJSON string

	// TODO: this might be a little drastic to drop back to Background context,
	// consider using ctx.WithoutCancel to keep all but the cancellation of the
	// parent: https://pkg.go.dev/context#WithoutCancel
	// e.g. for telemetry and such.

	// We do not use the request ctx on purpose because this method runs in the background.
	ctx := context.Background()

	defer func(start time.Time) {
		// The deleted-packages list was already persisted before any datastore
		// mutation; re-set it here to refresh its TTL so the client has the full
		// window to read it even after a long-running batch. Best-effort only: at
		// this point the batch may have already committed, so a Redis failure must
		// not mark it as failed.
		if batchErr == nil && deletedPackagesJSON != "" {
			if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID+batchSoftwareDeletedSuffix, deletedPackagesJSON, 10*time.Minute); err != nil {
				svc.logger.WarnContext(ctx, "failed to refresh deleted-packages result; the deletion report may be missing from the batch result",
					"request_uuid", requestUUID,
					"err", err,
				)
			}
		}
		status := batchSetCompleted
		if batchErr != nil {
			status = fmt.Sprintf("%s%s", batchSetFailedPrefix, batchErr)
		}
		logger := svc.logger.With(
			"request_uuid", requestUUID,
			"team_id", teamID,
			"payloads", len(payloads),
			"status", status,
			"took", time.Since(start),
		)
		logger.InfoContext(ctx, "software batch done")
		// Give 10m for the client to read the result (it overrides the previos expiration time).
		if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID, status, 10*time.Minute); err != nil {
			logger.ErrorContext(ctx, "failed to set result", "err", err)
		}
	}(time.Now())

	// Periodically refresh the expiration on the batch install process so that, even when downloading/uploading
	// large installers, we ensure the server doesn't lose track of the batch. This way, the only time a batch times
	// out is if the server goes offline during running the batch.
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(keyExpireTime / 3) // Running keepalive much more often since we don't retry set errors
		defer ticker.Stop()
		for {
			select {
			// at this point we're done with the batch, at which point the caller will set the job in Redis as complete
			// with a longer TTL, so we don't need to do anything here
			case <-done:
				return
			case <-ticker.C:
				_ = svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID, batchSetProcessing, keyExpireTime)
			}
		}
	}()
	defer close(done)

	maxInstallerSize := svc.config.Server.MaxInstallerSizeBytes
	downloadURLFn := func(ctx context.Context, downloadURL string, ifNoneMatch string) (*http.Response, *fleet.TempFileReader, error) {
		return downloadInstallerURL(ctx, downloadURL, ifNoneMatch, maxInstallerSize)
	}

	// retryDownload wraps downloadURLFn with the standard retry policy.
	// Note: a 304 response returns nil error and is treated as success (not retried).
	retryDownload := func(ctx context.Context, downloadURL, ifNoneMatch string) (*http.Response, *fleet.TempFileReader, error) {
		var resp *http.Response
		var tfr *fleet.TempFileReader
		err := retry.Do(func() error {
			// Close resources from a previous attempt to avoid leaking
			// file descriptors, temp files, and HTTP connections.
			if tfr != nil {
				tfr.Close()
				tfr = nil
			}
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
				resp = nil
			}
			var retryErr error
			resp, tfr, retryErr = downloadURLFn(ctx, downloadURL, ifNoneMatch)
			return retryErr
		}, retry.WithMaxAttempts(fleet.BatchDownloadMaxRetries), retry.WithInterval(fleet.BatchSoftwareInstallerRetryInterval()))
		return resp, tfr, err
	}

	var manualAgentInstall bool
	tmID := ptr.ValOrZero(teamID)
	if tmID == 0 {
		ac, err := svc.ds.AppConfig(ctx)
		if err != nil {
			batchErr = fmt.Errorf("Couldn't get app config: %w", err)
			return
		}
		manualAgentInstall = ac.MDM.MacOSSetup.ManualAgentInstall.Value
	} else {
		team, err := svc.ds.TeamLite(ctx, tmID)
		if err != nil {
			batchErr = fmt.Errorf("Couldn't get team for team ID %d: %w", tmID, err)
			return
		}
		manualAgentInstall = team.Config.MDM.MacOSSetup.ManualAgentInstall.Value
	}

	var g errgroup.Group
	g.SetLimit(1) // TODO: consider whether we can increase this limit, see https://github.com/fleetdm/fleet/issues/22704#issuecomment-2397407837

	// the reason for this struct with extra installers support is that:
	// - ih-house apps match multiple installers to a single source installer
	//   payload (because an .ipa creates entries for iOS and iPadOS)
	// - the for loop over each entry in the payload is executed in a goroutine
	//   that can only write to its pre-allocated index in the installers slice, so
	//   any extra installer for a given payload must be part of a single value
	//   inserted in that slice.
	type installerPayloadWithExtras struct {
		*fleet.UploadSoftwareInstallerPayload
		ExtraInstallers []*fleet.UploadSoftwareInstallerPayload
	}

	// critical to avoid data race, the slices are pre-allocated and each
	// goroutine only writes to its index.
	installers := make([]*installerPayloadWithExtras, len(payloads))
	toBeClosedTFRs := make([]*fleet.TempFileReader, len(payloads))

	for i, p := range payloads {
		i, p := i, p

		g.Go(func() error {
			// NOTE: cannot defer tfr.Close() here because the reader needs to be
			// available after the goroutine completes. Instead, all temp file
			// readers are collected in toBeClosedTFRs and will have their Close
			// deferred after the join/wait of goroutines.
			installer := &fleet.UploadSoftwareInstallerPayload{
				TeamID:             teamID,
				InstallScript:      p.InstallScript,
				PreInstallQuery:    p.PreInstallQuery,
				PostInstallScript:  p.PostInstallScript,
				UninstallScript:    p.UninstallScript,
				SelfService:        p.SelfService,
				UserID:             userID,
				URL:                p.URL,
				InstallDuringSetup: p.InstallDuringSetup,
				LabelsIncludeAny:   p.LabelsIncludeAny,
				LabelsExcludeAny:   p.LabelsExcludeAny,
				LabelsIncludeAll:   p.LabelsIncludeAll,
				ValidatedLabels:    p.ValidatedLabels,
				Categories:         p.Categories.Value,
				DisplayName:        p.DisplayName,
				RollbackVersion:    p.RollbackVersion,
				AlwaysDownload:     p.AlwaysDownload,
				Configuration:      p.Configuration,
			}

			var extraInstallers []*fleet.UploadSoftwareInstallerPayload

			categories, catIDs, err := svc.removeDuplicateOrMissingCategories(ctx, tmID, p.Categories.Value)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "filtering software installer categories")
			}
			installer.Categories = categories
			installer.CategoryIDs = catIDs

			// check if we already have the installer based on the SHA256 and URL
			teamIDs, err := svc.ds.GetTeamsWithInstallerByHash(ctx, p.SHA256, p.URL)
			if err != nil {
				return err
			}

			foundInstallers, ok := teamIDs[tmID]
			switch {
			case ok:
				// Perfect match: existing installer on the same team
				foundInstaller := foundInstallers[0]

				if foundInstaller.Extension == "exe" || foundInstaller.Extension == "tar.gz" {
					if p.InstallScript == "" {
						return fmt.Errorf("Couldn't edit. Install script is required for .%s packages.", foundInstaller.Extension)
					}

					if p.UninstallScript == "" {
						return fmt.Errorf("Couldn't edit. Uninstall script is required for .%s packages.", foundInstaller.Extension)
					}
				}

				// make a copy of the installer without filled fields in case we add
				// extra installers
				extraInstallerBase := *installer
				if err := svc.fillSoftwareInstallerPayloadFromExisting(ctx, installer, foundInstaller, p.SHA256); err != nil {
					return err
				}
				for _, extraInstaller := range foundInstallers[1:] {
					extraPayload := extraInstallerBase
					if err := svc.fillSoftwareInstallerPayloadFromExisting(ctx, &extraPayload, extraInstaller, p.SHA256); err != nil {
						return err
					}
					extraInstallers = append(extraInstallers, &extraPayload)
				}

			case !ok && len(teamIDs) > 0:
				// Installer(s) exists, but for another team. We should copy it over to this team
				// (if we have access to the other team).
				user, err := svc.ds.UserByID(ctx, userID)
				if err != nil {
					return err
				}

				userctx := viewer.NewContext(ctx, viewer.Viewer{User: user})

				for tmID, teamInstallers := range teamIDs {
					// use the first one to which this user has access; the specific one shouldn't
					// matter because they're all the same installer bytes
					var tmIDPtr *uint
					if tmID != 0 {
						tmIDPtr = ptr.Uint(tmID)
					}
					if authErr := svc.authz.Authorize(userctx, &fleet.SoftwareInstaller{TeamID: tmIDPtr}, fleet.ActionWrite); authErr != nil {
						continue
					}

					teamInstaller := teamInstallers[0]
					if teamInstaller.Extension == "exe" || teamInstaller.Extension == "zip" {
						if p.InstallScript == "" {
							ext := teamInstaller.Extension
							return fmt.Errorf("Couldn't edit. Install script is required for .%s packages.", ext)
						}

						if p.UninstallScript == "" {
							ext := teamInstaller.Extension
							return fmt.Errorf("Couldn't edit. Uninstall script is required for .%s packages.", ext)
						}
					}

					// make a copy of the installer without filled fields in case we add
					// extra installers
					extraInstallerBase := *installer
					if err := svc.fillSoftwareInstallerPayloadFromExisting(ctx, installer, teamInstaller, p.SHA256); err != nil {
						return err
					}
					for _, extraInstaller := range teamInstallers[1:] {
						extraPayload := extraInstallerBase
						if err := svc.fillSoftwareInstallerPayloadFromExisting(ctx, &extraPayload, extraInstaller, p.SHA256); err != nil {
							return err
						}
						extraInstallers = append(extraInstallers, &extraPayload)
					}

					break
				}
			}

			// For FMA installers, check if this version is already cached for this team.
			var fmaVersionCached bool
			if p.Slug != nil && *p.Slug != "" && p.MaintainedApp != nil && p.MaintainedApp.Version != "" {
				cached, err := svc.ds.HasFMAInstallerVersion(ctx, teamID, p.MaintainedApp.ID, p.MaintainedApp.Version)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "check cached FMA version")
				}
				fmaVersionCached = cached
				installer.FMAVersionCached = cached
			}

			var installerBytesExist bool
			if !fmaVersionCached && p.SHA256 != "" {
				installerBytesExist, err = svc.softwareInstallStore.Exists(ctx, installer.StorageID)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "check if installer exists in store")
				}
			}

			// no accessible matching installer was found, so attempt to download it from URL.
			if !fmaVersionCached && (installer.StorageID == "" || !installerBytesExist) {
				if p.SHA256 != "" && p.URL == "" {
					return fmt.Errorf("package not found with hash %s", p.SHA256)
				}

				var tfr *fleet.TempFileReader

				// Handle script packages from path (script:// URL scheme)
				if filename, ok := strings.CutPrefix(p.URL, "script://"); ok {
					ext := strings.ToLower(filepath.Ext(filename))
					ext = strings.TrimPrefix(ext, ".")

					if !fleet.IsScriptPackage(ext) {
						return fmt.Errorf("script:// URL must reference a .sh or .ps1 file, got: %s", filename)
					}

					if p.InstallScript == "" {
						return fmt.Errorf("script package %s has no install script content", filename)
					}

					scriptContent := []byte(p.InstallScript)
					tfr, err = fleet.NewTempFileReader(bytes.NewReader(scriptContent), nil)
					if err != nil {
						return fmt.Errorf("creating temp file for script package %s: %w", filename, err)
					}

					installer.InstallerFile = tfr
					toBeClosedTFRs[i] = tfr
					installer.Filename = filename
				} else {
					// Conditional GET (default behavior, disabled by always_download: true).
					// Look up existing installer by URL for its ETag, only when
					// we're about to download (avoids wasted DB queries).
					var existingForCache *fleet.ExistingSoftwareInstaller
					var ifNoneMatch string
					if !p.AlwaysDownload && p.SHA256 == "" && p.URL != "" {
						// First try same-team lookup, then fall back to any team.
						existing, lookupErr := svc.ds.GetInstallerByTeamAndURL(ctx, &tmID, p.URL)
						if lookupErr != nil {
							svc.logger.WarnContext(ctx, "conditional download lookup failed, will download normally", "url", p.URL, "err", lookupErr)
						} else if existing == nil {
							// Cross-team fallback: another team may already have this URL cached.
							existing, lookupErr = svc.ds.GetInstallerByTeamAndURL(ctx, nil, p.URL)
							if lookupErr != nil {
								svc.logger.WarnContext(ctx, "cross-team conditional download lookup failed, will download normally", "url", p.URL, "err", lookupErr)
							}
						}
						if lookupErr == nil && existing != nil && existing.StorageID != "" &&
							existing.HTTPETag != nil && *existing.HTTPETag != "" &&
							existing.Extension != "ipa" && // skip conditional download for .ipa (multi-platform extraInstallers)
							validETag(*existing.HTTPETag) { // re-validate before use as defense-in-depth
							existingForCache = existing
							ifNoneMatch = *existing.HTTPETag
						}
					}

					resp, tfr, err := retryDownload(ctx, p.URL, ifNoneMatch)
					if err != nil {
						return err
					}

					// Handle 304 Not Modified (conditional download with matching ETag).
					// TRUST ASSUMPTION: conditional download trusts the origin server's
					// ETag as a content fingerprint, so we reuse the cached installer
					// bytes and metadata (filename, version, extension, etc.) without
					// re-extraction. Flow continues past the download-specific code so
					// that script fields from the user's GitOps config still pass
					// through the shared normalization/validation below.
					var cacheHit bool
					if resp != nil && resp.StatusCode == http.StatusNotModified && existingForCache != nil {
						bytesExist, existErr := svc.softwareInstallStore.Exists(ctx, existingForCache.StorageID)
						if existErr == nil && bytesExist {
							if err := svc.fillSoftwareInstallerPayloadFromExisting(ctx, installer, existingForCache, existingForCache.StorageID); err != nil {
								return err
							}
							installer.HTTPETag = existingForCache.HTTPETag
							// Propagate the existing hash so FMA hydration below
							// doesn't try to recompute it from the (nil) file
							// reader when the manifest uses noCheckHash.
							if p.MaintainedApp != nil {
								p.MaintainedApp.SHA256 = existingForCache.StorageID
							}
							cacheHit = true
						} else {
							svc.logger.WarnContext(ctx, "304 received but installer bytes missing, re-downloading", "url", p.URL)
							resp, tfr, err = retryDownload(ctx, p.URL, "")
							if err != nil {
								return err
							}
							if resp != nil && resp.StatusCode == http.StatusNotModified {
								return fmt.Errorf("server returned 304 on unconditional re-download of %q", p.URL)
							}
						}
					}

					if !cacheHit {
						// Protocol violation guards: downloadURLFn never returns nil resp
						// on success, but guard defensively for server misbehavior.
						if resp == nil || tfr == nil {
							statusCode := 0
							if resp != nil {
								statusCode = resp.StatusCode
							}
							return fmt.Errorf("download of %q returned no body (status %d)", p.URL, statusCode)
						}

						installer.InstallerFile = tfr
						toBeClosedTFRs[i] = tfr

						filename := maintained_apps.FilenameFromResponse(resp)
						installer.Filename = filename

						// Always capture ETag from download response so it's available
						// immediately if always_download is later disabled.
						if etag := resp.Header.Get("ETag"); etag != "" && validETag(etag) {
							installer.HTTPETag = &etag
						} else {
							svc.logger.DebugContext(ctx, "no usable ETag from server for conditional download", "url", p.URL, "etag", resp.Header.Get("ETag"))
						}

						// In-house apps (.ipa) don't support custom scripts or a
						// pre-install query; clear them.
						ext := strings.ToLower(filepath.Ext(filename))
						ext = strings.TrimPrefix(ext, ".")
						if ext == "ipa" {
							installer.InstallScript = ""
							installer.PostInstallScript = ""
							installer.UninstallScript = ""
							installer.PreInstallQuery = ""
						}
					}
				}
			}

			if p.Slug != nil && *p.Slug != "" {
				// Fleet maintained software hydration
				// This code should be extracted for common use from here and AddFleetMaintainedApp in maintained_apps.go
				// It's the same code and would be nice to get some reuse
				appName := p.MaintainedApp.UniqueIdentifier
				if p.MaintainedApp.Platform == "darwin" || appName == "" {
					appName = p.MaintainedApp.Name
				}
				if installer.Filename == "" {
					parsedURL, err := url.Parse(installer.URL)
					if err != nil {
						return fmt.Errorf("Error with maintained app, parsing URL: %v\n", err)
					}
					installer.Filename = path.Base(parsedURL.Path)
				}
				// noCheckHash is used by homebrew to signal that a hash shouldn't be checked
				// This comes from the manifest and is a special case for maintained apps
				// we need to generate the SHA256 from the installer file.
				// Skip when version is cached — the existing row already has the computed hash.
				if !fmaVersionCached && p.MaintainedApp.SHA256 == noCheckHash {
					// generate the SHA256 from the installer file
					if installer.InstallerFile == nil {
						return fmt.Errorf("maintained app %s requires hash to be generated but no installer file found", p.MaintainedApp.UniqueIdentifier)
					}
					p.MaintainedApp.SHA256, err = file.SHA256FromTempFileReader(installer.InstallerFile)
					if err != nil {
						return fmt.Errorf("maintained app %s error generating hash: %w", p.MaintainedApp.UniqueIdentifier, err)
					}
				}
				extension := strings.TrimLeft(filepath.Ext(installer.Filename), ".")
				installer.Title = appName
				installer.Version = p.MaintainedApp.Version

				// Some FMAs (e.g. Chrome for macOS) aren't version-pinned by URL, so we have to extract the
				// version from the package once we download it.
				// Skip when version is cached — the existing row already has the correct version.
				if !fmaVersionCached && installer.Version == "latest" && installer.InstallerFile != nil {
					meta, err := file.ExtractInstallerMetadata(installer.InstallerFile)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "extracting installer metadata")
					}

					// reset the reader (it was consumed to extract metadata)
					if err := installer.InstallerFile.Rewind(); err != nil {
						return ctxerr.Wrap(ctx, err, "resetting installer file reader")
					}

					installer.Version = meta.Version
				}

				installer.Platform = p.MaintainedApp.Platform
				installer.Source = p.MaintainedApp.Source()
				if installer.Source == "programs" && p.MaintainedApp.UpgradeCode != "" {
					installer.UpgradeCode = p.MaintainedApp.UpgradeCode
				}

				installer.Extension = extension
				installer.BundleIdentifier = p.MaintainedApp.BundleIdentifier()
				installer.StorageID = p.MaintainedApp.SHA256
				installer.FleetMaintainedAppID = &p.MaintainedApp.ID
				installer.PatchQuery = p.MaintainedApp.PatchQuery
			}

			var ext string
			if installer.FleetMaintainedAppID == nil && installer.InstallerFile != nil {
				ext, err = svc.addMetadataToSoftwarePayload(ctx, installer, true)
				if err != nil {
					return err
				}

				if p.SHA256 != "" && p.SHA256 != installer.StorageID {
					// this isn't the specified installer, so return an error
					return fmt.Errorf("downloaded installer hash does not match provided hash for installer with url %s", p.URL)
				}
			}

			// Managed app configuration is only supported for iOS / iPadOS in-house apps.
			if installer.Extension != "ipa" {
				installer.Configuration = nil
			}

			switch {
			case fleet.IsScriptPackage(installer.Extension):
				// Keep the file-derived install script and the provided post-install,
				// uninstall, and pre-install query; skip the default-script injection
				// below. Path-based script packages carry their filename in a
				// "script://" url — an internal placeholder, not a real download url,
				// so don't persist it.
				if strings.HasPrefix(installer.URL, "script://") {
					installer.URL = ""
				}

			case installer.Extension != "exe":
				// custom scripts only for exe installers and non-script packages
				installer.InstallScript = getInstallScript(installer.Extension, installer.PackageIDs, installer.InstallScript)

				if installer.UninstallScript == "" {
					installer.UninstallScript = file.GetUninstallScript(installer.Extension)
				}

			case installer.Extension == "ipa":
				installer.PostInstallScript = ""
				installer.UninstallScript = ""
				installer.PreInstallQuery = ""
				installer.InstallScript = ""
			}

			if fleet.IsMacOSPlatform(installer.Platform) && ptr.ValOrZero(installer.InstallDuringSetup) && manualAgentInstall {
				return errors.New(`Couldn't edit software. "setup_experience" cannot be used for macOS software if "macos_manual_agent_install" is enabled.`)
			}

			// Update $PACKAGE_ID/$UPGRADE_CODE in uninstall script
			if err := preProcessUninstallScript(installer); err != nil {
				return fmt.Errorf("processing uninstall script: %w", err)
			}

			// A script package's install script is the uploaded file, validated in
			// addScriptPackageMetadata, so only post-install/uninstall are checked here.
			scriptsToValidate := []struct {
				name    string
				content string
			}{
				{"post-install script", installer.PostInstallScript},
				{"uninstall script", installer.UninstallScript},
			}
			if !fleet.IsScriptPackage(installer.Extension) {
				scriptsToValidate = append(scriptsToValidate, struct {
					name    string
					content string
				}{"install script", installer.InstallScript})
			}
			for _, sv := range scriptsToValidate {
				if err := fleet.ValidateSoftwareInstallerScript(sv.content, installer.Platform); err != nil {
					return fmt.Errorf("Couldn't edit software. %s validation failed: %s", sv.name, err.Error())
				}
			}

			// if filename was empty, try to extract it from the URL with the
			// now-known extension
			if installer.Filename == "" {
				installer.Filename = file.ExtractFilenameFromURLPath(p.URL, ext)
			}
			// if empty, resort to a default name
			if installer.Filename == "" {
				installer.Filename = fmt.Sprintf("package.%s", ext)
			}
			if installer.Title == "" && installer.Extension != "ipa" {
				// If an IPA is specified via hash rather than downloaded via URL, we won't have a title populated,
				// and should try to pull the title from the database if it exists. If we can't extract title name for
				// some reason, filename should only be used after attempting to pull data from the database.
				installer.Title = installer.Filename
			}

			// if this is an .ipa and there is no extra installer, create it here
			if installer.Extension == "ipa" && len(extraInstallers) == 0 {
				extraPayload := *installer
				switch installer.Platform {
				case string(fleet.IOSPlatform):
					extraPayload.Platform = string(fleet.IPadOSPlatform)
					extraPayload.Source = "ipados_apps"
				case string(fleet.IPadOSPlatform):
					extraPayload.Platform = string(fleet.IOSPlatform)
					extraPayload.Source = "ios_apps"
				}
				extraInstallers = append(extraInstallers, &extraPayload)
			}

			installers[i] = &installerPayloadWithExtras{
				UploadSoftwareInstallerPayload: installer,
				ExtraInstallers:                extraInstallers,
			}

			return nil
		})
	}

	waitErr := g.Wait()

	// defer close for any valid temp file reader
	for _, tfr := range toBeClosedTFRs {
		if tfr != nil {
			defer tfr.Close()
		}
	}

	if waitErr != nil {
		// NOTE: intentionally not wrapping to avoid polluting user errors.
		batchErr = waitErr
		return
	}

	// Compute which existing packages this batch will delete (dry run: would
	// delete): the installers on the team whose title matches no incoming
	// payload, mirroring the title-based deletion in ds.BatchSetSoftwareInstallers.
	incoming := make([]fleet.SoftwareTitleIdentifier, 0, len(installers))
	for _, payloadWithExtras := range installers {
		for _, p := range append([]*fleet.UploadSoftwareInstallerPayload{payloadWithExtras.UploadSoftwareInstallerPayload}, payloadWithExtras.ExtraInstallers...) {
			incoming = append(incoming, fleet.SoftwareTitleIdentifier{
				UniqueIdentifier: p.UniqueIdentifier(),
				Source:           p.Source,
			})
		}
	}
	deletedPackages, err := svc.ds.GetSoftwareInstallersPendingDeletion(ctx, teamID, incoming)
	if err != nil {
		batchErr = fmt.Errorf("computing software packages pending deletion: %w", err)
		return
	}
	if len(deletedPackages) > 0 {
		deletedJSON, err := json.Marshal(deletedPackages)
		if err != nil {
			batchErr = fmt.Errorf("encoding software packages pending deletion: %w", err)
			return
		}
		deletedPackagesJSON = string(deletedJSON)
		// Persist before any datastore mutation: a failure here fails the batch
		// while it is still safe to retry (nothing has been applied or deleted),
		// so deletion warnings are never silently missing. The defer refreshes
		// this key's TTL on completion for long-running batches.
		if err := svc.keyValueStore.Set(ctx, batchSoftwarePrefix+requestUUID+batchSoftwareDeletedSuffix, deletedPackagesJSON, 10*time.Minute); err != nil {
			batchErr = fmt.Errorf("recording software packages pending deletion: %w", err)
			return
		}
	}

	if dryRun {
		return
	}

	var inHouseInstallers, softwareInstallers []*fleet.UploadSoftwareInstallerPayload
	for _, payloadWithExtras := range installers {
		payload := payloadWithExtras.UploadSoftwareInstallerPayload
		if !payload.FMAVersionCached {
			batchErr = retry.Do(func() error {
				if retryErr := svc.storeSoftware(ctx, payload); retryErr != nil {
					return fmt.Errorf("storing software installer %q: %w", payload.Filename, retryErr)
				}

				return nil
			}, retry.WithMaxAttempts(fleet.BatchUploadMaxRetries), retry.WithInterval(fleet.BatchSoftwareInstallerRetryInterval()))
		}
		if payload.Extension == "ipa" {
			inHouseInstallers = append(inHouseInstallers, payload)
			inHouseInstallers = append(inHouseInstallers, payloadWithExtras.ExtraInstallers...)
		} else {
			softwareInstallers = append(softwareInstallers, payload)
			softwareInstallers = append(softwareInstallers, payloadWithExtras.ExtraInstallers...)
		}
	}

	if err := svc.ds.BatchSetSoftwareInstallers(ctx, teamID, softwareInstallers); err != nil {
		batchErr = fmt.Errorf("batch set software installers: %w", err)
		return
	}
	if err := svc.ds.BatchSetInHouseAppsInstallers(ctx, teamID, inHouseInstallers); err != nil {
		batchErr = fmt.Errorf("batch set in-house apps installers: %w", err)
		return
	}

	// Note: per @noahtalerman we don't want activity items for CLI actions
	// anymore, so that's intentionally skipped.
}

func (svc *Service) fillSoftwareInstallerPayloadFromExisting(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload, existing *fleet.ExistingSoftwareInstaller, sha256Hash string) error {
	payload.Extension = existing.Extension
	payload.Filename = existing.Filename
	payload.Version = existing.Version
	payload.Platform = existing.Platform
	payload.Source = existing.Source
	if existing.BundleIdentifier != nil {
		payload.BundleIdentifier = *existing.BundleIdentifier
	}
	payload.Title = existing.Title
	payload.StorageID = sha256Hash
	payload.PackageIDs = existing.PackageIDs

	if fleet.IsScriptPackage(existing.Extension) {
		contents, err := svc.ds.GetAnyScriptContents(ctx, existing.InstallScriptContentID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "fetch install script for hash-matched script package")
		}
		payload.InstallScript = string(contents)
	}

	return nil
}

// validETag checks if an ETag value is a strong ETag per RFC 7232
// section 2.3: a quoted opaque-tag without the weak validator prefix.
// Weak ETags (W/"...") are rejected because they indicate semantic
// equivalence rather than byte-for-byte identity, which is insufficient
// for validating cached binary installers.
// The opaque-tag body must consist of RFC 7232 etagc characters
// (%x21 / %x23-7E), which excludes control chars, spaces, inner DQUOTEs,
// and DEL. We reject obs-text (>0x7F) for defense-in-depth. Values over
// 512 bytes are rejected.
func validETag(etag string) bool {
	if len(etag) > 512 {
		return false
	}
	// Reject weak ETags — they don't guarantee byte-identical content.
	if strings.HasPrefix(etag, "W/") {
		return false
	}
	e := etag
	if len(e) < 2 || e[0] != '"' || e[len(e)-1] != '"' {
		return false
	}
	for i := 1; i < len(e)-1; i++ {
		c := e[i]
		// RFC 7232 etagc = %x21 / %x23-7E / obs-text. Reject obs-text
		// (>0x7F) for defense-in-depth.
		if c != 0x21 && (c < 0x23 || c > 0x7E) {
			return false
		}
	}
	return true
}

func (svc *Service) GetBatchSetSoftwareInstallersResult(ctx context.Context, tmName string, requestUUID string, dryRun bool) (string, string, []fleet.SoftwarePackageResponse, []fleet.DeletedSoftwarePackage, []string, error) {
	// We've already authorized in the POST /api/latest/fleet/software/batch,
	// but adding it here so we don't need to worry about a special case endpoint.
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return "", "", nil, nil, nil, err
	}

	result, err := svc.keyValueStore.Get(ctx, batchSoftwarePrefix+requestUUID)
	if err != nil {
		return "", "", nil, nil, nil, ctxerr.Wrap(ctx, err, "failed to get result")
	}
	if result == nil {
		return "", "", nil, nil, nil, ctxerr.Wrap(ctx, &notFoundError{}, "request_uuid not found")
	}

	// getDeletedPackages loads the packages the batch deleted (dry run: would
	// delete). A missing or expired key degrades to an empty list, not an error.
	getDeletedPackages := func() ([]fleet.DeletedSoftwarePackage, error) {
		deletedJSON, err := svc.keyValueStore.Get(ctx, batchSoftwarePrefix+requestUUID+batchSoftwareDeletedSuffix)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to get deleted packages result")
		}
		if deletedJSON == nil || *deletedJSON == "" {
			return nil, nil
		}
		var deletedPackages []fleet.DeletedSoftwarePackage
		if err := json.Unmarshal([]byte(*deletedJSON), &deletedPackages); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal deleted packages result")
		}
		return deletedPackages, nil
	}

	// getCategories loads the self-service categories the batch's software references
	getCategories := func() ([]string, error) {
		categoriesJSON, err := svc.keyValueStore.Get(ctx, batchSoftwarePrefix+requestUUID+batchSoftwareCategoriesSuffix)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "failed to get categories result")
		}
		if categoriesJSON == nil || *categoriesJSON == "" {
			return nil, nil
		}
		var categories []string
		if err := json.Unmarshal([]byte(*categoriesJSON), &categories); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "unmarshal categories result")
		}
		return categories, nil
	}

	switch {
	case *result == batchSetCompleted:
		// fall through to retrieving the (deleted) software packages below.
	case *result == batchSetProcessing:
		return fleet.BatchSetSoftwareInstallersStatusProcessing, "", nil, nil, nil, nil
	case strings.HasPrefix(*result, batchSetFailedPrefix):
		message := strings.TrimPrefix(*result, batchSetFailedPrefix)
		return fleet.BatchSetSoftwareInstallersStatusFailed, message, nil, nil, nil, nil
	default:
		return "", "", nil, nil, nil, ctxerr.New(ctx, "invalid status")
	}

	var (
		teamID    uint  // GetSoftwareInstallers uses 0 for "No team"
		ptrTeamID *uint // Authorize uses *uint for "No team" teamID
	)
	if tmName != "" {
		team, err := svc.ds.TeamByName(ctx, tmName)
		if err != nil {
			return "", "", nil, nil, nil, ctxerr.Wrap(ctx, err, "load team by name")
		}
		teamID = team.ID
		ptrTeamID = &team.ID
	}

	// We've already authorized in the POST /api/latest/fleet/software/batch,
	// but adding it here so we don't need to worry about a special case endpoint.
	//
	// We use fleet.ActionWrite because this method is the counterpart of the POST
	// /api/latest/fleet/software/batch. This applies to dry runs too, since the
	// deleted-packages list exposes team-scoped software data.
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: ptrTeamID}, fleet.ActionWrite); err != nil {
		return "", "", nil, nil, nil, ctxerr.Wrap(ctx, err, "validating authorization")
	}

	deletedPackages, err := getDeletedPackages()
	if err != nil {
		return "", "", nil, nil, nil, err
	}

	categories, err := getCategories()
	if err != nil {
		return "", "", nil, nil, nil, err
	}

	if dryRun {
		return fleet.BatchSetSoftwareInstallersStatusCompleted, "", nil, deletedPackages, categories, nil
	}

	softwarePackages, err := svc.ds.GetSoftwareInstallers(ctx, teamID)
	if err != nil {
		return "", "", nil, nil, nil, ctxerr.Wrap(ctx, err, "get software installers")
	}

	return fleet.BatchSetSoftwareInstallersStatusCompleted, "", softwarePackages, deletedPackages, categories, nil
}

func (svc *Service) SelfServiceInstallSoftwareTitle(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	// User-enrolled (BYOD) iOS/iPadOS hosts are no longer blocked from
	// self-service. The downstream VPP install flow handles user-scoped
	// licensing via clientUserIds. End-to-end success still depends on the
	// main install-gate removal landing (#31138 subtask 01).
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

		scoped, err := svc.ds.IsSoftwareInstallerLabelScoped(ctx, installer.InstallerID, host.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "checking label scoping during software install attempt")
		}

		if !scoped {
			return &fleet.BadRequestError{
				Message: "Couldn't install. Host isn't member of the labels defined for this software title.",
			}
		}

		ext, requiredPlatform := installerRequiredPlatform(installer)
		if requiredPlatform == "" {
			// this should never happen
			return ctxerr.Errorf(ctx, "software installer has unsupported type %s", ext)
		}

		if host.FleetPlatform() != requiredPlatform {
			// Allow .sh scripts for any unix-like platform (linux and darwin)
			if !(ext == ".sh" && fleet.IsUnixLike(host.Platform)) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf("Package (%s) can be installed only on %s hosts.", ext, requiredPlatform),
					InternalErr: ctxerr.WrapWithData(
						ctx, err, "invalid host platform for requested installer",
						map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
					),
				}
			}
		}

		if err := svc.ds.ResetNonPolicyInstallAttempts(ctx, host.ID, installer.InstallerID); err != nil {
			return ctxerr.Wrap(ctx, err, "reset install attempts before self-service install")
		}

		_, err = svc.ds.InsertSoftwareInstallRequest(ctx, host.ID, installer.InstallerID, fleet.HostSoftwareInstallOptions{
			SelfService: true,
			WithRetries: true,
		})
		return ctxerr.Wrap(ctx, err, "inserting self-service software install request")
	}

	vppApp, err := svc.ds.GetVPPAppByTeamAndTitleID(ctx, host.TeamID, softwareTitleID)
	if err != nil {
		// if we couldn't find an installer or a VPP app, try an in-house app
		if fleet.IsNotFound(err) {
			return svc.selfServiceInstallInHouseApp(ctx, host, softwareTitleID)
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

	scoped, err := svc.ds.IsVPPAppLabelScoped(ctx, vppApp.VPPAppTeam.AppTeamID, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking vpp label scoping during software install attempt")
	}

	if !scoped {
		return &fleet.BadRequestError{
			Message: "Couldn't install. This software is not available for this host.",
		}
	}

	platform := host.FleetPlatform()
	mobileAppleDevice := fleet.InstallableDevicePlatform(platform) == fleet.IOSPlatform || fleet.InstallableDevicePlatform(platform) == fleet.IPadOSPlatform

	_, err = svc.installSoftwareFromVPP(ctx, host, vppApp, mobileAppleDevice || fleet.InstallableDevicePlatform(platform) == fleet.MacOSPlatform, fleet.HostSoftwareInstallOptions{
		SelfService: true,
	})
	return err
}

func (svc *Service) SelfServiceInstallAllSoftwareTitles(ctx context.Context, host *fleet.Host, categoryID *uint) error {
	// get available self-service titles sorted by name
	titles, categoryName, err := svc.ds.GetSoftwareTitlesForInstallAll(ctx, host, categoryID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get software titles for install all")
	}

	// Queue individual install activities for each title. If any errors occurred while
	// queuing this title we log them and continue to the next software title.
	var queuedCount uint
	for _, title := range titles {
		if err := svc.SelfServiceInstallSoftwareTitle(ctx, host, title.ID); err != nil {
			svc.logger.ErrorContext(ctx, "enqueuing software install", "title_id", title.ID, "err", err)
			continue
		}
		queuedCount++
	}

	if queuedCount == 0 {
		return nil
	}

	if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeInstalledAllSelfServiceSoftware{
		HostID:                  host.ID,
		HostDisplayName:         host.DisplayName(),
		SelfServiceCategoryID:   categoryID,
		SelfServiceCategoryName: categoryName,
		SoftwareTitlesCount:     queuedCount,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating installed all self-service software activity")
	}

	return nil
}

// branching out this call so it doesn't conflict with work in parallel in the
// self-service install method, and it would be good to isolate the installers
// and VPP apps logic too later on.
func (svc *Service) selfServiceInstallInHouseApp(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	iha, err := svc.ds.GetInHouseAppMetadataByTeamAndTitleID(ctx, host.TeamID, softwareTitleID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return &fleet.BadRequestError{
				Message: "Couldn't install software. Software title is not available for install. Please add software package or App Store app to install.",
				InternalErr: ctxerr.WrapWithData(
					ctx, err, "couldn't find an installer, VPP app or in-house app for software title",
					map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
				),
			}
		}
		return ctxerr.Wrap(ctx, err, "install in house app: get metadata")
	}

	if !iha.SelfService {
		return &fleet.BadRequestError{
			Message: "Software title is not available through self-service",
			InternalErr: ctxerr.NewWithData(
				ctx, "software title not available through self-service",
				map[string]any{"host_id": host.ID, "team_id": host.TeamID, "title_id": softwareTitleID},
			),
		}
	}

	scoped, err := svc.ds.IsInHouseAppLabelScoped(ctx, iha.InstallerID, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking label scoping during in-house app install attempt")
	}

	if !scoped {
		return &fleet.BadRequestError{
			Message: "Couldn't install. This software is not available for this host.",
		}
	}

	opts := fleet.HostSoftwareInstallOptions{SelfService: true}
	cfg, err := svc.ds.GetInHouseAppConfiguration(ctx, iha.InstallerID)
	if err != nil && !fleet.IsNotFound(err) {
		return ctxerr.Wrap(ctx, err, "get in-house app configuration for pre-flight check")
	}
	switch err := svc.precheckAppConfigResolvable(ctx, host, cfg); {
	case errors.Is(err, apple_mdm.ErrUnresolvableAppConfigVar):
		return svc.recordFailedInHouseInstall(ctx, host.ID, iha.InstallerID, opts, unresolvableAppConfigFailureReason(err))
	case err != nil:
		return ctxerr.Wrap(ctx, err, "pre-flight substitute fleet variables in in-house app configuration")
	}

	err = svc.ds.InsertHostInHouseAppInstall(ctx, host.ID, iha.InstallerID, softwareTitleID, uuid.NewString(), opts)
	return ctxerr.Wrap(ctx, err, "insert in house app install")
}

// installerRequiredPlatform returns the file extension and the platform used for
// platform validation. The installer's stored Platform is used when set (e.g.
// .zip installers may target windows or darwin). Note that `.sh` installers are
// stored as platform=linux but are allowed on any unix-like host by callers.
func installerRequiredPlatform(installer *fleet.SoftwareInstaller) (ext, requiredPlatform string) {
	ext = filepath.Ext(installer.Name)
	if installer.Platform != "" {
		return ext, installer.Platform
	}
	return ext, packageExtensionToPlatform(ext)
}

// packageExtensionToPlatform returns the platform name based on the
// package extension. Returns an empty string if there is no match. This is only
// used as a fallback by installerRequiredPlatform when an installer has no
// stored Platform; prefer the stored Platform, which is authoritative.
//
// .msix is included for Fleet-maintained Windows apps only; custom package
// upload still rejects .msix (see addMetadataToSoftwarePayload and
// SoftwareInstallerPlatformFromExtension).
//
// .zip is intentionally omitted: it is ambiguous across platforms (a Windows
// installer or a macOS app bundle), so the stored Platform must be used. Both
// FMAs and uploads always set Platform for .zip, so this fallback is never hit
// for zip.
func packageExtensionToPlatform(ext string) string {
	var requiredPlatform string
	switch ext {
	case ".msi", ".exe", ".ps1", ".msix":
		requiredPlatform = "windows"
	case ".pkg", ".dmg":
		requiredPlatform = "darwin"
	case ".deb", ".rpm", ".gz", ".tgz", ".sh":
		requiredPlatform = "linux"
	default:
		return ""
	}

	return requiredPlatform
}

func UpgradeCodeMigration(
	ctx context.Context,
	ds fleet.Datastore,
	softwareInstallStore fleet.SoftwareInstallerStore,
	logger *slog.Logger,
) error {
	// Find MSI installers without upgrade_code
	idMap, err := ds.GetMSIInstallersWithoutUpgradeCode(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting msi installers without upgrade_code")
	}
	if len(idMap) == 0 {
		return nil
	}

	upgradeCodesByStorageID := map[string]string{}

	// Download each package and parse it, if we haven't already
	for id, storageID := range idMap {
		if _, hasParsedUpgradeCode := upgradeCodesByStorageID[storageID]; !hasParsedUpgradeCode {
			// check if the installer exists in the store
			exists, err := softwareInstallStore.Exists(ctx, storageID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "checking if installer exists")
			}
			if !exists {
				logger.WarnContext(ctx, "software installer not found in store", "software_installer_id", id, "storage_id", storageID)
				upgradeCodesByStorageID[storageID] = "" // set to empty string to avoid duplicating work
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
				logger.WarnContext(ctx, "extracting metadata from installer",
					"software_installer_id", id, "storage_id", storageID, "err", err)
				upgradeCodesByStorageID[storageID] = ""
				continue
			}
			meta, err := file.ExtractInstallerMetadata(tfr)
			_ = tfr.Close() // best-effort closing and deleting of temp file
			if err != nil {
				logger.WarnContext(ctx, "extracting metadata from installer",
					"software_installer_id", id, "storage_id", storageID, "err", err)
				upgradeCodesByStorageID[storageID] = ""
				continue
			}
			if meta.UpgradeCode == "" {
				logger.DebugContext(ctx, "no upgrade code found in metadata", "software_installer_id", id, "storage_id", storageID)
			} // fall through since we're going to set the upgrade code even if it's blank

			upgradeCodesByStorageID[storageID] = meta.UpgradeCode
		}

		if upgradeCode, hasParsedUpgradeCode := upgradeCodesByStorageID[storageID]; hasParsedUpgradeCode && upgradeCode != "" {
			// Update the upgrade_code of the software package if we have one
			if err := ds.UpdateInstallerUpgradeCode(ctx, id, upgradeCode); err != nil {
				logger.WarnContext(ctx, "failed to update upgrade code", "software_installer_id", id, "error", err)
				continue
			}
		}
	}

	return nil
}

func UninstallSoftwareMigration(
	ctx context.Context,
	ds fleet.Datastore,
	softwareInstallStore fleet.SoftwareInstallerStore,
	logger *slog.Logger,
) error {
	// Find software installers that should have their uninstall script populated
	idMap, err := ds.GetSoftwareInstallersPendingUninstallScriptPopulation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting software installers to modufy")
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
			logger.WarnContext(ctx, "software installer not found in store", "software_installer_id", id, "storage_id", storageID)
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
			logger.WarnContext(ctx, "extracting metadata from installer",
				"software_installer_id", id, "storage_id", storageID, "err", err)
			continue
		}
		meta, err := file.ExtractInstallerMetadata(tfr)
		_ = tfr.Close() // best-effort closing and deleting of temp file
		if err != nil {
			logger.WarnContext(ctx, "extracting metadata from installer",
				"software_installer_id", id, "storage_id", storageID, "err", err)
			continue
		}
		if len(meta.PackageIDs) == 0 {
			logger.WarnContext(ctx, "no package_id found in metadata", "software_installer_id", id, "storage_id", storageID)
			continue
		}
		if meta.Extension == "" {
			logger.WarnContext(ctx, "no extension found in metadata", "software_installer_id", id, "storage_id", storageID)
			continue
		}
		payload := fleet.UploadSoftwareInstallerPayload{
			PackageIDs: meta.PackageIDs,
			Extension:  meta.Extension,
		}
		payload.UninstallScript = file.GetUninstallScript(payload.Extension)

		// Update $PACKAGE_ID in uninstall script
		if err := preProcessUninstallScript(&payload); err != nil {
			return ctxerr.Wrap(ctx, err, "applying uninstall script template")
		}

		// Update the package_id and extension in the software installer and the uninstall script
		if err := ds.UpdateSoftwareInstallerWithoutPackageIDs(ctx, id, payload); err != nil {
			return ctxerr.Wrap(ctx, err, "updating package_id in software installer")
		}
	}

	return nil
}

func activitySoftwareLabelsFromValidatedLabels(validatedLabels *fleet.LabelIdentsWithScope) (includeAny, excludeAny, includeAll []fleet.ActivitySoftwareLabel) {
	if validatedLabels == nil || len(validatedLabels.ByName) == 0 {
		return nil, nil, nil
	}

	labels := make([]fleet.ActivitySoftwareLabel, 0, len(validatedLabels.ByName))
	for _, lbl := range validatedLabels.ByName {
		labels = append(labels, fleet.ActivitySoftwareLabel{
			ID:   lbl.LabelID,
			Name: lbl.LabelName,
		})
	}
	switch validatedLabels.LabelScope {
	case fleet.LabelScopeIncludeAny:
		includeAny = labels
	case fleet.LabelScopeExcludeAny:
		excludeAny = labels
	case fleet.LabelScopeIncludeAll:
		includeAll = labels
	}
	return includeAny, excludeAny, includeAll
}

func activitySoftwareLabelsFromSoftwareScopeLabels(includeAnyScopeLabels, excludeAnyScopeLabels, includeAllScopeLabels []fleet.SoftwareScopeLabel) (includeAny, excludeAny, includeAll []fleet.ActivitySoftwareLabel) {
	for _, label := range includeAnyScopeLabels {
		includeAny = append(includeAny, fleet.ActivitySoftwareLabel{
			ID:   label.LabelID,
			Name: label.LabelName,
		})
	}
	for _, label := range excludeAnyScopeLabels {
		excludeAny = append(excludeAny, fleet.ActivitySoftwareLabel{
			ID:   label.LabelID,
			Name: label.LabelName,
		})
	}
	for _, label := range includeAllScopeLabels {
		includeAll = append(includeAll, fleet.ActivitySoftwareLabel{
			ID:   label.LabelID,
			Name: label.LabelName,
		})
	}
	return includeAny, excludeAny, includeAll
}

// getInstallScript returns the install script for a software installer,
// using a special script for fleetd packages to handle macOS in-band upgrades.
func getInstallScript(extension string, packageIDs []string, currentScript string) string {
	if extension == "pkg" && file.IsFleetdPkg(packageIDs) {
		return file.InstallPkgFleetdScript
	}
	if currentScript != "" {
		return currentScript
	}
	return file.GetInstallScript(extension)
}

// batchAddSelfServiceCategories only adds categories, because it is used across both the installer and vpp
// endpoints and we cannot know what categories to delete before those are both done.
func (svc *Service) batchAddSelfServiceCategories(ctx context.Context, teamID *uint, categoryNames []string, dryRun bool) ([]string, error) {
	// Compare names with fleet.SoftwareCategoryNamesEqual rather than a plain
	// case-insensitive comparison: the software_categories unique index uses the
	// utf8mb4_unicode_ci collation, which ignores variation selectors, so two
	// names Go considers distinct (e.g. "🖥️ Productivity" with vs. without U+FE0F)
	// are the same row to MySQL. Deduping/matching on the DB's terms here avoids
	// attempting an insert that would fail with a 1062 duplicate-entry error.
	var allCategories []string
	for _, name := range fleet.TranslateLegacySoftwareCategoryNames(categoryNames) {
		if slices.ContainsFunc(allCategories, func(c string) bool { return fleet.SoftwareCategoryNamesEqual(c, name) }) {
			continue
		}
		allCategories = append(allCategories, name)
	}

	if len(allCategories) == 0 {
		return allCategories, nil
	}

	existingCategories, err := svc.ds.ListSoftwareCategories(ctxdb.RequirePrimary(ctx, true), ptr.ValOrZero(teamID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing existing software categories")
	}

	var categoriesToInsert []string
	for _, name := range allCategories {
		if !slices.ContainsFunc(existingCategories, func(c fleet.SoftwareCategory) bool { return fleet.SoftwareCategoryNamesEqual(c.Name, name) }) {
			categoriesToInsert = append(categoriesToInsert, name)
		}
	}

	if dryRun {
		return allCategories, nil
	}

	if err := svc.ds.BatchNewSoftwareCategories(ctx, ptr.ValOrZero(teamID), categoriesToInsert); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "creating self-service categories")
	}
	return allCategories, nil
}

func parsePinnedVersion(ctx context.Context, version string) (trimmedVersion string, usesCaret bool, err error) {
	trimmedVersion, usesCaret = strings.CutPrefix(version, "^")
	if usesCaret {
		if len(trimmedVersion) == 0 {
			return "", false, fleet.NewUserMessageError(errEmptyCaretVersion, http.StatusBadRequest)
		}
		if _, err := strconv.ParseUint(trimmedVersion, 10, 64); err != nil {
			return "", false, fleet.NewUserMessageError(errNonMajorVersion, http.StatusBadRequest)
		}
	}
	return trimmedVersion, usesCaret, nil
}

func versionMatchesMajor(version string, majorVersion string) bool {
	versionMajor, _, _ := strings.Cut(version, ".")
	return versionMajor == majorVersion
}
