package apple_mdm

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/ee/server/service/scep"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/variables"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
)

// This file handles preprocessing, enqueing and sending MDM apple profiles.

// ProcessAndEnqueueProfiles preprocesses the profile contents to replace any
// `onFailedEnqueue` is called for each command that could not be enqueued
func ProcessAndEnqueueProfiles(
	ctx context.Context,
	ds fleet.Datastore,
	logger kitlog.Logger,
	appConfig *fleet.AppConfig,
	commander *MDMAppleCommander,
	installTargets, removeTargets map[string]*CmdTarget,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	onFailedEnqueue func(cmdUUID string, err error),
	onSuccessfulEnqueue func(cmdUUID string),
) error {
	// Grab the contents of all the profiles we need to install
	profileUUIDs := make([]string, 0, len(installTargets)+len(removeTargets))
	for pUUID := range installTargets {
		profileUUIDs = append(profileUUIDs, pUUID)
	}

	profileContents, err := ds.GetMDMAppleProfilesContents(ctx, profileUUIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get profile contents")
	}

	groupedCAs, err := ds.GetGroupedCertificateAuthorities(ctx, true)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting grouped certificate authorities")
	}

	// Insert variables into profile contents of install targets. Variables may be host-specific.
	err = preprocessProfileContents(ctx, appConfig, ds,
		scep.NewSCEPConfigService(logger, nil),
		digicert.NewService(digicert.WithLogger(logger)),
		logger, installTargets, profileContents, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, groupedCAs)
	if err != nil {
		return err
	}

	// Find the profiles containing secret variables.
	profilesWithSecrets, err := findProfilesWithSecrets(logger, installTargets, profileContents)
	if err != nil {
		return err
	}

	type remoteResult struct {
		Err     error
		CmdUUID string
	}

	// Send the install/remove commands for each profile.
	var wgProd, wgCons sync.WaitGroup
	ch := make(chan remoteResult)

	execCmd := func(profUUID string, target *CmdTarget, op fleet.MDMOperationType) {
		defer wgProd.Done()

		var err error
		switch op {
		case fleet.MDMOperationTypeInstall:
			if _, ok := profilesWithSecrets[profUUID]; ok {
				err = commander.EnqueueCommandInstallProfileWithSecrets(ctx, target.EnrollmentIDs, profileContents[profUUID], target.CmdUUID)
			} else {
				err = commander.InstallProfile(ctx, target.EnrollmentIDs, profileContents[profUUID], target.CmdUUID)
			}
		case fleet.MDMOperationTypeRemove:
			err = commander.RemoveProfile(ctx, target.EnrollmentIDs, target.ProfileIdentifier, target.CmdUUID)
		}

		var e *APNSDeliveryError
		switch {
		case errors.As(err, &e):
			level.Debug(logger).Log("err", "sending push notifications, profiles still enqueued", "details", err)
		case err != nil:
			level.Error(logger).Log("err", fmt.Sprintf("enqueue command to %s profiles", op), "details", err)
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

	wgCons.Add(1)
	go func() {
		defer wgCons.Done()

		for resp := range ch {
			if resp.Err == nil && onSuccessfulEnqueue != nil {
				onSuccessfulEnqueue(resp.CmdUUID)
			} else if resp.Err != nil && onFailedEnqueue != nil {
				onFailedEnqueue(resp.CmdUUID, resp.Err)
			}
		}
	}()

	wgProd.Wait()
	close(ch) // done sending at this point, this triggers end of for loop in consumer
	wgCons.Wait()

	return nil
}

func preprocessProfileContents(
	ctx context.Context,
	appConfig *fleet.AppConfig,
	ds fleet.Datastore,
	scepConfig fleet.SCEPConfigService,
	digiCertService fleet.DigiCertService,
	logger kitlog.Logger,
	targets map[string]*CmdTarget,
	profileContents map[string]mobileconfig.Mobileconfig,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
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

	var addedTargets map[string]*CmdTarget
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
		for fleetVar := range fleetVars {
			if fleetVar == string(fleet.FleetVarSCEPRenewalID) ||
				fleetVar == string(fleet.FleetVarNDESSCEPChallenge) || fleetVar == string(fleet.FleetVarNDESSCEPProxyURL) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)) ||
				strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)) {
				// Give a few minutes leeway to account for clock skew
				variablesUpdatedAt = ptr.Time(time.Now().UTC().Add(-3 * time.Minute))
				break
			}
		}

	initialFleetVarLoop:
		for fleetVar := range fleetVars {
			switch {
			case fleetVar == string(fleet.FleetVarNDESSCEPChallenge) || fleetVar == string(fleet.FleetVarNDESSCEPProxyURL):
				configured, err := isNDESSCEPConfigured(ctx, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, target)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking NDES SCEP configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			case fleetVar == string(fleet.FleetVarHostEndUserEmailIDP) || fleetVar == string(fleet.FleetVarHostHardwareSerial) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPUsername) || fleetVar == string(fleet.FleetVarHostEndUserIDPUsernameLocalPart) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) || fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) || fleetVar == string(fleet.FleetVarSCEPRenewalID) ||
				fleetVar == string(fleet.FleetVarHostEndUserIDPFullname):
				// No extra validation needed for these variables

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)):
				var caName string
				if strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)) {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix))
				} else {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix))
				}
				if digiCertCAs == nil {
					digiCertCAs = make(map[string]*fleet.DigiCertCA)
				}
				configured, err := isDigiCertConfigured(ctx, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, digiCertCAs, profUUID, target, caName, fleetVar)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking DigiCert configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
				var caName string
				if strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)) {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix))
				} else {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
				}
				if customSCEPCAs == nil {
					customSCEPCAs = make(map[string]*fleet.CustomSCEPProxyCA)
				}
				configured, err := isCustomSCEPConfigured(ctx, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, customSCEPCAs, profUUID, target, caName,
					fleetVar)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "checking custom SCEP configuration")
				}
				if !configured {
					valid = false
					break initialFleetVarLoop
				}

			case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)) || strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)):
				if smallstepCAs == nil {
					smallstepCAs = make(map[string]*fleet.SmallstepSCEPProxyCA)
				}
				var caName string
				if strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)) {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix))
				} else {
					caName = strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix))
				}
				configured, err := isSmallstepSCEPConfigured(ctx, groupedCAs, ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, smallstepCAs, profUUID, target, caName,
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
				_, err := markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, detail, variablesUpdatedAt)
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
			addedTargets = make(map[string]*CmdTarget, 1)
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
			if !ok { // Should never happen
				continue
			}
			// Fetch the host UUID, which may not be the same as the Enrollment ID, from the profile
			hostUUID := profile.HostUUID
			profile.CommandUUID = tempCmdUUID
			profile.VariablesUpdatedAt = variablesUpdatedAt

			hostContents := contentsStr
			failed := false
		fleetVarLoop:
			for fleetVar := range fleetVars {
				var err error
				switch {
				case fleetVar == string(fleet.FleetVarNDESSCEPChallenge):
					if ndesConfig == nil {
						ndesConfig = groupedCAs.NDESSCEP
					}
					// Insert the SCEP challenge into the profile contents
					challenge, err := scepConfig.GetNDESSCEPChallenge(ctx, *ndesConfig)
					if err != nil {
						detail := ""
						switch {
						case errors.As(err, &scep.NDESInvalidError{}):
							detail = fmt.Sprintf("Invalid NDES admin credentials. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please update credentials in Settings > Integrations > Mobile Device Management > Simple Certificate Enrollment Protocol.",
								fleet.FleetVarNDESSCEPChallenge)
						case errors.As(err, &scep.NDESPasswordCacheFullError{}):
							detail = fmt.Sprintf("The NDES password cache is full. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please increase the number of cached passwords in NDES and try again.",
								fleet.FleetVarNDESSCEPChallenge)
						case errors.As(err, &scep.NDESInsufficientPermissionsError{}):
							detail = fmt.Sprintf("This account does not have sufficient permissions to enroll with SCEP. "+
								"Fleet couldn't populate $FLEET_VAR_%s. "+
								"Please update the account with NDES SCEP enroll permissions and try again.",
								fleet.FleetVarNDESSCEPChallenge)
						default:
							detail = fmt.Sprintf("Fleet couldn't populate $FLEET_VAR_%s. %s", fleet.FleetVarNDESSCEPChallenge, err.Error())
						}
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

					hostContents = replaceFleetVariableInXML(fleetVarNDESSCEPChallengeRegexp, hostContents, challenge)

				case fleetVar == string(fleet.FleetVarNDESSCEPProxyURL):
					// Insert the SCEP URL into the profile contents
					proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), SCEPProxyPath,
						url.PathEscape(fmt.Sprintf("%s,%s,NDES", hostUUID, profUUID)))
					hostContents = replaceFleetVariableInXML(fleetVarNDESSCEPProxyURLRegexp, hostContents, proxyURL)

				case fleetVar == string(fleet.FleetVarSCEPRenewalID):
					// Insert the SCEP renewal ID into the SCEP Payload CN
					fleetRenewalID := "fleet-" + profUUID
					hostContents = replaceFleetVariableInXML(fleetVarSCEPRenewalIDRegexp, hostContents, fleetRenewalID)

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPChallengePrefix))
					ca, ok := customSCEPCAs[caName]
					if !ok {
						level.Error(logger).Log("msg", "Custom SCEP CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarCustomSCEPChallengePrefix), ca.Name, hostContents, ca.Challenge)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP challenge")
					}

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
					ca, ok := customSCEPCAs[caName]
					if !ok {
						level.Error(logger).Log("msg", "Custom SCEP CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
					// Generate a new SCEP challenge for the profile
					challenge, err := ds.NewChallenge(ctx)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "generating SCEP challenge")
					}
					// Insert the SCEP URL into the profile contents
					proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), SCEPProxyPath,
						url.PathEscape(fmt.Sprintf("%s,%s,%s,%s", hostUUID, profUUID, caName, challenge)))
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarCustomSCEPProxyURLPrefix), ca.Name, hostContents, proxyURL)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP proxy URL")
					}
					managedCertificatePayloads = append(managedCertificatePayloads, &fleet.MDMManagedCertificate{
						HostUUID:    hostUUID,
						ProfileUUID: profUUID,
						Type:        fleet.CAConfigCustomSCEPProxy,
						CAName:      caName,
					})

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPChallengePrefix))
					ca, ok := smallstepCAs[caName]
					if !ok {
						level.Error(logger).Log("msg", "Smallstep SCEP CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
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
					level.Info(logger).Log("msg", "retrieved SCEP challenge from Smallstep", "host_uuid", hostUUID, "profile_uuid", profUUID)

					payload := &fleet.MDMManagedCertificate{
						HostUUID:             hostUUID,
						ProfileUUID:          profUUID,
						ChallengeRetrievedAt: ptr.Time(time.Now()),
						Type:                 fleet.CAConfigSmallstep,
						CAName:               caName,
					}
					managedCertificatePayloads = append(managedCertificatePayloads, payload)
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarSmallstepSCEPChallengePrefix), ca.Name, hostContents, challenge)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Smallstep SCEP challenge variable")
					}

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix)):
					// Insert the SCEP URL into the profile contents
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarSmallstepSCEPProxyURLPrefix))
					proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), SCEPProxyPath,
						url.PathEscape(fmt.Sprintf("%s,%s,%s", hostUUID, profUUID, caName)))
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarSmallstepSCEPProxyURLPrefix), caName, hostContents, proxyURL)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Smallstep SCEP URL variable")
					}

				case fleetVar == string(fleet.FleetVarHostEndUserEmailIDP):
					email, ok, err := getIDPEmail(ctx, ds, target, hostUUID)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting IDP email")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					hostContents = replaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, hostContents, email)

				case fleetVar == string(fleet.FleetVarHostHardwareSerial):
					hardwareSerial, ok, err := getHostHardwareSerial(ctx, ds, target, hostUUID)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting host hardware serial")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					hostContents = replaceFleetVariableInXML(fleetVarHostHardwareSerialRegexp, hostContents, hardwareSerial)

				case fleetVar == string(fleet.FleetVarHostEndUserIDPUsername) || fleetVar == string(fleet.FleetVarHostEndUserIDPUsernameLocalPart) ||
					fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) || fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) ||
					fleetVar == string(fleet.FleetVarHostEndUserIDPFullname):
					user, ok, err := getHostEndUserIDPUser(ctx, ds, target, hostUUID, fleetVar, hostIDForUUIDCache)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "getting host end user IDP username")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}

					var rx *regexp.Regexp
					var value string
					switch fleetVar {
					case string(fleet.FleetVarHostEndUserIDPUsername):
						rx = fleetVarHostEndUserIDPUsernameRegexp
						value = user.IdpUserName
					case string(fleet.FleetVarHostEndUserIDPUsernameLocalPart):
						rx = fleetVarHostEndUserIDPUsernameLocalPartRegexp
						value = getEmailLocalPart(user.IdpUserName)
					case string(fleet.FleetVarHostEndUserIDPGroups):
						rx = fleetVarHostEndUserIDPGroupsRegexp
						value = strings.Join(user.IdpGroups, ",")
					case string(fleet.FleetVarHostEndUserIDPDepartment):
						rx = fleetVarHostEndUserIDPDepartmentRegexp
						value = user.Department
					case string(fleet.FleetVarHostEndUserIDPFullname):
						rx = fleetVarHostEndUserIDPFullnameRegexp
						value = strings.TrimSpace(user.IdpFullName)
					}
					hostContents = replaceFleetVariableInXML(rx, hostContents, value)

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertPasswordPrefix)):
					// We will replace the password when we populate the certificate data

				case strings.HasPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix)):
					caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarDigiCertDataPrefix))
					ca, ok := digiCertCAs[caName]
					if !ok {
						level.Error(logger).Log("msg", "Custom DigiCert CA not found. "+
							"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
						continue
					}
					caCopy := *ca

					// Populate Fleet vars in the CA fields
					caVarsCache := make(map[string]string)
					ok, err := replaceFleetVarInItem(ctx, ds, target, hostUUID, caVarsCache, &caCopy.CertificateCommonName)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					ok, err = replaceFleetVarInItem(ctx, ds, target, hostUUID, caVarsCache, &caCopy.CertificateSeatID)
					if err != nil {
						return ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
					}
					if !ok {
						failed = true
						break fleetVarLoop
					}
					if len(caCopy.CertificateUserPrincipalNames) > 0 {
						for i := range caCopy.CertificateUserPrincipalNames {
							ok, err = replaceFleetVarInItem(ctx, ds, target, hostUUID, caVarsCache, &caCopy.CertificateUserPrincipalNames[i])
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
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarDigiCertDataPrefix), caName, hostContents,
						base64.StdEncoding.EncodeToString(cert.PfxData))
					if err != nil {
						return ctxerr.Wrap(ctx, err, "replacing Fleet variable for DigiCert data")
					}
					hostContents, err = replaceExactFleetPrefixVariableInXML(string(fleet.FleetVarDigiCertPasswordPrefix), caName, hostContents, cert.Password)
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
				addedTargets[tempProfUUID] = &CmdTarget{
					CmdUUID:           tempCmdUUID,
					ProfileIdentifier: target.ProfileIdentifier,
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
		for profUUID, target := range addedTargets {
			targets[profUUID] = target
		}
	}
	return nil
}

