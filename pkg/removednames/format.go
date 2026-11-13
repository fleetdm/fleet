package removednames

import (
	"fmt"
	"strings"
)

// Format builds the error message for a removed name. Every message links to the migration guide.
type Format interface {
	Message(e Entry) string
}

// ParameterFormat is for a removed or renamed request parameter, form field, or body key. The replacement is left to the guide.
//
//	The "team_id" parameter is not available anymore. What to use instead: <guide>
type ParameterFormat struct{}

// Message implements Format.
func (ParameterFormat) Message(e Entry) string {
	return fmt.Sprintf(`The "%s" parameter is not available anymore. What to use instead: %s`, e.Old, GuideURL)
}

// KeyFormat is for a removed or renamed key in a YAML file. The replacement is left to the guide.
//
//	The "team_settings" key is not available anymore. What to use instead: <guide>
type KeyFormat struct{}

// Message implements Format.
func (KeyFormat) Message(e Entry) string {
	return fmt.Sprintf(`The "%s" key is not available anymore. What to use instead: %s`, e.Old, GuideURL)
}

// NounFormat is for a removed fleetctl flag, command, environment variable, or flag value, a SAML attribute, or a Fleet variable. Noun
// says which one it is. A flag is shown with its leading dashes. The replacement is left to the guide.
//
//	The "--team" flag is not available anymore. What to use instead: <guide>
//	The "fleetctl get teams" command is not available anymore. What to use instead: <guide>
//	The "FLEET_JIT_USER_ROLE_TEAM_<ID>" SAML attribute is not available anymore. What to use instead: <guide>
//	The "$FLEET_VAR_HOST_END_USER_EMAIL_IDP" variable is not available anymore. What to use instead: <guide>
type NounFormat struct {
	Noun string
}

// Message implements Format.
func (f NounFormat) Message(e Entry) string {
	return fmt.Sprintf(`The "%s" %s is not available anymore. What to use instead: %s`, displayName(e.Kind, e.Old), f.Noun, GuideURL)
}

// ReplacedFormat names the replacement, or just the removal when the entry has no replacement. It is the default for the kinds that
// the error spec does not cover yet (CSV columns, spec kinds, file names, YAML values, and activity types).
//
//	team_id was removed in Fleet 5. Use fleet_id instead. See <guide>
//	team_id was removed in Fleet 5. See <guide>
type ReplacedFormat struct{}

// Message implements Format.
func (ReplacedFormat) Message(e Entry) string {
	return Message(displayName(e.Kind, e.Old), displayName(e.Kind, e.New))
}

// EndpointFormat is for a removed endpoint (KindPath). It shows the HTTP method from the entry's Context, when set, and leaves the
// replacement to the guide.
//
//	The "POST /mdm/apple/profiles/match" endpoint is not available anymore. What to use instead: <guide>
type EndpointFormat struct {
	// ByDefault is for an endpoint that a setting can turn back on.
	ByDefault bool
}

// Message implements Format.
func (f EndpointFormat) Message(e Entry) string {
	endpoint := strings.TrimPrefix(e.Old, pathPrefix)
	if e.Context != "" {
		endpoint = e.Context + " " + endpoint
	}
	byDefault := ""
	if f.ByDefault {
		byDefault = " by default"
	}
	return fmt.Sprintf(`The "%s" endpoint is not available anymore%s. What to use instead: %s`, endpoint, byDefault, GuideURL)
}

// defaultFormat is the format of an entry that does not set one, which follows the Fleet 5 error spec: one wording for endpoints,
// parameters, YAML keys, and fleetctl flags, commands, environment variables, and values.
func defaultFormat(kind Kind) Format {
	switch kind {
	case KindPath:
		return EndpointFormat{}
	case KindQueryParam, KindFormField, KindJSONKey, KindSortKey, KindTargetType:
		return ParameterFormat{}
	case KindYAMLKey:
		return KeyFormat{}
	case KindFlag:
		return NounFormat{Noun: "flag"}
	case KindCommand:
		return NounFormat{Noun: "command"}
	case KindEnvVar:
		return NounFormat{Noun: "environment variable"}
	case KindFlagValue:
		return NounFormat{Noun: "value"}
	case KindSAMLAttribute:
		return NounFormat{Noun: "SAML attribute"}
	case KindFleetVariable:
		return NounFormat{Noun: "variable"}
	default:
		return ReplacedFormat{}
	}
}

// displayName is how a name appears in a message. Only flags are displayed differently: they get their leading dashes, one for a
// single-letter flag such as -t and two for the rest.
func displayName(kind Kind, name string) string {
	switch {
	case kind != KindFlag || name == "":
		return name
	case len(name) == 1:
		return "-" + name
	default:
		return "--" + name
	}
}
