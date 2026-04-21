package apple_mdm

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/profiles"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/variables"
	"github.com/google/uuid"
)

// LEGACY VARIABLE
var fleetVarHostEndUserEmailIDPRegexp = regexp.MustCompile(fmt.Sprintf(`(\$FLEET_VAR_%s)|(\${FLEET_VAR_%[1]s})`, fleet.FleetVarHostEndUserEmailIDP))

// EnqueueResult holds the results of profile enqueue operations.
type EnqueueResult struct {
	// FailedCmdUUIDs maps command UUIDs that failed to enqueue to their errors.
	FailedCmdUUIDs map[string]error
	// SucceededCmdUUIDs contains the command UUIDs that were enqueued successfully.
	SucceededCmdUUIDs []string
}

func ProcessAndEnqueueProfiles(ctx context.Context,
	ds fleet.Datastore,
	logger *slog.Logger,
	appConfig *fleet.AppConfig,
	commander *MDMAppleCommander,
	installTargets, removeTargets map[string]*fleet.CmdTarget,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	profileContents map[string]mobileconfig.Mobileconfig,
) (*EnqueueResult, error) {
	// Grab the contents of all the profiles we need to install, if not already provided.
	if profileContents == nil {
		profileUUIDs := make([]string, 0, len(installTargets))
		for pUUID := range installTargets {
			profileUUIDs = append(profileUUIDs, pUUID)
		}

		var err error
		profileContents, err = ds.GetMDMAppleProfilesContents(ctx, profileUUIDs)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get profile contents")
		}
	}

	groupedCAs, err := ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	// Insert variables into profile contents of install targets. Variables may be host-specific.
	err = preprocessProfileContents(ctx, appConfig, ds,
		scep.NewSCEPConfigService(logger, nil),
		digicert.NewService(digicert.WithLogger(logger)),
		logger, installTargets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	if err != nil {
		return nil, err
	}

	// Find the profiles containing secret variables.
	profilesWithSecrets, err := fleet.FindProfilesWithSecrets(ctx, logger, installTargets, profileContents)
	if err != nil {
		return nil, err
	}

	type remoteResult struct {
		Err     error
		CmdUUID string
	}

	// Send the install/remove commands for each profile.
	var wgProd, wgCons sync.WaitGroup
	ch := make(chan remoteResult)

	execCmd := func(profUUID string, target *fleet.CmdTarget, op fleet.MDMOperationType) {
		defer wgProd.Done()

		var err error
		switch op {
		case fleet.MDMOperationTypeInstall:
			if _, ok := profilesWithSecrets[profUUID]; ok {
				err = commander.EnqueueCommandInstallProfileWithSecrets(ctx, target.EnrollmentIDs, profileContents[profUUID], target.CmdUUID, target.ProfileName)
			} else {
				err = commander.InstallProfile(ctx, target.EnrollmentIDs, profileContents[profUUID], target.CmdUUID, target.ProfileName)
			}
		case fleet.MDMOperationTypeRemove:
			err = commander.RemoveProfile(ctx, target.EnrollmentIDs, target.ProfileIdentifier, target.CmdUUID, target.ProfileName)
		}

		// Determine whether the command was enqueued (even if push notification failed).
		var e *APNSDeliveryError
		switch {
		case errors.As(err, &e):
			logger.DebugContext(ctx, "failed sending push notifications, profiles still enqueued", "details", err)
			ch <- remoteResult{nil, target.CmdUUID}
			// this is fine to pass as success here, since we have sent the command but just didn't notify the client, but when the client checks back in it will process this profile.
		case err != nil:
			logger.ErrorContext(ctx, fmt.Sprintf("enqueue command to %s profiles", op), "details", err)
			ch <- remoteResult{err, target.CmdUUID}
		default:
			ch <- remoteResult{nil, target.CmdUUID}
		}

	}
	for profUUID, target := range installTargets {
		wgProd.Add(1)
		go execCmd(profUUID, target, fleet.MDMOperationTypeInstall)
	}
	for profUUID, target := range removeTargets {
		wgProd.Add(1)
		go execCmd(profUUID, target, fleet.MDMOperationTypeRemove)
	}

	result := &EnqueueResult{
		FailedCmdUUIDs:    make(map[string]error),
		SucceededCmdUUIDs: []string{},
	}

	wgCons.Go(func() {
		for resp := range ch {
			if resp.Err == nil {
				result.SucceededCmdUUIDs = append(result.SucceededCmdUUIDs, resp.CmdUUID)
			} else {
				result.FailedCmdUUIDs[resp.CmdUUID] = resp.Err
			}
		}
	})

	wgProd.Wait()
	close(ch) // done sending at this point, this triggers end of for loop in consumer
	wgCons.Wait()
	return result, nil
}

