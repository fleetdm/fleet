package service

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/variables"
)

func (svc *Service) NewMDMWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, data []byte, labelsInclude []string, labelsMembershipMode fleet.MDMLabelsMode, labelsExcludeAny []string) (*fleet.MDMWindowsConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	cp, usesFleetVars, teamName, err := svc.parseAndValidateWindowsConfigProfile(ctx, teamID, profileName, data, labelsInclude, labelsMembershipMode, labelsExcludeAny)
	if err != nil {
		return nil, err
	}

	newCP, err := svc.ds.NewMDMWindowsConfigProfile(ctx, *cp, usesFleetVars)
	if err != nil {
		if _, ok := errors.AsType[endpointer.ExistsErrorInterface](err); ok {
			err = fleet.NewInvalidArgumentError("profile", SameProfileNameUploadErrorMsg).
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedWindowsProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: newCP.Name,
		}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm windows config profile")
	}

	return newCP, nil
}

// parseAndValidateWindowsConfigProfile runs the validation shared by the
// create and update paths. It returns the constructed profile (with labels
// set), the Fleet variable names it uses, and the team's name (empty string
// for no team).
func (svc *Service) parseAndValidateWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, data []byte, labelsInclude []string, labelsMembershipMode fleet.MDMLabelsMode, labelsExcludeAny []string) (*fleet.MDMWindowsConfigProfile, []fleet.FleetVarName, string, error) {
	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, nil, "", ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	lic, err := svc.License(ctx)
	if err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, err, "checking license")
	}

	var teamName string
	if teamID > 0 {
		if lic == nil || !lic.IsPremium() {
			return nil, nil, "", ctxerr.Wrap(ctx, fleet.ErrMissingLicense)
		}
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, nil, "", ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	if len(labelsInclude) > 0 || len(labelsExcludeAny) > 0 {
		if lic == nil || !lic.IsPremium() {
			return nil, nil, "", ctxerr.Wrap(ctx, fleet.NewLicenseErrorWithCause(fleet.ConfigProfileLabelScopingPremiumCauseMsg), "checking license for profile label scoping")
		}
	}

	cp := fleet.MDMWindowsConfigProfile{
		TeamID: &teamID,
		Name:   profileName,
		SyncML: data,
	}
	if err := cp.ValidateUserProvided(svc.config.MDM.IsCustomDiskEncryptionEnabled()); err != nil {
		msg := err.Error()
		if strings.Contains(msg, syncml.DiskEncryptionProfileRestrictionErrMsg) {
			return nil, nil, "", ctxerr.Wrap(ctx,
				&fleet.BadRequestError{Message: msg + " To control these settings use disk encryption endpoint."})
		}

		// this is not great, but since the validations are shared between the CLI
		// and the API, we must make some changes to error message here.
		if ix := strings.Index(msg, "To control these settings,"); ix >= 0 {
			msg = strings.TrimSpace(msg[:ix])
		}
		err := &fleet.BadRequestError{Message: "Couldn't add. " + msg}
		return nil, nil, "", ctxerr.Wrap(ctx, err, "validate profile")
	}

	if overlap := fleet.LabelOverlap(labelsInclude, labelsExcludeAny); overlap != "" {
		return nil, nil, "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("labels", fmt.Sprintf("label %q cannot appear in both include and exclude lists", overlap)))
	}
	includeLabels, excludeLabels, err := svc.validateProfileLabelSets(ctx, &teamID, labelsInclude, labelsExcludeAny)
	if err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, err, "validating labels")
	}
	switch labelsMembershipMode {
	case fleet.LabelsIncludeAny:
		cp.LabelsIncludeAny = includeLabels
	default:
		// default include all
		cp.LabelsIncludeAll = includeLabels
	}
	cp.LabelsExcludeAny = excludeLabels

	if err := fleet.ValidateEmbeddedSecretsAndCustomHostVitals(ctx, svc.ds, []string{string(cp.SyncML)}); err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", err.Error()))
	}

	groupedCAs, err := svc.ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	foundVars, err := validateWindowsProfileFleetVariables(string(cp.SyncML), lic, groupedCAs)
	if err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, err)
	}

	// Collect Fleet variables used in the profile
	var usesFleetVars []fleet.FleetVarName
	for _, varName := range foundVars {
		usesFleetVars = append(usesFleetVars, fleet.FleetVarName(varName))
	}

	if err := svc.handleWindowsProfileSoftwareUpdate(ctx, cp.SyncML, teamID); err != nil {
		return nil, nil, "", ctxerr.Wrap(ctx, err, "handling windows profile software update")
	}

	return &cp, usesFleetVars, teamName, nil
}

