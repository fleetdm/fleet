package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"al.essio.dev/pkg/shellescape"
	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/ee/server/service/digicert"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/go-kit/log/level"
)

func isDigiCertConfiguredForScript(ctx context.Context, appConfig *fleet.AppConfig,
	ds fleet.Datastore, digiCertCAs map[string]*fleet.DigiCertIntegration, caName string,
	fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return false, ctxerr.Errorf(ctx, "DigiCert integration requires a Fleet Premium license.")
	}
	if _, ok := digiCertCAs[caName]; ok {
		return true, nil
	}

	configured := false
	var digiCertCA *fleet.DigiCertIntegration
	if appConfig.Integrations.DigiCert.Valid {
		for _, ca := range appConfig.Integrations.DigiCert.Value {
			if ca.Name == caName {
				digiCertCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || digiCertCA == nil {
		return false,
			ctxerr.Errorf(ctx, "Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName)
	}

	// Get the API token
	asset, err := ds.GetCAConfigAsset(ctx, digiCertCA.Name, fleet.CAConfigDigiCert)
	switch {
	case fleet.IsNotFound(err):
		return false,
			ctxerr.Errorf(ctx, "DigiCert CA '%s' is missing API token. Please configure in Settings > Integrations > Certificates.", caName)
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "getting CA config asset")
	}
	digiCertCA.APIToken = string(asset.Value)
	digiCertCAs[caName] = digiCertCA

	return true, nil
}

func isCustomSCEPConfiguredForScript(ctx context.Context, appConfig *fleet.AppConfig, ds fleet.Datastore,
	customSCEPCAs map[string]*fleet.CustomSCEPProxyIntegration, caName string, fleetVar string,
) (bool, error) {
	if !license.IsPremium(ctx) {
		return false, ctxerr.Errorf(ctx, "Custom SCEP integration requires a Fleet Premium license.")
	}
	if _, ok := customSCEPCAs[caName]; ok {
		return true, nil
	}
	configured := false
	var scepCA *fleet.CustomSCEPProxyIntegration
	if appConfig.Integrations.CustomSCEPProxy.Valid {
		for _, ca := range appConfig.Integrations.CustomSCEPProxy.Value {
			if ca.Name == caName {
				scepCA = &ca
				configured = true
				break
			}
		}
	}
	if !configured || scepCA == nil {
		return false,
			ctxerr.Errorf(ctx, "Fleet couldn't populate $%s because %s certificate authority doesn't exist.")
	}

	// Get the challenge
	asset, err := ds.GetCAConfigAsset(ctx, scepCA.Name, fleet.CAConfigCustomSCEPProxy)
	switch {
	case fleet.IsNotFound(err):
		return false,
			ctxerr.Errorf(ctx, "Custom SCEP CA '%s' is missing a challenge. Please configure in Settings > Integrations > Certificates.", caName)
	case err != nil:
		return false, ctxerr.Wrap(ctx, err, "getting custom SCEP CA config asset")
	}
	scepCA.Challenge = string(asset.Value)
	customSCEPCAs[caName] = scepCA

	return true, nil
}

func replaceFleetVariableInShellScript(regExp *regexp.Regexp, contents string, replacement string) string {
	return regExp.ReplaceAllLiteralString(contents, shellescape.Quote(replacement))
}

func replaceExactFleetPrefixVariableInShellScript(prefix string, suffix string, contents string, replacement string) (string, error) {
	replacement = shellescape.Quote(replacement)

	// We are replacing an exact variable, which should be present in XML like: <something>$FLEET_VAR_OUR_VAR</something>
	// We strip the leading/trailing whitespace since we don't want them to remain in XML
	// Our plist parser ignores spaces in <data> type. We don't catch this issue at profile validation, so we handle it here.
	fleetVar := "FLEET_VAR_" + prefix + suffix
	re, err := regexp.Compile(fmt.Sprintf(`((\$%s)|(\${%s}))`, fleetVar, fleetVar))
	if err != nil {
		return "", err
	}
	return re.ReplaceAllLiteralString(contents, replacement), nil
}

func replaceFleetVarInItemFromHostInfo(ctx context.Context, ds fleet.Datastore, hostInfo *fleet.Host, caVarsCache map[string]string, item *string,
) (bool, error) {
	caFleetVars := findFleetVariables(*item)
	for caVar := range caFleetVars {
		switch caVar {
		case fleet.FleetVarHostEndUserEmailIDP:
			return false, ctxerr.New(ctx, "end user email not supported yet")
		case fleet.FleetVarHostHardwareSerial:
			*item = replaceFleetVariableInShellScript(fleetVarHostHardwareSerialRegexp, *item, hostInfo.HardwareSerial)
		default:
			// We should not reach this since we validated the variables when saving app config
		}
	}
	return true, nil
}

func (svc *Service) processFleetVariables(ctx context.Context, execID, contents string) (string, error) {
	fleetVars := findFleetVariables(contents)
	if len(fleetVars) == 0 {
		return contents, nil
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err)
	}

	hostInfo, ok := hostctx.FromContext(ctx)
	if !ok {
		return "", ctxerr.Errorf(ctx, "unable to infer host from context")
	}

	var (
		// Copy of NDES SCEP config which will contain unencrypted password, if needed
		ndesConfig    *fleet.NDESSCEPProxyIntegration
		digiCertCAs   map[string]*fleet.DigiCertIntegration
		customSCEPCAs map[string]*fleet.CustomSCEPProxyIntegration
	)

	scepConfig := eeservice.NewSCEPConfigService(svc.logger, nil)
	digiCertService := digicert.NewService(digicert.WithLogger(svc.logger))

	valid := true
	for fleetVar := range fleetVars {
		switch {
		/*
			case fleetVar == fleet.FleetVarNDESSCEPChallenge || fleetVar == fleet.FleetVarNDESSCEPProxyURL:
				configured, err := isNDESSCEPConfigured(ctx, appConfig, svc.ds, hostProfilesToInstallMap, userEnrollmentsToHostUUIDsMap, profUUID, target)
				if err != nil {
					return "", ctxerr.Wrap(ctx, err, "checking NDES SCEP configuration")
				}
				if !configured {
					valid = false
					break
				}
		*/

		case fleetVar == fleet.FleetVarHostEndUserEmailIDP || fleetVar == fleet.FleetVarHostHardwareSerial ||
			fleetVar == fleet.FleetVarHostEndUserIDPUsername || fleetVar == fleet.FleetVarHostEndUserIDPUsernameLocalPart ||
			fleetVar == fleet.FleetVarHostEndUserIDPGroups || fleetVar == fleet.FleetVarHostEndUserIDPDepartment || fleetVar == fleet.FleetVarSCEPRenewalID:
			// No extra validation needed for these variables

		case strings.HasPrefix(fleetVar, fleet.FleetVarDigiCertPasswordPrefix) || strings.HasPrefix(fleetVar, fleet.FleetVarDigiCertDataPrefix):
			var caName string
			if strings.HasPrefix(fleetVar, fleet.FleetVarDigiCertPasswordPrefix) {
				caName = strings.TrimPrefix(fleetVar, fleet.FleetVarDigiCertPasswordPrefix)
			} else {
				caName = strings.TrimPrefix(fleetVar, fleet.FleetVarDigiCertDataPrefix)
			}
			if digiCertCAs == nil {
				digiCertCAs = make(map[string]*fleet.DigiCertIntegration)
			}
			configured, err := isDigiCertConfiguredForScript(ctx, appConfig, svc.ds, digiCertCAs, caName, fleetVar)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "checking DigiCert configuration")
			}
			if !configured {
				valid = false
				break
			}

		case strings.HasPrefix(fleetVar, fleet.FleetVarCustomSCEPChallengePrefix) || strings.HasPrefix(fleetVar, fleet.FleetVarCustomSCEPProxyURLPrefix):
			var caName string
			if strings.HasPrefix(fleetVar, fleet.FleetVarCustomSCEPChallengePrefix) {
				caName = strings.TrimPrefix(fleetVar, fleet.FleetVarCustomSCEPChallengePrefix)
			} else {
				caName = strings.TrimPrefix(fleetVar, fleet.FleetVarCustomSCEPProxyURLPrefix)
			}
			if customSCEPCAs == nil {
				customSCEPCAs = make(map[string]*fleet.CustomSCEPProxyIntegration)
			}
			configured, err := isCustomSCEPConfiguredForScript(ctx, appConfig, svc.ds, customSCEPCAs, caName,
				fleetVar)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "checking custom SCEP configuration")
			}
			if !configured {
				valid = false
				break
			}

		default:
			// Otherwise, error out since this variable is unknown
			return "", ctxerr.Errorf(
				ctx, "Unknown Fleet variable $FLEET_VAR_%s found in profile. Please update or remove.",
				fleetVar)
		}
	}

	if !valid {
		return "", ctxerr.Errorf(ctx, "profile references one or more variables corresponding to unconfigured settings")
	}

	// Fetch the host UUID, which may not be the same as the Enrollment ID, from the profile
	hostUUID := hostInfo.UUID
	failed := false