func preprocessProfileContents(
	ctx context.Context,
	appConfig *fleet.AppConfig,
	ds fleet.Datastore,
	scepConfig fleet.SCEPConfigService,
	digiCertService fleet.DigiCertService,
	logger *slog.Logger,
	targets map[string]*fleet.CmdTarget,
	profileContents map[string]mobileconfig.Mobileconfig,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	groupedCAs *fleet.GroupedCertificateAuthorities,
) error {
	// This method replaces Fleet variables ($FLEET_VAR_<NAME>) in the profile
	// contents, generating a unique profile for each host. For a 2KB profile and
	// 30K hosts, this method may generate ~60MB of profile data in memory.

	var (
		// Copy of NDES SCEP config which will contain unencrypted password, if needed
		ndesConfig    *fleet.NDESSCEPProxyCA
		digiCertCAs   map[string]*fleet.DigiCertCA
		customSCEPCAs map[string]*fleet.CustomSCEPProxyCA
		smallstepCAs  map[string]*fleet.SmallstepSCEPProxyCA
	)

	// this is used to cache the host ID corresponding to the UUID, so we don't
	// need to look it up more than once per host.
	hostIDForUUIDCache := make(map[string]uint)

	var addedTargets map[string]*fleet.CmdTarget
	for profUUID, target := range targets {
		contents, ok := profileContents[profUUID]
		if !ok {
			// This should never happen
			continue
		}

		// Check if Fleet variables are present.
		contentsStr := string(contents)
		fleetVars := variables.Find(contentsStr)
		if len(fleetVars) == 0 {
			continue
		}

		var variablesUpdatedAt *time.Time

		// Do common validation that applies to all hosts in the target
		valid := true
		// Check if there are any CA variables first so that if a non-CA variable causes
		// preprocessing to fail, we still set the variablesUpdatedAt timestamp so that
		// validation works as expected
		// In the future we should expand variablesUpdatedAt logic to include non-CA variables as
		// well
		for _, fleetVar := range fleetVars {
			if fleetVar == string(fleet.FleetVarSCEPRenewalID) ||
				fleetVar == string(fleet.FleetVarNDESSCEPChallenge) || fleetVar == string(fleet.FleetVarNDESSCEPProxyURL) || fleetVar == string(fleet.FleetVarHostUUID) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)) {
				// Give a few minutes leeway to account for clock skew
				variablesUpdatedAt = ptr.Time(time.Now().UTC().Add(-3 * time.Minute))
				break
			}
		}

	initialFleetVarLoop:
		for _, fleetVar := range fleetVars {
			switch {
			case fleetVar == string(fleet.FleetVarNDESSCEPChallenge) || fleetVar == string(fleet.FleetVarNDESSCEPProxyURL):
				configured, err := isNDESSCEPConfigured(ctx, logger, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, target)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking NDES SCEP configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			case fleetVar == string(fleet.FleetVarHostEndUserEmailIDP) || fleetVar == string(fleet.FleetVarHostHardwareSerial) || fleetVar == string(fleet.FleetVarHostPlatform) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPUsername) || fleetVar == string(fleet.FleetVarHostEndUserIDPUsernameLocalPart) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) || fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) || fleetVar == string(fleet.FleetVarSCEPRenewalID) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPFullname) || fleetVar == string(fleet.FleetVarHostUUID):
				// No extra validation needed for these variables

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)):
				caName, found := strings.CutPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix))
				if !found {
					caName, _ = strings.CutPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix))
				}

				if digiCertCAs == nil {
					digiCertCAs = make(map[string]*fleet.DigiCertCA)
				}
				configured, err := isDigiCertConfigured(ctx, logger, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, digiCertCAs, profUUID, target, caName, fleetVar)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking DigiCert configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
				caName, found := strings.CutPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix))
				if !found {
					caName, _ = strings.CutPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
				}

				if customSCEPCAs == nil {
					customSCEPCAs = make(map[string]*fleet.CustomSCEPProxyCA)
					if groupedCAs != nil {
						for _, ca := range groupedCAs.CustomScepProxy {
							customSCEPCAs[ca.Name] = &ca
						}
					}
				}
				err := profiles.IsCustomSCEPConfigured(ctx, customSCEPCAs, caName, fleetVar, func(errMsg string) error {
					_, err := fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, errMsg, ptr.Time(time.Now().UTC()))
					return err
				})
				if err != nil {
					valid = false
					break initialFleetVarLoop
				}

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)):
				if smallstepCAs == nil {
					smallstepCAs = make(map[string]*fleet.SmallstepSCEPProxyCA)
				}
				caName, found := strings.CutPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix))
				if !found {
					caName, _ = strings.CutPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix))
				}

				configured, err := isSmallstepSCEPConfigured(ctx, logger, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, smallstepCAs, profUUID, target, caName,
					fleetVar)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking Smallstep SCEP configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			default:
				// Otherwise, error out since this variable is unknown
				detail := fmt.Sprintf("Unknown Fleet variable $FLEET_VAR_%s found in profile. Please update or remove.",
					fleetVar)
				_, err := fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, detail, variablesUpdatedAt)
				if err != nil {
					return err
				}
				valid = false
			}
		}
		if !valid {
			// We marked the profile as failed, so we will not do any additional processing on it
			delete(targets, profUUID)
			continue
		}

		// Currently, all supported Fleet variables are unique per host, so we split the profile into multiple profiles.
		// We generate a new temporary profileUUID which is currently only used to install the profile.
		// The profileUUID in host_mdm_apple_profiles is still the original profileUUID.
		// We also generate a new commandUUID which is used to install the profile via nano_commands table.
		if addedTargets == nil {
			addedTargets = make(map[string]*fleet.CmdTarget, 1)
		}
		// We store the timestamp when the challenge was retrieved to know if it has expired.
		var managedCertificatePayloads []*fleet.MDMManagedCertificate
		// We need to update the profiles of each host with the new command UUID
		profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.EnrollmentIDs))
		for _, enrollmentID := range target.EnrollmentIDs {
			tempProfUUID := uuid.NewString()
			// Use the same UUID for command UUID, which will be the primary key for nano_commands
			tempCmdUUID := tempProfUUID
			profile, ok := getHostProfileToInstallByEnrollmentID(hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, enrollmentID, profUUID)
			if !ok || profile == nil { // Should never happen
				continue
			}
			// Fetch the host UUID, which may not be the same as the Enrollment ID, from the profile
			hostUUID := profile.HostUUID

			// some variables need more information about the host; build a skeleton host and hydrate if we need more info
			hostLite := fleet.Host{UUID: hostUUID}
			onMismatchedHostCount := func(hostCount int) error {
				return ctxerr.Wrap(ctx, ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
					CommandUUID:        target.CmdUUID,
					HostUUID:           hostLite.UUID,
					Status:             &fleet.MDMDeliveryFailed,
					Detail:             fmt.Sprintf("Unexpected number of hosts (%d) for UUID %s.", hostCount, hostLite.UUID),
					OperationType:      fleet.MDMOperationTypeInstall,
					VariablesUpdatedAt: variablesUpdatedAt,
				}), "could not retrieve host by UUID for profile variable substitution")
			}

			profile.CommandUUID = tempCmdUUID
			profile.VariablesUpdatedAt = variablesUpdatedAt

			hostContents := contentsStr
			failed := false

		fleetVarLoop:
			for _, fleetVar := range fleetVars {
				var err error
				switch {
				case fleetVar == string(fleet.FleetVarNDESSCEPChallenge):
					if ndesConfig == nil {
						if groupedCAs == nil || groupedCAs.NDESSCEP == nil {
							logger.ErrorContext(ctx, "missing NDES CA configuration for profile with NDES variables", "host_uuid", hostUUID, "profile_uuid", profUUID)
							continue
						}
						ndesConfig = groupedCAs.NDESSCEP
					}
					logger.DebugContext(ctx, "fetching NDES challenge", "host_uuid", hostUUID, "profile_uuid", profUUID)
					// Insert the SCEP challenge into the profile contents
					challenge, err := scepConfig.GetNDESSCEPChallenge(ctx, *ndesConfig)
					if err != nil {
						detail := scep.NDESChallengeErrorToDetail(err)
						err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID:        target.CmdUUID,
							HostUUID:           hostUUID,
							Status:             &fleet.MDMDeliveryFailed,
							Detail:             detail,
							OperationType:      fleet.MDMOperationTypeInstall,
							VariablesUpdatedAt: variablesUpdatedAt,
						})
						if err != nil {
							return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for NDES SCEP challenge")
						}
						failed = true
						break fleetVarLoop
					}
					payload := &fleet.MDMManagedCertificate{
						HostUUID:             hostUUID,
						ProfileUUID:          profUUID,
						ChallengeRetrievedAt: ptr.Time(time.Now()),
						Type:                 fleet.CAConfigNDES,
						CAName:               "NDES",
					}
					managedCertificatePayloads = append(managedCertificatePayloads, payload)

					hostContents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarNDESSCEPChallengeRegexp, hostContents, challenge)

				case fleetVar == string(fleet.FleetVarNDESSCEPProxyURL):
					// Insert the SCEP URL into the profile contents
					hostContents = profiles.ReplaceNDESSCEPProxyURLVariable(appConfig.MDMUrl(), hostUUID, profUUID, hostContents)

				case fleetVar == string(fleet.FleetVarSCEPRenewalID):
					// Insert the SCEP renewal ID into the SCEP Payload CN or OU
					fleetRenewalID := "fleet-" + profUUID
					hostContents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarSCEPRenewalIDRegexp, hostContents, fleetRenewalID)

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)):
					replacedContents, replacedVariable, err := profiles.ReplaceCustomSCEPChallengeVariable(ctx, logger, fleetVar, customSCEPCAs, hostContents)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing custom SCEP challenge variable")
					}
					if !replacedVariable {
						continue
					}
					hostContents = replacedContents

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
					replacedContents, managedCertificate, replacedVariable, err := profiles.ReplaceCustomSCEPProxyURLVariable(ctx, logger, ds, appConfig, fleetVar, customSCEPCAs, hostContents, hostUUID, profUUID)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing custom SCEP proxy URL variable")
					}
					if !replacedVariable {
						continue
					}
					hostContents = replacedContents
					managedCertificatePayloads = append(managedCertificatePayloads, managedCertificate)

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix))
					ca, ok := smallstepCAs[caName]
					if !ok {
						logger.ErrorContext(ctx, "Smallstep SCEP CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
					logger.DebugContext(ctx, "fetching Smallstep SCEP challenge", "host_uuid", hostUUID, "profile_uuid", profUUID)
					challenge, err := scepConfig.GetSmallstepSCEPChallenge(ctx, *ca)
					if err != nil {
						detail := fmt.Sprintf("Fleet couldn't populate $FLEET_VAR_%s. %s", fleet.FleetVarSmallstepSCEPChallengePrefix, err.Error())
						err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID:        target.CmdUUID,
							HostUUID:           hostUUID,
							Status:             &fleet.MDMDeliveryFailed,
							Detail:             detail,
							OperationType:      fleet.MDMOperationTypeInstall,
							VariablesUpdatedAt: variablesUpdatedAt,
						})
						if err != nil {
							return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for Smallstep SCEP challenge")
						}
						failed = true
						break fleetVarLoop
					}
					logger.InfoContext(ctx, "retrieved SCEP challenge from Smallstep", "host_uuid", hostUUID, "profile_uuid", profUUID)

					payload := &fleet.MDMManagedCertificate{
						HostUUID:             hostUUID,
						ProfileUUID:          profUUID,
						ChallengeRetrievedAt: ptr.Time(time.Now()),
						Type:                 fleet.CAConfigSmallstep,
						CAName:               caName,
					}
					managedCertificatePayloads = append(managedCertificatePayloads, payload)
					hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarSmallstepSCEPChallengePrefix), ca.Name, hostContents, challenge)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Smallstep SCEP challenge variable")
					}

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)):
					// Insert the SCEP URL into the profile contents
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix))
					proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), SCEPProxyPath,
						url.PathEscape(fmt.Sprintf("%s,%s,%s", hostUUID, profUUID, caName)))
					hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarSmallstepSCEPProxyURLPrefix), caName, hostContents, proxyURL)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Smallstep SCEP URL variable")
					}

				case fleetVar == string(fleet.FleetVarHostEndUserEmailIDP):
					// FIXME: if this is used together with a CA, and fail inside getFirstIDPEmail, the profile will fail, but not get the correct variablesUpdatedAt var.
					email, ok, err := getFirstIDPEmail(ctx, ds, target, hostUUID)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting IDP email")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					hostContents = profiles.ReplaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, hostContents, email)

				case fleetVar == string(fleet.FleetVarHostHardwareSerial):
					hostLite, ok, err = profiles.HydrateHost(ctx, ds, hostLite, onMismatchedHostCount)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting host hardware serial")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					hostContents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostHardwareSerialRegexp, hostContents, hostLite.HardwareSerial)
				case fleetVar == string(fleet.FleetVarHostPlatform):
					hostLite, ok, err = profiles.HydrateHost(ctx, ds, hostLite, onMismatchedHostCount)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting host platform")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					platform := hostLite.Platform
					if platform == "darwin" {
						platform = "macos"
					}

					hostContents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostPlatformRegexp, hostContents, platform)
				case fleetVar == string(fleet.FleetVarHostEndUserIDPUsername) || fleetVar == string(fleet.FleetVarHostEndUserIDPUsernameLocalPart) ||
					fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) || fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) ||
					fleetVar == string(fleet.FleetVarHostEndUserIDPFullname):
					replacedContents, replacedVariable, err := profiles.ReplaceHostEndUserIDPVariables(ctx, ds, fleetVar, hostContents, hostUUID, hostIDForUUIDCache, func(errMsg string) error {
						err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID:        target.CmdUUID,
							HostUUID:           hostUUID,
							Status:             &fleet.MDMDeliveryFailed,
							Detail:             errMsg,
							OperationType:      fleet.MDMOperationTypeInstall,
							VariablesUpdatedAt: variablesUpdatedAt,
						})
						return err
					})
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing host end user IDP variables")
					}
					if !replacedVariable {
						failed = true
						break fleetVarLoop
					}

					hostContents = replacedContents

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)):
					// We will replace the password when we populate the certificate data

				case fleetVar == string(fleet.FleetVarHostUUID):
					hostContents = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostUUIDRegexp, hostContents, hostUUID)

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix))
					ca, ok := digiCertCAs[caName]
					if !ok {
						logger.ErrorContext(ctx, "Custom DigiCert CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
					caCopy := *ca
					// Deep copy the UPN slice to prevent cross-host contamination: a
					// shallow copy shares the backing array, so in-place substitutions for
					// one host would corrupt the cached CA used by subsequent hosts.
					caCopy.CertificateUserPrincipalNames = slices.Clone(ca.CertificateUserPrincipalNames)

					// Populate Fleet vars in the CA fields
					caVarsCache := make(map[string]string)

					ok, err := replaceFleetVarInItem(ctx, ds, target, hostLite, caVarsCache, &caCopy.CertificateCommonName, onMismatchedHostCount)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					ok, err = replaceFleetVarInItem(ctx, ds, target, hostLite, caVarsCache, &caCopy.CertificateSeatID, onMismatchedHostCount)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					if len(caCopy.CertificateUserPrincipalNames) > 0 {
						for i := range caCopy.CertificateUserPrincipalNames {
							ok, err = replaceFleetVarInItem(ctx, ds, target, hostLite, caVarsCache, &caCopy.CertificateUserPrincipalNames[i], onMismatchedHostCount)
							if err != nil {
								return ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
							}
							if !ok {
								failed = true
								break fleetVarLoop
							}
						}
					}

					cert, err := digiCertService.GetCertificate(ctx, caCopy)
					if err != nil {
						detail := fmt.Sprintf("Couldn't get certificate from DigiCert for %s. %s", caCopy.Name, err)
						err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
							CommandUUID:        target.CmdUUID,
							HostUUID:           hostUUID,
							Status:             &fleet.MDMDeliveryFailed,
							Detail:             detail,
							OperationType:      fleet.MDMOperationTypeInstall,
							VariablesUpdatedAt: variablesUpdatedAt,
						})
						if err != nil {
							return ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for DigiCert")
						}
						failed = true
						break fleetVarLoop
					}
					hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarDigiCertDataPrefix), caName, hostContents,
						base64.StdEncoding.EncodeToString(cert.PfxData))
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Fleet variable for DigiCert data")
					}
					hostContents, err = profiles.ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarDigiCertPasswordPrefix), caName, hostContents, cert.Password)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Fleet variable for DigiCert password")
					}
					managedCertificatePayloads = append(managedCertificatePayloads, &fleet.MDMManagedCertificate{
						HostUUID:       hostUUID,
						ProfileUUID:    profUUID,
						NotValidBefore: &cert.NotValidBefore,
						NotValidAfter:  &cert.NotValidAfter,
						Type:           fleet.CAConfigDigiCert,
						CAName:         caName,
						Serial:         &cert.SerialNumber,
					})

				default:
					// This was handled in the above switch statement, so we should never reach this case
				}
			}
			if !failed {
				addedTargets[tempProfUUID] = &fleet.CmdTarget{
					CmdUUID:           tempCmdUUID,
					ProfileIdentifier: target.ProfileIdentifier,
					ProfileName:       target.ProfileName,
					EnrollmentIDs:     []string{enrollmentID},
				}
				profileContents[tempProfUUID] = mobileconfig.Mobileconfig(hostContents)
				profilesToUpdate = append(profilesToUpdate, profile)
			}
		}
		// Update profiles with the new command UUID
		if len(profilesToUpdate) > 0 {
			if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
				return ctxerr.Wrap(ctx, err, "updating host profiles")
			}
		}
		if len(managedCertificatePayloads) != 0 {
			// TODO: We could filter out failed profiles, but at the moment we don't, see Windows impl. for how it's done there.
			err := ds.BulkUpsertMDMManagedCertificates(ctx, managedCertificatePayloads)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "updating managed certificates")
			}
		}
		// Remove the parent target, since we will use host-specific targets
		delete(targets, profUUID)
	}
	if len(addedTargets) > 0 {
		// Add the new host-specific targets to the original targets map
		maps.Copy(targets, addedTargets)
	}
	return nil
}

