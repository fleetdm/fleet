package fleet

import (
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

// fleetVarsSupportedInHostNameTemplates is the allow-list of Fleet variables that
// may be used in a host name template.
var fleetVarsSupportedInHostNameTemplates = []FleetVarName{
	FleetVarHostHardwareSerial,
	FleetVarHostUUID,
	FleetVarHostPlatform,
}

// nameTemplateVarRegexp matches every supported name-template variable in both
// its $FLEET_VAR_NAME and ${FLEET_VAR_NAME} forms, in a single pattern. It is
// derived from fleetVarsSupportedInHostNameTemplates so it stays in sync with the
// allow-list. The unbraced form uses a trailing word boundary so that an
// unsupported longer name (e.g. HOST_UUID_EXTRA) is not partially matched.
var nameTemplateVarRegexp = func() *regexp.Regexp {
	alts := make([]string, len(fleetVarsSupportedInHostNameTemplates))
	for i, v := range fleetVarsSupportedInHostNameTemplates {
		alts[i] = regexp.QuoteMeta(string(v))
	}
	alt := strings.Join(alts, "|")
	return regexp.MustCompile(fmt.Sprintf(`\$FLEET_VAR_(%[1]s)\b|\$\{FLEET_VAR_(%[1]s)\}`, alt))
}()

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

	// Secret variables ($FLEET_SECRET_*) are never allowed in name templates.
	if len(ContainsPrefixVars(tmpl, ServerSecretPrefix)) > 0 {
		return "", NewInvalidArgumentError("name_template", "Secret variables aren't supported in host name templates.")
	}

	// Every Fleet variable used must be in the allow-list.
	for _, v := range variables.Find(tmpl) {
		if !slices.Contains(fleetVarsSupportedInHostNameTemplates, FleetVarName(v)) {
			return "", NewInvalidArgumentError("name_template",
				"Fleet variable $FLEET_VAR_"+v+" is not supported in host name templates.")
		}
	}

	// The resolved name must fit Apple's device name limit. Stripping the
	// variables yields the shortest a resolved name can be (a variable may
	// resolve to an empty value), so if the fixed text alone exceeds the limit no
	// host can ever get a valid name — reject it now rather than silently failing
	// every host at resolve time. Per-host overflow from variable expansion is
	// still caught by the cron when it resolves against a host's actual values.
	if literal := nameTemplateVarRegexp.ReplaceAllString(tmpl, ""); len(literal) > MaxResolvedHostNameBytes {
		return "", NewInvalidArgumentError("name_template",
			fmt.Sprintf("Host name template's fixed text can't be longer than %d bytes (the device name limit).", MaxResolvedHostNameBytes))
	}

	return tmpl, nil
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

// ResolveHostNameTemplate resolves a host name template against a host by
// substituting the supported host-identity Fleet variables with the host's
// values.
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

	return nameTemplateVarRegexp.ReplaceAllStringFunc(tmpl, func(match string) string {
		groups := nameTemplateVarRegexp.FindStringSubmatch(match)
		// Exactly one of the two capture groups (unbraced/braced) is populated.
		name := groups[1]
		if name == "" {
			name = groups[2]
		}
		return values[FleetVarName(name)]
	})
}