// updateMDMWindowsConfigProfile implements the Windows branch of
// UpdateMDMConfigProfile. A profile's name cannot change here: unlike Apple
// profiles there is no separate identifier, so name is a Windows profile's
// only identity (GitOps likewise treats a rename as delete-then-insert, not
// an edit).
func (svc *Service) updateMDMWindowsConfigProfile(ctx context.Context, profileUUID string, profile []byte, labelsInclude []string, labelsMembershipMode fleet.MDMLabelsMode, labelsExcludeAny []string) error {
	// first we perform a basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	existing, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	teamID, teamName, err := svc.resolveProfileTeam(ctx, existing.TeamID)
	if err != nil {
		return err
	}

	// now we can do a specific authz check based on team id of profile before we update it
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: existing.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// prevent editing profiles that are managed by Fleet
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[existing.Name]; ok {
		return &fleet.BadRequestError{
			Message:     "profiles managed by Fleet can't be edited using this endpoint.",
			InternalErr: fmt.Errorf("editing profile %s for team %s not allowed because it's managed by Fleet", existing.Name, teamName),
		}
	}

	var cp *fleet.MDMWindowsConfigProfile
	var usesFleetVars []fleet.FleetVarName
	if len(profile) > 0 {
		cp, usesFleetVars, _, err = svc.parseAndValidateWindowsConfigProfile(ctx, teamID, existing.Name, profile, labelsInclude, labelsMembershipMode, labelsExcludeAny)
		if err != nil {
			return err
		}
	} else {
		// no new content -- only labels are being changed.
		if err := svc.checkLabelsOnlyProfileUpdate(ctx, labelsInclude, labelsExcludeAny); err != nil {
			return err
		}
		includeLabels, excludeLabels, err := svc.validateProfileLabelSets(ctx, &teamID, labelsInclude, labelsExcludeAny)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "validating labels")
		}
		cp = &fleet.MDMWindowsConfigProfile{
			Name:   existing.Name,
			TeamID: existing.TeamID,
		}
		switch labelsMembershipMode {
		case fleet.LabelsIncludeAll:
			cp.LabelsIncludeAll = includeLabels
		case fleet.LabelsIncludeAny:
			cp.LabelsIncludeAny = includeLabels
		}
		cp.LabelsExcludeAny = excludeLabels
	}
	cp.ProfileUUID = profileUUID

	if _, err := svc.ds.UpdateMDMWindowsConfigProfile(ctx, *cp, usesFleetVars); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.NewActivity(
		ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedWindowsProfile{
			TeamID:      actTeamID,
			TeamName:    actTeamName,
			ProfileName: cp.Name,
		}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edit mdm windows config profile")
	}

	return nil
}

// handleWindowsProfileSoftwareUpdate validates the preconditions for an OS-update
// (software update) profile: premium license and OS updates not already configured
// via settings. The "already exists" check and tracking-table insert happen
// atomically in ds.NewMDMWindowsConfigProfile.
func (svc *Service) handleWindowsProfileSoftwareUpdate(
	ctx context.Context,
	syncML []byte,
	teamID uint,
) error {
	if !fleet.ProfileTargetsReservedLocURI(syncML, syncml.FleetOSUpdateTargetLocURI) {
		return nil
	}

	lic, _ := license.FromContext(ctx)
	if lic == nil || !lic.IsPremium() {
		return fleet.ErrMissingLicense
	}

	osUpdatesConfigured, err := isWindowsOSUpdatesConfigured(ctx, teamID, svc)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if Windows OS updates are configured")
	}
	if osUpdatesConfigured {
		return &fleet.BadRequestError{
			Message: fleet.OSUpdatesAlreadyConfiguredErrorMessage,
		}
	}

	return nil
}