func replaceFleetVarInItem(ctx context.Context, ds fleet.Datastore, target *CmdTarget, hostUUID string, caVarsCache map[string]string, item *string,
) (bool, error) {
	caFleetVars := variables.Find(*item)
	for caVar := range caFleetVars {
		switch caVar {
		case string(fleet.FleetVarHostEndUserEmailIDP):
			email, ok := caVarsCache[string(fleet.FleetVarHostEndUserEmailIDP)]
			if !ok {
				var err error
				email, ok, err = getIDPEmail(ctx, ds, target, hostUUID)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "getting IDP email")
				}
				if !ok {
					return false, nil
				}
				caVarsCache[string(fleet.FleetVarHostEndUserEmailIDP)] = email
			}
			*item = replaceFleetVariableInXML(fleetVarHostEndUserEmailIDPRegexp, *item, email)
		case string(fleet.FleetVarHostHardwareSerial):
			hardwareSerial, ok := caVarsCache[string(fleet.FleetVarHostHardwareSerial)]
			if !ok {
				var err error
				hardwareSerial, ok, err = getHostHardwareSerial(ctx, ds, target, hostUUID)
				if err != nil {
					return false, ctxerr.Wrap(ctx, err, "getting host hardware serial")
				}
				if !ok {
					return false, nil
				}
				caVarsCache[string(fleet.FleetVarHostHardwareSerial)] = hardwareSerial
			}
			*item = replaceFleetVariableInXML(fleetVarHostHardwareSerialRegexp, *item, hardwareSerial)
		default:
			// We should not reach this since we validated the variables when saving app config
		}
	}
	return true, nil
}

