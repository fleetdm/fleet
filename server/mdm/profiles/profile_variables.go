package profiles

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/ptr"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

/*
This file contains functions to replace profile variables in MDM profiles, that are supported
on multiple platforms, so it can be shared.

Fleet variables supported across systems:
- $FLEET_VAR_CUSTOM_SCEP_CHALLENGE_<CA_NAME>
- $FLEET_VAR_CUSTOM_SCEP_PROXY_URL_<CA_NAME>
- $FLEET_VAR_HOST_END_USER_EMAIL_IDP

Once more is needed it should be placed here, and the main replacement logic can be taken from the apple_mdm.go
under server/service folder. Inside the `preprocessProfileContents` under the `fleetVarLoop` loop.
*/

func ReplaceCustomSCEPChallengeVariable(ctx context.Context, logger kitlog.Logger, fleetVariable string, customSCEPCAs map[string]*fleet.CustomSCEPProxyCA, profileContents string) (contents string, replacedVariable bool, err error) {
	caName := strings.TrimPrefix(fleetVariable, string(fleet.FleetVarCustomSCEPChallengePrefix))
	ca, ok := customSCEPCAs[caName]
	if !ok {
		level.Error(logger).Log("msg", "Custom SCEP CA not found. "+
			"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
		return "", false, nil
	}
	contents, err = ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarCustomSCEPChallengePrefix), ca.Name, profileContents, ca.Challenge)
	if err != nil {
		return "", false, ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP challenge")
	}
	return contents, true, nil
}

func ReplaceCustomSCEPProxyURLVariable(ctx context.Context, logger kitlog.Logger, ds fleet.Datastore, appConfig *fleet.AppConfig,
	fleetVar string, customSCEPCAs map[string]*fleet.CustomSCEPProxyCA, profileContents string,
	hostUUID string, profUUID string,
) (contents string, managedCertificate *fleet.MDMManagedCertificate, replacedVariable bool, err error) {
	caName := strings.TrimPrefix(fleetVar, string(fleet.FleetVarCustomSCEPProxyURLPrefix))
	ca, ok := customSCEPCAs[caName]
	if !ok {
		level.Error(logger).Log("msg", "Custom SCEP CA not found. "+
			"This error should never happen since we validated/populated CAs earlier", "ca_name", caName)
		return "", nil, false, nil
	}
	// Generate a new SCEP challenge for the profile
	challenge, err := ds.NewChallenge(ctx)
	if err != nil {
		return "", nil, false, ctxerr.Wrap(ctx, err, "generating SCEP challenge")
	}
	// Insert the SCEP URL into the profile contents
	proxyURL := fmt.Sprintf("%s%s%s", appConfig.MDMUrl(), apple_mdm.SCEPProxyPath,
		url.PathEscape(fmt.Sprintf("%s,%s,%s,%s", hostUUID, profUUID, caName, challenge)))
	contents, err = ReplaceExactFleetPrefixVariableInXML(string(fleet.FleetVarCustomSCEPProxyURLPrefix), ca.Name, profileContents, proxyURL)
	if err != nil {
		return "", nil, false, ctxerr.Wrap(ctx, err, "replacing Fleet variable for SCEP proxy URL")
	}

	managedCertificate = &fleet.MDMManagedCertificate{
		HostUUID:    hostUUID,
		ProfileUUID: profUUID,
		Type:        fleet.CAConfigCustomSCEPProxy,
		CAName:      caName,
	}

	return contents, managedCertificate, true, nil
}

