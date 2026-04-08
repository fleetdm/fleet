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
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	"github.com/fleetdm/fleet/v4/server/variables"
)

func (svc *Service) NewMDMWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, data []byte, labels []string, labelsMembershipMode fleet.MDMLabelsMode) (*fleet.MDMWindowsConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	var teamName string
	if teamID > 0 {
		lic, _ := license.FromContext(ctx)
		if lic == nil || !lic.IsPremium() {
			return nil, ctxerr.Wrap(ctx, fleet.ErrMissingLicense)
		}
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	cp := fleet.MDMWindowsConfigProfile{
		TeamID: &teamID,
		Name:   profileName,
		SyncML: data,
	}
	if err := cp.ValidateUserProvided(svc.config.MDM.EnableCustomOSUpdatesAndFileVault); err != nil {
		msg := err.Error()
		if strings.Contains(msg, syncml.DiskEncryptionProfileRestrictionErrMsg) {
			return nil, ctxerr.Wrap(ctx,
				&fleet.BadRequestError{Message: msg + " To control these settings use disk encryption endpoint."})
		}

		// this is not great, but since the validations are shared between the CLI
		// and the API, we must make some changes to error message here.
		if ix := strings.Index(msg, "To control these settings,"); ix >= 0 {
			msg = strings.TrimSpace(msg[:ix])
		}
		err := &fleet.BadRequestError{Message: "Couldn't add. " + msg}
		return nil, ctxerr.Wrap(ctx, err, "validate profile")
	}

	labelMap, err := svc.validateProfileLabels(ctx, &teamID, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating labels")
	}
	switch labelsMembershipMode {
	case fleet.LabelsIncludeAny:
		cp.LabelsIncludeAny = labelMap
	case fleet.LabelsExcludeAny:
		cp.LabelsExcludeAny = labelMap
	default:
		// default include all
		cp.LabelsIncludeAll = labelMap
	}

	if err := svc.ds.ValidateEmbeddedSecrets(ctx, []string{string(cp.SyncML)}); err != nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("profile", err.Error()))
	}

	// Get license for validation
	lic, err := svc.License(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "checking license")
	}

	groupedCAs, err := svc.ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	foundVars, err := validateWindowsProfileFleetVariables(string(cp.SyncML), lic, groupedCAs)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// Collect Fleet variables used in the profile
	var usesFleetVars []fleet.FleetVarName
	for _, varName := range foundVars {
		usesFleetVars = append(usesFleetVars, fleet.FleetVarName(varName))
	}

	newCP, err := svc.ds.NewMDMWindowsConfigProfile(ctx, cp, usesFleetVars)
	if err != nil {
		var existsErr endpointer.ExistsErrorInterface
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("profile", SameProfileNameUploadErrorMsg).
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}

	if _, err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newCP.ProfileUUID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
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

// fleetVarsSupportedInWindowsProfiles lists the Fleet variables that are
// supported in Windows configuration profiles.
// except prefix variables
var fleetVarsSupportedInWindowsProfiles = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarSCEPWindowsCertificateID,
	fleet.FleetVarSCEPRenewalID,
	fleet.FleetVarHostEndUserIDPUsername,
	fleet.FleetVarHostEndUserIDPUsernameLocalPart,
	fleet.FleetVarHostEndUserIDPFullname,
	fleet.FleetVarHostEndUserIDPDepartment,
	fleet.FleetVarHostEndUserIDPGroups,
	fleet.FleetVarHostPlatform,
	fleet.FleetVarNDESSCEPChallenge,
	fleet.FleetVarNDESSCEPProxyURL,
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

			dataContent := ""
			if cmd.Data != nil {
				dataContent = cmd.Data.Content
			}

			isChallenge := strings.HasSuffix(*cmd.Target, "/Install/Challenge")
			isServerURL := strings.HasSuffix(*cmd.Target, "/Install/ServerURL")
			isSubjectName := strings.HasSuffix(*cmd.Target, "/Install/SubjectName")

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
			if containsFleetVar(*cmd.Target, fleet.FleetVarNDESSCEPChallenge) ||
				containsFleetVar(*cmd.Target, fleet.FleetVarNDESSCEPProxyURL) {
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
			if isSubjectName &&
				!strings.Contains(dataContent, "OU="+fleet.FleetVarSCEPRenewalID.WithPrefix()) &&
				!strings.Contains(dataContent, "OU="+fleet.FleetVarSCEPRenewalID.WithBraces()) {
				return &fleet.BadRequestError{
					Message: fmt.Sprintf("SubjectName item must contain the %s variable in the OU field", fleet.FleetVarSCEPRenewalID.WithPrefix()),
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

			if strings.HasSuffix(*cmd.Target, "/Install/SubjectName") {
				// SubjectName item found, check that it contains the expected renewal ID variable
				if cmd.Data == nil {
					return errors.New("SubjectName item is missing data")
				}

				if !strings.Contains(cmd.Data.Content, "OU="+fleet.FleetVarSCEPRenewalID.WithPrefix()) && !strings.Contains(cmd.Data.Content, "OU="+fleet.FleetVarSCEPRenewalID.WithBraces()) {
					// Does not contain the renewal ID in any of it's two fleet var forms as the OU field
					return fmt.Errorf("SubjectName item must contain the %s variable in the OU field", fleet.FleetVarSCEPRenewalID.WithPrefix())
				}
			}
		}
	}

	return nil
}