func replaceFleetVariableInXML(regExp *regexp.Regexp, contents string, replacement string) string {
	// Escape XML characters since this replacement is intended for XML profile.
	b := make([]byte, 0, len(replacement))
	buf := bytes.NewBuffer(b)
	// error is always nil for Buffer.Write method, so we ignore it
	_ = xml.EscapeText(buf, []byte(replacement))
	return regExp.ReplaceAllLiteralString(contents, buf.String())
}

func replaceExactFleetPrefixVariableInXML(prefix string, suffix string, contents string, replacement string) (string, error) {
	// Escape XML characters since this replacement is intended for XML profile.
	b := make([]byte, 0, len(replacement))
	buf := bytes.NewBuffer(b)
	// error is always nil for Buffer.Write method, so we ignore it
	_ = xml.EscapeText(buf, []byte(replacement))

	// We are replacing an exact variable, which should be present in XML like: <something>$FLEET_VAR_OUR_VAR</something>
	// We strip the leading/trailing whitespace since we don't want them to remain in XML
	// Our plist parser ignores spaces in <data> type. We don't catch this issue at profile validation, so we handle it here.
	fleetVar := "FLEET_VAR_" + prefix + suffix
	re, err := regexp.Compile(fmt.Sprintf(`>\s*((\$%s)|(\${%s}))\s*<`, fleetVar, fleetVar))
	if err != nil {
		return "", err
	}
	return re.ReplaceAllLiteralString(contents, fmt.Sprintf(`>%s<`, buf.String())), nil
}

