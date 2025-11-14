package microsoft_mdm

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"hash/fnv"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/admx"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/wlanxml"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/variables"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

// LoopOverExpectedHostProfiles loops all the <LocURI> values on all the profiles for a
// given host. It provides to the callback function:
//
// - An `ExpectedMDMProfile` that references the profile owning the LocURI
// - A hash that's unique for each profile/uri combination
// - The LocURI string
// - The data (if any) of the first <Item> element of the current LocURI
func LoopOverExpectedHostProfiles(
	ctx context.Context,
	logger kitlog.Logger,
	ds fleet.Datastore,
	host *fleet.Host,
	fn func(profile *fleet.ExpectedMDMProfile, hash, locURI, data string),
) error {
	profileMap, err := ds.GetHostMDMProfilesExpectedForVerification(ctx, host)
	if err != nil {
		return fmt.Errorf("getting host profiles for verification: %w", err)
	}

	params := PreprocessingParameters{
		HostIDForUUIDCache: make(map[string]uint),
	}

	for _, expectedProf := range profileMap {
		expanded, err := ds.ExpandEmbeddedSecrets(ctx, string(expectedProf.RawProfile))
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "expanding embedded secrets for profile %s", expectedProf.Name)
		}

		// Process Fleet variables if present (similar to how it's done during profile deployment)
		// This ensures we compare what was actually sent to the device
		processedContent := PreprocessWindowsProfileContentsForVerification(ctx, logger, ds, host.UUID, expectedProf.ProfileUUID, expanded, params)
		expectedProf.RawProfile = []byte(processedContent)

		var prof fleet.SyncMLCmd
		wrappedBytes := fmt.Sprintf("<Atomic>%s</Atomic>", expectedProf.RawProfile)
		if err := xml.Unmarshal([]byte(wrappedBytes), &prof); err != nil {
			return fmt.Errorf("unmarshalling profile %s: %w", expectedProf.Name, err)
		}
		for _, rc := range prof.ReplaceCommands {
			locURI := rc.GetTargetURI()
			data := rc.GetNormalizedTargetDataForVerification()
			ref := HashLocURI(expectedProf.Name, locURI)
			fn(expectedProf, ref, locURI, data)
		}
		for _, ac := range prof.AddCommands {
			locURI := ac.GetTargetURI()
			data := ac.GetNormalizedTargetDataForVerification()
			ref := HashLocURI(expectedProf.Name, locURI)
			fn(expectedProf, ref, locURI, data)
		}
	}

	return nil
}

// HashLocURI creates a unique, consistent hash for a given profileName +
// locURI combination.
//
// FIXME: the mdm_bridge table decodes CmdID as `int`,
// so we encode the reference as an int32.
func HashLocURI(profileName, locURI string) string {
	hash := fnv.New32a()
	hash.Write([]byte(profileName + locURI))
	return fmt.Sprint(hash.Sum32())
}

// VerifyHostMDMProfiles performs the verification of the MDM profiles installed on a host and
// updates the verification status in the datastore. It is intended to be called by Fleet osquery
// service when the Fleet server ingests host details.
func VerifyHostMDMProfiles(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore, host *fleet.Host,
	rawProfileResultsSyncML []byte,
) error {
	profileResults, err := transformProfileResults(rawProfileResultsSyncML)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "transforming policy results")
	}

	existingProfiles, err := ds.GetHostMDMWindowsProfiles(ctx, host.UUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting existing windows host profiles")
	}

	verified, missing, err := compareResultsToExpectedProfiles(ctx, logger, ds, host, profileResults, existingProfiles)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "comparing results to expected profiles")
	}

	toFail, toRetry, err := splitMissingProfilesIntoFailAndRetryBuckets(ctx, ds, host, missing, verified)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "splitting missing profiles into fail and retry buckets")
	}

	err = ds.UpdateHostMDMProfilesVerification(ctx, host, slices.Collect(maps.Keys(verified)), toFail, toRetry)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "updating host mdm profiles during verification")
	}
	return nil
}

func splitMissingProfilesIntoFailAndRetryBuckets(ctx context.Context, ds fleet.ProfileVerificationStore, host *fleet.Host,
	missing map[string]struct{},
	verified map[string]struct{},
) ([]string, []string, error) {
	toFail := make([]string, 0, len(missing))
	toRetry := make([]string, 0, len(missing))
	if len(missing) > 0 {
		counts, err := ds.GetHostMDMProfilesRetryCounts(ctx, host)
		if err != nil {
			return nil, nil, fmt.Errorf("getting host profiles retry counts: %w", err)
		}
		retriesByProfileUUID := make(map[string]uint, len(counts))
		for _, r := range counts {
			retriesByProfileUUID[r.ProfileName] = r.Retries
		}
		for key := range missing {
			// if the profile is in missing, we failed to validate at
			// least one LocURI, delete it from the verified map
			delete(verified, key)
			if retriesByProfileUUID[key] < mdm.MaxProfileRetries {
				// if we haven't hit the max retries, we set
				// the host profile status to nil (which causes
				// an install profile command to be enqueued
				// the next time the profile manager cron runs)
				// and increment the retry count
				toRetry = append(toRetry, key)
				continue
			}
			// otherwise we set the host profile status to failed
			toFail = append(toFail, key)
		}
	}
	return toFail, toRetry, nil
}