func isWindowsOSUpdatesConfigured(ctx context.Context, teamID uint, svc *Service) (bool, error) {
	var windowsOSUpdates fleet.WindowsUpdates
	if teamID > 0 {
		teamConfig, err := svc.ds.TeamMDMConfig(ctx, teamID)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "getting team config")
		}
		windowsOSUpdates = teamConfig.WindowsUpdates
	} else {
		appConfig, err := svc.ds.AppConfig(ctx)
		if err != nil {
			return false, ctxerr.Wrap(ctx, err, "getting app config")
		}
		windowsOSUpdates = appConfig.MDM.WindowsUpdates
	}

	if windowsOSUpdates.Configured() {
		return true, nil
	}
	return false, nil
}

// fleetVarsSupportedInWindowsProfiles lists the Fleet variables that are
// supported in Windows configuration profiles.
// except prefix variables
var fleetVarsSupportedInWindowsProfiles = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarSCEPWindowsCertificateID,
	fleet.FleetVarSCEPRenewalID,
	fleet.FleetVarCertificateRenewalID,
	fleet.FleetVarHostEndUserIDPUsername,
	fleet.FleetVarHostEndUserIDPUsernameLocalPart,
	fleet.FleetVarHostEndUserIDPFullname,
	fleet.FleetVarHostEndUserIDPDepartment,
	fleet.FleetVarHostEndUserIDPGroups,
	fleet.FleetVarHostPlatform,
	fleet.FleetVarNDESSCEPChallenge,
	fleet.FleetVarNDESSCEPProxyURL,
}

// subjectNameHasRenewalIDMarker reports whether a SubjectName data string
// contains the renewal-ID variable in OU=. The legacy SCEP_RENEWAL_ID name
// is accepted alongside CERTIFICATE_RENEWAL_ID for back-compat.
func subjectNameHasRenewalIDMarker(data string) bool {
	for _, v := range []fleet.FleetVarName{fleet.FleetVarCertificateRenewalID, fleet.FleetVarSCEPRenewalID} {
		if strings.Contains(data, "OU="+v.WithPrefix()) || strings.Contains(data, "OU="+v.WithBraces()) {
			return true
		}
	}
	return false
}

func validateWindowsProfileFleetVariables(contents string, lic *fleet.LicenseInfo, groupedCAs *fleet.GroupedCertificateAuthorities) ([]string, error) {
	foundVars := variables.Find(contents)
	if len(foundVars) == 0 {
		return nil, nil
	}

	// Check for premium license if the profile contains Fleet variables
	if lic == nil || !lic.IsPremium() {
		return nil, fleet.ErrMissingLicense
	}

	// Check if all found variables are supported
	for _, varName := range foundVars {
		if !slices.Contains(fleetVarsSupportedInWindowsProfiles, fleet.FleetVarName(varName)) &&
			!strings.HasPrefix(varName, string(fleet.FleetVarCustomSCEPChallengePrefix)) &&
			!strings.HasPrefix(varName, string(fleet.FleetVarCustomSCEPProxyURLPrefix)) {
			return nil, fleet.NewInvalidArgumentError("profile", fmt.Sprintf("Fleet variable $FLEET_VAR_%s is not supported in Windows profiles.", varName))
		}
	}

	err := validateProfileCertificateAuthorityVariables(contents, lic, groupedCAs, nil, additionalCustomSCEPValidationForWindowsProfiles, additionalNDESValidationForWindowsProfiles, nil)
	if err != nil {
		return nil, err
	}

	// Do additional validation that both custom SCEP URL and challenge vars are provided and not using different CA names etc.

	return foundVars, nil
}

// collectAllSyncMLItems returns all CmdItems from a SyncMLCmd, including items from nested
// commands within Atomic blocks (ReplaceCommands, AddCommands, ExecCommands).
func collectAllSyncMLItems(cmd *fleet.SyncMLCmd) []fleet.CmdItem {
	items := cmd.Items
	for _, nested := range cmd.ReplaceCommands {
		items = append(items, nested.Items...)
	}
	for _, nested := range cmd.AddCommands {
		items = append(items, nested.Items...)
	}
	for _, nested := range cmd.ExecCommands {
		items = append(items, nested.Items...)
	}
	return items
}

// containsFleetVar checks if s contains the given Fleet variable in either $FLEET_VAR_ or ${FLEET_VAR_} form.
func containsFleetVar(s string, v fleet.FleetVarName) bool {
	return strings.Contains(s, v.WithPrefix()) || strings.Contains(s, v.WithBraces())
}