func getFirstIDPEmail(ctx context.Context, ds fleet.Datastore, target *fleet.CmdTarget, hostUUID string) (string, bool, error) {
	// Insert the end user email IDP into the profile contents
	emails, err := ds.GetHostEmails(ctx, hostUUID, fleet.DeviceMappingMDMIdpAccounts)
	if err != nil {
		// This is a server error, so we exit.
		return "", false, ctxerr.Wrap(ctx, err, "getting host emails")
	}
	if len(emails) == 0 {
		// We couldn't retrieve the end user email IDP, so mark the profile as failed with additional detail.
		err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			CommandUUID: target.CmdUUID,
			HostUUID:    hostUUID,
			Status:      &fleet.MDMDeliveryFailed,
			Detail: fmt.Sprintf("There is no IdP email for this host. "+
				"Fleet couldn't populate $FLEET_VAR_%s. "+
				"[Learn more](https://fleetdm.com/learn-more-about/idp-email)",
				fleet.FleetVarHostEndUserEmailIDP),
			OperationType: fleet.MDMOperationTypeInstall,
		})
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user email IdP")
		}
		return "", false, nil
	}
	return emails[0], true, nil
}

func replaceFleetVarInItem(ctx context.Context, ds fleet.Datastore, target *fleet.CmdTarget, hostLite fleet.Host, caVarsCache map[string]string, item *string, onMismatchedHostCount func(int) error) (bool, error) {
	caFleetVars := variables.Find(*item)
	for _, caVar := range caFleetVars {
		switch caVar {
		case string(fleet.FleetVarHostEndUserEmailIDP):
			email, ok := caVarsCache[string(fleet.FleetVarHostEndUserEmailIDP)]
			if !ok {
				var err error
				email, ok, err = getFirstIDPEmail(ctx, ds, target, hostLite.UUID)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "getting IDP email")
				}
				if !ok {
					return false, nil
				}
				caVarsCache[string(fleet.FleetVarHostEndUserEmailIDP)] = email
			}
			*item = profiles.ReplaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, *item, email)
		case string(fleet.FleetVarHostHardwareSerial):
			hardwareSerial, ok := caVarsCache[string(fleet.FleetVarHostHardwareSerial)]
			if !ok {
				var err error
				hostLite, ok, err = profiles.HydrateHost(ctx, ds, hostLite, onMismatchedHostCount)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "getting host hardware serial")
				}
				if !ok {
					return false, nil
				}
				hardwareSerial = hostLite.HardwareSerial
				caVarsCache[string(fleet.FleetVarHostHardwareSerial)] = hostLite.HardwareSerial
			}
			*item = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostHardwareSerialRegexp, *item, hardwareSerial)
		case string(fleet.FleetVarHostPlatform):
			platform, ok := caVarsCache[string(fleet.FleetVarHostPlatform)]
			if !ok {
				var err error
				hostLite, ok, err = profiles.HydrateHost(ctx, ds, hostLite, onMismatchedHostCount)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "getting host hardware serial")
				}
				if !ok {
					return false, nil
				}
				platform = hostLite.Platform
				if platform == "darwin" {
					platform = "macos"
				}

				caVarsCache[string(fleet.FleetVarHostPlatform)] = platform
			}
			*item = profiles.ReplaceFleetVariableInXML(fleet.FleetVarHostPlatformRegexp, *item, platform)
		default:
			// We should not reach this since we validated the variables when saving app config
		}
	}
	return true, nil
}