func ReplaceHostEndUserIDPVariables(ctx context.Context, ds fleet.Datastore,
	fleetVar string, profileContents string, hostUUID string,
	hostIDForUUIDCache map[string]uint,
	onError func(errMsg string) error,
) (replacedContents string, replacedVariable bool, err error) {
	user, ok, err := getHostEndUserIDPUser(ctx, ds, hostUUID, fleetVar, hostIDForUUIDCache, onError)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	var rx *regexp.Regexp
	var value string
	switch fleetVar {
	case string(fleet.FleetVarHostEndUserIDPUsername):
		rx = fleet.FleetVarHostEndUserIDPUsernameRegexp
		value = user.IdpUserName
	case string(fleet.FleetVarHostEndUserIDPUsernameLocalPart):
		rx = fleet.FleetVarHostEndUserIDPUsernameLocalPartRegexp
		value = getEmailLocalPart(user.IdpUserName)
	case string(fleet.FleetVarHostEndUserIDPGroups):
		rx = fleet.FleetVarHostEndUserIDPGroupsRegexp
		value = strings.Join(user.IdpGroups, ",")
	case string(fleet.FleetVarHostEndUserIDPDepartment):
		rx = fleet.FleetVarHostEndUserIDPDepartmentRegexp
		value = user.Department
	case string(fleet.FleetVarHostEndUserIDPFullname):
		rx = fleet.FleetVarHostEndUserIDPFullnameRegexp
		value = strings.TrimSpace(user.IdpFullName)
	}
	replacedContents = ReplaceFleetVariableInXML(rx, profileContents, value)

	return replacedContents, true, nil
}

func getHostEndUserIDPUser(ctx context.Context, ds fleet.Datastore,
	hostUUID, fleetVar string, hostIDForUUIDCache map[string]uint,
	onError func(errMsg string) error,
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
			// hosts with the same UUID.
			return nil, false, onError(fmt.Sprintf("Unexpected number of hosts (%d) for UUID %s. ", len(ids), hostUUID))
		}
		hostID = ids[0]
		hostIDForUUIDCache[hostUUID] = hostID
	}

	users, err := fleet.GetEndUsers(ctx, ds, hostID)
	if err != nil {
		return nil, false, ctxerr.Wrap(ctx, err, "get end users for host")
	}

	noGroupsErr := fmt.Sprintf("There are no IdP groups for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPGroups)
	noDepartmentErr := fmt.Sprintf("There is no IdP department for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPDepartment)
	noFullnameErr := fmt.Sprintf("There is no IdP full name for this host. Fleet couldn't populate $FLEET_VAR_%s.", fleet.FleetVarHostEndUserIDPFullname)
	if len(users) > 0 && users[0].IdpUserName != "" {
		idpUser := users[0]

		if fleetVar == string(fleet.FleetVarHostEndUserIDPGroups) && len(idpUser.IdpGroups) == 0 {
			return nil, false, onError(noGroupsErr)
		}
		if fleetVar == string(fleet.FleetVarHostEndUserIDPDepartment) && idpUser.Department == "" {
			return nil, false, onError(noDepartmentErr)
		}
		if fleetVar == string(fleet.FleetVarHostEndUserIDPFullname) && strings.TrimSpace(idpUser.IdpFullName) == "" {
			return nil, false, onError(noFullnameErr)
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
	return nil, false, onError(detail)
}

func getEmailLocalPart(email string) string {
	// if there is a "@" in the email, return the part before that "@", otherwise
	// return the string unchanged.
	local, _, _ := strings.Cut(email, "@")
	return local
}

func ReplaceExactFleetPrefixVariableInXML(prefix string, suffix string, contents string, replacement string) (string, error) {
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

func ReplaceFleetVariableInXML(regExp *regexp.Regexp, contents string, replacement string) string {
	// Escape XML characters since this replacement is intended for XML profile.
	b := make([]byte, 0, len(replacement))
	buf := bytes.NewBuffer(b)
	// error is always nil for Buffer.Write method, so we ignore it
	_ = xml.EscapeText(buf, []byte(replacement))
	return regExp.ReplaceAllLiteralString(contents, buf.String())
}

func IsCustomSCEPConfigured(ctx context.Context,
	customSCEPCAs map[string]*fleet.CustomSCEPProxyCA, caName string, fleetVar string,
	onError func(string) error, // A function that allows the caller to run some code on errors, if an error is returned it will be returned by IsCustomSCEPConfigured
) error {
	if !license.IsPremium(ctx) {
		return onError("Custom SCEP integration requires a Fleet Premium license.")
	}
	if _, ok := customSCEPCAs[caName]; !ok {
		return onError(fmt.Sprintf("Fleet couldn't populate $%s because %s certificate authority doesn't exist.", fleetVar, caName))
	}

	return nil
}