// isFleetVar checks if s is exactly the given Fleet variable in either $FLEET_VAR_ or ${FLEET_VAR_} form.
func isFleetVar(s string, v fleet.FleetVarName) bool {
	return s == v.WithPrefix() || s == v.WithBraces()
}

func additionalNDESValidationForWindowsProfiles(contents string, ndesVars *NDESVarsFound) error {
	if ndesVars == nil {
		return nil
	}

	var cmdMsg *fleet.SyncMLCmd
	dec := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace([]byte(contents))))
	for {
		if err := dec.Decode(&cmdMsg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("The payload isn't valid XML: %w", err)
		}
		if cmdMsg == nil {
			break
		}

		for _, cmd := range collectAllSyncMLItems(cmdMsg) {
			if cmd.Target == nil {
				continue
			}

			target := strings.TrimSpace(*cmd.Target)

			dataContent := ""
			if cmd.Data != nil {
				dataContent = cmd.Data.Content
			}

			isChallenge := strings.HasSuffix(target, "/Install/Challenge")
			isServerURL := strings.HasSuffix(target, "/Install/ServerURL")
			isSubjectName := strings.HasSuffix(target, "/Install/SubjectName")

			// Verify that each NDES variable appears ONLY in its expected field.
			// This prevents the one-time challenge or proxy URL from being placed in an unexpected field
			// where it could be exfiltrated, since variable replacement is a global string substitution.
			if !isChallenge && containsFleetVar(dataContent, fleet.FleetVarNDESSCEPChallenge) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf(
						"Variable %q must only be in the SCEP certificate's \"Challenge\" field.", fleet.FleetVarNDESSCEPChallenge.WithPrefix()),
				}
			}
			if !isServerURL && containsFleetVar(dataContent, fleet.FleetVarNDESSCEPProxyURL) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf(
						"Variable %q must only be in the SCEP certificate's \"ServerURL\" field.", fleet.FleetVarNDESSCEPProxyURL.WithPrefix()),
				}
			}

			// Variables must not appear in LocURI target paths.
			if containsFleetVar(target, fleet.FleetVarNDESSCEPChallenge) ||
				containsFleetVar(target, fleet.FleetVarNDESSCEPProxyURL) {
				return &fleet.BadRequestError{
					Message: "NDES Fleet variables must not appear in LocURI target paths.",
				}
			}

			// Verify the expected fields contain the correct variables.
			if isChallenge && !isFleetVar(dataContent, fleet.FleetVarNDESSCEPChallenge) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf(
						"Variable %q must be in the SCEP certificate's \"Challenge\" field.", fleet.FleetVarNDESSCEPChallenge.WithPrefix()),
				}
			}
			if isServerURL && !isFleetVar(dataContent, fleet.FleetVarNDESSCEPProxyURL) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf(
						"Variable %q must be in the SCEP certificate's \"ServerURL\" field.", fleet.FleetVarNDESSCEPProxyURL.WithPrefix()),
				}
			}
			if isSubjectName && !subjectNameHasRenewalIDMarker(dataContent) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf("SubjectName item must contain the %s variable in the OU field", fleet.FleetVarCertificateRenewalID.WithPrefix()),
				}
			}
		}
	}

	return nil
}

func additionalCustomSCEPValidationForWindowsProfiles(contents string, customSCEPVars *CustomSCEPVarsFound) error {
	if customSCEPVars == nil {
		return nil
	}

	var cmdMsg *fleet.SyncMLCmd
	dec := xml.NewDecoder(bytes.NewReader(bytes.TrimSpace([]byte(contents))))
	for {
		if err := dec.Decode(&cmdMsg); err != nil { // EOF is fine in this case
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("The payload isn't valid XML: %w", err)
		}
		if cmdMsg == nil {
			break
		}

		for _, cmd := range collectAllSyncMLItems(cmdMsg) {
			if cmd.Target == nil {
				continue
			}

			target := strings.TrimSpace(*cmd.Target)

			if strings.HasSuffix(target, "/Install/SubjectName") {
				// SubjectName item found, check that it contains the expected renewal ID variable
				if cmd.Data == nil {
					return errors.New("SubjectName item is missing data")
				}

				if !subjectNameHasRenewalIDMarker(cmd.Data.Content) {
					return fmt.Errorf("SubjectName item must contain the %s variable in the OU field", fleet.FleetVarCertificateRenewalID.WithPrefix())
				}
			}
		}
	}

	return nil
}