func getHostEndUserIDPUser(ctx context.Context, ds fleet.Datastore, target *CmdTarget,
	hostUUID, fleetVar string, hostIDForUUIDCache map[string]uint,
) (*fleet.HostEndUser, bool, error) {
	hostID, ok := hostIDForUUIDCache[hostUUID]
	if !ok {
		filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
		ids, err := ds.HostIDsByIdentifier(ctx, filter, []string{hostUUID})
		if err != nil {
			return nil, false, ctxerr.Wrap(ctx, err, "get host id from uuid")
		}

		if len(ids) != 1 {
			// Something went wrong. Maybe host was deleted, or we have multiple
			// hosts with the same UUID. Mark the profile as failed with additional
			// detail.
			err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
				CommandUUID:   target.CmdUUID,
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        fmt.Sprintf("Unexpected number of hosts (%d) for UUID %s. ", len(ids), hostUUID),
				OperationType: fleet.MDMOperationTypeInstall,
			})
			if err != nil {
				return nil, false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user IDP")
			}
			return nil, false, nil
		}
		hostID = ids[0]
		hostIDForUUIDCache[hostUUID] = hostID
	}

	users, err := fleet.GetEndUsers(ctx, ds, hostID)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "get end users for host")
	}

	noGroupsErr := fmt.Sprintf("There is no IdP groups for this host. Fleet couldn’t populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPGroups)
	noDepartmentErr := fmt.Sprintf("There is no IdP department for this host. Fleet couldn’t populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPDepartment)
	noFullnameErr := fmt.Sprintf("There is no IdP full name for this host. Fleet couldn’t populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPFullname)
	if len(users) > 0 && users[0].IdpUserName != "" {
		idpUser := users[0]

		if fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) && len(idpUser.IdpGroups) == 0 {
			err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
				CommandUUID:   target.CmdUUID,
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        noGroupsErr,
				OperationType: fleet.MDMOperationTypeInstall,
			})
			if err != nil {
				return nil, false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user IDP (no groups)")
			}
			return nil, false, nil
		}
		if fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) && idpUser.Department == "" {
			err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
				CommandUUID:   target.CmdUUID,
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        noDepartmentErr,
				OperationType: fleet.MDMOperationTypeInstall,
			})
			if err != nil {
				return nil, false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user IDP (no department)")
			}
			return nil, false, nil
		}
		if fleetVar == string(fleet.FleetVarHostEndUserIDPFullname) && strings.TrimSpace(idpUser.IdpFullName) == "" {
			err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
				CommandUUID:   target.CmdUUID,
				HostUUID:      hostUUID,
				Status:        &fleet.MDMDeliveryFailed,
				Detail:        noFullnameErr,
				OperationType: fleet.MDMOperationTypeInstall,
			})
			if err != nil {
				return nil, false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user IDP (no fullname)")
			}
			return nil, false, nil
		}

		return &idpUser, true, nil
	}

	// otherwise there's no IdP user, mark the profile as failed with the
	// appropriate detail message.
	var detail string
	switch fleetVar {
	case string(fleet.FleetVarHostEndUserIDPUsername), string(fleet.FleetVarHostEndUserIDPUsernameLocalPart):
		detail = fmt.Sprintf("There is no IdP username for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleetVar)
	case string(fleet.FleetVarHostEndUserIDPGroups):
		detail = noGroupsErr
	case string(fleet.FleetVarHostEndUserIDPDepartment):
		detail = noDepartmentErr
	case string(fleet.FleetVarHostEndUserIDPFullname):
		detail = noFullnameErr
	}
	err = ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
		CommandUUID:   target.CmdUUID,
		HostUUID:      hostUUID,
		Status:        &fleet.MDMDeliveryFailed,
		Detail:        detail,
		OperationType: fleet.MDMOperationTypeInstall,
	})
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for end user IDP")
	}
	return nil, false, nil
}

