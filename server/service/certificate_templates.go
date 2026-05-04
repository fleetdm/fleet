package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// Fleet variables supported in certificate template subject names and SANs.
var fleetVarsSupportedInCertificateTemplates = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarHostEndUserIDPUsername,
}

// maxCertificateTemplateSubjectAlternativeNameLength caps the SAN string length to prevent
// pathological inputs. 4096 bytes accommodates real-world SAN lists (a handful of DNS / UPN /
// EMAIL / IP / URI entries) with comfortable headroom.
const maxCertificateTemplateSubjectAlternativeNameLength = 4096

// subjectAlternativeNameAllowedKeys lists the SAN attribute KEYs the agent recognizes. The
// server validates KEY membership at create time so admins get fast feedback on typos.
var subjectAlternativeNameAllowedKeys = map[string]struct{}{
	"DNS":   {},
	"EMAIL": {},
	"UPN":   {},
	"IP":    {},
	"URI":   {},
}

func validateCertificateTemplateFleetVariables(subjectName string) error {
	fleetVars := variables.Find(subjectName)
	if len(fleetVars) == 0 {
		return nil
	}

	for _, fleetVar := range fleetVars {
		if !slices.Contains(fleetVarsSupportedInCertificateTemplates, fleet.FleetVarName(fleetVar)) {
			return fmt.Errorf("Fleet variable $FLEET_VAR_%s is not supported in certificate templates", fleetVar)
		}
	}

	return nil
}

// validateCertificateTemplateSubjectAlternativeName performs lightweight format-only validation
// of the SAN string. Empty / whitespace-only input is permitted (means no SAN). For non-empty
// values it checks the length cap, that each non-empty comma-separated token contains '=' with
// non-empty content on both sides, that each KEY is in the allow-list (DNS, EMAIL, UPN, IP,
// URI), and that at least one valid token is present (rejects separator-only inputs like ",").
// The value (right of '=') is otherwise not validated; value content is parsed by the Android
// agent at delivery time, where any $FLEET_VAR_* references have already been expanded.
func validateCertificateTemplateSubjectAlternativeName(san string) error {
	if strings.TrimSpace(san) == "" {
		return nil
	}
	if len(san) > maxCertificateTemplateSubjectAlternativeNameLength {
		return fmt.Errorf("subject_alternative_name is too long. Maximum is %d bytes.",
			maxCertificateTemplateSubjectAlternativeNameLength)
	}
	tokensSeen := 0
	for raw := range strings.SplitSeq(san, ",") {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		tokensSeen++
		eqIdx := strings.Index(token, "=")
		if eqIdx == -1 {
			return fmt.Errorf("subject_alternative_name token %q is missing '='", token)
		}
		if eqIdx == 0 {
			return fmt.Errorf("subject_alternative_name token %q has an empty key", token)
		}
		key := strings.ToUpper(strings.TrimSpace(token[:eqIdx]))
		if _, ok := subjectAlternativeNameAllowedKeys[key]; !ok {
			return fmt.Errorf(
				"subject_alternative_name has unsupported key %q. Allowed keys are DNS, EMAIL, UPN, IP, URI.",
				key)
		}
		if strings.TrimSpace(token[eqIdx+1:]) == "" {
			return fmt.Errorf("subject_alternative_name token %q has an empty value", token)
		}
	}
	if tokensSeen == 0 {
		return errors.New("subject_alternative_name contains no entries")
	}
	return nil
}

// replaceCertificateVariables replaces FLEET_VAR_* variables in the subject name with actual host values
func (svc *Service) replaceCertificateVariables(ctx context.Context, subjectName string, host *fleet.Host) (string, error) {
	fleetVars := variables.Find(subjectName)
	if len(fleetVars) == 0 {
		return subjectName, nil
	}

	result := subjectName
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostUUID):
			if host.UUID == "" {
				return "", ctxerr.Errorf(ctx, "host does not have a UUID for variable %s", fleetVar)
			}
			result = fleet.FleetVarHostUUIDRegexp.ReplaceAllString(result, host.UUID)
		case string(fleet.FleetVarHostHardwareSerial):
			if host.HardwareSerial == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have a hardware serial for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostHardwareSerialRegexp.ReplaceAllString(result, host.HardwareSerial)
		case string(fleet.FleetVarHostEndUserIDPUsername):
			users, err := fleet.GetEndUsers(ctx, svc.ds, host.ID)
			if err != nil {
				return "", ctxerr.Wrapf(ctx, err, "getting host end users for variable %s", fleetVar)
			}
			if len(users) == 0 || users[0].IdpUserName == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have an IDP username for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPUsernameRegexp.ReplaceAllString(result, users[0].IdpUserName)
		default:
			return "", ctxerr.Errorf(ctx, "unsupported Fleet variable %s in certificate template", fleetVar)
		}
	}

	return result, nil
}
