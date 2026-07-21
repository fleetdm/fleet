package fleet

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/variables"
)

const maxHostNameTemplateLength = 255

// MaxResolvedHostNameBytes is Apple's byte limit for a device name (the resolved
// host name). A resolved name longer than this can't be applied, so the cron
// fails those rows; ValidateHostNameTemplate also rejects a template whose fixed
// text alone already exceeds it.
const MaxResolvedHostNameBytes = 63

// hostIdentityVarsInNameTemplates are the built-in variables resolved purely from
// the in-memory host struct.
var hostIdentityVarsInNameTemplates = []FleetVarName{
	FleetVarHostHardwareSerial,
	FleetVarHostUUID,
	FleetVarHostPlatform,
}

// idpVarsInNameTemplates are the built-in IdP end-user variables.
var idpVarsInNameTemplates = []FleetVarName{
	FleetVarHostEndUserIDPUsername,
	FleetVarHostEndUserIDPUsernameLocalPart,
	FleetVarHostEndUserIDPGroups,
	FleetVarHostEndUserIDPDepartment,
	FleetVarHostEndUserIDPFullname,
}

// fleetVarsSupportedInHostNameTemplates is the allow-list of built-in Fleet
// variables that may be used in a host name template.
var fleetVarsSupportedInHostNameTemplates = slices.Concat(hostIdentityVarsInNameTemplates, idpVarsInNameTemplates)

// nameTemplateVarRegexp matches every supported built-in name-template variable
// (identity + IdP) in both its $FLEET_VAR_NAME and ${FLEET_VAR_NAME} forms.
var nameTemplateVarRegexp = varAlternationRegexp(fleetVarsSupportedInHostNameTemplates)

// nameTemplateIdentityVarRegexp matches only the host-identity variables.
// ResolveHostNameTemplate uses it so it substitutes identity variables and leaves
// IdP tokens untouched for the service-layer resolver.
var nameTemplateIdentityVarRegexp = varAlternationRegexp(hostIdentityVarsInNameTemplates)

// IsHostNameTemplateIDPVar reports whether name (a built-in variable name without
// the FLEET_VAR_ prefix, as returned by variables.Find) is an IdP end-user
// variable supported in host name templates.
func IsHostNameTemplateIDPVar(name string) bool {
	return slices.Contains(idpVarsInNameTemplates, FleetVarName(name))
}

// varAlternationRegexp builds a regexp matching any of the given variables in both
// the $FLEET_VAR_NAME and ${FLEET_VAR_NAME} forms.
func varAlternationRegexp(vars []FleetVarName) *regexp.Regexp {
	alts := make([]string, len(vars))
	for i, v := range vars {
		alts[i] = regexp.QuoteMeta(string(v))
	}
	alt := strings.Join(alts, "|")
	return regexp.MustCompile(fmt.Sprintf(`\$FLEET_VAR_(%[1]s)\b|\$\{FLEET_VAR_(%[1]s)\}`, alt))
}

// nameTemplateSecretRegexp matches a $FLEET_SECRET_NAME / ${FLEET_SECRET_NAME}
// custom (secret) variable token. Secret values are only known at resolve time
// so this is used to strip secret tokens out of a template when computing its fixed-text byte floor.
var nameTemplateSecretRegexp = regexp.MustCompile(`\$` + ServerSecretPrefix + `\w+|\$\{` + ServerSecretPrefix + `\w+\}`)

// nameTemplateVitalRegexp matches a $FLEET_HOST_VITAL_<id> / ${FLEET_HOST_VITAL_<id>}
// custom host vital token. Vital values, like secrets, are only known at
// resolve time, so this is used to strip vital tokens out of a template when
// computing its fixed-text byte floor.
var nameTemplateVitalRegexp = regexp.MustCompile(`\$` + CustomHostVitalPrefix + `\w+|\$\{` + CustomHostVitalPrefix + `\w+\}`)