func getEmailLocalPart(email string) string {
	// if there is a "@" in the email, return the part before that "@", otherwise
	// return the string unchanged.
	local, _, _ := strings.Cut(email, "@")
	return local
}

func getIDPEmail(ctx context.Context, ds fleet.Datastore, target *CmdTarget, hostUUID string) (string, bool, error) {
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

func getHostHardwareSerial(ctx context.Context, ds fleet.Datastore, target *CmdTarget, hostUUID string) (string, bool, error) {
	hosts, err := ds.ListHostsLiteByUUIDs(ctx, fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}, []string{hostUUID})
	if err != nil {
		return "", false, ctxerr.Wrap(ctx, err, "listing hosts")
	}
	if len(hosts) != 1 {
		// Something went wrong. Maybe host was deleted, or we have multiple hosts with the same UUID.
		// Mark the profile as failed with additional detail.
		err := ds.UpdateOrDeleteHostMDMAppleProfile(ctx, &fleet.HostMDMAppleProfile{
			CommandUUID:   target.CmdUUID,
			HostUUID:      hostUUID,
			Status:        &fleet.MDMDeliveryFailed,
			Detail:        fmt.Sprintf("Unexpected number of hosts (%d) for UUID %s. ", len(hosts), hostUUID),
			OperationType: fleet.MDMOperationTypeInstall,
		})
		if err != nil {
			return "", false, ctxerr.Wrap(ctx, err, "updating host MDM Apple profile for hardware serial")
		}
		return "", false, nil
	}
	hardwareSerial := hosts[0].HardwareSerial
	return hardwareSerial, true, nil
}