func compareResultsToExpectedProfiles(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore, host *fleet.Host,
	profileResults profileResultsTransform, existingProfiles []fleet.HostMDMWindowsProfile,
) (verified map[string]struct{}, missing map[string]struct{}, err error) {
	missing = map[string]struct{}{}
	verified = map[string]struct{}{}

	// Map existing profiles for this host by UUID for easier lookup for certain edge cases
	windowsProfilesByID := make(map[string]fleet.HostMDMWindowsProfile, len(existingProfiles))
	for _, existingProfile := range existingProfiles {
		windowsProfilesByID[existingProfile.ProfileUUID] = existingProfile
	}

	err = LoopOverExpectedHostProfiles(ctx, logger, ds, host, func(profile *fleet.ExpectedMDMProfile, ref, locURI, wantData string) {
		if strings.HasPrefix(strings.TrimSpace(locURI), "./Device/Vendor/MSFT/ClientCertificateInstall/SCEP") {
			verified[profile.Name] = struct{}{}
			// We delete here if by some accident it was marked as missing before
			delete(missing, profile.Name)
			return
		}

		// if we didn't get a status for a LocURI, mark the profile as missing.
		gotStatus, ok := profileResults.cmdRefToStatus[ref]
		if !ok {
			missing[profile.Name] = struct{}{}
			return
		}
		// it's okay if we didn't get a result
		gotResults := profileResults.cmdRefToResult[ref]
		// non-200 status don't have results. Consider it failed
		// unless it falls into a special case we know about.
		// TODO: There are likely more to be added
		var equal bool
		switch {
		case !strings.HasPrefix(gotStatus, "2"):
			equal = false
			// For unknown reasons these always return a 404 so mark as equal in that case if
			// the profile is verifying(meaning MDM protocol returned a good status) or verified
			if gotStatus == "404" && (IsADMXInstallConfigOperationCSP(locURI) || IsWin32OrDesktopBridgeADMXCSP(locURI)) {
				if existingProfile, ok := windowsProfilesByID[profile.ProfileUUID]; ok && existingProfile.Status != nil &&
					(*existingProfile.Status == fleet.MDMDeliveryVerified || *existingProfile.Status == fleet.MDMDeliveryVerifying) {
					level.Debug(logger).Log("msg", "ADMX policy install operation or Win32/Desktop Bridge ADMX policy returned 404, marking as verified", "profile_uuid", profile.ProfileUUID, "host_id", host.ID, "locuri", locURI)
					equal = true
				}
			}
		case wantData == gotResults:
			equal = true
		case wlanxml.IsWLANXML(wantData):
			equal, err = wlanxml.Equal(wantData, gotResults)
			if err != nil {
				err = fmt.Errorf("comparing WLAN XML profiles: %w", err)
				return
			}
		case admx.IsADMX(wantData):
			equal, err = admx.Equal(wantData, gotResults)
			if err != nil {
				err = fmt.Errorf("comparing ADMX policies: %w", err)
				return
			}
		}
		if !equal {
			level.Debug(logger).Log("msg", "Windows profile verification failed", "profile", profile.Name, "host_id", host.ID)
			withinGracePeriod := profile.IsWithinGracePeriod(host.DetailUpdatedAt)
			if !withinGracePeriod {
				missing[profile.Name] = struct{}{}
			}
			return
		}
		verified[profile.Name] = struct{}{}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("looping host mdm LocURIs: %w", err)
	}
	return verified, missing, nil
}

type profileResultsTransform struct {
	cmdRefToStatus map[string]string
	cmdRefToResult map[string]string
}

func transformProfileResults(rawProfileResultsSyncML []byte) (profileResultsTransform, error) {
	var syncML fleet.SyncML
	decoder := xml.NewDecoder(bytes.NewReader(rawProfileResultsSyncML))
	// the DLL used by the `mdm_bridge` extension sends the response with
	// <?xml version="1.0" encoding="utf-16"?>, however if you use
	// `charset.NewReaderLabel` it fails to unmarshal (!?) for now, I'm
	// relying on this hack.
	decoder.CharsetReader = func(encoding string, input io.Reader) (io.Reader, error) {
		return input, nil
	}

	if err := decoder.Decode(&syncML); err != nil {
		return profileResultsTransform{}, fmt.Errorf("decoding provided syncML: %w", err)
	}

	// TODO: what if more than one profile has the same
	// target uri but a different value? (product question)
	transform := profileResultsTransform{
		cmdRefToStatus: map[string]string{},
		cmdRefToResult: map[string]string{},
	}
	for _, r := range syncML.GetOrderedCmds() {
		if r.Cmd.CmdRef == nil {
			continue
		}
		ref := *r.Cmd.CmdRef
		if r.Verb == fleet.CmdStatus && r.Cmd.Data != nil {
			transform.cmdRefToStatus[ref] = *r.Cmd.Data
		}

		if r.Verb == fleet.CmdResults {
			transform.cmdRefToResult[ref] = r.Cmd.GetTargetData()
		}
	}
	return transform, nil
}

// These two methods are for detection of ADMX ingestion and Win32/Desktop Bridge ADMX policies.
// Documentation here: https://learn.microsoft.com/en-us/windows/client-management/win32-and-centennial-app-policy-configuration
// For reasons not entirely clear, attempting to use the Get verb to fetch the results of either the
// ADMXInstall operatiion or the config then installed against it will return a 404 so for now the best
// we can do is detect them and mark them as verified.
func IsADMXInstallConfigOperationCSP(locURI string) bool {
	normalizedLocURI := strings.ToLower(locURI)
	return strings.HasPrefix(normalizedLocURI, "./vendor/msft/policy/configoperations/admxinstall/") || strings.HasPrefix(normalizedLocURI, "./device/vendor/msft/policy/configoperations/admxinstall")
}

func IsWin32OrDesktopBridgeADMXCSP(locURI string) bool {
	normalizedLocURI := strings.ToLower(locURI)
	if strings.HasPrefix(normalizedLocURI, "./vendor/msft/policy/config/") || strings.HasPrefix(normalizedLocURI, "./user/vendor/msft/policy/config/") || strings.HasPrefix(normalizedLocURI, "./device/vendor/msft/policy/config/") {
		return strings.Contains(normalizedLocURI, "~")
	}
	return false
}

// PreprocessingParameters holds parameters needed for preprocessing Windows profiles, for both verification and deployment only.
// It should only contain helper stuff, and not core values such as hostUUID, profileUUID, etc.
type PreprocessingParameters struct {
	// a lookup map to avoid repeated datastore calls for hostID from hostUUID
	HostIDForUUIDCache map[string]uint
}

// PreprocessWindowsProfileContentsForVerification processes Windows configuration profiles to replace Fleet variables
// with the given host UUID for verification purposes.
//
// This function is similar to PreprocessWindowsProfileContentsForDeployment, but it does not require
// a datastore or logger since it only replaces certain fleet variables to avoid datastore unnecessary work.
func PreprocessWindowsProfileContentsForVerification(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore, hostUUID string, profileUUID string, profileContents string, params PreprocessingParameters) string {
	replacedContents, _ := preprocessWindowsProfileContents(ctx, logger, ds, nil, true, hostUUID, profileUUID, nil, profileContents, nil, params)
	// ^ We ignore the error here, and rely on the fact that the function will return the original contents if no replacements were made.
	// So verification fails on individual profile level, instead of entire verification failing.
	return replacedContents
}

// PreprocessWindowsProfileContentsForDeployment processes Windows configuration profiles to replace Fleet variables
// with their actual values for each host during profile deployment.
func PreprocessWindowsProfileContentsForDeployment(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore,
	appConfig *fleet.AppConfig, hostCmdUUID string, profileUUID string,
	groupedCAs *fleet.GroupedCertificateAuthorities, profileContents string,
	managedCertificatePayloads *[]*fleet.MDMManagedCertificate,
	params PreprocessingParameters,
) (string, error) {
	// TODO: Should we avoid iterating this list for every profile?
	customSCEPCAs := make(map[string]*fleet.CustomSCEPProxyCA, len(groupedCAs.CustomScepProxy))
	for _, ca := range groupedCAs.CustomScepProxy {
		customSCEPCAs[ca.Name] = &ca
	}

	return preprocessWindowsProfileContents(ctx, logger, ds, appConfig, false, hostCmdUUID, profileUUID, customSCEPCAs, profileContents, managedCertificatePayloads, params)
}

// This error type is used to indicate errors during Microsoft profile processing, such as variable replacement failures.
// It should not break the entire deployment flow, but rather be handled gracefully at the profile level, setting it to failed and detail = Error()
type MicrosoftProfileProcessingError struct {
	message string
}

func (e *MicrosoftProfileProcessingError) Error() string {
	return e.message
}

// preprocessWindowsProfileContents processes Windows configuration profiles to replace Fleet variables
// with their actual values for each host. This function is used both during profile deployment
// and during profile verification to ensure consistency.
//
// The function handles XML escaping to prevent injection attacks.
//
// Currently supported variables:
//   - $FLEET_VAR_HOST_UUID or ${FLEET_VAR_HOST_UUID}: Replaced with the host's UUID
//   - $FLEET_VAR_HOST_END_USER_EMAIL_IDP or ${FLEET_VAR_HOST_END_USER_EMAIL_IDP}: Replaced with the host's end user email from the IDP
//   - $FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID or ${FLEET_VAR_SCEP_WINDOWS_CERTIFICATE_ID}: Replaced with the profile UUID for SCEP certificate
//   - $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME> or ${FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>}: Replaced with the challenge for the specified custom SCEP CA
//   - $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME> or ${FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>}: Replaced with the proxy URL for the specified custom SCEP CA
//
// Why we don't use Go templates here:
//  1. Error handling: Go templates don't provide fine-grained error handling for individual variable
//     replacements. We need to handle failures per-host and per-variable gracefully.
//  2. Variable dependencies: Some variables may be related or have dependencies on each other. With
//     manual processing, we can control the order of variable replacement precisely.
//  3. Performance: Templates must be compiled every time they're used, adding overhead when processing
//     thousands of host profiles. Direct string replacement is more efficient for our use case.
//  4. XML escaping: We need XML-specific escaping for values, which is simpler to control with direct
//     string replacement rather than template functions.
func preprocessWindowsProfileContents(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore, appConfig *fleet.AppConfig,
	isVerifying bool, hostUUID string, profileUUID string,
	customSCEPCAs map[string]*fleet.CustomSCEPProxyCA, profileContents string,
	managedCertificatePayloads *[]*fleet.MDMManagedCertificate,
	params PreprocessingParameters,
) (string, error) {
	// Check if Fleet variables are present
	fleetVars := variables.Find(profileContents)
	if len(fleetVars) == 0 {
		// No variables to replace, return original content
		return profileContents, nil
	}

	// Process each Fleet variable
	result := profileContents
	for _, fleetVar := range fleetVars {
		if fleetVar == string(fleet.FleetVarHostUUID) {
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostUUIDRegexp, result, hostUUID)
		} else if slices.Contains(fleet.IDPFleetVariables, fleet.FleetVarName(fleetVar)) {
			replacedContents, replacedVariable, err := profiles.ReplaceHostEndUserIDPVariables(ctx, ds, fleetVar, result, hostUUID, params.HostIDForUUIDCache, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			if !replacedVariable {
				return profileContents, ctxerr.Wrap(ctx, err, "host end user IDP variable replacement failed for variable")
			}
			result = replacedContents
		}

		// We skip some variables during verification, to avoid unnecessary datastore calls
		// or processing that is not needed for verification.
		if isVerifying {
			continue
		}

		switch {
		case fleetVar == string(fleet.FleetVarSCEPWindowsCertificateID):
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarSCEPWindowsCertificateIDRegexp, result, profileUUID)
		case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)):
			caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix))
			err := profiles.IsCustomSCEPConfigured(ctx, customSCEPCAs, caName, fleetVar, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			replacedContents, replacedVariable, err := profiles.ReplaceCustomSCEPChallengeVariable(ctx, logger, fleetVar, customSCEPCAs, result)
			if err != nil {
				return profileContents, ctxerr.Wrap(ctx, err, "replacing custom SCEP challenge variable")
			}
			if !replacedVariable {
				return profileContents, &MicrosoftProfileProcessingError{message: fmt.Sprintf("Custom SCEP challenge variable replacement failed for variable %s", fleetVar)}
			}
			result = replacedContents
		case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
			caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
			err := profiles.IsCustomSCEPConfigured(ctx, customSCEPCAs, caName, fleetVar, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			replacedContents, managedCertificate, replacedVariable, err := profiles.ReplaceCustomSCEPProxyURLVariable(ctx, logger, ds, appConfig, fleetVar, customSCEPCAs, result, hostUUID, profileUUID)
			if err != nil {
				return profileContents, ctxerr.Wrap(ctx, err, "replacing custom SCEP challenge variable")
			}
			if !replacedVariable {
				return profileContents, &MicrosoftProfileProcessingError{message: fmt.Sprintf("Custom SCEP challenge variable replacement failed for variable %s", fleetVar)}
			}
			result = replacedContents

			*managedCertificatePayloads = append(*managedCertificatePayloads, managedCertificate)
		}

		// Add other Fleet variables here as they are implemented, identify if it can be skipped for verification.
	}

	return result, nil
}
