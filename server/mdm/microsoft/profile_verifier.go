package microsoft_mdm

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// PreprocessWindowsProfileContentsForDeployment processes Windows configuration profiles to replace Fleet variables
// with their actual values for each host during profile deployment.
func PreprocessWindowsProfileContentsForDeployment(deps ProfilePreprocessDependencies, params ProfilePreprocessParams, profileContents string) (string, error) {
	return preprocessWindowsProfileContents(deps, params, profileContents)
}

// MicrosoftProfileProcessingError is used to indicate errors during Microsoft profile processing, such as variable replacement failures.
// It should not break the entire deployment flow, but rather be handled gracefully at the profile level, setting it to failed and detail = Error()
type MicrosoftProfileProcessingError struct {
	message string
}

func (e *MicrosoftProfileProcessingError) Error() string {
	return e.message
}

type ProfilePreprocessDependencies struct {
	Context                    context.Context
	Logger                     *slog.Logger
	DataStore                  fleet.Datastore
	HostIDForUUIDCache         map[string]uint
	AppConfig                  *fleet.AppConfig
	CustomSCEPCAs              map[string]*fleet.CustomSCEPProxyCA
	ManagedCertificatePayloads *[]*fleet.MDMManagedCertificate
}

type ProfilePreprocessParams struct {
	HostUUID    string
	ProfileUUID string
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
//
// If you need another dependency that should be reused across profiles, add it to a ProfilePreprocessDependencies
// implementation and to the interface if it's required for both verification and deployment. For new dependencies that
// vary profile-to-profile, add them to ProfilePreprocessParams.
func preprocessWindowsProfileContents(deps ProfilePreprocessDependencies, params ProfilePreprocessParams, profileContents string) (string, error) {
	// Check if Fleet variables are present
	fleetVars := variables.Find(profileContents)
	if len(fleetVars) == 0 {
		// No variables to replace, return original content
		return profileContents, nil
	}

	// Process each Fleet variable
	result := profileContents
	for _, fleetVar := range fleetVars {
		switch {
		case fleetVar == string(fleet.FleetVarHostUUID):
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostUUIDRegexp, result, params.HostUUID)
		case fleetVar == string(fleet.FleetVarHostPlatform):
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostPlatformRegexp, result, "windows")
		case fleetVar == string(fleet.FleetVarHostHardwareSerial):
			hostLite, _, err := profiles.HydrateHost(deps.Context, deps.DataStore, fleet.Host{UUID: params.HostUUID}, func(hostCount int) error {
				return &MicrosoftProfileProcessingError{message: fmt.Sprintf("Found %d hosts with UUID %s. Profile variable substitution for %s requires exactly one host", hostCount, params.HostUUID, fleet.FleetVarHostHardwareSerial.WithPrefix())}
			})
			if err != nil {
				return profileContents, err
			}
			if hostLite.HardwareSerial == "" {
				return profileContents, &MicrosoftProfileProcessingError{message: fmt.Sprintf("There is no serial number for this host. Fleet couldn't populate %s.", fleet.FleetVarHostHardwareSerial.WithPrefix())}
			}

			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostHardwareSerialRegexp, result, hostLite.HardwareSerial)
		case slices.Contains(fleet.IDPFleetVariables, fleet.FleetVarName(fleetVar)):
			replacedContents, replacedVariable, err := profiles.ReplaceHostEndUserIDPVariables(deps.Context, deps.DataStore, fleetVar, result, params.HostUUID, deps.HostIDForUUIDCache, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			if !replacedVariable {
				return profileContents, ctxerr.Wrap(deps.Context, err, "host end user IDP variable replacement failed for variable")
			}
			result = replacedContents
		}

		switch {
		case fleetVar == string(fleet.FleetVarSCEPWindowsCertificateID):
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarSCEPWindowsCertificateIDRegexp, result, params.ProfileUUID)
		case fleetVar == string(fleet.FleetVarSCEPRenewalID):
			result = profiles.ReplaceFleetVariableInXML(fleet.FleetVarSCEPRenewalIDRegexp, result, "fleet-"+params.ProfileUUID)
		case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)):
			caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix))
			err := profiles.IsCustomSCEPConfigured(deps.Context, deps.CustomSCEPCAs, caName, fleetVar, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			replacedContents, replacedVariable, err := profiles.ReplaceCustomSCEPChallengeVariable(deps.Context, deps.Logger, fleetVar, deps.CustomSCEPCAs, result)
			if err != nil {
				return profileContents, ctxerr.Wrap(deps.Context, err, "replacing custom SCEP challenge variable")
			}
			if !replacedVariable {
				return profileContents, &MicrosoftProfileProcessingError{message: fmt.Sprintf("Custom SCEP challenge variable replacement failed for variable %s", fleetVar)}
			}
			result = replacedContents
		case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
			caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
			err := profiles.IsCustomSCEPConfigured(deps.Context, deps.CustomSCEPCAs, caName, fleetVar, func(errMsg string) error {
				return &MicrosoftProfileProcessingError{message: errMsg}
			})
			if err != nil {
				return profileContents, err
			}
			replacedContents, managedCertificate, replacedVariable, err := profiles.ReplaceCustomSCEPProxyURLVariable(deps.Context, deps.Logger, deps.DataStore, deps.AppConfig, fleetVar, deps.CustomSCEPCAs, result, params.HostUUID, params.ProfileUUID)
			if err != nil {
				return profileContents, ctxerr.Wrap(deps.Context, err, "replacing custom SCEP challenge variable")
			}
			if !replacedVariable {
				return profileContents, &MicrosoftProfileProcessingError{message: fmt.Sprintf("Custom SCEP challenge variable replacement failed for variable %s", fleetVar)}
			}
			result = replacedContents

			*deps.ManagedCertificatePayloads = append(*deps.ManagedCertificatePayloads, managedCertificate)
		}

		// Add other Fleet variables here as they are implemented, identify if it can be skipped for verification.
	}

	return result, nil
}
