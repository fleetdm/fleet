//go:build darwin
// +build darwin

package app_sso_platform

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/fleetdm/fleet/v4/orbit/pkg/user"
	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// Columns is the schema of the table.
func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		// Extension identifier of the Platform SSO extension (e.g. "com.microsoft.CompanyPortalMac.ssoextension").
		// Required column, currently supports setting this once per query.
		table.TextColumn("extension_identifier"),
		// Realm of the user that logged via Platform SSO (e.g. "KERBEROS.MICROSOFTONLINE.COM").
		// Required column, currently supports setting this once per query.
		table.TextColumn("realm"),
		// Device ID extracted from "Device Configuration" -> "deviceSigningCertificate" -> Subject -> CommonName.
		table.TextColumn("device_id"),
		// User principal name of the user that logged in via Platform SSO.
		table.TextColumn("user_principal_name"),
	}
}

// Generate is called to return the results for the table at query time.
//
// Constraints for generating can be retrieved from the queryContext.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	extensionIdentifierConstraints, ok := queryContext.Constraints["extension_identifier"]
	if !ok || len(extensionIdentifierConstraints.Constraints) == 0 {
		return nil, errors.New("missing extension_identifier")
	}

	var expectedExtensionIdentifiers []string
	for _, constraint := range extensionIdentifierConstraints.Constraints {
		if constraint.Operator != table.OperatorEquals {
			return nil, errors.New("only supported operator for 'extension_identifier' is '='")
		}
		if constraint.Expression == "" {
			continue
		}
		expectedExtensionIdentifiers = append(expectedExtensionIdentifiers, constraint.Expression)
	}
	if len(expectedExtensionIdentifiers) == 0 {
		return nil, errors.New("missing extension_identifier")
	} else if len(expectedExtensionIdentifiers) > 1 {
		return nil, errors.New("only one extension_identifier can be set")
	}

	realmConstraints, ok := queryContext.Constraints["realm"]
	if !ok || len(realmConstraints.Constraints) == 0 {
		return nil, errors.New("missing realm")
	}

	var expectedRealms []string
	for _, constraint := range realmConstraints.Constraints {
		if constraint.Operator != table.OperatorEquals {
			return nil, errors.New("only supported operator for 'realm' is '='")
		}
		if constraint.Expression == "" {
			continue
		}
		expectedRealms = append(expectedRealms, constraint.Expression)
	}
	if len(expectedRealms) == 0 {
		return nil, errors.New("missing realm")
	} else if len(expectedRealms) > 1 {
		return nil, errors.New("only one realm can be set")
	}

	loggedInUser, err := user.UserLoggedInViaGui()
	if err != nil {
		return nil, fmt.Errorf("failed to check user logged in: %w", err)
	}
	if loggedInUser == nil || *loggedInUser == "" {
		// User is not logged in, nothing to do so we return no results.
		return nil, nil
	}

	output, err := executeAppSSOPlatform(*loggedInUser)
	if err != nil {
		return nil, fmt.Errorf("failed to execute app-sso platform: %w", err)
	}

	appSSOPlatform, err := parseAppSSOPlatformCommandOutput(output, expectedExtensionIdentifiers[0], expectedRealms[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse \"app-sso platform --state\" output: %w", err)
	}
	if appSSOPlatform == nil {
		// Device not registered, nothing to do so we return no results.
		return nil, nil
	}

	return []map[string]string{{
		"extension_identifier": appSSOPlatform.extensionIdentifier,
		"realm":                appSSOPlatform.realm,
		"device_id":            appSSOPlatform.deviceID,
		"user_principal_name":  appSSOPlatform.userPrincipalName,
	}}, nil
}

func executeAppSSOPlatform(loggedInUser string) ([]byte, error) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf(`launchctl asuser $(id -u "%s") sudo -iu "%s" /usr/bin/app-sso platform --state`, loggedInUser, loggedInUser)) // #nosec G20: loggedInUser is not controlled by user.
	return cmd.Output()
}

var (
	// deviceRe extracts JSON after "Device Configuration:" and before "Login Configuration:"
	deviceRe = regexp.MustCompile(`(?s)Device Configuration:\n\s(\{.*?\}|\(null\))\n\nLogin Configuration:`)
	// userRe extracts JSON after "User Configuration:" and before "SSO Tokens:" (or end of string)
	userRe = regexp.MustCompile(`(?s)User Configuration:\n\s(\{.*?\}|\(null\))\n\n`)
)

