package service

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/variables"
)

// escapeDNValue escapes special characters in a string value being substituted
// into an X.500 Distinguished Name or SAN, per RFC 4514 §2.4.
func escapeDNValue(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		switch {
		case r == ',' || r == '+' || r == '"' || r == '\\' || r == '<' || r == '>' || r == ';':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '#' && i == 0:
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == ' ' && (i == 0 || i == len(s)-1):
			b.WriteByte('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Fleet variables supported in certificate template subject names and SANs.
var fleetVarsSupportedInCertificateTemplates = []fleet.FleetVarName{
	fleet.FleetVarHostUUID,
	fleet.FleetVarHostHardwareSerial,
	fleet.FleetVarHostPlatform,
	fleet.FleetVarHostEndUserIDPUsername,
	fleet.FleetVarHostEndUserIDPUsernameLocalPart,
	fleet.FleetVarHostEndUserIDPGroups,
	fleet.FleetVarHostEndUserIDPDepartment,
	fleet.FleetVarHostEndUserIDPFullname,
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
// non-empty content on both sides, that each KEY is in the allowlist (DNS, EMAIL, UPN, IP,
// URI), and that at least one valid token is present (rejects separator-only inputs like ",").
// The value (right of '=') is otherwise not validated; value content is parsed by the Android
// agent at delivery time, where any $FLEET_VAR_* references have already been expanded.
//
// certName is suffixed onto each error reason as "(certificate <name>)" for GitOps multi-cert
// clarity; pass "" from single-cert callers like CreateCertificateTemplate where the failing
// cert is unambiguous. Returns a typed *fleet.InvalidArgumentError on failure (HTTP 422,
// errors[].name = "subject_alternative_name") or nil on success.
func validateCertificateTemplateSubjectAlternativeName(san, certName string) error {
	const field = "subject_alternative_name"
	mkErr := func(reason string) error {
		if certName != "" {
			reason = fmt.Sprintf("%s (certificate %s)", reason, certName)
		}
		return fleet.NewInvalidArgumentError(field, reason)
	}
	if strings.TrimSpace(san) == "" {
		return nil
	}
	if len(san) > maxCertificateTemplateSubjectAlternativeNameLength {
		return mkErr(fmt.Sprintf("is too long. Maximum is %d bytes",
			maxCertificateTemplateSubjectAlternativeNameLength))
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
			return mkErr(fmt.Sprintf("token %q is missing '='", token))
		}
		if eqIdx == 0 {
			return mkErr(fmt.Sprintf("token %q has an empty key", token))
		}
		key := strings.ToUpper(strings.TrimSpace(token[:eqIdx]))
		if _, ok := subjectAlternativeNameAllowedKeys[key]; !ok {
			return mkErr(fmt.Sprintf("has unsupported key %q. Allowed keys are DNS, EMAIL, UPN, IP, URI", key))
		}
		if strings.TrimSpace(token[eqIdx+1:]) == "" {
			return mkErr(fmt.Sprintf("token %q has an empty value", token))
		}
	}
	if tokensSeen == 0 {
		return mkErr("contains no entries")
	}
	return nil
}

// replaceCertificateVariables replaces FLEET_VAR_* variables in the input string with actual
// host values. endUsersMemo is an optional cross-call cache for the host's end-user list — pass
// the same `*[]fleet.HostEndUser` (with `*memo == nil` initially) into successive calls for the
// same host to avoid re-fetching from the datastore. The IDP related variable is the only one
// that triggers a DB round-trip; UUID and hardware serial come from the in-memory host struct.
func (svc *Service) replaceCertificateVariables(ctx context.Context, input string, host *fleet.Host, endUsersMemo *[]fleet.HostEndUser) (string, error) {
	fleetVars := variables.Find(input)
	if len(fleetVars) == 0 {
		return input, nil
	}

	// fetchEndUsers lazily fetches and caches the host's end-user list.
	fetchEndUsers := func(fleetVar string) ([]fleet.HostEndUser, error) {
		if endUsersMemo != nil && *endUsersMemo != nil {
			return *endUsersMemo, nil
		}
		fetched, err := fleet.GetEndUsers(ctx, svc.ds, host.ID)
		if err != nil {
			return nil, ctxerr.Wrapf(ctx, err, "getting host end users for variable %s", fleetVar)
		}
		if endUsersMemo != nil {
			if fetched == nil {
				fetched = []fleet.HostEndUser{}
			}
			*endUsersMemo = fetched
		}
		return fetched, nil
	}

	// requireIDPUser fetches end users and returns the first IDP user, or an error if none.
	requireIDPUser := func(fleetVar string) (*fleet.HostEndUser, error) {
		users, err := fetchEndUsers(fleetVar)
		if err != nil {
			return nil, err
		}
		if len(users) == 0 || users[0].IdpUserName == "" {
			return nil, ctxerr.Errorf(ctx, "host %s does not have an IDP user for variable %s", host.UUID, fleetVar)
		}
		return &users[0], nil
	}

	result := input
	for _, fleetVar := range fleetVars {
		switch fleetVar {
		case string(fleet.FleetVarHostUUID):
			if host.UUID == "" {
				return "", ctxerr.Errorf(ctx, "host does not have a UUID for variable %s", fleetVar)
			}
			result = fleet.FleetVarHostUUIDRegexp.ReplaceAllString(result, escapeDNValue(host.UUID))
		case string(fleet.FleetVarHostHardwareSerial):
			if host.HardwareSerial == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have a hardware serial for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostHardwareSerialRegexp.ReplaceAllString(result, escapeDNValue(host.HardwareSerial))
		case string(fleet.FleetVarHostPlatform):
			if host.Platform == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have a platform for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostPlatformRegexp.ReplaceAllString(result, escapeDNValue(host.Platform))
		case string(fleet.FleetVarHostEndUserIDPUsername):
			user, err := requireIDPUser(fleetVar)
			if err != nil {
				return "", err
			}
			result = fleet.FleetVarHostEndUserIDPUsernameRegexp.ReplaceAllString(result, escapeDNValue(user.IdpUserName))
		case string(fleet.FleetVarHostEndUserIDPUsernameLocalPart):
			user, err := requireIDPUser(fleetVar)
			if err != nil {
				return "", err
			}
			local, _, _ := strings.Cut(user.IdpUserName, "@")
			result = fleet.FleetVarHostEndUserIDPUsernameLocalPartRegexp.ReplaceAllString(result, escapeDNValue(local))
		case string(fleet.FleetVarHostEndUserIDPGroups):
			user, err := requireIDPUser(fleetVar)
			if err != nil {
				return "", err
			}
			if len(user.IdpGroups) == 0 {
				return "", ctxerr.Errorf(ctx, "host %s does not have IDP groups for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPGroupsRegexp.ReplaceAllString(result, escapeDNValue(strings.Join(user.IdpGroups, ",")))
		case string(fleet.FleetVarHostEndUserIDPDepartment):
			user, err := requireIDPUser(fleetVar)
			if err != nil {
				return "", err
			}
			if user.Department == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have an IDP department for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPDepartmentRegexp.ReplaceAllString(result, escapeDNValue(user.Department))
		case string(fleet.FleetVarHostEndUserIDPFullname):
			user, err := requireIDPUser(fleetVar)
			if err != nil {
				return "", err
			}
			fullName := strings.TrimSpace(user.IdpFullName)
			if fullName == "" {
				return "", ctxerr.Errorf(ctx, "host %s does not have an IDP full name for variable %s", host.UUID, fleetVar)
			}
			result = fleet.FleetVarHostEndUserIDPFullnameRegexp.ReplaceAllString(result, escapeDNValue(fullName))
		default:
			return "", ctxerr.Errorf(ctx, "unsupported Fleet variable %s in certificate template", fleetVar)
		}
	}

	return result, nil
}

// expandCertVar runs replaceCertificateVariables on `input` and on success returns
// (expanded, true, nil). On expansion failure (host missing UUID / hardware serial / IdP user, or
// an unsupported variable), it persists a CertificateStatusFailed update for the host's row in
// host_certificate_templates and mutates `certificate` in place: status -> failed, SCEPChallenge
// and FleetChallenge -> nil so a previously-delivered template's active challenges do not ride
// along on the failed response. The caller should check ok and return the failed certificate
// without further processing.
//
// detailPrefix is appended to the persisted Detail field, e.g.
// "Could not replace certificate variables" for subject_name, or
// "Could not replace certificate variables in subject_alternative_name" for SAN. It is the
// caller's responsibility to keep these prefixes stable across releases since downstream tooling
// may match on them.
func (svc *Service) expandCertVar(
	ctx context.Context,
	certificate *fleet.CertificateTemplateResponseForHost,
	input string,
	detailPrefix string,
	host *fleet.Host,
	endUsersMemo *[]fleet.HostEndUser,
) (expanded string, ok bool, err error) {
	expanded, expandErr := svc.replaceCertificateVariables(ctx, input, host, endUsersMemo)
	if expandErr == nil {
		return expanded, true, nil
	}
	errorMsg := fmt.Sprintf("%s: %s", detailPrefix, expandErr.Error())
	if upsertErr := svc.ds.UpsertCertificateStatus(ctx, &fleet.CertificateStatusUpdate{
		HostUUID:              host.UUID,
		CertificateTemplateID: certificate.ID,
		Status:                fleet.MDMDeliveryFailed,
		Detail:                &errorMsg,
		OperationType:         fleet.MDMOperationTypeInstall,
	}); upsertErr != nil {
		return "", false, upsertErr
	}
	certificate.Status = fleet.CertificateTemplateFailed
	certificate.SCEPChallenge = nil
	certificate.FleetChallenge = nil
	return "", false, nil
}
