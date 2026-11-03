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
	KindPath         Kind = "path"          // URL path template, starting with /api/_version_/fleet
	KindQueryParam   Kind = "query_param"   // URL query parameter
	KindJSONKey      Kind = "json_key"      // key in a JSON request or response body
	KindFormField    Kind = "form_field"    // multipart form field
	KindCSVColumn    Kind = "csv_column"    // column header of a CSV export
	KindSortKey      Kind = "sort_key"      // value of an order_key parameter
	KindTargetType   Kind = "target_type"   // value of a target type
	KindFlag         Kind = "flag"          // fleetctl flag, without the leading dashes
	KindFlagValue    Kind = "flag_value"    // fleetctl flag and value, verbatim
	KindEnvVar       Kind = "env_var"       // environment variable read by fleetctl
	KindCommand      Kind = "command"       // fleetctl command, verbatim
	KindYAMLKey      Kind = "yaml_key"      // GitOps or apply YAML key path, dot notation, [] for array elements
	KindSpecKind     Kind = "spec_kind"     // kind of a fleetctl apply spec
	KindActivityType Kind = "activity_type" // activity type name
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
	// Format builds the error message. Nil means ReplacedFormat.
	Format Format
}

// Message returns the error message for the removed name, for example
// "team_id was removed in Fleet 5. Use fleet_id instead. See https://fleetdm.com/learn-more-about/fleet-5-migration".
func (e Entry) Message() string {
	if e.Format == nil {
		return ReplacedFormat{}.Message(e)
	}
	return e.Format.Message(e)
}

// MessageForVersion is Message for a request to a removed path, with the API version the request used (for example "v1" or "latest")
// in place of the _version_ segment of the paths in the message.
func (e Entry) MessageForVersion(version string) string {
	return strings.ReplaceAll(e.Message(), "/_version_/", "/"+version+"/")
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

func findIn(list []Entry, kind Kind, oldName, context string) (Entry, bool) {
	for _, e := range list {
		if e.Kind == kind && e.Old == oldName && (e.Context == "" || e.Context == context) {
			return e, true
		}
	}
	return Entry{}, false
}