fleetVarLoop:
	for fleetVar := range fleetVars {
		var err error
		switch {
		case fleetVar == fleet.FleetVarNDESSCEPChallenge:
			if ndesConfig == nil {
				// Retrieve the NDES admin password. This is done once per run.
				configAssets, err := svc.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetNDESPassword}, nil)
				if err != nil {
					return "", ctxerr.Wrap(ctx, err, "getting NDES password")
				}
				// Copy config struct by value
				configWithPassword := appConfig.Integrations.NDESSCEPProxy.Value
				configWithPassword.Password = string(configAssets[fleet.MDMAssetNDESPassword].Value)
				// Store the config with the password for later use
				ndesConfig = &configWithPassword
			}
			// Insert the SCEP challenge into the profile contents
			challenge, err := scepConfig.GetNDESSCEPChallenge(ctx, *ndesConfig)
			if err != nil {
				detail := ""
				switch {
				case errors.As(err, &eeservice.NDESInvalidError{}):
					detail = fmt.Sprintf("Invalid NDES admin credentials. "+
						"Fleet couldn't populate $FLEET_VAR_%s. "+
						"Please update credentials in Settings > Integrations > Mobile Device Management > Simple Certificate Enrollment Protocol.",
						fleet.FleetVarNDESSCEPChallenge)
				case errors.As(err, &eeservice.NDESPasswordCacheFullError{}):
					detail = fmt.Sprintf("The NDES password cache is full. "+
						"Fleet couldn't populate $FLEET_VAR_%s. "+
						"Please increase the number of cached passwords in NDES and try again.",
						fleet.FleetVarNDESSCEPChallenge)
				case errors.As(err, &eeservice.NDESInsufficientPermissionsError{}):
					detail = fmt.Sprintf("This account does not have sufficient permissions to enroll with SCEP. "+
						"Fleet couldn't populate $FLEET_VAR_%s. "+
						"Please update the account with NDES SCEP enroll permissions and try again.",
						fleet.FleetVarNDESSCEPChallenge)
				default:
					detail = fmt.Sprintf("Fleet couldn't populate $FLEET_VAR_%s. %s", fleet.FleetVarNDESSCEPChallenge, err.Error())
				}
				return "", ctxerr.Wrap(ctx, err, detail)
			}

			contents = replaceFleetVariableInShellScript(fleetVarNDESSCEPChallengeRegexp, contents, challenge)

		case fleetVar == fleet.FleetVarNDESSCEPProxyURL:
			// Insert the SCEP URL into the profile contents
			proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), apple_mdm.SCEPProxyPath,
				url.PathEscape(fmt.Sprintf("%s,%s,NDES", hostUUID, execID)))
			contents = replaceFleetVariableInShellScript(fleetVarNDESSCEPProxyURLRegexp, contents, proxyURL)

		case fleetVar == fleet.FleetVarSCEPRenewalID:
			// Insert the SCEP renewal ID into the SCEP Payload CN
			fleetRenewalID := "fleet-" + execID
			contents = replaceFleetVariableInShellScript(fleetVarSCEPRenewalIDRegexp, contents, fleetRenewalID)

		case strings.HasPrefix(fleetVar, fleet.FleetVarCustomSCEPChallengePrefix):
			caName := strings.TrimPrefix(fleetVar, fleet.FleetVarCustomSCEPChallengePrefix)
			ca, ok := customSCEPCAs[caName]
			if !ok {
				level.Error(svc.logger).Log("msg", "Custom SCEP CA not found. "+
					"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
				continue
			}
			contents, err = replaceExactFleetPrefixVariableInShellScript(fleet.FleetVarCustomSCEPChallengePrefix, ca.Name, contents, ca.Challenge)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP challenge")
			}

		case strings.HasPrefix(fleetVar, fleet.FleetVarCustomSCEPProxyURLPrefix):
			caName := strings.TrimPrefix(fleetVar, fleet.FleetVarCustomSCEPProxyURLPrefix)
			ca, ok := customSCEPCAs[caName]
			if !ok {
				level.Error(svc.logger).Log("msg", "Custom SCEP CA not found. "+
					"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
				continue
			}
			// Generate a new SCEP challenge for the profile
			challenge, err := svc.ds.NewChallenge(ctx)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "generating SCEP challenge")
			}
			// Insert the SCEP URL into the profile contents
			proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), apple_mdm.SCEPProxyPath,
				url.PathEscape(fmt.Sprintf("%s,%s,%s,%s", hostUUID, execID, caName, challenge)))
			contents, err = replaceExactFleetPrefixVariableInShellScript(fleet.FleetVarCustomSCEPProxyURLPrefix, ca.Name, contents, proxyURL)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP proxy URL")
			}

		case fleetVar == fleet.FleetVarHostEndUserEmailIDP:
			return "", ctxerr.New(ctx, "User email not supported in shell scripts")

		case fleetVar == fleet.FleetVarHostHardwareSerial:
			contents = replaceFleetVariableInShellScript(
				fleetVarHostHardwareSerialRegexp,
				contents,
				hostInfo.HardwareSerial)

		case fleetVar == fleet.FleetVarHostEndUserIDPUsername || fleetVar == fleet.FleetVarHostEndUserIDPUsernameLocalPart ||
			fleetVar == fleet.FleetVarHostEndUserIDPGroups || fleetVar == fleet.FleetVarHostEndUserIDPDepartment:
			return "", ctxerr.New(ctx, "IDP fields not supported in shell scripts")

		case strings.HasPrefix(fleetVar, fleet.FleetVarDigiCertPasswordPrefix):
			// We will replace the password when we populate the certificate data

		case strings.HasPrefix(fleetVar, fleet.FleetVarDigiCertDataPrefix):
			caName := strings.TrimPrefix(fleetVar, fleet.FleetVarDigiCertDataPrefix)
			ca, ok := digiCertCAs[caName]
			if !ok {
				level.Error(svc.logger).Log("msg", "Custom DigiCert CA not found. "+
					"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
				continue
			}
			caCopy := *ca

			// Populate Fleet vars in the CA fields
			caVarsCache := make(map[string]string)
			ok, err := replaceFleetVarInItemFromHostInfo(ctx, svc.ds, hostInfo, caVarsCache, &caCopy.CertificateCommonName)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
			}
			if !ok {
				failed = true
				break fleetVarLoop
			}
			ok, err = replaceFleetVarInItemFromHostInfo(ctx, svc.ds, hostInfo, caVarsCache, &caCopy.CertificateSeatID)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
			}
			if !ok {
				failed = true
				break fleetVarLoop
			}
			if len(caCopy.CertificateUserPrincipalNames) > 0 {
				for i := range caCopy.CertificateUserPrincipalNames {
					ok, err = replaceFleetVarInItemFromHostInfo(ctx, svc.ds, hostInfo, caVarsCache, &caCopy.CertificateUserPrincipalNames[i])
					if err != nil {
						return "", ctxerr.Wrap(ctx, err, "populating Fleet variables in DigiCert CA common name")
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
				return "", ctxerr.New(ctx, detail)
			}
			contents, err = replaceExactFleetPrefixVariableInShellScript(fleet.FleetVarDigiCertDataPrefix, caName, contents,
				base64.StdEncoding.EncodeToString(cert.PfxData))
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "replacing Fleet variable for DigiCert data")
			}
			contents, err = replaceExactFleetPrefixVariableInShellScript(fleet.FleetVarDigiCertPasswordPrefix, caName, contents, cert.Password)
			if err != nil {
				return "", ctxerr.Wrap(ctx, err, "replacing Fleet variable for DigiCert password")
			}

		default:
			// This was handled in the above switch statement, so we should never reach this case
		}
	}
	if failed {
		return "", ctxerr.New(ctx, "variable injection failed")
	}

	return contents, nil
}