// extractJSONSections finds JSON blocks for "Device Configuration" and "User Configuration".
func extractJSONSections(s []byte) (deviceConfig string, userConfig string, err error) {
	deviceMatch := deviceRe.FindSubmatch(s)
	userMatch := userRe.FindSubmatch(s)

	if len(deviceMatch) < 2 {
		return "", "", errors.New("match for \"Device Configuration\" not found")
	}
	if len(userMatch) < 2 {
		return "", "", errors.New("match for \"User Configuration\" JSON not found")
	}

	return string(deviceMatch[1]), string(userMatch[1]), nil
}

type appSSOPlatformData struct {
	extensionIdentifier string
	deviceID            string
	realm               string
	userPrincipalName   string
}

func parseAppSSOPlatformCommandOutput(output []byte, expectedExtensionIdentifier string, expectedRealm string) (*appSSOPlatformData, error) {
	deviceConfigJSON, userConfigJSON, err := extractJSONSections(output)
	if err != nil {
		return nil, fmt.Errorf("could not extract JSON sections: %w", err)
	}
	if deviceConfigJSON == "(null)" {
		log.Debug().Msg("device not registered")
		return nil, nil
	}
	deviceConfig := struct {
		DeviceSigningCertificate string `json:"deviceSigningCertificate"`
		ExtensionIdentifier      string `json:"extensionIdentifier"`
	}{}
	if err := json.Unmarshal([]byte(deviceConfigJSON), &deviceConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal \"Device Configuration\" JSON: %w", err)
	}
	if expectedExtensionIdentifier != deviceConfig.ExtensionIdentifier {
		log.Debug().Str("extensionIdentifier", deviceConfig.ExtensionIdentifier).Msg("device registered, but found unmatched extension")
		return nil, nil
	}
	dsc, err := base64.RawURLEncoding.DecodeString(deviceConfig.DeviceSigningCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to decode \"deviceSigningCertificate\": %w", err)
	}
	deviceSigningCertificate, err := x509.ParseCertificate(dsc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse \"deviceSigningCertificate\": %w", err)
	}
	if deviceSigningCertificate.Subject.CommonName == "" {
		return nil, errors.New("empty subject common name in \"deviceSigningCertificate\"")
	}
	log.Debug().Str(
		"\"Device Configuration\"", deviceSigningCertificate.Subject.CommonName,
	).Msg("found device ID")
	userConfig := struct {
		KerberosStatus []map[string]any `json:"kerberosStatus"`
	}{}
	if userConfigJSON == "(null)" {
		log.Debug().Msg("user not registered")
		return &appSSOPlatformData{
			extensionIdentifier: deviceConfig.ExtensionIdentifier,
			deviceID:            deviceSigningCertificate.Subject.CommonName,
			realm:               expectedRealm,
			userPrincipalName:   "",
		}, nil
	}
	if err := json.Unmarshal([]byte(userConfigJSON), &userConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal \"User Configuration\" JSON: %w", err)
	}
	if len(userConfig.KerberosStatus) == 0 {
		return nil, errors.New("\"kerberosStatus\" has no entries")
	}
	realm_, ok := userConfig.KerberosStatus[0]["realm"]
	if !ok {
		return nil, errors.New("missing \"realm\" key in \"kerberosStatus\"")
	}
	realm, ok := realm_.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for \"realm\" key in \"kerberosStatus\": %T", err)
	}
	upn_, ok := userConfig.KerberosStatus[0]["upn"]
	if !ok {
		return nil, errors.New("missing \"upn\" key in \"kerberosStatus\"")
	}
	upn, ok := upn_.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected type for \"upn\" key in \"kerberosStatus\": %T", err)
	}
	if upn == "" {
		return nil, errors.New("empty \"upn\" key in \"kerberosStatus\"")
	}
	if expectedRealm != realm {
		log.Debug().Str("realm", realm).Msg("user registered, but found unmatched realm")
		return &appSSOPlatformData{
			extensionIdentifier: deviceConfig.ExtensionIdentifier,
			deviceID:            deviceSigningCertificate.Subject.CommonName,
			realm:               expectedRealm,
			userPrincipalName:   "",
		}, nil
	}
	suffix := fmt.Sprintf("@%s", realm)
	upn = strings.TrimSuffix(upn, suffix)
	upn = strings.ReplaceAll(upn, "\\@", "@")
	log.Debug().Str(
		"extension_identifier", deviceConfig.ExtensionIdentifier,
	).Str(
		"device_id", deviceSigningCertificate.Subject.CommonName,
	).Str(
		"realm", realm,
	).Str(
		"user_principal_name", upn,
	).Msg("device and user found")
	return &appSSOPlatformData{
		extensionIdentifier: deviceConfig.ExtensionIdentifier,
		deviceID:            deviceSigningCertificate.Subject.CommonName,
		realm:               realm,
		userPrincipalName:   upn,
	}, nil
}
