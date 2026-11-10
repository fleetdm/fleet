package removednames

import (
	"fmt"
	"slices"
	"strings"
)

// GuideURL is the Fleet 5 migration guide that every removal message links to.
const GuideURL = "https://fleetdm.com/learn-more-about/fleet-5-migration"

// pathPrefix is the start of every KindPath entry. Entries keep it so they are full paths, and messages for removed endpoints drop it.
const pathPrefix = "/api/_version_/fleet"

// Kind is the kind of thing a removed name was.
type Kind string

const (
	KindPath          Kind = "path"           // URL path template, starting with /api/_version_/fleet
	KindQueryParam    Kind = "query_param"    // URL query parameter
	KindJSONKey       Kind = "json_key"       // key in a JSON request or response body
	KindFormField     Kind = "form_field"     // multipart form field
	KindCSVColumn     Kind = "csv_column"     // column header of a CSV export
	KindSortKey       Kind = "sort_key"       // value of an order_key parameter
	KindTargetType    Kind = "target_type"    // value of a target type
	KindFlag          Kind = "flag"           // fleetctl flag, without the leading dashes
	KindFlagValue     Kind = "flag_value"     // fleetctl flag and value, verbatim
	KindEnvVar        Kind = "env_var"        // environment variable read by fleetctl
	KindCommand       Kind = "command"        // fleetctl command, verbatim
	KindYAMLKey       Kind = "yaml_key"       // GitOps or apply YAML key path, dot notation, [] for array elements
	KindSpecKind      Kind = "spec_kind"      // kind of a fleetctl apply spec
	KindFileName      Kind = "file_name"      // name of a GitOps file, without its directory
	KindYAMLValue     Kind = "yaml_value"     // value of a YAML key, written "key: value" and compared without regard to case
	KindActivityType  Kind = "activity_type"  // activity type name
	KindSAMLAttribute Kind = "saml_attribute" // SAML attribute an identity provider sends, written with <ID> where a suffix varies
)

// Scope says where a removed name appears, so each consumer loads only the entries it needs.
type Scope uint8

const (
	// ScopeRequest is a name a client sends (paths, parameters, request bodies, flags, YAML).
	ScopeRequest Scope = 1 << iota
	// ScopeResponse is a name the server returns (response bodies, CSV columns).
	ScopeResponse
	// ScopeActivityDetails is a key inside the stored details of activities.
	ScopeActivityDetails
)

// Entry is one removed name and, when there is one, its replacement.
type Entry struct {
	Kind  Kind
	Scope Scope
	Old   string
	// New is the replacement. It is empty for a name that was removed without one.
	New string
	// Context narrows the entry to one fleetctl command (for example "fleetctl get hosts") or, for a path, to one HTTP method.
	// Empty means it applies everywhere.
	Context string
	// Format builds the error message. Nil means the default for the Kind: EndpointFormat for a path, ParameterFormat for a query
	// parameter, form field, JSON key, sort key, or target type, KeyFormat for a YAML key, NounFormat for a flag, command, environment
	// variable, or flag value, and ReplacedFormat for the rest.
	Format Format
}

// Message returns the error message for the removed name, for example
// `The "team_id" parameter is not available anymore. What to use instead: https://fleetdm.com/learn-more-about/fleet-5-migration`.
func (e Entry) Message() string {
	if e.Format == nil {
		return defaultFormat(e.Kind).Message(e)
	}
	return e.Format.Message(e)
}

// Message formats the removal message for an arbitrary old name and its replacement, which may be empty.
func Message(oldName, newName string) string {
	if newName == "" {
		return fmt.Sprintf("%s was removed in Fleet 5. See %s", oldName, GuideURL)
	}
	return fmt.Sprintf("%s was removed in Fleet 5. Use %s instead. See %s", oldName, newName, GuideURL)
}

// All returns a copy of every entry.
func All() []Entry {
	return slices.Clone(entries)
}

// Filter returns the entries of the given kind that apply in at least one of the given scopes. A zero scope matches every scope.
func Filter(kind Kind, scope Scope) []Entry {
	var out []Entry
	for _, e := range entries {
		if e.Kind == kind && (scope == 0 || e.Scope&scope != 0) {
			out = append(out, e)
		}
	}
	return out
}

// Find returns the entry for a removed name. An entry with an empty Context matches every context; otherwise the context must match.
func Find(kind Kind, oldName, context string) (Entry, bool) {
	return findIn(entries, kind, oldName, context)
}

// FindSAMLAttribute returns the entry for a removed SAML attribute. A KindSAMLAttribute entry is written with a trailing <ID> for the
// part that varies (FLEET_JIT_USER_ROLE_TEAM_<ID>), so it matches any attribute name that starts with the text before it.
func FindSAMLAttribute(name string) (Entry, bool) {
	for _, e := range entries {
		if prefix, ok := strings.CutSuffix(e.Old, "<ID>"); e.Kind == KindSAMLAttribute && ok && strings.HasPrefix(name, prefix) {
			return e, true
		}
	}
	return Entry{}, false
}

// sameName reports whether name is the old name of e. A KindYAMLValue entry is written "key: value": its value is compared without
// regard to case and its key exactly, because YAML keys are case-sensitive.
func sameName(e Entry, name string) bool {
	if e.Kind != KindYAMLValue {
		return e.Old == name
	}
	key, value, ok := strings.Cut(e.Old, ": ")
	nameKey, nameValue, nameOk := strings.Cut(name, ": ")
	return ok && nameOk && key == nameKey && strings.EqualFold(value, nameValue)
}

func findIn(list []Entry, kind Kind, oldName, context string) (Entry, bool) {
	for _, e := range list {
		if e.Kind == kind && sameName(e, oldName) && (e.Context == "" || e.Context == context) {
			return e, true
		}
	}
	return Entry{}, false
}