// ValidateHostNameTemplate validates a host name template and returns the
// normalized (trimmed) template that callers should persist.
func ValidateHostNameTemplate(tmpl string) (string, error) {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return "", NewInvalidArgumentError("name_template", "Host name template can't be empty.")
	}
	if !utf8.ValidString(tmpl) {
		return "", NewInvalidArgumentError("name_template", "Host name template must be valid UTF-8.")
	}
	if utf8.RuneCountInString(tmpl) > maxHostNameTemplateLength {
		return "", NewInvalidArgumentError("name_template", "Host name template can't be longer than 255 characters.")
	}
	for _, r := range tmpl {
		// Reject C0/C1 control characters (Cc) as well as Unicode "format"
		// characters (Cf, e.g. bidi overrides and zero-width joiners) that can be
		// used to spoof a name displayed to admins in the UI.
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return "", NewInvalidArgumentError("name_template", "Host name template can't contain control characters.")
		}
	}

	// Every built-in Fleet variable used must be in the allow-list.
	for _, v := range variables.Find(tmpl) {
		if !slices.Contains(fleetVarsSupportedInHostNameTemplates, FleetVarName(v)) {
			return "", NewInvalidArgumentError("name_template",
				"Fleet variable $FLEET_VAR_"+v+" is not supported in host name templates.")
		}
	}

	// The resolved name must fit Apple's device name limit. Stripping the
	// variables yields the shortest a resolved name can be (a variable may
	// resolve to an empty value — including a secret, which can be set to an
	// empty string), so if the fixed text alone exceeds the limit no host can
	// ever get a valid name — reject it now rather than silently failing every
	// host at resolve time. Per-host overflow from variable/secret expansion is
	// still caught by the cron when it resolves against a host's actual values.
	literal := nameTemplateVarRegexp.ReplaceAllString(tmpl, "")
	literal = nameTemplateSecretRegexp.ReplaceAllString(literal, "")
	literal = nameTemplateVitalRegexp.ReplaceAllString(literal, "")
	if len(literal) > MaxResolvedHostNameBytes {
		return "", NewInvalidArgumentError("name_template",
			fmt.Sprintf("Host name template's fixed text can't be longer than %d bytes (the device name limit).", MaxResolvedHostNameBytes))
	}

	return tmpl, nil
}

// ValidateHostNameTemplateWithSecrets validates a host name template
// syntactically (see ValidateHostNameTemplate) and additionally verifies that
// every custom (secret, $FLEET_SECRET_*) variable it references is defined in
// the datastore, mirroring how scripts and profiles validate embedded secrets at
// save time, and that every custom host vital ($FLEET_HOST_VITAL_<id>) it
// references is a known vital ID. It returns the normalized template to
// persist.
func ValidateHostNameTemplateWithSecrets(ctx context.Context, ds Datastore, tmpl string) (string, error) {
	validated, err := ValidateHostNameTemplate(tmpl)
	if err != nil {
		return "", err
	}
	if len(ContainsPrefixVars(validated, ServerSecretPrefix)) > 0 {
		if err := ds.ValidateEmbeddedSecrets(ctx, []string{validated}); err != nil {
			// A referenced-but-undefined secret is a user input error (422); surface
			// the underlying message (which names the missing secret) as an
			// invalid-argument error. Any other error (e.g. a DB failure) is an
			// infrastructure problem and must propagate as-is (500), not be
			// misreported as invalid input.
			if IsMissingSecretsError(err) {
				return "", NewInvalidArgumentError("name_template", err.Error())
			}
			return "", err
		}
	}
	// Vital IDs are dynamic (not a fixed allow-list), so an unknown or malformed
	// $FLEET_HOST_VITAL_<id> reference is only caught here, same as scripts and
	// profiles validate their own embedded vital references.
	if err := ds.ValidateReferencedCustomHostVitals(ctx, []string{validated}); err != nil {
		if IsInvalidReferencedCustomHostVitalsError(err) {
			return "", NewInvalidArgumentError("name_template", err.Error())
		}
		return "", err
	}
	return validated, nil
}

// hostNameTemplatePlatformDisplayNames maps a host's osquery platform to the
// display name shown when $FLEET_VAR_HOST_PLATFORM is resolved in a host name
// template. Host name templates only apply to Apple devices, so only the Apple
// platforms are mapped here; it mirrors the frontend's
// APPLE_PLATFORM_DISPLAY_NAMES. Any other platform resolves to its raw value.
var hostNameTemplatePlatformDisplayNames = map[string]string{
	"darwin": "macOS",
	"ios":    "iOS",
	"ipados": "iPadOS",
}

// ResolveHostNameTemplate substitutes the host-identity built-in variables
// ($FLEET_VAR_HOST_HARDWARE_SERIAL, _HOST_UUID, _HOST_PLATFORM) with the host's
// values. IdP end-user variables are left untouched — they require a datastore
// lookup and are resolved separately in the service layer.
func ResolveHostNameTemplate(tmpl string, host *Host) string {
	if host == nil {
		return tmpl
	}

	platform := host.Platform
	if display, ok := hostNameTemplatePlatformDisplayNames[platform]; ok {
		platform = display
	}

	values := map[FleetVarName]string{
		FleetVarHostHardwareSerial: host.HardwareSerial,
		FleetVarHostUUID:           host.UUID,
		FleetVarHostPlatform:       platform,
	}

	return nameTemplateIdentityVarRegexp.ReplaceAllStringFunc(tmpl, func(match string) string {
		groups := nameTemplateIdentityVarRegexp.FindStringSubmatch(match)
		// Exactly one of the two capture groups (unbraced/braced) is populated.
		name := groups[1]
		if name == "" {
			name = groups[2]
		}
		return values[FleetVarName(name)]
	})
}