func isNDESSCEPConfigured(ctx context.Context, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload, userEnrollmentsToHostUUIDsMap map[string]string, profUUID string, target *CmdTarget,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "NDES SCEP Proxy requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if groupedCAs.NDESSCEP == nil {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			"NDES SCEP Proxy is not configured. Please configure in Settings > Integrations > Certificates.", ptr.Time(time.Now().UTC()))
	}
	return true, nil
}

func isCustomSCEPConfigured(ctx context.Context, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	existingCustomSCEPCAs map[string]*fleet.CustomSCEPProxyCA, profUUID string, target *CmdTarget, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "Custom SCEP integration requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if _, ok := existingCustomSCEPCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var scepCA *fleet.CustomSCEPProxyCA
	if len(groupedCAs.CustomScepProxy) > 0 {
		for _, ca := range groupedCAs.CustomScepProxy {
			if ca.Name == caName {
				scepCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || scepCA == nil {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName), ptr.Time(time.Now().UTC()))
	}

	existingCustomSCEPCAs[caName] = scepCA
	return true, nil
}

func isSmallstepSCEPConfigured(ctx context.Context, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	existingSmallstepSCEPCAs map[string]*fleet.SmallstepSCEPProxyCA, profUUID string, target *CmdTarget, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "Smallstep SCEP integration requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if _, ok := existingSmallstepSCEPCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var scepCA *fleet.SmallstepSCEPProxyCA
	if len(groupedCAs.Smallstep) > 0 {
		for _, ca := range groupedCAs.Smallstep {
			if ca.Name == caName {
				scepCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || scepCA == nil {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName), ptr.Time(time.Now().UTC()))
	}

	existingSmallstepSCEPCAs[caName] = scepCA
	return true, nil
}

func isDigiCertConfigured(ctx context.Context, groupedCAs *fleet.GroupedCertificateAuthorities, ds fleet.Datastore,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	existingDigiCertCAs map[string]*fleet.DigiCertCA, profUUID string, target *CmdTarget, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, "DigiCert integration requires a Fleet Premium license.", ptr.Time(time.Now().UTC()))
	}
	if _, ok := existingDigiCertCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var digiCertCA *fleet.DigiCertCA
	if len(groupedCAs.DigiCert) > 0 {
		for _, ca := range groupedCAs.DigiCert {
			if ca.Name == caName {
				digiCertCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || digiCertCA == nil {
		return markProfilesFailed(ctx, ds, target, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID,
			fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName), ptr.Time(time.Now().UTC()))
	}

	existingDigiCertCAs[caName] = digiCertCA
	return true, nil
}

// install/removeTargets are maps from profileUUID -> command uuid and host
// UUIDs as the underlying MDM services are optimized to send one command to
// multiple hosts at the same time. Note that the same command uuid is used
// for all hosts in a given install/remove target operation.
type CmdTarget struct {
	CmdUUID           string
	ProfileIdentifier string
	EnrollmentIDs     []string
}

type HostProfileUUID struct {
	HostUUID    string
	ProfileUUID string
}

func findProfilesWithSecrets(
	logger kitlog.Logger,
	installTargets map[string]*CmdTarget,
	profileContents map[string]mobileconfig.Mobileconfig,
) (map[string]struct{}, error) {
	profilesWithSecrets := make(map[string]struct{})
	for profUUID := range installTargets {
		p, ok := profileContents[profUUID]
		if !ok { // Should never happen
			level.Error(logger).Log("msg", "profile content not found in ReconcileAppleProfiles", "profile_uuid", profUUID)
			continue
		}
		profileStr := string(p)
		vars := fleet.ContainsPrefixVars(profileStr, fleet.ServerSecretPrefix)
		if len(vars) > 0 {
			profilesWithSecrets[profUUID] = struct{}{}
		}
	}
	return profilesWithSecrets, nil
}

func markProfilesFailed(
	ctx context.Context,
	ds fleet.Datastore,
	target *CmdTarget,
	hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	profUUID string,
	detail string,
	variablesUpdatedAt *time.Time,
) (bool, error) {
	profilesToUpdate := make([]*fleet.MDMAppleBulkUpsertHostProfilePayload, 0, len(target.EnrollmentIDs))
	for _, enrollmentID := range target.EnrollmentIDs {
		profile, ok := getHostProfileToInstallByEnrollmentID(hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, enrollmentID, profUUID)
		if !ok {
			// If sending to the user channel the enrollmentID will have to be mapped back to the host UUID.
			hostUUID, ok := userEnrollmentsToHostUUIDsMap[enrollmentID]
			if ok {
				profile, ok = hostProfilesToInstallMap[HostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
			}
			if !ok {
				continue
			}
		}
		profile.Status = &fleet.MDMDeliveryFailed
		profile.Detail = detail
		profile.VariablesUpdatedAt = variablesUpdatedAt
		profilesToUpdate = append(profilesToUpdate, profile)
	}
	if err := ds.BulkUpsertMDMAppleHostProfiles(ctx, profilesToUpdate); err != nil {
		return false, fmt.Errorf("marking host profiles failed: %w", err)
	}
	return false, nil
}

func getHostProfileToInstallByEnrollmentID(hostProfilesToInstallMap map[HostProfileUUID]*fleet.MDMAppleBulkUpsertHostProfilePayload,
	userEnrollmentsToHostUUIDsMap map[string]string,
	enrollmentID,
	profUUID string,
) (*fleet.MDMAppleBulkUpsertHostProfilePayload, bool) {
	profile, ok := hostProfilesToInstallMap[HostProfileUUID{HostUUID: enrollmentID, ProfileUUID: profUUID}]
	if !ok {
		var hostUUID string
		// If sending to the user channel the enrollmentID will have to be mapped back to the host UUID.
		hostUUID, ok = userEnrollmentsToHostUUIDsMap[enrollmentID]
		if ok {
			profile, ok = hostProfilesToInstallMap[HostProfileUUID{HostUUID: hostUUID, ProfileUUID: profUUID}]
		}
	}
	return profile, ok
}