func isDigiCertConfigured(ctx context.Context, logger *slog.Logger, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	existingDigiCertCAs map[string]*fleet.DigiCertCA, profUUID string, target *fleet.CmdTarget, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "DigiCert integration requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if _, ok := existingDigiCertCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var digiCertCA *fleet.DigiCertCA
	if groupedCAs != nil && len(groupedCAs.DigiCert) > 0 {
		for _, ca := range groupedCAs.DigiCert {
			if ca.Name == caName {
				digiCertCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || digiCertCA == nil {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName), ptr.Time(time.Now().UTC()))
	}

	existingDigiCertCAs[caName] = digiCertCA
	return true, nil
}

func isNDESSCEPConfigured(ctx context.Context, logger *slog.Logger, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, userEnrollmentsToHostUUIDsMap map[string]string, profUUID string, target *fleet.CmdTarget,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "NDES SCEP Proxy requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if groupedCAs == nil || groupedCAs.NDESSCEP == nil {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			"NDES SCEP Proxy is not configured. Please configure in Settings > Integrations > Certificates.", ptr.Time(time.Now().UTC()))
	}
	return true, nil
}

func isSmallstepSCEPConfigured(ctx context.Context, logger *slog.Logger, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	existingSmallstepSCEPCAs map[string]*fleet.SmallstepSCEPProxyCA, profUUID string, target *fleet.CmdTarget, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "Smallstep SCEP integration requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if _, ok := existingSmallstepSCEPCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var scepCA *fleet.SmallstepSCEPProxyCA
	if groupedCAs != nil && len(groupedCAs.Smallstep) > 0 {
		for _, ca := range groupedCAs.Smallstep {
			if ca.Name == caName {
				scepCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || scepCA == nil {
		return fleet.MarkProfilesFailed(ctx, ds, logger, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName), ptr.Time(time.Now().UTC()))
	}

	existingSmallstepSCEPCAs[caName] = scepCA
	return true, nil
}

func getHostProfileToInstallByEnrollmentID(hostProfilesToInstallMap map[fleet.HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	enrollmentID,
	profUUID string,
) (*fleet.MDMAppleBulkUpsertHostProfilePayload, bool) {
	profile, ok := hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: enrollmentID, ProfileUUID: profUUID}]
	if !ok {
		var hostUUID string
		// If sending to the user channel the enrollmentID will have to be mapped back to the host UUID.
		hostUUID, ok = userEnrollmentsToHostUUIDsMap[enrollmentID]
		if ok {
			profile, ok = hostProfilesToInstallMap[fleet.HostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
		}
	}
	return profile, ok
}
